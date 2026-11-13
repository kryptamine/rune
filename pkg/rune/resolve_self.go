package rune

import (
	"rune/pkg/ast"
	"rune/pkg/value"
)

// varUses records how a mutable local, or a method's self, is read: each
// read is true if it only reads or sets a field, or calls a method on
// it. props names the property each read takes, calls the method each
// call names. A self has its method, whose struct type is known.
type varUses struct {
	local  *ast.Local
	method *ast.FunctionStmt
	shape  *value.Shape
	reads  map[*ast.VarExpr]bool
	props  map[*ast.VarExpr]string
	calls  map[*ast.VarExpr]string
	// assigned is set when a self is assigned a whole new value.
	assigned bool
}

func newVarUses(l *ast.Local) *varUses {
	return &varUses{local: l, reads: map[*ast.VarExpr]bool{}, props: map[*ast.VarExpr]string{}, calls: map[*ast.VarExpr]string{}}
}

// usesOf returns e and its record if it reads a tracked variable.
func (p *Resolver) usesOf(e ast.Expr) (*ast.VarExpr, *varUses) {
	v, ok := e.(*ast.VarExpr)
	if !ok || v.Local == nil {
		return nil, nil
	}

	if p.self != nil && v.Local == p.self.method.Parameters[0].Local {
		return v, p.self
	}

	return v, p.locals[v.Local]
}

// trackLocal records the uses of a mutable local of a function.
func (p *Resolver) trackLocal(stmt *ast.VarStmt) {
	if stmt.Mutable && stmt.Local != nil && !stmt.Local.TopLevel {
		p.locals[stmt.Local] = newVarUses(stmt.Local)
	}
}

// useVar records a read of e, safe when it only has a field or an
// element set; a read a parent takes apart records itself again.
func (p *Resolver) useVar(e ast.Expr, safe bool) {
	if v, u := p.usesOf(e); u != nil {
		u.reads[v] = safe
	}
}

// useProp records a read of e.key or e[key].
func (p *Resolver) useProp(expr *ast.IndexExpr) {
	v, u := p.usesOf(expr.Array)
	if u == nil {
		return
	}

	u.reads[v] = true

	if key, ok := expr.Index.(*ast.LiteralExpr); ok && expr.Property {
		u.props[v] = key.Value.Str()
	}
}

// useCall records a call of the method prop names.
func (p *Resolver) useCall(prop *ast.IndexExpr) {
	if v, u := p.usesOf(prop.Array); u != nil {
		u.reads[v] = true
		delete(u.props, v)
		u.calls[v] = prop.Token.Lexeme
	}
}

func (p *Resolver) useAssign(expr *ast.AssignExpr) {
	if p.self != nil && expr.Local == p.self.method.Parameters[0].Local {
		p.self.assigned = true
	}
}

// resolveMethods resolves the methods of stmt, recording the uses of the
// mut self of those of a struct.
func (p *Resolver) resolveMethods(stmt *ast.ImplStmt) {
	shape := p.structs[p.mod.topLevel[stmt.Type.Lexeme]]

	for _, m := range stmt.Methods {
		if shape == nil || shape.Enum || !m.Method.Self {
			p.resolveFn(m.Fn)
			continue
		}

		u := newVarUses(nil)
		u.method, u.shape = m.Fn, shape
		p.self = u
		p.resolveFn(m.Fn)
		p.self = nil

		u.local = m.Fn.Parameters[0].Local
		p.selves = append(p.selves, u)
	}
}

// ownVars marks the variables that may change a struct in place once
// they hold a copy of their own: the mutable locals and mut selves that
// never let it go. Such a variable is only read for its fields, has its
// fields set, or has methods called on it that keep self too; it is
// never captured, nor, for a self, assigned.
func (p *Resolver) ownVars() {
	byName := map[*value.Shape]map[string]*varUses{}
	escapes := map[*varUses]bool{}

	for _, u := range p.selves {
		if byName[u.shape] == nil {
			byName[u.shape] = map[string]*varUses{}
		}

		byName[u.shape][u.method.Name.Lexeme] = u
		escapes[u] = u.assigned || leaks(u, func(name string) bool { return u.shape.Index(name) < 0 })
	}

	for changed := true; changed; {
		changed = false

		for _, u := range p.selves {
			if escapes[u] {
				continue
			}

			for _, name := range u.calls {
				if callee := byName[u.shape][name]; callee == nil || escapes[callee] {
					escapes[u], changed = true, true
					break
				}
			}
		}
	}

	// A local's type is unknown: a name it calls must name no method that
	// lets self go, and one it reads no method at all, which would bind it.
	methods, leaking := map[string]bool{}, map[string]bool{}

	for _, u := range p.selves {
		name := u.method.Name.Lexeme
		methods[name] = true
		leaking[name] = leaking[name] || escapes[u]
		u.local.Owned = u.method.Parameters[0].Mutable && !escapes[u]
	}

	for _, u := range p.locals {
		u.local.Owned = !leaks(u, func(name string) bool { return methods[name] }) && !callsAny(u, leaking)
	}

	p.selves, p.locals = nil, map[*ast.Local]*varUses{}
}

// leaks reports whether u's variable is captured, read as a whole, or has
// a property read that bound names.
func leaks(u *varUses, bound func(name string) bool) bool {
	if u.local.Captured {
		return true
	}

	for v, safe := range u.reads {
		if name, ok := u.props[v]; !safe || ok && bound(name) {
			return true
		}
	}

	return false
}

func callsAny(u *varUses, names map[string]bool) bool {
	for _, name := range u.calls {
		if names[name] {
			return true
		}
	}

	return false
}
