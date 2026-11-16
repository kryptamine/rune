package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/value"
	"slices"
)

func (fs *funcState) stmts(stmts []ast.Stmt) { fs.stmtsThen(stmts, nil) }

// stmtsThen compiles stmts and then last, if any, before the defers among
// stmts run.
func (fs *funcState) stmtsThen(stmts []ast.Stmt, last func()) {
	for i, s := range stmts {
		if d, ok := s.(*ast.DeferStmt); ok {
			fs.deferStmt(d, func() { fs.stmtsThen(stmts[i+1:], last) })
			return
		}

		fs.stmt(s)
		fs.free = fs.nactive
	}

	if last != nil {
		last()
	}
}

// deferStmt compiles rest, what follows d, then d's body: on the way out
// of rest, and if rest fails, before the error goes on up.
func (fs *funcState) deferStmt(d *ast.DeferStmt, rest func()) {
	at := fs.nactive
	run := func() { fs.runDefer(d.Body) }
	try := fs.handled(ins{op: opTry, a: at, x: 1}, &unwind{leave: run, try: true}, rest)
	run()
	fs.rethrowAfter(try, at, run)
}

// unwind is what leaving the code being compiled passes, besides the
// handler of a try: code to run on the way out, under a handler of its
// own if try, which goes first.
type unwind struct {
	leave func()
	try   bool
}

// handled emits try and compiles body under its handler, with entry on
// unwinds; it returns try's index, for onError.
func (fs *funcState) handled(try ins, entry *unwind, body func()) int {
	at := fs.emit(try, nil)

	fs.unwinds = append(fs.unwinds, entry)
	fs.scoped(body)
	fs.unwinds = fs.unwinds[:len(fs.unwinds)-1]

	fs.emit(ins{op: opEndTry, b: 1}, nil)

	return at
}

// onError compiles handle where the try at index try resumes, with the
// registers from at, which the error is put in, free.
func (fs *funcState) onError(try int, at int32, handle func()) {
	fs.patch([]int{try})

	nactive, free := fs.nactive, fs.free
	fs.nactive, fs.free = at, at
	fs.scoped(handle)
	fs.nactive, fs.free = nactive, free
}

// rethrowAfter jumps over a handler compiled where the try at index try
// resumes, which runs handle and passes the error on up.
func (fs *funcState) rethrowAfter(try int, at int32, handle func()) {
	end := fs.emit(ins{op: opJmp}, nil)

	fs.onError(try, at, func() {
		fs.declare(nil)
		fs.declare(nil) // the error
		handle()
		fs.emit(ins{op: opRethrow, a: at + 1}, nil)
	})
	fs.patch([]int{end})
}

func (fs *funcState) runDefer(body ast.Stmt) { fs.scoped(func() { fs.stmt(body) }) }

// runDefers runs the bodies of the defers above unwinds[depth], each after
// dropping the handlers above it; it returns how many handlers are left.
func (fs *funcState) runDefers(depth int) (handlers int) {
	nactive, free := fs.nactive, fs.free
	fs.nactive = free

	for i := len(fs.unwinds) - 1; i >= depth; i-- {
		u := fs.unwinds[i]
		if u == nil {
			handlers++
			continue
		}

		if !u.try {
			u.leave()
			continue
		}

		fs.emit(ins{op: opEndTry, b: int32(handlers + 1)}, nil)
		handlers = 0

		unwinds := fs.unwinds
		fs.unwinds = unwinds[:i:i]
		u.leave()
		fs.unwinds = unwinds
	}

	fs.nactive, fs.free = nactive, free

	return handlers
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
	case *ast.PatternStmt:
		fs.patternStmt(s)
	case *ast.BlockStmt:
		fs.block(s, noReg)
	case *ast.IfStmt:
		fs.ifStmt(s, noReg)
	case *ast.WhileStmt:
		fs.whileStmt(s)
	case *ast.ForInStmt:
		fs.forIn(s)
	case *ast.FunctionStmt:
		fs.functionStmt(s)
	case *ast.ReturnStmt:
		fs.returnStmt(s)
	case *ast.TryStmt:
		fs.tryStmt(s)
	case *ast.DeferStmt:
		fs.deferStmt(s, func() {})
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

// returnStmt runs the defers after computing the value, which they may
// not change, and leaves the handlers to the return to drop.
func (fs *funcState) returnStmt(s *ast.ReturnStmt) {
	if s.Value == nil {
		fs.runDefers(0)
		fs.returnSelf()
		fs.emit(ins{op: opReturnNil}, nil)

		return
	}

	v := fs.exprAny(s.Value)

	if v < fs.nactive && slices.ContainsFunc(fs.unwinds, func(u *unwind) bool { return u != nil }) {
		t := fs.alloc()
		fs.move(t, v)
		v = t
	}

	fs.runDefers(0)

	fs.returnSelf()

	ret := ins{op: opReturn, a: v, x: uint8(boolNum(!fresh(s.Value)))}
	if p := paramMove(s.Value); p > 0 {
		ret.x, ret.b = 2, int32(p-1)
	}

	fs.emit(ret, nil)
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

// leaveLoop leaves the trys and defers a break or continue jumps out of.
func (fs *funcState) leaveLoop() *loop {
	l := fs.loops[len(fs.loops)-1]
	if n := fs.runDefers(l.unwinds); n > 0 {
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
		fs.share(s.Initializer, r)
		fs.free = r
		fs.declare(l)

		return
	}

	fs.predeclare(l)
	fs.initialize(l, func(dst int32) {
		fs.exprOrNil(s.Initializer, dst)
		fs.share(s.Initializer, dst)
	})
}

// unpackTo takes the first n elements of the array R[src], of which
// required must be there, into new temporaries, and with rest the others
// as an array after them; it returns the first temporary. Without rest
// there may be no others, unless open.
func (fs *funcState) unpackTo(src int32, n, required int, rest, open bool, tok *ast.Token) int32 {
	if n > maxUnpack {
		panic(tooLarge{fs.proto.name})
	}

	base := fs.free
	for range n + int(boolNum(rest)) {
		fs.alloc()
	}

	fs.emit(ins{op: opUnpack, a: base, b: src, c: int32(n | required<<unpackShift), x: uint8(boolNum(rest)), y: uint8(boolNum(open))}, tok)

	return base
}

// orDefault computes def into dst if dst is undefined, for a missing element
// or property.
func (fs *funcState) orDefault(dst int32, def ast.Expr) {
	if def == nil {
		return
	}

	skip := fs.emit(ins{op: opJmpDefined, a: dst}, nil)
	fs.copyExpr(def, dst)
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

// ifStmt compiles s, with the value of the branch taken into dst unless
// it is noReg.
func (fs *funcState) ifStmt(s *ast.IfStmt, dst int32) {
	var skip []int

	fs.condScope(s.Condition, func() {
		skip = fs.jumpIf(s.Condition, false)
		fs.block(s.Then, dst)
	})

	if s.El == nil && dst == noReg {
		fs.patch(skip)
		return
	}

	end := fs.emit(ins{op: opJmp}, nil)
	fs.patch(skip)

	switch el := s.El.(type) {
	case nil:
		fs.emit(ins{op: opLoadK, a: dst, c: fs.constant(value.NilValue)}, nil)
	case *ast.IfStmt:
		fs.ifStmt(el, dst)
	case *ast.BlockStmt:
		fs.block(el, dst)
	}

	fs.patch([]int{end})
}

// block compiles b, with its value into dst unless it is noReg. Its
// locals go above the temporaries live in the expression it may be in.
func (fs *funcState) block(b *ast.BlockStmt, dst int32) {
	free := fs.free
	defer func() { fs.free = free }()

	fs.scoped(func() {
		fs.nactive = free

		if dst == noReg {
			fs.stmts(b.Stmts)
			return
		}

		stmts := b.Stmts
		if b.Valued {
			stmts = stmts[:len(stmts)-1]
		}

		fs.stmtsThen(stmts, func() {
			if v := b.Value(); v != nil {
				fs.expr(v, dst)
			} else {
				fs.emit(ins{op: opLoadK, a: dst, c: fs.constant(value.NilValue)}, nil)
			}
		})
	})
}

// whileStmt tests at the bottom, so an iteration runs one jump.
func (fs *funcState) whileStmt(s *ast.WhileStmt) {
	fs.condScope(s.Condition, func() { fs.while(s) })
}

func (fs *funcState) while(s *ast.WhileStmt) {
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
	l := &loop{unwinds: len(fs.unwinds)}
	fs.loops = append(fs.loops, l)
	fs.scoped(body)
	fs.loops = fs.loops[:len(fs.loops)-1]

	return l
}

func (fs *funcState) forIn(s *ast.ForInStmt) {
	fs.scoped(func() {
		var base int32

		var coll *lval

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
		} else if s.Value.Mutable {
			coll = fs.collection(s)
			base = fs.declare(nil)
			fs.move(base, coll.reg)
			fs.emit(ins{op: opForIter, a: base, x: 1 | forMut, y: uint8(boolNum(fs.alone(s, coll)))}, &s.Keyword)

			key, val = fs.declare(nil), fs.declare(nil)
			next.a = base
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

		// A for mut's key is the cursor's, which a cell may take over.
		at := key
		if coll != nil && s.Key != nil && s.Key.Local.Captured {
			at = fs.declare(nil)
		}

		enter := fs.emit(ins{op: opJmp}, nil)
		body := fs.here()

		var handler func()

		l := fs.loop(func() {
			fs.move(at, key)

			for _, b := range []*ast.Binding{s.Key, &s.Value} {
				if b != nil && b.Local.Captured {
					r := fs.locals[b.Local]
					fs.emit(ins{op: opNewCell, a: r, b: r}, nil)
				}
			}

			if coll == nil {
				fs.stmt(s.Body)
				return
			}

			handler = fs.writeBackLoop(s, coll, at, val)
		})

		fs.patch([]int{enter})
		fs.patch(l.continues)

		next.c = body
		fs.emit(next, &s.Keyword)
		fs.patch(l.breaks)
		if handler != nil {
			handler()
		}
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
	fs.tryCatch(s.Catches, fs.nactive, func() { fs.stmt(s.Body) }, func(c *ast.CatchClause, r int32) {
		fs.bindCatch(c, r)
		fs.stmts(c.Body)
	})
}

// tryExpr puts the error above every live register, dst included.
func (fs *funcState) tryExpr(t *ast.TryExpr, dst int32) {
	fs.tryCatch(t.Catches, max(fs.free, dst+1), func() { fs.expr(t.Value, dst) }, func(c *ast.CatchClause, r int32) {
		fs.bindCatch(c, r)
		fs.expr(c.Value, dst)
	})
}

// tryCatch compiles body under a handler that puts the error in
// registers from at, then the catch clauses, each by handle.
func (fs *funcState) tryCatch(catches []ast.CatchClause, at int32, body func(), handle func(c *ast.CatchClause, r int32)) {
	rethrows := len(catches[len(catches)-1].Types) > 0
	bind := slices.ContainsFunc(catches, func(c ast.CatchClause) bool { return c.Param != nil || len(c.Types) > 0 })
	try := fs.handled(ins{op: opTry, a: at, x: uint8(boolNum(rethrows)), y: uint8(boolNum(bind))}, nil, body)
	ends := []int{fs.emit(ins{op: opJmp}, nil)}

	fs.onError(try, at, func() {
		if bind {
			fs.declare(nil)
		}

		if rethrows {
			fs.declare(nil) // the caught error
		}

		for i := range catches {
			c := &catches[i]
			if len(c.Types) == 0 {
				fs.scoped(func() { handle(c, at) })
				break
			}

			miss := fs.isOneOf(at, c.Shapes)
			fs.scoped(func() { handle(c, at) })
			ends = append(ends, fs.emit(ins{op: opJmp}, nil))
			fs.patch(miss)
		}

		if rethrows {
			fs.emit(ins{op: opRethrow, a: at + 1}, nil)
		}
	})
	fs.patch(ends)
}

// bindCatch binds the catch variable of c, if any, to register r.
func (fs *funcState) bindCatch(c *ast.CatchClause, r int32) {
	if c.Param != nil {
		fs.locals[c.ParamLocal] = r
		if c.ParamLocal.Captured {
			fs.emit(ins{op: opNewCell, a: r, b: r}, nil)
		}
	}
}
