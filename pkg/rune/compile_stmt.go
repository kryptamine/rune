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
	case *ast.VarStmt:
		fs.varStmt(s)
	case *ast.DestructureStmt:
		fs.destructure(s)
	case *ast.ArrayDestructureStmt:
		fs.unpack(s)
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
			fs.returnSelf()
			fs.emit(ins{op: opReturnNil}, nil)
		} else {
			v := fs.exprAny(s.Value)

			// Register 1 takes the flag of an owned self.
			if fs.selfReg != 0 && v == 1 {
				v = fs.alloc()
				fs.move(v, 1)
			}

			fs.returnSelf()
			fs.emit(ins{op: opReturn, a: v}, nil)
		}
	case *ast.TryStmt:
		fs.tryStmt(s)
	case *ast.MatchStmt:
		fs.match(s.Match, noReg)
	case *ast.StructStmt, *ast.ImplStmt:
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

// varStmt computes a plain local's value before declaring it, so a call
// there can put its result in the local's register with no move.
func (fs *funcState) varStmt(s *ast.VarStmt) {
	if s.Const {
		return
	}

	l := s.Local

	if !l.TopLevel && !l.Captured {
		r := fs.alloc()
		fs.exprOrNil(s.Initializer, r)
		fs.free = r
		fs.declare(l)

		if l.Owned {
			fs.ownership(fs.declare(nil), s.Initializer)
		}

		return
	}

	fs.predeclare(l)
	fs.initialize(l, func(dst int32) { fs.exprOrNil(s.Initializer, dst) })

	if l.Owned {
		fs.ownership(fs.locals[l]+1, s.Initializer)
	}
}

func (fs *funcState) destructure(s *ast.DestructureStmt) {
	for _, b := range ast.Declared(s) {
		fs.predeclare(b.Local)
	}

	src := fs.exprAny(s.Initializer)

	for i := range s.Fields {
		f := &s.Fields[i]
		if f.Discards() {
			continue
		}
		fs.initialize(f.Local, func(dst int32) {
			fs.emit(ins{op: opGetProp, a: dst, b: src, c: fs.constant(value.NewString(f.Key.Lexeme)), x: uint8(boolNum(f.Default != nil))}, &f.Key)
			fs.orDefault(dst, f.Default)
		})
	}

	if r := s.Rest; r != nil && !r.Discards() {
		keys := make([]string, len(s.Fields))
		for i, f := range s.Fields {
			keys[i] = f.Key.Lexeme
		}

		c := fs.keyList(keys)
		fs.initialize(r.Local, func(dst int32) {
			fs.emit(ins{op: opOmit, a: dst, b: src, c: c}, &s.Brace)
		})
	}
}

// unpack takes the elements into temporaries, then the variables.
func (fs *funcState) unpack(s *ast.ArrayDestructureStmt) {
	for _, b := range ast.Declared(s) {
		fs.predeclare(b.Local)
	}

	src := fs.exprAny(s.Initializer)
	n := len(s.Elems)
	rest := s.Rest != nil
	base := fs.unpackTo(src, n, s.Required(), rest, &s.Bracket)

	for i, e := range s.Elems {
		if e != nil {
			fs.initialize(e.Local, func(dst int32) {
				fs.move(dst, base+int32(i))
				fs.orDefault(dst, e.Default)
			})
		}
	}

	if rest {
		fs.initialize(s.Rest.Local, func(dst int32) { fs.move(dst, base+int32(n)) })
	}
}

// unpackTo takes the first n elements of the array R[src], of which
// required must be there, into new temporaries, and with rest the others
// as an array after them; it returns the first temporary.
func (fs *funcState) unpackTo(src int32, n, required int, rest bool, tok *ast.Token) int32 {
	if n > maxUnpack {
		panic(tooLarge{fs.proto.name})
	}

	base := fs.free
	for range n + int(boolNum(rest)) {
		fs.alloc()
	}

	fs.emit(ins{op: opUnpack, a: base, b: src, c: int32(n | required<<unpackShift), x: uint8(boolNum(rest))}, tok)

	return base
}

// orDefault computes def into dst if dst is undefined, for a missing element
// or property.
func (fs *funcState) orDefault(dst int32, def ast.Expr) {
	if def == nil {
		return
	}

	skip := fs.emit(ins{op: opJmpDefined, a: dst}, nil)
	fs.expr(def, dst)
	fs.patch([]int{skip})
}

// predeclare gives a local its register before its value is computed, in
// a cell if captured, since a closure there may capture it.
func (fs *funcState) predeclare(l *ast.Local) {
	switch {
	case l.TopLevel:
	case l.Captured:
		fs.declareAs(l, value.UndefinedValue)
	default:
		fs.declare(l)
	}
}

// initialize stores what emit computes into dst in the declared variable l.
func (fs *funcState) initialize(l *ast.Local, emit func(dst int32)) {
	r := fs.locals[l]

	if l.Captured {
		t := fs.alloc()
		emit(t)
		fs.emit(ins{op: opSetCell, a: r, b: t}, nil)
		fs.free = t
	} else {
		emit(r)
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
		fs.emit(next, &s.Keyword)
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
		fs.emit(ins{op: opLoadK, a: base + 2, c: fs.constant(value.NewInt(1))}, nil)
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

			miss := fs.isOneOf(r, c.Shapes)
			fs.scoped(func() { fs.catchBody(c, r) })
			ends = append(ends, fs.emit(ins{op: opJmp}, nil))
			fs.patch(miss)
		}

		if rethrows {
			fs.emit(ins{op: opRethrow, a: r + 1}, nil)
		}
	})

	fs.patch(ends)
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
