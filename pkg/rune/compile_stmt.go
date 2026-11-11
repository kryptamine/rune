package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/value"
	"slices"
)

func (fs *funcState) stmts(stmts []ast.Stmt) {
	for _, s := range stmts {
		fs.stmt(s)
		fs.free = fs.nactive
	}
}

// scoped runs body and then frees the locals it declared.
func (fs *funcState) scoped(body func()) {
	nactive := fs.nactive
	body()
	fs.nactive, fs.free = nactive, nactive
}

func (fs *funcState) stmt(stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		fs.exprStmt(s.Expr)
	case *ast.PrintStmt:
		fs.emit(ins{op: opPrint, a: fs.exprAny(s.Expr)}, nil)
	case *ast.VarStmt:
		fs.varStmt(s)
	case *ast.BlockStmt:
		fs.scoped(func() { fs.stmts(s.Stmts) })
	case *ast.IfStmt:
		fs.ifStmt(s)
	case *ast.WhileStmt:
		fs.whileStmt(s)
	case *ast.ForInStmt:
		fs.forIn(s)
	case *ast.FunctionStmt:
		fs.functionStmt(s)
	case *ast.ReturnStmt:
		if s.Value == nil {
			fs.emit(ins{op: opReturnNil}, nil)
		} else {
			fs.emit(ins{op: opReturn, a: fs.exprAny(s.Value)}, nil)
		}
	case *ast.TryStmt:
		fs.tryStmt(s)
	case *ast.ThrowStmt:
		fs.emit(ins{op: opThrow, a: fs.exprAny(s.Value)}, &s.Keyword)
	case *ast.StructStmt:
	case *ast.BreakStmt:
		l := fs.leaveLoop()
		l.breaks = append(l.breaks, fs.emit(ins{op: opJmp}, nil))
	case *ast.ContinueStmt:
		l := fs.leaveLoop()
		l.continues = append(l.continues, fs.emit(ins{op: opJmp}, nil))
	default:
		panic(fmt.Sprintf("compile: unknown statement %T", stmt))
	}
}

// exprStmt drops the value, so x++ needs no copy of the old x.
func (fs *funcState) exprStmt(e ast.Expr) {
	if u, ok := e.(*ast.UpdateExpr); ok {
		prefix := *u
		prefix.Prefix = true
		fs.update(&prefix, noReg)

		return
	}

	fs.exprAny(e)
}

// leaveLoop drops the handlers of the trys a break or continue jumps out of.
func (fs *funcState) leaveLoop() *loop {
	l := fs.loops[len(fs.loops)-1]
	if n := fs.tries - l.tries; n > 0 {
		fs.emit(ins{op: opEndTry, b: int32(n)}, nil)
	}

	return l
}

// varStmt binds a captured variable's cell before its initializer runs,
// since a closure there may capture it.
func (fs *funcState) varStmt(s *ast.VarStmt) {
	l := s.Local

	switch {
	case !l.TopLevel && !l.Captured:
		r := fs.alloc()
		fs.exprOrNil(s.Initializer, r)
		fs.free = r
		fs.declare(l)

		return
	case !l.TopLevel:
		fs.declareAs(l, value.UndefinedValue)
	}

	r := fs.locals[l]

	if l.Captured {
		t := fs.alloc()
		fs.exprOrNil(s.Initializer, t)
		fs.emit(ins{op: opSetCell, a: r, b: t}, nil)
		fs.free = t
	} else {
		fs.exprOrNil(s.Initializer, r)
	}

	fs.defined[l] = true
}

func (fs *funcState) exprOrNil(e ast.Expr, dst int32) {
	if e == nil {
		fs.emit(ins{op: opLoadK, a: dst, c: fs.constant(value.NilValue)}, nil)
		return
	}

	fs.expr(e, dst)
}

func (fs *funcState) functionStmt(s *ast.FunctionStmt) {
	if s.Local.TopLevel {
		defer func() { fs.defined[s.Local] = true }()
	}

	r, ok := fs.locals[s.Local]
	if !ok {
		r = fs.declare(s.Local)
	}

	if s.Local.TopLevel && s.Local.Captured {
		t := fs.alloc()
		fs.emit(ins{op: opClosure, a: t, c: fs.function(s)}, nil)
		fs.emit(ins{op: opSetCell, a: r, b: t}, nil)

		return
	}

	if !s.Local.Captured {
		fs.emit(ins{op: opClosure, a: r, c: fs.function(s)}, nil)
		return
	}

	// The cell exists before the closure, so the function can call itself.
	fs.emit(ins{op: opLoadK, a: r, c: fs.constant(value.NilValue)}, nil)
	fs.emit(ins{op: opNewCell, a: r, b: r}, nil)

	t := fs.alloc()
	fs.emit(ins{op: opClosure, a: t, c: fs.function(s)}, nil)
	fs.emit(ins{op: opSetCell, a: r, b: t}, nil)
}

func (fs *funcState) ifStmt(s *ast.IfStmt) {
	skip := fs.jumpIf(s.Condition, false)
	fs.scoped(func() { fs.stmt(s.Then) })

	if s.El == nil {
		fs.patch(skip)
		return
	}

	end := fs.emit(ins{op: opJmp}, nil)
	fs.patch(skip)
	fs.scoped(func() { fs.stmt(s.El) })
	fs.patch([]int{end})
}

// whileStmt tests at the bottom, so an iteration runs one jump.
func (fs *funcState) whileStmt(s *ast.WhileStmt) {
	enter := fs.emit(ins{op: opJmp}, nil)
	body := fs.here()

	l := fs.loop(func() { fs.stmt(s.Body) })

	fs.patch([]int{enter})
	fs.patch(l.continues)

	for _, j := range fs.jumpIf(s.Condition, true) {
		fs.code[j].c = body
	}

	fs.patch(l.breaks)
}

func (fs *funcState) loop(body func()) *loop {
	l := &loop{tries: fs.tries}
	fs.loops = append(fs.loops, l)
	fs.scoped(body)
	fs.loops = fs.loops[:len(fs.loops)-1]

	return l
}

func (fs *funcState) forIn(s *ast.ForInStmt) {
	fs.scoped(func() {
		var base int32

		var key, val int32

		next := ins{op: opForIterNext}

		if r, ok := s.Iterable.(*ast.RangeExpr); ok {
			base = fs.rangeParts(r)
			fs.free = base + 3
			fs.alloc()
			fs.nactive = fs.free
			fs.emit(ins{op: opForRange, a: base}, &r.Operator)

			key, val = fs.declare(nil), fs.declare(nil)
			next = ins{op: opForRangeNext, a: base, x: uint8(boolNum(r.Inclusive))}
		} else {
			base = fs.declare(nil)
			fs.expr(s.Iterable, base)
			fs.emit(ins{op: opForIter, a: base, x: uint8(boolNum(s.Key != nil))}, &s.Keyword)

			key, val = fs.declare(nil), fs.declare(nil)
			next.a = base
		}

		bind := func(b *ast.Binding, r int32) {
			if b != nil {
				fs.locals[b.Local] = r
			}
		}

		bind(s.Key, key)
		bind(&s.Value, val)

		enter := fs.emit(ins{op: opJmp}, nil)
		body := fs.here()

		l := fs.loop(func() {
			for _, b := range []*ast.Binding{s.Key, &s.Value} {
				if b != nil && b.Local.Captured {
					r := fs.locals[b.Local]
					fs.emit(ins{op: opNewCell, a: r, b: r}, nil)
				}
			}

			fs.stmt(s.Body)
		})

		fs.patch([]int{enter})
		fs.patch(l.continues)

		next.c = body
		fs.emit(next, nil)
		fs.patch(l.breaks)
	})
}

// rangeParts evaluates a range's start, end and step into three new registers.
func (fs *funcState) rangeParts(r *ast.RangeExpr) int32 {
	base := fs.alloc()
	fs.alloc()
	fs.alloc()

	fs.expr(r.Start, base)
	fs.expr(r.End, base+1)

	if r.Step != nil {
		fs.expr(r.Step, base+2)
	} else {
		fs.emit(ins{op: opLoadK, a: base + 2, c: fs.constant(value.NewNumber(1))}, nil)
	}

	return base
}

func (fs *funcState) tryStmt(s *ast.TryStmt) {
	rethrows := len(s.Catches[len(s.Catches)-1].Types) > 0
	bind := slices.ContainsFunc(s.Catches, func(c ast.CatchClause) bool { return c.Param != nil || len(c.Types) > 0 })
	try := fs.emit(ins{op: opTry, a: fs.nactive, x: uint8(boolNum(rethrows)), y: uint8(boolNum(bind))}, nil)

	fs.tries++
	fs.scoped(func() { fs.stmt(s.Body) })
	fs.tries--

	fs.emit(ins{op: opEndTry, b: 1}, nil)
	ends := []int{fs.emit(ins{op: opJmp}, nil)}
	fs.patch([]int{try})

	fs.scoped(func() {
		r := fs.nactive
		if bind {
			fs.declare(nil)
		}

		if rethrows {
			fs.declare(nil) // the caught error
		}

		for i := range s.Catches {
			c := &s.Catches[i]
			if len(c.Types) == 0 {
				fs.scoped(func() { fs.catchBody(c, r) })
				break
			}

			hits := fs.isAny(r, c.Types)
			miss := fs.emit(ins{op: opJmp}, nil)
			fs.patch(hits)
			fs.scoped(func() { fs.catchBody(c, r) })
			ends = append(ends, fs.emit(ins{op: opJmp}, nil))
			fs.patch([]int{miss})
		}

		if rethrows {
			fs.emit(ins{op: opRethrow, a: r + 1}, nil)
		}
	})

	fs.patch(ends)
}

// isAny emits a jump, for each of types, taken if R[r] is of it.
func (fs *funcState) isAny(r int32, types []*ast.VarExpr) []int {
	hits := make([]int, len(types))

	for i, t := range types {
		tmp := fs.alloc()
		fs.emit(ins{op: opIs, a: tmp, b: r, c: fs.exprAny(t)}, &t.Name)
		hits[i] = fs.emit(ins{op: opJmpIf, a: tmp, y: 1}, nil)
		fs.free = fs.nactive
	}

	return hits
}

// catchBody compiles c with the caught value in register r.
func (fs *funcState) catchBody(c *ast.CatchClause, r int32) {
	if c.Param != nil {
		fs.locals[c.ParamLocal] = r
		if c.ParamLocal.Captured {
			fs.emit(ins{op: opNewCell, a: r, b: r}, nil)
		}
	}

	fs.stmts(c.Body)
}
