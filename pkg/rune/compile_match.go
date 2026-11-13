package rune

import (
	"rune/pkg/ast"
	"rune/pkg/value"
)

// match compiles m into dst; for a statement (dst noReg) the value is
// dropped. No arm matching is an error either way.
func (fs *funcState) match(m *ast.MatchExpr, dst int32) {
	fs.scoped(func() {
		var guards []ast.Expr
		for _, arm := range m.Arms {
			if arm.Guard != nil {
				guards = append(guards, arm.Guard)
			}
		}

		subj := fs.operand(m.Subject, guards...)
		fs.nactive = fs.free

		last := &m.Arms[len(m.Arms)-1]
		fails := !reachedAll(last)

		var ends []int

		for i := range m.Arms {
			arm := &m.Arms[i]

			fs.scoped(func() {
				miss := fs.arm(arm, subj, dst)

				if arm != last || fails {
					ends = append(ends, fs.emit(ins{op: opJmp}, nil))
				}

				fs.patch(miss)
			})
		}

		if fails {
			fs.emit(ins{op: opNoMatch, a: subj}, &m.Keyword)
		}

		fs.patch(ends)
	})
}

// arm compiles arm against R[subj] and returns the jumps taken when it
// does not match.
func (fs *funcState) arm(arm *ast.MatchArm, subj, dst int32) []int {
	bindings := ast.Bindings(arm.Pattern)
	for _, b := range bindings {
		fs.declare(b.Local)
	}

	miss := fs.pattern(arm.Pattern, subj)
	fs.free = fs.nactive

	for _, b := range bindings {
		if b.Local.Captured {
			r := fs.locals[b.Local]
			fs.emit(ins{op: opNewCell, a: r, b: r}, nil)
		}
	}

	if arm.Guard != nil {
		miss = append(miss, fs.jumpIf(arm.Guard, false)...)
	}

	if e, ok := arm.Body.(*ast.ExprStmt); ok && dst != noReg {
		fs.expr(e.Expr, dst)
		return miss
	}

	fs.stmt(arm.Body)

	if dst != noReg {
		fs.emit(ins{op: opLoadK, a: dst, c: fs.constant(value.NilValue)}, nil)
	}

	return miss
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
	case *ast.OrPattern:
		return fs.anyOf(len(n.Alts), func(i int) []int { return fs.pattern(n.Alts[i], s) })
	case *ast.StructPattern:
		miss := []int{fs.typeTest(s, n.Shape)}
		for _, f := range n.Fields {
			miss = append(miss, fs.fieldTest(s, f, false)...)
		}

		return miss
	case *ast.TypePattern:
		return append(fs.isOneOf(s, n.Shapes), fs.pattern(n.Inner, s)...)
	case *ast.ObjectPattern:
		var miss []int
		for _, f := range n.Fields {
			miss = append(miss, fs.fieldTest(s, f, true)...)
		}

		return miss
	}

	panic("compile: unknown pattern")
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

	k, ok := fs.operandConst(n.Value, true, maxAB)
	if ok {
		return fs.emit(ins{op: opJmpCmpK, x: cmpNE, y: 1, a: s, b: k}, &ne)
	}

	t := fs.alloc()
	fs.emit(ins{op: opLoadK, a: t, c: k}, nil)

	return fs.emit(ins{op: opJmpCmp, x: cmpNE, y: 1, a: s, b: t}, &ne)
}

// fieldTest matches field f of R[s]: of a struct already known to have
// it, or of an object or struct that may not.
func (fs *funcState) fieldTest(s int32, f ast.FieldPattern, mayMiss bool) []int {
	if _, ok := f.Pattern.(*ast.WildcardPattern); ok && !mayMiss {
		return nil
	}

	free := fs.free
	defer func() { fs.free = free }()

	b, bind := f.Pattern.(*ast.BindPattern)

	var t int32
	if bind {
		t = fs.locals[b.Local]
	} else {
		t = fs.alloc()
	}

	key := fs.constant(value.NewString(f.Key.Lexeme))

	var miss []int

	if mayMiss {
		fs.emit(ins{op: opTestProp, a: t, b: s, c: key}, nil)
		miss = append(miss, fs.emit(ins{op: opJmp}, nil))
	} else {
		fs.emit(ins{op: opGetProp, a: t, b: s, c: key}, &f.Key)
	}

	if bind {
		return miss
	}

	return append(miss, fs.pattern(f.Pattern, t)...)
}
