package rune

import (
	"rune/pkg/ast"
	"rune/pkg/value"
)

// place is the target of a write: t[i], or for a property t.K[i].
type place struct {
	fs   *funcState
	t    *lval
	i    int32
	prop bool
	tok  *ast.Token
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
		pl.fs.emit(ins{op: opCheckProp, a: pl.t.reg, b: pl.i, y: uint8(boolNum(pl.t.isMutSelf()))}, pl.tok)
	}
}

func (pl *place) load(dst int32) {
	pl.fs.emit(ins{op: pick(pl.prop, opGetProp, opGetIndex), a: dst, b: pl.t.reg, c: pl.i}, pl.tok)
}

// store writes v. Only a mut self takes a field: as a copy, which goes
// back to self unless the target turns out to be an object or array.
func (pl *place) store(v int32) {
	fs, t := pl.fs, pl.t
	self := t.isMutSelf()
	set := fs.emit(ins{op: pick(pl.prop, opSetProp, opSetIndex), a: t.reg, b: pl.i, c: v, y: uint8(boolNum(self))}, pl.tok)

	if !self || t.inPlace() {
		return
	}

	fs.code[set].x = 1
	skip := fs.emit(ins{op: opJmp}, pl.tok)
	t.writeBack()
	fs.patch([]int{skip})
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
		if n.Module == nil && n.Local != nil {
			lv.v = n
		}
	case *ast.IndexExpr:
		if n.Member == nil && n.Export == nil && !n.Optional {
			lv.at = fs.place(n.Array, n.Index, n.Property, &n.Token, later...)
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

func (lv *lval) isMutSelf() bool { return lv.v != nil && lv.fs.isMutSelf(lv.v.Local) }

// writeBack puts reg back where it was read from.
func (lv *lval) writeBack() {
	fs := lv.fs

	switch {
	case lv.at != nil:
		lv.at.store(lv.reg)
	case lv.v != nil && !lv.inPlace():
		kind, r := fs.lookup(lv.v.Local)
		fs.store(kind, r, lv.reg, fs.checked(lv.v.Local), &lv.v.Name)
	}
}

// inPlace reports whether reg is the variable's own register, so it needs
// no writing back.
func (lv *lval) inPlace() bool {
	kind, r := lv.fs.lookup(lv.v.Local)
	return kind == varLocal && r == lv.reg
}
