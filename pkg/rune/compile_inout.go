package rune

import "rune/pkg/ast"

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

// shareInouts shares each level of a path another holder shares.
func (fs *funcState) shareInouts(places []*lval) {
	for _, lv := range places {
		if lv != nil {
			lv.shareDown()
		}
	}
}

// inoutMask has bit+i for each of places, as A of the jump after a call.
func inoutMask(places []*lval, bit int) int32 {
	var mask int32
	for i, lv := range places {
		if lv != nil && bit+i < 16 {
			mask |= 1 << (bit + i)
		}
	}

	return mask
}

// writeBacks puts each argument the call wrote back, from the registers
// from first on, back where places read it.
func (fs *funcState) writeBacks(places []*lval, first int32, bit int, tok *ast.Token) {
	for i, lv := range places {
		if lv == nil || bit+i >= 16 {
			continue
		}

		skip := fs.emit(ins{op: opJmpNoInout, b: int32(bit + i)}, tok)
		lv.reg = first + int32(i)
		lv.writeBack(true)
		fs.patch([]int{skip})
	}
}

// protects reports whether a write-back must also happen when the call
// fails: something sees the places after the error, a handler or defer
// of this function, or anyone for a variable in a cell.
func (fs *funcState) protects(places []*lval) bool {
	if places == nil {
		return false
	}

	if len(fs.unwinds) > 0 {
		return true
	}

	for _, lv := range places {
		if lv != nil && fs.inCell(lv.e) {
			return true
		}
	}

	return false
}

// protect compiles call, which emits the call and its write-backs, under
// a handler that does the write-backs again if the call fails.
func (fs *funcState) protect(places []*lval, first int32, bit int, tok *ast.Token, call func()) {
	if !fs.protects(places) {
		call()
		return
	}

	at := fs.free
	try := fs.emit(ins{op: opTry, a: at, x: 1}, nil)
	call()
	fs.emit(ins{op: opEndTry, b: 1}, nil)
	end := fs.emit(ins{op: opJmp}, nil)

	fs.onError(try, at, func() {
		fs.declare(nil)
		fs.declare(nil) // the error
		fs.writeBacks(places, first, bit, tok)
		fs.emit(ins{op: opRethrow, a: at + 1}, nil)
	})
	fs.patch([]int{end})
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
			coll.writeBack(true)
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

// writeBack is the unwinds entry of a for mut body: leaving the body puts
// the variable back into its element, and so does an error, if handled.
type writeBack struct {
	put     func()
	handled bool
	try     int
	at      int32
}

func (*writeBack) Accept(ast.StmtVisitor) error { return nil }

// writeBackLoop compiles the body of for mut s, the variable in val going
// back into coll at key after it.
func (fs *funcState) writeBackLoop(s *ast.ForInStmt, coll *lval, key, val int32) *writeBack {
	wb := &writeBack{put: func() { fs.putBack(s, coll, key, val) }}
	wb.handled = len(fs.unwinds) > 0 || fs.inCell(coll.e)

	if wb.handled {
		wb.at = fs.free
		wb.try = fs.emit(ins{op: opTry, a: wb.at, x: 1}, nil)
	}

	fs.unwinds = append(fs.unwinds, wb)
	fs.scoped(func() { fs.stmt(s.Body) })
	fs.unwinds = fs.unwinds[:len(fs.unwinds)-1]

	if wb.handled {
		fs.emit(ins{op: opEndTry, b: 1}, nil)
	}

	wb.put()

	return wb
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
	pl.set(v, true)
}

// handler compiles the handler of a for mut body, after the loop.
func (wb *writeBack) handler(fs *funcState) {
	if wb == nil || !wb.handled {
		return
	}

	end := fs.emit(ins{op: opJmp}, nil)

	fs.onError(wb.try, wb.at, func() {
		fs.declare(nil)
		fs.declare(nil) // the error
		wb.put()
		fs.emit(ins{op: opRethrow, a: wb.at + 1}, nil)
	})
	fs.patch([]int{end})
}
