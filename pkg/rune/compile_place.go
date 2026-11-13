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
		pl.fs.emit(ins{op: opCheckProp, a: pl.t.reg, b: pl.i, y: uint8(boolNum(pl.t.isSelf()))}, pl.tok)
	}
}

func (pl *place) load(dst int32) {
	pl.fs.emit(ins{op: pick(pl.prop, opGetProp, opGetIndex), a: dst, b: pl.t.reg, c: pl.i}, pl.tok)
}

// store writes v. Only a mut self takes a field set, as a copy that goes
// back to self unless self is changed in place or is no struct.
func (pl *place) store(v int32) {
	fs, t := pl.fs, pl.t
	self := t.isSelf()

	if pl.prop && self && t.v.Local.Owned && t.inPlace() {
		fs.emit(ins{op: opSetOwned, a: t.reg, b: pl.i, c: v}, pl.tok)
		return
	}

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
		if n.Module == nil && n.Local != nil && !n.Local.IsConst {
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

// guard checks that the variable a write-back after a call goes to still
// holds what was read from it: a closure or a function the call runs can
// change one in a cell or an upvalue, and the write-back would lose that.
// A field set (self.x = f()) has the same hazard, unchecked: its SetProp
// copies over the register holding what was read.
type guard struct {
	fs   *funcState
	v    *ast.VarExpr
	read int32
	copy bool
}

// guard is nil unless lv's root variable can change behind the call. A
// receiver that is the variable itself is loaded into the call's
// registers, which the call overwrites, so guard reserves a register
// to keep it in; call it before allocating those.
func (fs *funcState) guard(lv *lval) *guard {
	root := lv
	for root != nil && root.at != nil {
		root = root.at.t
	}

	if root == nil || root.v == nil {
		return nil
	}

	if kind, _ := fs.lookup(root.v.Local); kind != varCell && kind != varUpval {
		return nil
	}

	if root == lv {
		return &guard{fs: fs, v: root.v, read: fs.alloc(), copy: true}
	}

	return &guard{fs: fs, v: root.v, read: root.reg}
}

// keep saves the receiver loaded into self.
func (g *guard) keep(self int32) {
	if g != nil && g.copy {
		g.fs.move(g.read, self)
	}
}

func (g *guard) check(tok *ast.Token) {
	if g == nil {
		return
	}

	cur := g.fs.alloc()
	g.fs.varExpr(g.v, cur)
	g.fs.emit(ins{op: opSame, a: cur, b: g.read, c: g.fs.constant(value.NewString(g.v.Name.Lexeme))}, tok)
}

// isSelf reports whether reg is read from the self of a mut self method.
func (lv *lval) isSelf() bool { return lv.v != nil && lv.v.Local.MutSelf }

// inPlace reports whether reg is the variable's own register, so it needs
// no writing back.
func (lv *lval) inPlace() bool {
	if lv.v == nil {
		return false
	}

	kind, r := lv.fs.lookup(lv.v.Local)
	return kind == varLocal && r == lv.reg
}

// ownership sets the flag in register own, after an owned variable's,
// telling whether it holds a struct of its own: one a literal just made.
func (fs *funcState) ownership(own int32, v ast.Expr) {
	var fresh bool
	if v != nil {
		_, fresh = unwrap(v).(*ast.StructExpr)
	}

	fs.emit(ins{op: opLoadK, a: own, c: fs.constant(value.NewBool(fresh))}, nil)
}
