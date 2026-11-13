package rune

import (
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/value"
)

func (p *Resolver) VisitMatchStmt(stmt *ast.MatchStmt) error {
	p.resolveExpr(stmt.Match)
	return nil
}

func (p *Resolver) VisitMatchExpr(expr *ast.MatchExpr) (value.Value, error) {
	p.resolveExpr(expr.Subject)

	for i := range expr.Arms {
		arm := &expr.Arms[i]

		if i > 0 && reachedAll(&expr.Arms[i-1]) {
			p.error(arm.Token, "Unreachable match arm.")
		}

		p.beginScope()
		arm.Pattern = p.resolvePattern(arm.Pattern)

		for _, b := range ast.Bindings(arm.Pattern) {
			b.Local = p.declare(*b)
			p.define(b.Name)
		}

		if arm.Guard != nil {
			p.resolveExpr(arm.Guard)
		}

		p.resolveStmt(arm.Body)
		p.endScope()
	}

	return value.NilValue, nil
}

// reachedAll reports whether arm matches every value, so no arm after it runs.
func reachedAll(arm *ast.MatchArm) bool {
	return arm.Guard == nil && ast.Irrefutable(arm.Pattern)
}

// resolvePattern returns pat with each bare name replaced by a test of the
// struct type it names, or else a binding.
func (p *Resolver) resolvePattern(pat ast.Pattern) ast.Pattern {
	switch n := pat.(type) {
	case *ast.BindPattern:
		if name := n.Name.Lexeme; p.structNamed(name) != nil {
			p.error(n.Name, "'"+name+"' is a struct type; test it with '_ is "+name+"'.")
		}
	case *ast.TypePattern:
		n.Shapes = p.resolveTypes(n.Types)
		n.Inner = p.resolvePattern(n.Inner)
	case *ast.OrPattern:
		for i, alt := range n.Alts {
			n.Alts[i] = p.resolvePattern(alt)

			if bs := ast.Bindings(n.Alts[i]); len(bs) > 0 {
				p.error(bs[0].Name, "Cannot bind variables in '|' patterns.")
			}
		}
	case *ast.StructPattern:
		if n.Module != nil {
			t := &ast.VarExpr{Module: n.Module, Name: n.Name}
			p.resolveExpr(t)
			n.Shape = p.structType(t)
		} else if n.Shape = p.structNamed(n.Name.Lexeme); n.Shape == nil {
			p.error(n.Name, "'"+n.Name.Lexeme+"' is not a struct.")
		}

		if n.Shape != nil && n.Shape.Enum {
			p.error(n.Name, notStruct(n.Shape))
		}

		for _, f := range n.Fields {
			if n.Shape != nil && n.Shape.Index(f.Key.Lexeme) < 0 {
				p.error(f.Key, errors.NoField(n.Shape.Name, f.Key.Lexeme))
			}
		}

		p.resolveFields(n.Fields)
	case *ast.ObjectPattern:
		p.resolveFields(n.Fields)
	case *ast.VariantPattern:
		return p.resolveVariantPattern(n)
	}

	return pat
}

// resolveTypes returns the type each of types names, reporting those
// that name something else; a missing module member is reported already.
func (p *Resolver) resolveTypes(types []*ast.VarExpr) []*value.Shape {
	shapes := make([]*value.Shape, len(types))

	for i, t := range types {
		if shapes[i] = p.typeNamed(t); shapes[i] == nil && (t.Module == nil || t.Local != nil) {
			p.error(t.Name, "'"+qualified(t.Module, t.Name)+"' is not a type.")
		}
	}

	return shapes
}

// typeNamed resolves t and returns the type it names: a struct type, or
// a built-in one such as string; nil for anything else.
func (p *Resolver) typeNamed(t *ast.VarExpr) *value.Shape {
	if s := p.kindType(t); s != nil {
		return s
	}

	p.resolveExpr(t)

	return p.structs[t.Local]
}

var kindTypes = func() map[string]*value.Shape {
	ts := map[string]*value.Shape{}
	for name, k := range builtinKinds {
		ts[name] = &value.Shape{Name: name, Builtin: true, Kind: k}
	}

	ts["nil"] = &value.Shape{Name: "nil", Builtin: true, Kind: value.Nil}
	ts["struct"] = &value.Shape{Name: "struct", Builtin: true, Kind: value.Type}
	ts["enum"] = &value.Shape{Name: "enum", Builtin: true, Kind: value.Type, Enum: true}

	return ts
}()

// kindType returns the built-in type t names, as in v is string, unless
// a declaration or an import hides it.
func (p *Resolver) kindType(t ast.Expr) *value.Shape {
	v, ok := t.(*ast.VarExpr)
	if !ok || v.Module != nil {
		return nil
	}

	name := v.Name.Lexeme
	if p.lookupScopes(name) != nil || p.mod.topLevel[name] != nil || p.mod.imports[name] != nil {
		return nil
	}

	return kindTypes[name]
}

func (p *Resolver) resolveFields(fields []ast.FieldPattern) {
	for i := range fields {
		fields[i].Pattern = p.resolvePattern(fields[i].Pattern)
	}
}

// structNamed returns the struct type name denotes here, if it is one.
func (p *Resolver) structNamed(name string) *value.Shape {
	if p.lookupScopes(name) != nil {
		return nil
	}

	if l := p.mod.topLevel[name]; l != nil {
		return p.structs[l]
	}

	if v, ok := builtinValues()[name]; ok && v.Kind() == value.Type {
		return v.TypeShape()
	}

	return nil
}
