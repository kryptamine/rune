package rune

import (
	"rune/pkg/ast"
	"rune/pkg/value"
	"slices"
)

func (fs *funcState) patternStmt(s *ast.PatternStmt) {
	for _, b := range ast.Declared(s) {
		fs.predeclare(b.Local)
	}

	fs.destructure(s.Pattern, fs.exprAny(s.Initializer))
}

// destructure binds the variables of pat to the parts of R[src], each of
// which must be there unless it has a default.
func (fs *funcState) destructure(pat ast.Pattern, src int32) {
	switch n := pat.(type) {
	case *ast.BindPattern:
		fs.initialize(n.Local, func(dst int32) { fs.move(dst, src) })
	case *ast.ObjectPattern:
		for _, f := range n.Fields {
			fs.destructureField(src, f)
		}

		if r := n.Rest; r != nil {
			c := fs.keyList(fieldKeys(n.Fields), nil)
			fs.initialize(r.Local, func(dst int32) {
				fs.emit(ins{op: opOmit, a: dst, b: src, c: c}, &n.Brace)
			})
		}
	case *ast.ArrayPattern:
		count := len(n.Elems)
		base := fs.unpackTo(src, count, n.Required(), n.Rest != nil, n.Open && n.Rest == nil, &n.Bracket)

		for i, e := range n.Elems {
			e, def := unwrapDefault(e)
			fs.orDefault(base+int32(i), def)
			fs.destructure(e, base+int32(i))
		}

		if n.Rest != nil {
			fs.destructure(n.Rest, base+int32(count))
		}
	}
}

// destructureField binds the variables of f to its property of R[src].
func (fs *funcState) destructureField(src int32, f ast.FieldPattern) {
	p, def := unwrapDefault(f.Pattern)
	if _, ok := p.(*ast.WildcardPattern); ok {
		return
	}

	get := func(dst int32) {
		fs.emit(ins{op: opGetProp, a: dst, b: src, c: fs.constant(value.NewString(f.Key.Lexeme)), x: uint8(boolNum(def != nil))}, &f.Key)
		fs.orDefault(dst, def)
		fs.emit(ins{op: opShare, a: dst}, nil)
	}

	if b, ok := p.(*ast.BindPattern); ok {
		fs.initialize(b.Local, get)
		return
	}

	free := fs.free
	t := fs.alloc()
	get(t)
	fs.destructure(p, t)
	fs.free = free
}

func unwrapDefault(p ast.Pattern) (ast.Pattern, ast.Expr) {
	if d, ok := p.(*ast.DefaultPattern); ok {
		return d.Pattern, d.Default
	}

	return p, nil
}

func fieldKeys(fields []ast.FieldPattern) []string {
	keys := make([]string, len(fields))
	for i, f := range fields {
		keys[i] = f.Key.Lexeme
	}

	return keys
}

// condScope compiles body, the condition cond and what it guards, with
// the variables its is tests bind declared first, above the temporaries
// live around it.
func (fs *funcState) condScope(cond ast.Expr, body func()) {
	var binds []*ast.Binding
	for _, t := range condTests(cond) {
		binds = append(binds, ast.Bindings(t.Pattern)...)
	}

	if len(binds) == 0 {
		body()
		return
	}

	nactive, free := fs.nactive, fs.free
	fs.nactive = fs.free

	for _, b := range binds {
		fs.declare(b.Local)
	}

	body()
	fs.nactive, fs.free = nactive, free
}

// cellBindings puts each variable pat binds that a closure captures in a cell.
func (fs *funcState) cellBindings(pat ast.Pattern) {
	for _, b := range ast.Bindings(pat) {
		if b.Local.Captured {
			r := fs.locals[b.Local]
			fs.emit(ins{op: opNewCell, a: r, b: r}, nil)
		}
	}
}

// pattern emits the tests of R[s] against p, binding its variables, and
// returns the jumps taken on a mismatch.
func (fs *funcState) pattern(p ast.Pattern, s int32) []int {
	switch n := p.(type) {
	case *ast.WildcardPattern:
		return nil
	case *ast.BindPattern:
		fs.move(fs.locals[n.Local], s)

		return nil
	case *ast.LiteralPattern:
		return []int{fs.literalTest(s, n)}
	case *ast.RangePattern:
		return fs.rangeTest(s, n)
	case *ast.OrPattern:
		return fs.anyOf(len(n.Alts), func(i int) []int { return fs.pattern(n.Alts[i], s) })
	case *ast.StructPattern:
		miss := []int{fs.typeTest(s, n.Shape)}
		for _, f := range n.Fields {
			miss = append(miss, fs.fieldTest(s, f, false)...)
		}

		return miss
	case *ast.IsPattern:
		return append(fs.pattern(n.Test, s), fs.pattern(n.Inner, s)...)
	case *ast.TypeTestPattern:
		return []int{fs.typeTestOf(s, n)}
	case *ast.ObjectPattern:
		var miss []int
		if n.Rest != nil || slices.ContainsFunc(n.Fields, hasDefault) {
			miss = append(miss, fs.typeTest(s, value.RecordType))
		}

		for _, f := range n.Fields {
			miss = append(miss, fs.fieldTest(s, f, true)...)
		}

		if n.Rest != nil {
			c := fs.keyList(fieldKeys(n.Fields), nil)
			fs.emit(ins{op: opOmit, a: fs.locals[n.Rest.Local], b: s, c: c}, &n.Brace)
		}

		return miss
	case *ast.ArrayPattern:
		return fs.arrayTest(s, n)
	}

	panic("compile: unknown pattern")
}

func hasDefault(f ast.FieldPattern) bool {
	_, ok := f.Pattern.(*ast.DefaultPattern)
	return ok
}

// arrayTest matches R[s] against [p, q = d, ...rest]: tests the length,
// then unpacks the elements into temporaries and matches each.
func (fs *funcState) arrayTest(s int32, n *ast.ArrayPattern) []int {
	count, required := len(n.Elems), n.Required()

	var miss []int
	if required == count {
		miss = []int{fs.emit(ins{op: opJmpNotLen, a: s, b: int32(count), x: uint8(boolNum(n.Open))}, nil)}
	} else {
		miss = []int{fs.emit(ins{op: opJmpNotLen, a: s, b: int32(required), x: lenAtLeast}, nil)}
		if !n.Open {
			miss = append(miss, fs.emit(ins{op: opJmpNotLen, a: s, b: int32(count), x: lenAtMost}, nil))
		}
	}

	if count == 0 && n.Rest == nil {
		return miss
	}

	free := fs.free
	defer func() { fs.free = free }()

	// The length is tested already.
	base := fs.unpackTo(s, count, required, n.Rest != nil, true, &n.Bracket)

	for i, e := range n.Elems {
		e, def := unwrapDefault(e)
		fs.orDefault(base+int32(i), def)
		miss = append(miss, fs.pattern(e, base+int32(i))...)
	}

	if n.Rest != nil {
		fs.pattern(n.Rest, base+int32(count))
	}

	return miss
}

// anyOf emits n tests, each giving the jumps it takes on a mismatch, and
// returns the jumps taken when all of them miss.
func (fs *funcState) anyOf(n int, test func(i int) []int) []int {
	var hits, miss []int

	for i := range n {
		miss = test(i)
		if i < n-1 {
			hits = append(hits, fs.emit(ins{op: opJmp}, nil))
			fs.patch(miss)
		}
	}

	fs.patch(hits)

	return miss
}

// isOneOf returns the jumps taken unless R[s] is a struct of one of shapes.
func (fs *funcState) isOneOf(s int32, shapes []*value.Shape) []int {
	return fs.anyOf(len(shapes), func(i int) []int { return []int{fs.typeTest(s, shapes[i])} })
}

// typeTestOf jumps unless R[s] is of the type n names, which may be
// evaluated only when it runs.
func (fs *funcState) typeTestOf(s int32, n *ast.TypeTestPattern) int {
	if n.Shape != nil {
		return fs.typeTest(s, n.Shape)
	}

	t := fs.exprAny(n.Type)
	r := fs.alloc()
	fs.emit(ins{op: opIs, a: r, b: s, c: t}, &n.Token)

	return fs.emit(ins{op: opJmpIf, a: r, y: 0}, nil)
}

// typeTest jumps unless R[s] is a struct of shape; a type past the
// constants JMPNOTTYPE reaches is tested with IS.
func (fs *funcState) typeTest(s int32, shape *value.Shape) int {
	k, ok := fs.operandConst(value.NewType(shape), true, maxAB)
	if ok {
		return fs.emit(ins{op: opJmpNotType, a: s, b: k}, nil)
	}

	t := fs.alloc()
	fs.emit(ins{op: opLoadK, a: t, c: k}, nil)
	fs.emit(ins{op: opIs, a: t, b: s, c: t}, nil)

	return fs.emit(ins{op: opJmpIf, a: t, y: 0}, nil)
}

// literalTest jumps unless R[s] == the literal.
func (fs *funcState) literalTest(s int32, n *ast.LiteralPattern) int {
	ne := n.Token
	ne.TokenType = ast.BANG_EQUAL

	return fs.jumpCmpConst(s, n.Value, cmpNE, 1, &ne)
}

// rangeTest jumps unless R[s] is a number within the bounds of n.
func (fs *funcState) rangeTest(s int32, n *ast.RangePattern) []int {
	miss := []int{fs.typeTest(s, kindTypes["number"])}

	bound := func(b ast.Pattern, cmp uint8) {
		if b != nil {
			miss = append(miss, fs.jumpCmpConst(s, b.(*ast.LiteralPattern).Value, cmp, 0, &n.Op))
		}
	}

	bound(n.Lo, cmpGE)

	if n.Inclusive {
		bound(n.Hi, cmpLE)
	} else {
		bound(n.Hi, cmpLT)
	}

	return miss
}

// jumpCmpConst jumps if (R[s] <cmp> v) == y.
func (fs *funcState) jumpCmpConst(s int32, v value.Value, cmp, y uint8, tok *ast.Token) int {
	k, ok := fs.operandConst(v, true, maxAB)
	if ok {
		return fs.emit(ins{op: opJmpCmpK, x: cmp, y: y, a: s, b: k}, tok)
	}

	t := fs.alloc()
	fs.emit(ins{op: opLoadK, a: t, c: k}, nil)

	return fs.emit(ins{op: opJmpCmp, x: cmp, y: y, a: s, b: t}, tok)
}

// fieldTest matches field f of R[s]: of a struct already known to have
// it, or of an object or struct that may not.
func (fs *funcState) fieldTest(s int32, f ast.FieldPattern, mayMiss bool) []int {
	if _, ok := f.Pattern.(*ast.WildcardPattern); ok && !mayMiss {
		return nil
	}

	free := fs.free
	defer func() { fs.free = free }()

	p, def := unwrapDefault(f.Pattern)
	b, bind := p.(*ast.BindPattern)

	var t int32
	if bind {
		t = fs.locals[b.Local]
	} else {
		t = fs.alloc()
	}

	key := fs.constant(value.NewString(f.Key.Lexeme))

	var miss []int

	switch {
	case !mayMiss:
		fs.emit(ins{op: opGetProp, a: t, b: s, c: key}, &f.Key)
	case def == nil:
		fs.emit(ins{op: opTestProp, a: t, b: s, c: key}, nil)
		miss = append(miss, fs.emit(ins{op: opJmp}, nil))
	default:
		fs.emit(ins{op: opTestProp, a: t, b: s, c: key}, nil)
		missing := fs.emit(ins{op: opJmp}, nil)
		found := fs.emit(ins{op: opJmp}, nil)
		fs.patch([]int{missing})
		fs.copyExpr(def, t)
		fs.patch([]int{found})
	}

	if bind {
		return miss
	}

	return append(miss, fs.pattern(p, t)...)
}

// isExpr sets R[dst] to whether the value of n matches its pattern.
func (fs *funcState) isExpr(n *ast.IsExpr, dst int32) {
	if types, ok := typeTests(n.Pattern); ok {
		fs.isTypes(n, types, dst)
		return
	}

	miss := fs.isMatch(n)
	fs.emit(ins{op: opLoadK, a: dst, c: fs.constant(value.NewBool(!n.Negate))}, nil)
	end := fs.emit(ins{op: opJmp}, nil)
	fs.patch(miss)
	fs.emit(ins{op: opLoadK, a: dst, c: fs.constant(value.NewBool(n.Negate))}, nil)
	fs.patch([]int{end})
}

// isMatch emits the test of n's pattern, binding its variables, declared
// already, and returns the jumps taken on a mismatch.
func (fs *funcState) isMatch(n *ast.IsExpr) []int {
	v := fs.operand(n.Value, ast.Exprs(n.Pattern)...)
	miss := fs.pattern(n.Pattern, v)
	fs.cellBindings(n.Pattern)

	return miss
}

// typeTests returns the types of pat, if it only tests for types.
func typeTests(pat ast.Pattern) ([]*ast.TypeTestPattern, bool) {
	switch n := pat.(type) {
	case *ast.TypeTestPattern:
		return []*ast.TypeTestPattern{n}, true
	case *ast.OrPattern:
		var types []*ast.TypeTestPattern
		for _, alt := range n.Alts {
			t, ok := alt.(*ast.TypeTestPattern)
			if !ok {
				return nil, false
			}

			types = append(types, t)
		}

		return types, true
	}

	return nil, false
}

// isTypes sets R[dst] to whether the value of n is of one of types,
// testing them in order until one decides.
func (fs *funcState) isTypes(n *ast.IsExpr, types []*ast.TypeTestPattern, dst int32) {
	v := fs.operand(n.Value, ast.Exprs(n.Pattern)...)
	neg := uint8(boolNum(n.Negate))
	r := dst
	if len(types) > 1 {
		r = fs.alloc()
	}

	var ends []int

	for i, typ := range types {
		var t int32
		if typ.Shape != nil {
			t = fs.alloc()
			fs.emit(ins{op: opLoadK, a: t, c: fs.constant(value.NewType(typ.Shape))}, nil)
		} else {
			t = fs.exprAny(typ.Type)
		}

		fs.emit(ins{op: opIs, a: r, b: v, c: t, y: neg}, &typ.Token)

		if i < len(types)-1 {
			ends = append(ends, fs.emit(ins{op: opJmpIf, a: r, y: 1 - neg}, nil))
		}
	}

	fs.patch(ends)

	if r != dst {
		fs.emit(ins{op: opMove, a: dst, b: r}, nil)
	}
}
