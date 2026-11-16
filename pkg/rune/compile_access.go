package rune

import (
	"rune/pkg/ast"
	"rune/pkg/value"
	"slices"
)

// nilJump makes the optional chain being compiled nil if R[r] is.
func (fs *funcState) nilJump(r int32) {
	fs.chain = append(fs.chain, fs.emit(ins{op: opJmpNil, a: r, y: 1}, nil))
}

func (fs *funcState) optional(n *ast.OptionalExpr, dst int32) {
	outer := fs.chain
	fs.chain = nil

	fs.expr(n.Expr, dst)

	jumps := fs.chain
	fs.chain = outer

	end := fs.emit(ins{op: opJmp}, nil)
	fs.patch(jumps)
	fs.emit(ins{op: opLoadK, a: dst, c: fs.constant(value.NilValue)}, nil)
	fs.patch([]int{end})
}

func (fs *funcState) index(n *ast.IndexExpr, dst int32) {
	if n.Export != nil {
		fs.varExpr(n.Export, dst)
		return
	}

	if n.Member != nil {
		fs.emit(ins{op: opLoadK, a: dst, c: fs.constant(*n.Member)}, nil)
		return
	}

	if n.Property {
		t := fs.exprAny(n.Array)
		if n.Optional {
			fs.nilJump(t)
		}

		fs.emit(ins{op: pick(n.Optional, opGetPropOpt, opGetProp), a: dst, b: t, c: fs.constant(value.NewString(n.Token.Lexeme))}, &n.Token)

		return
	}

	t := fs.operand(n.Array, n.Index)
	if n.Optional {
		fs.nilJump(t)
	}

	i := fs.exprAny(n.Index)
	fs.emit(ins{op: pick(n.Optional, opGetIndexOpt, opGetIndex), a: dst, b: t, c: i}, &n.Token)
}

// in compiles x in xs; a range is tested without making its array.
func (fs *funcState) in(n *ast.BinaryExpr, dst int32) {
	negate := uint8(boolNum(n.Operator.TokenType == ast.BANG_IN))

	x := fs.operand(n.Left, n.Right)

	if r, ok := unwrap(n.Right).(*ast.RangeExpr); ok {
		fs.emit(ins{op: opInRange, a: dst, b: x, c: fs.rangeParts(r), x: uint8(boolNum(r.Inclusive)), y: negate}, &n.Operator)
		return
	}

	fs.emit(ins{op: opIn, a: dst, b: x, c: fs.exprAny(n.Right), y: negate}, &n.Operator)
}

func (fs *funcState) slice(n *ast.SliceExpr, dst int32) {
	r := n.Range

	t := fs.operand(n.Target, r.Start, r.End)
	if n.Optional {
		fs.nilJump(t)
	}

	base := fs.alloc()
	fs.alloc()

	flags := uint8(0)
	if r.Inclusive {
		flags |= sliceInclusive
	}

	if r.Start != nil {
		fs.expr(r.Start, base)
		flags |= sliceStart
	}

	if r.End != nil {
		fs.expr(r.End, base+1)
		flags |= sliceEnd
	}

	fs.emit(ins{op: opSlice, a: dst, b: t, c: base, x: flags}, &r.Operator)
}

// setIndex returns the register holding the assigned value.
func (fs *funcState) setIndex(n *ast.SetIndexExpr) int32 {
	pl := fs.place(n.Array, n.Index, n.Property, &n.Token, n.Value)

	if n.Op != nil || !fs.simple(n.Value) {
		pl.check()
	}

	if n.Op == nil {
		var v int32
		if sameRoot(n.Value, n.Array) {
			// The write puts a copy in the variable, which v must not be.
			v = fs.alloc()
			fs.expr(n.Value, v)
		} else {
			v = fs.exprAny(n.Value)
		}

		fs.share(n.Value, v)
		pl.store(v, quiet(n))

		return v
	}

	// The old value is read before the right-hand side runs, as in
	// "a[i] = a[i] op v".
	old := fs.alloc()
	pl.load(old)

	v := fs.exprAny(n.Value)
	fs.emit(binaryIns(n.Op.TokenType, old, old, v), n.Op)
	pl.store(old, quiet(n))

	return old
}

// simple reports whether e is a literal or a plain local.
func (fs *funcState) simple(e ast.Expr) bool {
	switch n := unwrap(e).(type) {
	case *ast.LiteralExpr:
		return true
	case *ast.VarExpr:
		if isConst(n) {
			return true
		}

		kind, _ := fs.lookup(n.Local)
		return kind == varLocal
	}

	return false
}

func (fs *funcState) update(n *ast.UpdateExpr, dst int32) {
	op := &n.Operator
	incr := ins{op: opIncr, x: uint8(boolNum(op.TokenType == ast.MINUS_MINUS))}

	if t, ok := n.Target.(*ast.VarExpr); ok {
		kind, r := fs.lookup(t.Local)

		if kind == varLocal {
			fs.checkDefined(t.Local, r, &t.Name)

			if !n.Prefix {
				fs.move(dst, r)
			}

			incr.a, incr.b = r, r
			fs.emit(incr, op)

			if n.Prefix {
				fs.move(dst, r)
			}

			return
		}

		old := fs.alloc()
		fs.varExpr(t, old)

		v := old
		if !n.Prefix {
			v = fs.alloc()
		}

		incr.a, incr.b = v, old
		fs.emit(incr, op)
		fs.store(kind, r, v, false, &t.Name)
		fs.move(dst, old)

		return
	}

	target := n.Target.(*ast.IndexExpr)
	pl := fs.place(target.Array, target.Index, target.Property, &target.Token)
	pl.check()

	old := fs.alloc()
	pl.load(old)

	v := old
	if !n.Prefix {
		v = fs.alloc()
	}

	incr.a, incr.b = v, old
	fs.emit(incr, op)
	pl.store(v, reads(target))
	fs.move(dst, old)
}

// quiet reports whether the write n shares nothing between loading the
// path it writes and writing: its indices and value only read, and the
// value is no part of the variable written.
func quiet(n *ast.SetIndexExpr) bool {
	return reads(n.Array) && reads(n.Index) && reads(n.Value) && !sameRoot(n.Value, n.Array)
}

// reads reports whether e only reads: no call, assignment or container
// it makes holds anything.
func reads(e ast.Expr) bool {
	switch n := e.(type) {
	case nil, *ast.LiteralExpr, *ast.VarExpr:
		return true
	case *ast.GroupingExpr:
		return reads(n.Expr)
	case *ast.UnaryExpr:
		return reads(n.Right)
	case *ast.BinaryExpr:
		return reads(n.Left) && reads(n.Right)
	case *ast.LogicalExpr:
		return reads(n.Left) && reads(n.Right)
	case *ast.ConditionalExpr:
		return reads(n.Cond) && reads(n.Then) && reads(n.Else)
	case *ast.IndexExpr:
		return reads(n.Array) && reads(n.Index)
	case *ast.InterpolationExpr:
		return !slices.ContainsFunc(n.Parts, func(p ast.Expr) bool { return !reads(p) })
	}

	return false
}
