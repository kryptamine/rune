package rune

import (
	"rune/pkg/ast"
	"slices"
)

// inouts evaluates where each argument of n that a mut parameter may
// write back to is read from, before the registers of the call.
func (fs *funcState) inouts(n *ast.CallExpr) []*lval {
	if n.Inout == 0 {
		return nil
	}

	later := append([]ast.Expr{n.Callee}, n.Args...)
	places := make([]*lval, len(n.Args))

	for i, arg := range n.Args {
		if n.Inout&(1<<i) != 0 {
			places[i] = fs.ref(arg, later...)
		}
	}

	return places
}

// inoutArg loads argument i, read from places[i], into dst. The function
// changes it in place, unless the call can reach it another way: read
// from a cell, or from the variable of another argument or of recv.
func (fs *funcState) inoutArg(n *ast.CallExpr, recv ast.Expr, places []*lval, i int, dst int32) {
	lv := places[i]
	lv.load(dst)

	shared := fs.inCell(lv.e) || recv != nil && sameRoot(lv.e, recv)
	for j, arg := range n.Args {
		shared = shared || j != i && sameRoot(arg, lv.e)
	}

	if shared {
		fs.emit(ins{op: opShare, a: dst}, nil)
	}
}

// inoutCall emits call, whose callee's registers hold places from first
// on, then the write-backs of places, which a mut parameter leaves in
// those registers. guarded are the places that also go back if the call
// fails.
func (fs *funcState) inoutCall(call ins, places, guarded []*lval, first int32, tok *ast.Token) {
	for _, lv := range places {
		if lv != nil {
			lv.shareDown()
		}
	}

	if places != nil {
		call.x |= inoutSite
	}

	fs.protect(guarded, first, tok, func() {
		fs.emit(call, tok)

		if places != nil {
			skip := fs.emit(ins{op: opJmp, a: inoutMask(places)}, tok)
			fs.writeBacks(places, first, tok)
			fs.patch([]int{skip})
		}
	})
}

// inoutMask has bit i for each of places, as A of the jump after a call.
func inoutMask(places []*lval) int32 {
	var mask int32
	for i, lv := range places {
		if lv != nil {
			mask |= 1 << i
		}
	}

	return mask
}

// writeBacks puts each argument the call wrote back, from the registers
// from first on, back where places read it.
func (fs *funcState) writeBacks(places []*lval, first int32, tok *ast.Token) {
	for i, lv := range places {
		if lv == nil {
			continue
		}

		skip := fs.emit(ins{op: opJmpNoInout, b: int32(i)}, tok)
		lv.reg = first + int32(i)
		lv.writeBack()
		fs.patch([]int{skip})
	}
}

// seenOnError reports whether something sees what e reads after an
// error: a handler or defer of this function, or anyone for a variable
// in a cell.
func (fs *funcState) seenOnError(e ast.Expr) bool {
	return len(fs.unwinds) > 0 || fs.inCell(e)
}

// protect compiles call, which emits the call and its write-backs, under
// a handler that does the write-backs again if the call fails, if
// anything sees places after the error.
func (fs *funcState) protect(places []*lval, first int32, tok *ast.Token, call func()) {
	if !slices.ContainsFunc(places, func(lv *lval) bool { return lv != nil && fs.seenOnError(lv.e) }) {
		call()
		return
	}

	at := fs.free
	try := fs.emit(ins{op: opTry, a: at, x: 1}, nil)
	call()
	fs.emit(ins{op: opEndTry, b: 1}, nil)
	fs.rethrowAfter(try, at, func() { fs.writeBacks(places, first, tok) })
}

// collection reads the collection of a for mut, which the variable goes
// back into. A loop that has it to itself first makes it a copy of its
// own, so each element is changed in place.
func (fs *funcState) collection(s *ast.ForInStmt) *lval {
	coll := fs.lvalue(s.Iterable)
	fs.nactive = fs.free

	if fs.alone(s, coll) {
		coll.shareDown()
		fs.emit(ins{op: opUnshare, a: coll.reg, x: uint8(boolNum(coll.held()))}, &s.Keyword)

		if coll.held() {
			skip := fs.emit(ins{op: opJmp}, &s.Keyword)
			coll.writeBack()
			fs.patch([]int{skip})
		}
	}

	return coll
}

// alone reports whether nothing but the variable of for mut s can read its
// collection while the loop runs.
func (fs *funcState) alone(s *ast.ForInStmt, coll *lval) bool {
	return s.Alone && !fs.inCell(coll.e)
}

// writeBackLoop compiles the body of for mut s, the variable in val going
// back into coll at key after it, and on leaving the body. If an error
// is seen, it goes back then too, by the handler it returns, which is
// compiled after the loop.
func (fs *funcState) writeBackLoop(s *ast.ForInStmt, coll *lval, key, val int32) (handler func()) {
	wb := &unwind{leave: func() { fs.putBack(s, coll, key, val) }, try: fs.seenOnError(coll.e)}
	body := func() { fs.stmt(s.Body) }

	if !wb.try {
		fs.unwinds = append(fs.unwinds, wb)
		fs.scoped(body)
		fs.unwinds = fs.unwinds[:len(fs.unwinds)-1]
		wb.leave()

		return nil
	}

	at := fs.free
	try := fs.handled(ins{op: opTry, a: at, x: 1}, wb, body)
	wb.leave()

	return func() { fs.rethrowAfter(try, at, wb.leave) }
}

// putBack writes the variable of for mut s in val into coll at key.
func (fs *funcState) putBack(s *ast.ForInStmt, coll *lval, key, val int32) {
	v := val
	if s.Value.Local.Captured {
		v = fs.alloc()
		fs.emit(ins{op: opGetCell, a: v, b: val}, nil)
		fs.emit(ins{op: opShare, a: v}, nil)
	}

	pl := &place{fs: fs, t: coll, i: key, tok: &s.Keyword}
	pl.put(v)
}
