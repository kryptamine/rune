package rune

import (
	"rune/pkg/ast"
	"rune/pkg/value"
	"slices"
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
	// deep names the methods called on a place below the variable, and
	// sets is set when a field of it, or below it, is set.
	deep []string
	sets bool
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

// trackLocal records the uses of a mutable variable or parameter.
func (p *Resolver) trackLocal(b ast.Binding, l *ast.Local) {
	if b.Mutable && l != nil && !b.Discards() {
		p.locals[l] = newVarUses(l)
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
	} else if u := p.rootUses(prop.Array); u != nil {
		u.deep = append(u.deep, prop.Token.Lexeme)
	}
}

// rootUses returns the record of the tracked variable e is a place of.
func (p *Resolver) rootUses(e ast.Expr) *varUses {
	if v := rootVar(e); v != nil {
		_, u := p.usesOf(v)
		return u
	}

	return nil
}

// useSet records the set of a field of a variable, or below it.
func (p *Resolver) useSet(expr *ast.SetIndexExpr) {
	if u := p.rootUses(expr.Array); u != nil && expr.Property {
		u.sets = true
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

			if m.Method.Self {
				delete(p.locals, m.Fn.Parameters[0].Local)
			}

			continue
		}

		u := newVarUses(nil)
		u.method, u.shape = m.Fn, shape
		p.self = u
		p.resolveFn(m.Fn)
		p.self = nil

		u.local = m.Fn.Parameters[0].Local
		delete(p.locals, u.local)
		p.selves = append(p.selves, u)
	}
}

// ownVars marks the variables that may change a struct in place once
// they hold a copy of their own. A mut self must never let it go: it is
// only read for its fields, has its fields set, or has methods called on
// it that keep self too; it is never captured or assigned. Another
// mutable variable, if not captured or exported, gives its copy up on
// each read that lets the value go, which ownVars marks; it is owned
// only if that saves a copy, of a field set or a mutating call.
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

	// A variable's type is unknown: a name it calls lets it go if any
	// method of that name lets self go, and one it reads if it names a
	// method at all, which would bind it.
	methods, leaking, mutating := map[string]bool{}, map[string]bool{}, map[string]bool{}

	for _, u := range p.selves {
		name, mut := u.method.Name.Lexeme, u.method.Parameters[0].Mutable
		methods[name] = true
		leaking[name] = leaking[name] || escapes[u]
		mutating[name] = mutating[name] || mut
		u.local.Owned = mut && !escapes[u]
	}

	for _, u := range p.selves {
		for v, name := range u.calls {
			v.Lends = u.local.Owned && mutating[name]
		}
	}

	exported := map[*ast.Local]bool{}
	for _, m := range p.modules {
		for _, l := range m.exports {
			exported[l] = true
		}
	}

	for _, u := range p.locals {
		if u.local.Captured || exported[u.local] || !u.sets && !callsAny(u, mutating) {
			continue
		}

		u.local.Owned = true

		for v, safe := range u.reads {
			v.Leaks = !safe || methods[u.props[v]] || leaking[u.calls[v]]
			v.Lends = mutating[u.calls[v]]
		}
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

// callsAny reports whether a method named in names is called on u's
// variable, or on a place below it.
func callsAny(u *varUses, names map[string]bool) bool {
	for _, name := range u.calls {
		if names[name] {
			return true
		}
	}

	return slices.ContainsFunc(u.deep, func(name string) bool { return names[name] })
}
