package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/helpers"
	"rune/pkg/value"
)

var arithOps = map[ast.TokenType]Op{
	ast.PLUS: opAdd, ast.MINUS: opSub, ast.STAR: opMul, ast.SLASH: opDiv, ast.PERCENT: opMod,
}

var cmpOps = map[ast.TokenType]uint8{
	ast.LESS: cmpLT, ast.LESS_EQUAL: cmpLE, ast.GREATER: cmpGT, ast.GREATER_EQUAL: cmpGE,
	ast.EQUAL_EQUAL: cmpEQ, ast.BANG_EQUAL: cmpNE,
}

// exprAny returns a register holding e: a plain local's own register, or
// a new temporary.
func (fs *funcState) exprAny(e ast.Expr) int32 {
	switch n := unwrap(e).(type) {
	case *ast.VarExpr:
		if kind, r := fs.lookup(n.Local); kind == varLocal {
			fs.checkDefined(n.Local, r, &n.Name)
			return r
		}
	case *ast.AssignExpr:
		return fs.assign(n)
	case *ast.SetIndexExpr:
		return fs.setIndex(n)
	}

	r := fs.alloc()
	fs.expr(e, r)

	return r
}

// operand is exprAny for an operand evaluated before others: a local
// they may assign to is copied first.
func (fs *funcState) operand(e ast.Expr, later ...ast.Expr) int32 {
	r := fs.exprAny(e)
	if r >= fs.nactive {
		return r
	}

	for _, l := range later {
		if assigns(l) {
			t := fs.alloc()
			fs.emit(ins{op: opMove, a: t, b: r}, nil)

			return t
		}
	}

	return r
}

// assigns reports whether e may contain an assignment to a variable.
func assigns(e ast.Expr) bool {
	switch n := e.(type) {
	case *ast.LiteralExpr, *ast.VarExpr, *ast.FunctionExpr:
		return false
	case *ast.AssignExpr:
		return true
	case *ast.UpdateExpr:
		_, ok := n.Target.(*ast.VarExpr)
		return ok || assigns(n.Target)
	case *ast.GroupingExpr:
		return assigns(n.Expr)
	case *ast.BinaryExpr:
		return assigns(n.Left) || assigns(n.Right)
	case *ast.LogicalExpr:
		return assigns(n.Left) || assigns(n.Right)
	case *ast.UnaryExpr:
		return assigns(n.Right)
	case *ast.CallExpr:
		return assigns(n.Callee) || anyAssigns(n.Args)
	case *ast.ArrayExpr:
		return anyAssigns(n.Items)
	case *ast.ObjectExpr:
		for _, p := range n.Pairs {
			if assigns(p.Value) {
				return true
			}
		}

		return false
	case *ast.StructExpr:
		for _, p := range n.Pairs {
			if assigns(p.Value) {
				return true
			}
		}

		return false
	case *ast.IndexExpr:
		return assigns(n.Array) || assigns(n.Index)
	case *ast.SetIndexExpr:
		return assigns(n.Array) || assigns(n.Index) || assigns(n.Value)
	case *ast.RangeExpr:
		return assigns(n.Start) || assigns(n.End) || n.Step != nil && assigns(n.Step)
	}

	return true
}

func anyAssigns(es []ast.Expr) bool {
	for _, e := range es {
		if assigns(e) {
			return true
		}
	}

	return false
}

func unwrap(e ast.Expr) ast.Expr {
	for {
		g, ok := e.(*ast.GroupingExpr)
		if !ok {
			return e
		}

		e = g.Expr
	}
}

// literalOf returns the value of a literal, a negated number included.
func literalOf(e ast.Expr) (value.Value, bool) {
	switch n := unwrap(e).(type) {
	case *ast.LiteralExpr:
		return n.Value, true
	case *ast.UnaryExpr:
		if v, ok := literalOf(n.Right); ok && n.Operator.TokenType == ast.MINUS && v.IsNumber() {
			return value.NewNumber(-v.Num()), true
		}
	}

	return value.NilValue, false
}

// noReg is the destination of a value nobody reads.
const noReg = -1

func (fs *funcState) move(dst, src int32) {
	if dst != src && dst != noReg {
		fs.emit(ins{op: opMove, a: dst, b: src}, nil)
	}
}

// expr compiles e into dst. dst is written last, after every read, except
// by the expressions writesEarly lists.
func (fs *funcState) expr(e ast.Expr, dst int32) {
	free := fs.free
	defer func() { fs.free = max(free, dst+1) }()

	switch n := e.(type) {
	case *ast.LiteralExpr:
		fs.emit(ins{op: opLoadK, a: dst, c: fs.constant(n.Value)}, nil)
	case *ast.GroupingExpr:
		fs.expr(n.Expr, dst)
	case *ast.VarExpr:
		fs.varExpr(n, dst)
	case *ast.AssignExpr:
		fs.move(dst, fs.assign(n))
	case *ast.BinaryExpr:
		fs.binary(n, dst)
	case *ast.LogicalExpr:
		fs.expr(n.Left, dst)
		j := fs.emit(ins{op: opJmpIf, a: dst, y: uint8(boolNum(n.Op.TokenType == ast.OR))}, nil)
		fs.expr(n.Right, dst)
		fs.patch([]int{j})
	case *ast.UnaryExpr:
		fs.unary(n, dst)
	case *ast.CallExpr:
		fs.call(n, dst)
	case *ast.ArrayExpr:
		fs.literal(dst, len(n.Items), func(i int) ast.Expr { return n.Items[i] }, func(from, to int, base int32) ins {
			return ins{op: opArray, b: base, c: int32(to - from)}
		})
	case *ast.ObjectExpr:
		fs.literal(dst, len(n.Pairs), func(i int) ast.Expr { return n.Pairs[i].Value }, func(from, to int, base int32) ins {
			keys := make([]string, 0, to-from)
			for _, pair := range n.Pairs[from:to] {
				keys = append(keys, pair.Key)
			}

			fs.proto.keys = append(fs.proto.keys, keys)

			return ins{op: opObject, b: base, c: int32(len(fs.proto.keys) - 1)}
		})
	case *ast.StructExpr:
		fs.structLiteral(n, dst)
	case *ast.IndexExpr:
		fs.index(n, dst)
	case *ast.SetIndexExpr:
		fs.move(dst, fs.setIndex(n))
	case *ast.UpdateExpr:
		fs.update(n, dst)
	case *ast.RangeExpr:
		base := fs.rangeParts(n)
		fs.emit(ins{op: opRange, a: dst, b: base, x: uint8(boolNum(n.Inclusive))}, &n.Operator)
	case *ast.FunctionExpr:
		fs.emit(ins{op: opClosure, a: dst, c: fs.function(n.Decl)}, nil)
	default:
		panic(fmt.Sprintf("compile: unknown expression %T", e))
	}
}

// literalChunk caps the registers an array or object literal takes; a
// longer one is built a chunk at a time, each adding to the first (X).
const literalChunk = 256

func (fs *funcState) literal(dst int32, n int, item func(i int) ast.Expr, build func(from, to int, base int32) ins) {
	target := dst
	if n > literalChunk {
		target = fs.alloc()
	}

	for from := 0; from == 0 || from < n; from += literalChunk {
		to := min(from+literalChunk, n)
		base := fs.free

		for i := from; i < to; i++ {
			fs.expr(item(i), fs.alloc())
		}

		in := build(from, to, base)
		in.a, in.x = target, uint8(boolNum(from > 0))
		fs.emit(in, nil)
		fs.free = base
	}

	fs.move(dst, target)
}

// structLiteral evaluates the fields in source order into registers in
// slot order, defaults filling the rest.
func (fs *funcState) structLiteral(n *ast.StructExpr, dst int32) {
	base := fs.free
	given := make([]bool, len(n.Decl.Fields))

	for range n.Decl.Fields {
		fs.alloc()
	}

	for _, pair := range n.Pairs {
		given[pair.Slot] = true
		fs.expr(pair.Value, base+int32(pair.Slot))
	}

	for i, f := range n.Decl.Fields {
		if !given[i] {
			fs.expr(f.Default, base+int32(i))
		}
	}

	fs.emit(ins{op: opStruct, a: dst, b: base, c: fs.constant(value.NewType(fs.unit.shapes[n.Decl]))}, nil)
}

// writesEarly reports whether compiling e into a register writes it
// before e is fully evaluated, so it must not target a live variable.
func writesEarly(e ast.Expr) bool {
	switch n := unwrap(e).(type) {
	case *ast.LogicalExpr, *ast.AssignExpr, *ast.SetIndexExpr:
		return true
	case *ast.UpdateExpr:
		return !n.Prefix
	}

	return false
}

func (fs *funcState) checkDefined(l *ast.Local, r int32, name *ast.Token) {
	if fs.checked(l) {
		fs.emit(ins{op: opCheckDefined, a: r}, name)
	}
}

func (fs *funcState) varExpr(n *ast.VarExpr, dst int32) {
	switch kind, r := fs.lookup(n.Local); kind {
	case varLocal:
		fs.checkDefined(n.Local, r, &n.Name)
		fs.move(dst, r)
	case varCell:
		fs.emit(ins{op: opGetCell, a: dst, b: r}, &n.Name)
	case varUpval:
		fs.emit(ins{op: opGetUpval, a: dst, b: r}, &n.Name)
	default:
		fs.emit(ins{op: opUndefined}, &n.Name)
	}
}

// store writes v to the variable that lookup placed at kind and r,
// checking it is defined if checked.
func (fs *funcState) store(kind varKind, r, v int32, checked bool, name *ast.Token) {
	switch kind {
	case varLocal:
		if checked {
			fs.emit(ins{op: opCheckDefined, a: r}, name)
		}

		fs.move(r, v)
	case varCell:
		fs.emit(ins{op: pick(checked, opSetCellChecked, opSetCell), a: r, b: v}, name)
	case varUpval:
		fs.emit(ins{op: pick(checked, opSetUpvalChecked, opSetUpval), a: v, b: r}, name)
	default:
		fs.emit(ins{op: opUndefined}, name)
	}
}

func (fs *funcState) assignLocal(n *ast.AssignExpr, r int32) {
	if !writesEarly(n.Value) {
		fs.expr(n.Value, r)
		return
	}

	t := fs.alloc()
	fs.expr(n.Value, t)
	fs.move(r, t)
}

// assign returns the register holding the assigned value.
func (fs *funcState) assign(n *ast.AssignExpr) int32 {
	kind, r := fs.lookup(n.Local)
	checked := fs.checked(n.Local)

	if kind == varLocal && !checked {
		fs.assignLocal(n, r)
		return r
	}

	v := fs.exprAny(n.Value)
	fs.store(kind, r, v, checked, &n.Name)

	if kind == varLocal {
		return r
	}

	return v
}

func (fs *funcState) binary(n *ast.BinaryExpr, dst int32) {
	op := &n.Operator

	lc, lConst := literalOf(n.Left)
	rc, rConst := literalOf(n.Right)

	// A fold that fails is left to fail at run time.
	if lConst && rConst {
		if v, err := binaryOp(op, lc, rc); err == nil {
			fs.emit(ins{op: opLoadK, a: dst, c: fs.constant(v)}, nil)
			return
		}
	}

	arith, isArith := arithOps[op.TokenType]
	cmp := cmpOps[op.TokenType]

	if k, ok := fs.operandConst(rc, rConst, maxC); ok {
		l := fs.operand(n.Left)

		if isArith {
			fs.emit(ins{op: arith + (opAddK - opAdd), a: dst, b: l, c: k}, op)
		} else {
			fs.emit(ins{op: opCmpK, x: cmp, a: dst, b: l, c: k}, op)
		}

		return
	}

	l := fs.operand(n.Left, n.Right)
	r := fs.exprAny(n.Right)

	if isArith {
		fs.emit(ins{op: arith, a: dst, b: l, c: r}, op)
	} else {
		fs.emit(ins{op: opCmp, x: cmp, a: dst, b: l, c: r}, op)
	}
}

// operandConst returns the constant index of a literal right operand
// that fits in limit.
func (fs *funcState) operandConst(v value.Value, isConst bool, limit int32) (int32, bool) {
	if !isConst {
		return 0, false
	}

	k := fs.constant(v)

	return k, k <= limit
}

func (fs *funcState) unary(n *ast.UnaryExpr, dst int32) {
	op := &n.Operator

	if v, ok := literalOf(n); ok {
		fs.emit(ins{op: opLoadK, a: dst, c: fs.constant(v)}, nil)
		return
	}

	if c, ok := literalOf(n.Right); ok && op.TokenType == ast.BANG {
		fs.emit(ins{op: opLoadK, a: dst, c: fs.constant(value.NewBool(!helpers.IsTruthy(c)))}, nil)
		return
	}

	r := fs.exprAny(n.Right)

	if op.TokenType == ast.BANG {
		fs.emit(ins{op: opNot, a: dst, b: r}, nil)
	} else {
		fs.emit(ins{op: opNeg, a: dst, b: r}, op)
	}
}

func (fs *funcState) call(n *ast.CallExpr, dst int32) {
	// The callee goes in dst when nothing above it is in use.
	base := dst
	if dst < fs.nactive || dst+1 != fs.free {
		base = fs.alloc()
	}

	fs.expr(n.Callee, base)

	for _, arg := range n.Args {
		fs.expr(arg, fs.alloc())
	}

	fs.emit(ins{op: opCall, a: base, b: int32(len(n.Args))}, &n.Token)
	fs.move(dst, base)
}

func (fs *funcState) index(n *ast.IndexExpr, dst int32) {
	if n.Property {
		t := fs.exprAny(n.Array)
		fs.emit(ins{op: opGetProp, a: dst, b: t, c: fs.constant(value.NewString(n.Token.Lexeme))}, &n.Token)

		return
	}

	t := fs.operand(n.Array, n.Index)
	i := fs.exprAny(n.Index)
	fs.emit(ins{op: opGetIndex, a: dst, b: t, c: i}, &n.Token)
}

// place is the target of a write: t[i], or for a property t.K[i].
type place struct {
	fs   *funcState
	t, i int32
	prop bool
	tok  *ast.Token
}

// place evaluates the target (and for t[i] the index) of a write that
// evaluates later before writing.
func (fs *funcState) place(array, index ast.Expr, prop bool, tok *ast.Token, later ...ast.Expr) place {
	pl := place{fs: fs, prop: prop, tok: tok}

	if prop {
		pl.t = fs.operand(array, later...)
		pl.i = fs.constant(value.NewString(tok.Lexeme))
	} else {
		pl.t = fs.operand(array, append([]ast.Expr{index}, later...)...)
		pl.i = fs.operand(index, later...)
	}

	return pl
}

// check makes a property write fail on a target that is not an object
// before more code runs.
func (pl place) check() {
	if pl.prop {
		pl.fs.emit(ins{op: opCheckProp, a: pl.t}, pl.tok)
	}
}

func (pl place) load(dst int32) {
	pl.fs.emit(ins{op: pick(pl.prop, opGetProp, opGetIndex), a: dst, b: pl.t, c: pl.i}, pl.tok)
}

func (pl place) store(v int32) {
	pl.fs.emit(ins{op: pick(pl.prop, opSetProp, opSetIndex), a: pl.t, b: pl.i, c: v}, pl.tok)
}

// setIndex returns the register holding the assigned value.
func (fs *funcState) setIndex(n *ast.SetIndexExpr) int32 {
	pl := fs.place(n.Array, n.Index, n.Property, &n.Token, n.Value)

	if n.Op != nil || !fs.simple(n.Value) {
		pl.check()
	}

	if n.Op == nil {
		v := fs.exprAny(n.Value)
		pl.store(v)

		return v
	}

	// The old value is read before the right-hand side runs, as in
	// "a[i] = a[i] op v".
	old := fs.alloc()
	pl.load(old)

	v := fs.exprAny(n.Value)
	fs.emit(ins{op: arithOps[n.Op.TokenType], a: old, b: old, c: v}, n.Op)
	pl.store(old)

	return old
}

// simple reports whether e is a literal or a plain local.
func (fs *funcState) simple(e ast.Expr) bool {
	switch n := unwrap(e).(type) {
	case *ast.LiteralExpr:
		return true
	case *ast.VarExpr:
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
	pl.store(v)
	fs.move(dst, old)
}

// jumpIf emits a test of e that jumps when its truthiness is sense and
// falls through otherwise, and returns the jumps to patch.
func (fs *funcState) jumpIf(e ast.Expr, sense bool) []int {
	free := fs.free
	defer func() { fs.free = free }()

	y := uint8(boolNum(sense))

	switch n := unwrap(e).(type) {
	case *ast.LiteralExpr:
		if helpers.IsTruthy(n.Value) == sense {
			return []int{fs.emit(ins{op: opJmp}, nil)}
		}

		return nil

	case *ast.UnaryExpr:
		if n.Operator.TokenType == ast.BANG {
			return fs.jumpIf(n.Right, !sense)
		}

	case *ast.LogicalExpr:
		// "a and b" jumps on false when either does; on true when b does,
		// once a has not jumped past it. "or" mirrors that.
		if (n.Op.TokenType == ast.OR) == sense {
			return append(fs.jumpIf(n.Left, sense), fs.jumpIf(n.Right, sense)...)
		}

		skip := fs.jumpIf(n.Left, !sense)
		out := fs.jumpIf(n.Right, sense)
		fs.patch(skip)

		return out

	case *ast.BinaryExpr:
		cmp, ok := cmpOps[n.Operator.TokenType]
		if !ok {
			break
		}

		_, lConst := literalOf(n.Left)
		rc, rConst := literalOf(n.Right)

		if lConst && rConst {
			break
		}

		op := &n.Operator

		if k, ok := fs.operandConst(rc, rConst, maxAB); ok {
			l := fs.operand(n.Left)
			return []int{fs.emit(ins{op: opJmpCmpK, x: cmp, y: y, a: l, b: k}, op)}
		}

		l := fs.operand(n.Left, n.Right)
		r := fs.exprAny(n.Right)

		return []int{fs.emit(ins{op: opJmpCmp, x: cmp, y: y, a: l, b: r}, op)}
	}

	r := fs.exprAny(e)

	return []int{fs.emit(ins{op: opJmpIf, a: r, y: y}, nil)}
}

func pick(cond bool, a, b Op) Op {
	if cond {
		return a
	}

	return b
}
