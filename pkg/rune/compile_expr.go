package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
)

var cmpOps = map[ast.TokenType]uint8{
	ast.LESS: cmpLT, ast.LESS_EQUAL: cmpLE, ast.GREATER: cmpGT, ast.GREATER_EQUAL: cmpGE,
	ast.EQUAL_EQUAL: cmpEQ, ast.BANG_EQUAL: cmpNE,
}

// flipped is each comparison with its operands swapped.
var flipped = [...]uint8{cmpLT: cmpGT, cmpLE: cmpGE, cmpGT: cmpLT, cmpGE: cmpLE, cmpEQ: cmpEQ, cmpNE: cmpNE}

// exprAny returns a register holding e: a plain local's own register, or
// a new temporary.
func (fs *funcState) exprAny(e ast.Expr) int32 {
	switch n := unwrap(e).(type) {
	case *ast.VarExpr:
		if isConst(n) {
			break
		}

		if kind, r := fs.lookup(n.Local); kind == varLocal {
			fs.checkDefined(n.Local, r, &n.Name)
			fs.leak(n)

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
	case *ast.GroupingExpr:
		return assigns(n.Expr)
	case *ast.OptionalExpr:
		return assigns(n.Expr)
	case *ast.SpreadExpr:
		return assigns(n.Expr)
	case *ast.BinaryExpr:
		return assigns(n.Left) || assigns(n.Right)
	case *ast.LogicalExpr:
		return assigns(n.Left) || assigns(n.Right)
	case *ast.ConditionalExpr:
		return assigns(n.Cond) || assigns(n.Then) || assigns(n.Else)
	case *ast.ThrowExpr:
		return assigns(n.Value)
	case *ast.UnaryExpr:
		return assigns(n.Right)
	case *ast.CallExpr:
		return n.Writable || assigns(n.Callee) || anyAssigns(n.Args)
	case *ast.ArrayExpr:
		return anyAssigns(n.Items)
	case *ast.InterpolationExpr:
		return anyAssigns(n.Parts)
	case *ast.ObjectExpr:
		for _, p := range n.Pairs {
			if assigns(p.Value) {
				return true
			}
		}

		return false
	case *ast.StructExpr:
		return pairsAssign(n.Pairs)
	case *ast.IsExpr:
		return assigns(n.Value) || assigns(n.Type)
	case *ast.IndexExpr:
		return assigns(n.Array) || assigns(n.Index)
	case *ast.RangeExpr:
		return assigns(n.Start) || assigns(n.End) || n.Step != nil && assigns(n.Step)
	}

	return true
}

func pairsAssign(pairs []ast.StructPair) bool {
	for _, p := range pairs {
		if assigns(p.Value) {
			return true
		}
	}

	return false
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

// isConst reports whether n names a constant, which has no register.
func isConst(n *ast.VarExpr) bool { return n.Local != nil && n.Local.IsConst }

// literalOf returns the value of a literal, or of operators on literals
// that fold without an error.
func literalOf(e ast.Expr) (value.Value, bool) {
	switch n := unwrap(e).(type) {
	case *ast.LiteralExpr:
		return n.Value, true
	case *ast.VarExpr:
		if isConst(n) {
			return n.Local.Const, true
		}
	case *ast.IndexExpr:
		if n.Member != nil && n.Member.IsNumber() {
			return *n.Member, true
		}

		if n.Export != nil {
			return literalOf(n.Export)
		}
	case *ast.UnaryExpr:
		v, ok := literalOf(n.Right)
		if !ok {
			break
		}

		switch n.Operator.TokenType {
		case ast.BANG:
			return value.NewBool(!helpers.IsTruthy(v)), true
		case ast.MINUS:
			v, err := negate(&n.Operator, v)
			return v, err == nil
		case ast.TILDE:
			v, err := bitNot(&n.Operator, v)
			return v, err == nil
		}
	case *ast.BinaryExpr:
		l, ok := literalOf(n.Left)
		if !ok {
			break
		}

		if r, ok := literalOf(n.Right); ok {
			v, err := binaryOp(&n.Operator, l, r)
			return v, err == nil
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

		var j int
		if n.Op.TokenType == ast.QUESTION_QUESTION {
			j = fs.emit(ins{op: opJmpNil, a: dst}, nil)
		} else {
			j = fs.emit(ins{op: opJmpIf, a: dst, y: uint8(boolNum(n.Op.TokenType == ast.OR))}, nil)
		}

		fs.expr(n.Right, dst)
		fs.patch([]int{j})
	case *ast.OptionalExpr:
		fs.optional(n, dst)
	case *ast.UnaryExpr:
		fs.unary(n, dst)
	case *ast.CallExpr:
		fs.call(n, dst)
	case *ast.ArrayExpr:
		fs.literal(dst, len(n.Items), func(i int) ast.Expr { return n.Items[i] }, arrayChunk)
	case *ast.InterpolationExpr:
		fs.literal(dst, len(n.Parts), func(i int) ast.Expr { return n.Parts[i] }, func(from, to int, base int32) ins {
			return ins{op: opConcat, b: base, c: int32(to - from)}
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
	case *ast.IsExpr:
		if n.Shape != nil {
			fs.isType(n, dst)
			break
		}

		v := fs.operand(n.Value, n.Type)
		fs.emit(ins{op: opIs, a: dst, b: v, c: fs.exprAny(n.Type)}, &n.Token)
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
	case *ast.MatchExpr:
		fs.match(n, dst)
	case *ast.ConditionalExpr:
		toElse := fs.jumpIf(n.Cond, false)
		fs.expr(n.Then, dst)
		end := fs.emit(ins{op: opJmp}, nil)
		fs.patch(toElse)
		fs.expr(n.Else, dst)
		fs.patch([]int{end})
	case *ast.ThrowExpr:
		fs.expr(n.Value, dst)
		fs.emit(ins{op: opThrow, a: dst}, &n.Keyword)
	default:
		panic(fmt.Sprintf("compile: unknown expression %T", e))
	}
}

func arrayChunk(from, to int, base int32) ins {
	return ins{op: opArray, b: base, c: int32(to - from)}
}

// literalChunk caps the registers an array or object literal takes; a
// longer one is built a chunk at a time, each adding to the first (X).
const literalChunk = 256

func (fs *funcState) literal(dst int32, n int, item func(i int) ast.Expr, build func(from, to int, base int32) ins) {
	for i := range n {
		if ast.IsSpread(item(i)) {
			fs.spreadLiteral(dst, n, item, build)
			return
		}
	}

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

// spreadLiteral builds a literal with spreads from an empty one, adding
// each run of plain items a chunk at a time and each spread on its own.
func (fs *funcState) spreadLiteral(dst int32, n int, item func(i int) ast.Expr, build func(from, to int, base int32) ins) {
	target := dst
	if dst < fs.nactive {
		target = fs.alloc()
	}

	empty := build(0, 0, fs.free)
	empty.a = target
	fs.emit(empty, nil)

	for from := 0; from < n; {
		base := fs.free

		if sp, ok := item(from).(*ast.SpreadExpr); ok {
			fs.emit(ins{op: opSpread, a: target, b: fs.exprAny(sp.Expr)}, &sp.Token)
			from++
		} else {
			to := from
			for to < n && to-from < literalChunk && !ast.IsSpread(item(to)) {
				fs.expr(item(to), fs.alloc())
				to++
			}

			in := build(from, to, base)
			in.a, in.x = target, 1
			fs.emit(in, nil)
			from = to
		}

		fs.free = base
	}

	fs.move(dst, target)
}

// structLiteral evaluates the fields in source order into registers in
// slot order, defaults filling the rest.
func (fs *funcState) structLiteral(n *ast.StructExpr, dst int32) {
	shape := n.Shape
	base := fs.free

	for range shape.Fields {
		fs.alloc()
	}

	if slices.ContainsFunc(n.Pairs, func(p ast.StructPair) bool { return ast.IsSpread(p.Value) }) {
		fs.spreadStruct(n, base, dst)
		return
	}

	given := make([]bool, len(shape.Fields))

	for _, pair := range n.Pairs {
		given[pair.Slot] = true
		fs.expr(pair.Value, base+int32(pair.Slot))
	}

	for i := range shape.Fields {
		if !given[i] {
			def, _ := shape.Default(i)
			op := helpers.If(value.Mutable(def), opLoadFresh, opLoadK)
			fs.emit(ins{op: op, a: base + int32(i), c: fs.constant(def)}, nil)
		}
	}

	fs.emit(ins{op: opStruct, a: dst, b: base, c: fs.constant(value.NewType(shape))}, nil)
}

// spreadStruct fills the fields in source order, each spread copying
// those it has over the ones before.
func (fs *funcState) spreadStruct(n *ast.StructExpr, base, dst int32) {
	typ := fs.constant(value.NewType(n.Shape))
	fs.emit(ins{op: opUndef, a: base, b: int32(len(n.Shape.Fields))}, nil)

	for _, pair := range n.Pairs {
		sp, ok := pair.Value.(*ast.SpreadExpr)
		if !ok {
			fs.expr(pair.Value, base+int32(pair.Slot))
			continue
		}

		free := fs.free
		fs.emit(ins{op: opStructSpread, a: base, b: fs.exprAny(sp.Expr), c: typ}, &sp.Token)
		fs.free = free
	}

	fs.emit(ins{op: opStructFill, a: dst, b: base, c: typ}, &n.Type.Name)
}

// writesEarly reports whether compiling e into a register writes it
// before e is fully evaluated, so it must not target a live variable.
func writesEarly(e ast.Expr) bool {
	switch n := unwrap(e).(type) {
	case *ast.LogicalExpr, *ast.AssignExpr, *ast.SetIndexExpr, *ast.ThrowExpr:
		return true
	case *ast.UpdateExpr:
		return !n.Prefix
	case *ast.ConditionalExpr:
		return writesEarly(n.Then) || writesEarly(n.Else)
	}

	return false
}

func (fs *funcState) checkDefined(l *ast.Local, r int32, name *ast.Token) {
	if fs.checked(l) {
		fs.emit(ins{op: opCheckDefined, a: r}, name)
	}
}

// leak clears the flag of an owned variable whose value the read n lets go.
func (fs *funcState) leak(n *ast.VarExpr) {
	if n.Leaks {
		fs.ownership(fs.locals[n.Local]+1, nil)
	}
}

func (fs *funcState) varExpr(n *ast.VarExpr, dst int32) {
	if v, ok := literalOf(n); ok {
		fs.emit(ins{op: opLoadK, a: dst, c: fs.constant(v)}, nil)
		return
	}

	switch kind, r := fs.lookup(n.Local); kind {
	case varLocal:
		fs.checkDefined(n.Local, r, &n.Name)
		fs.move(dst, r)
		fs.leak(n)
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

		if n.Local.Owned {
			fs.ownership(r+1, n.Value)
		}

		return r
	}

	v := fs.exprAny(n.Value)
	fs.store(kind, r, v, checked, &n.Name)

	if kind == varLocal {
		if n.Local.Owned {
			fs.ownership(r+1, n.Value)
		}

		return r
	}

	return v
}

func (fs *funcState) binary(n *ast.BinaryExpr, dst int32) {
	op := &n.Operator

	// A fold that fails is left to fail at run time.
	if v, ok := literalOf(n); ok {
		fs.emit(ins{op: opLoadK, a: dst, c: fs.constant(v)}, nil)
		return
	}

	rc, rConst := literalOf(n.Right)

	arith, isArith := arithOps[op.TokenType]
	cmp, isCmp := cmpOps[op.TokenType]

	// A number on the left of an operator that commutes goes right.
	left, kcmp := n.Left, cmp
	if lc, ok := literalOf(n.Left); ok && !rConst && lc.IsNumber() && (isCmp || isArith && (arith == value.OpAdd || arith == value.OpMul)) {
		left, rc, rConst, kcmp = n.Right, lc, true, flipped[cmp]
	}

	if k, ok := fs.operandConst(rc, rConst, maxC); ok && (isArith || isCmp) {
		l := fs.operand(left)

		if isArith {
			fs.emit(ins{op: arithOp(arith, true), a: dst, b: l, c: k}, op)
		} else {
			fs.emit(ins{op: opCmpK, x: kcmp, a: dst, b: l, c: k}, op)
		}

		return
	}

	l := fs.operand(n.Left, n.Right)
	r := fs.exprAny(n.Right)

	if !isCmp {
		fs.emit(binaryIns(op.TokenType, dst, l, r), op)
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

	r := fs.exprAny(n.Right)

	if op.TokenType == ast.BANG {
		fs.emit(ins{op: opNot, a: dst, b: r}, nil)
	} else if op.TokenType == ast.TILDE {
		fs.emit(ins{op: opBitNot, a: dst, b: r}, op)
	} else {
		fs.emit(ins{op: opNeg, a: dst, b: r}, op)
	}
}

func (fs *funcState) call(n *ast.CallExpr, dst int32) {
	spread := slices.ContainsFunc(n.Args, ast.IsSpread)

	if prop, ok := n.Callee.(*ast.IndexExpr); ok && isMethodCall(prop) && !n.Optional && !spread {
		fs.invoke(n, prop, dst)
		return
	}

	// The callee goes in dst when nothing above it is in use.
	base := dst
	if dst < fs.nactive || dst+1 != fs.free {
		base = fs.alloc()
	}

	fs.expr(n.Callee, base)

	if n.Optional {
		fs.nilJump(base)
	}

	if !spread {
		for _, arg := range n.Args {
			fs.expr(arg, fs.alloc())
		}

		fs.emit(ins{op: opCall, a: base, b: int32(len(n.Args))}, &n.Token)
		fs.move(dst, base)

		return
	}

	// The arguments go in an array, unless a lone spread gives one.
	args := fs.alloc()
	if sp, ok := n.Args[0].(*ast.SpreadExpr); ok && len(n.Args) == 1 {
		fs.expr(sp.Expr, args)
	} else {
		fs.literal(args, len(n.Args), func(i int) ast.Expr { return n.Args[i] }, arrayChunk)
	}

	fs.emit(ins{op: opCallArray, a: base}, &n.Token)
	fs.move(dst, base)
}

// invoke calls method prop of its receiver, which goes in the register
// after the callee's. A writable receiver gets self back after a mutating
// method returns.
func (fs *funcState) invoke(n *ast.CallExpr, prop *ast.IndexExpr, dst int32) {
	var lv *lval
	if n.Writable {
		lv = fs.ref(prop.Array, n.Args...)
	}

	base := dst
	if dst < fs.nactive || dst+1 != fs.free {
		base = fs.alloc()
	}

	// A place that cannot take the new self fails at the write-back,
	// which only a mutating method reaches.
	self := fs.alloc()
	if lv != nil {
		lv.load(self)
	} else {
		fs.expr(prop.Array, self)
	}

	if prop.Optional {
		fs.nilJump(self)
	}

	for _, arg := range n.Args {
		fs.expr(arg, fs.alloc())
	}

	// An owned variable passes its flag and takes it back from the
	// register after self.
	owned := lv != nil && lv.v != nil && lv.v.Lends
	if owned {
		if len(n.Args) == 0 {
			fs.alloc()
		}

		fs.emit(ins{op: opMove, a: base, b: fs.locals[lv.v.Local] + 1}, nil)
	}

	name := fs.constant(value.NewString(prop.Token.Lexeme))
	x := uint8(boolNum(n.Frozen)) | uint8(boolNum(owned))<<1
	fs.emit(ins{op: opInvoke, a: base, b: int32(len(n.Args)), c: name, x: x, y: uint8(boolNum(n.Writable))}, &prop.Token)

	if lv != nil {
		skip := fs.emit(ins{op: opJmp}, &prop.Token)

		if owned {
			fs.emit(ins{op: opMove, a: fs.locals[lv.v.Local] + 1, b: self + 1}, &prop.Token)
		}

		lv.writeBack()
		fs.patch([]int{skip})
	}

	fs.move(dst, base)
}

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
	fs.emit(binaryIns(n.Op.TokenType, old, old, v), n.Op)
	pl.store(old)

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
		if n.Op.TokenType == ast.QUESTION_QUESTION {
			break
		}

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

		lc, lConst := literalOf(n.Left)
		rc, rConst := literalOf(n.Right)

		if lConst && rConst {
			break
		}

		op := &n.Operator

		left, kcmp := n.Left, cmp
		if lConst && lc.IsNumber() {
			left, rc, rConst, kcmp = n.Right, lc, true, flipped[cmp]
		}

		if k, ok := fs.operandConst(rc, rConst, maxAB); ok {
			l := fs.operand(left)
			return []int{fs.emit(ins{op: opJmpCmpK, x: kcmp, y: y, a: l, b: k}, op)}
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

// isType sets R[dst] to whether the value of n is of the type n.Shape.
func (fs *funcState) isType(n *ast.IsExpr, dst int32) {
	v := fs.exprAny(n.Value)
	t := fs.alloc()
	fs.emit(ins{op: opLoadK, a: t, c: fs.constant(value.NewType(n.Shape))}, nil)
	fs.emit(ins{op: opIs, a: dst, b: v, c: t}, &n.Token)
}
