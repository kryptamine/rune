package rune

import (
	"rune/pkg/ast"
	"rune/pkg/value"
	"slices"
)

// place is the target of a write: t[i], or for a property t.K[i].
type place struct {
	fs       *funcState
	t        *lval
	i        int32
	prop     bool
	optional bool
	tok      *ast.Token
}

// place evaluates the target (and for t[i] the index) of a write that
// evaluates later before writing.
func (fs *funcState) place(array, index ast.Expr, prop bool, tok *ast.Token, later ...ast.Expr) *place {
	pl := &place{fs: fs, prop: prop, tok: tok}

	if prop {
		pl.t = fs.lvalue(array, later...)
		pl.i = fs.constant(value.NewString(tok.Lexeme))
	} else {
		pl.t = fs.lvalue(array, append([]ast.Expr{index}, later...)...)
		pl.i = fs.operand(index, later...)
	}

	return pl
}

// check makes a property write fail on a target that cannot take it
// before more code runs.
func (pl *place) check() {
	if pl.prop {
		pl.fs.emit(ins{op: opCheckProp, a: pl.t.reg, b: pl.i, y: uint8(boolNum(pl.t.fromSelf()))}, pl.tok)
	}
}

func (pl *place) load(dst int32) {
	if pl.optional {
		pl.fs.nilJump(pl.t.reg)
	}

	pl.fs.emit(ins{op: pick(pl.prop, opGetProp, opGetIndex), a: dst, b: pl.t.reg, c: pl.i}, pl.tok)
}

// store writes v. Only a path from a mut self takes a field set. A set
// on a shared target copies it, and the copy goes back where it was read.
func (pl *place) store(v int32) {
	pl.t.shareDown()
	pl.put(v)
}

// put is store for what goes back on a path shared already.
func (pl *place) put(v int32) {
	fs, t := pl.fs, pl.t

	set := fs.emit(ins{op: pick(pl.prop, opSetProp, opSetIndex), a: t.reg, b: pl.i, c: v, y: uint8(boolNum(t.fromSelf()))}, pl.tok)

	if !t.held() {
		return
	}

	// A call on the right can change a variable in a cell, so what was
	// read from one always goes back, whether copied or not.
	if fs.inCell(t.e) {
		t.writeBack()
		return
	}

	fs.code[set].x = 1
	skip := fs.emit(ins{op: opJmp}, pl.tok)
	t.writeBack()
	fs.patch([]int{skip})
}

// inCell reports whether e is read from a variable in a cell or an
// upvalue, which a call can change behind the reader.
func (fs *funcState) inCell(e ast.Expr) bool {
	for {
		switch n := unwrap(e).(type) {
		case *ast.IndexExpr:
			if n.Export != nil {
				return true
			}

			e = n.Array
		case *ast.VarExpr:
			if n.Local == nil || n.Local.IsConst {
				return false
			}

			kind, _ := fs.lookup(n.Local)
			return kind == varCell || kind == varUpval
		default:
			return false
		}
	}
}

// lval is a value in reg read from a variable or a place, where a new
// struct can go back.
type lval struct {
	fs  *funcState
	e   ast.Expr
	reg int32
	v   *ast.VarExpr
	at  *place
}

// ref evaluates where e is read from, but not e, which load reads.
func (fs *funcState) ref(e ast.Expr, later ...ast.Expr) *lval {
	e = unwrap(e)
	lv := &lval{fs: fs, e: e, reg: noReg}

	switch n := e.(type) {
	case *ast.VarExpr:
		if n.Module == nil && n.Local != nil && !n.Local.IsConst {
			lv.v = n
		}
	case *ast.IndexExpr:
		switch {
		case n.Export != nil:
			if !n.Export.Local.IsConst {
				lv.v = n.Export
			}
		case n.Member == nil:
			lv.at = fs.place(n.Array, n.Index, n.Property, &n.Token, later...)
			lv.at.optional = n.Optional
		}
	}

	return lv
}

// lvalue is ref and load, reading a plain local in its own register.
func (fs *funcState) lvalue(e ast.Expr, later ...ast.Expr) *lval {
	lv := fs.ref(e, later...)
	if lv.at == nil {
		lv.reg = fs.operand(e, later...)
	} else {
		lv.load(fs.alloc())
	}

	return lv
}

func (lv *lval) load(dst int32) {
	if lv.at != nil {
		lv.at.load(dst)
	} else {
		lv.fs.expr(lv.e, dst)
	}

	lv.reg = dst
}

// check fails before the call on a write-back that cannot happen.
func (lv *lval) check() {
	if lv.at != nil {
		lv.at.check()
		lv.at.t.check()
	}
}

// writeBack puts reg back where it was read from.
func (lv *lval) writeBack() {
	fs := lv.fs

	switch {
	case lv.at != nil:
		lv.at.put(lv.reg)
	case lv.v != nil && !lv.inPlace():
		kind, r := fs.lookup(lv.v.Local)
		fs.store(kind, r, lv.reg, fs.checked(lv.v.Local), &lv.v.Name)
	}
}

// shareDown shares each level of the path reg was read along whose
// holder is shared, from the top: a write then copies every level that
// another holder has too, each copy going back into the level above; see
// frozenStruct.
func (lv *lval) shareDown() {
	var path []*lval
	for l := lv; l.at != nil; l = l.at.t {
		path = append(path, l)
	}

	x := uint8(boolNum(!lv.fromSelf()) * frozenStruct)
	for _, l := range slices.Backward(path) {
		lv.fs.emit(ins{op: opShareIf, a: l.reg, b: l.at.t.reg, x: x}, nil)
	}
}

// held reports whether reg was read from somewhere it must go back to.
func (lv *lval) held() bool { return lv.at != nil || lv.v != nil && !lv.inPlace() }

// fromSelf reports whether reg is read from the self of a mut self
// method or a place below it.
func (lv *lval) fromSelf() bool {
	for lv.at != nil {
		lv = lv.at.t
	}

	return lv.v != nil && lv.v.Local.MutSelf
}

// inPlace reports whether reg is the variable's own register, so it needs
// no writing back.
func (lv *lval) inPlace() bool {
	if lv.v == nil {
		return false
	}

	kind, r := lv.fs.lookup(lv.v.Local)
	return kind == varLocal && r == lv.reg
}
