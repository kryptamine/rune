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
	case *ast.NamePattern:
		if shape := p.structNamed(n.Name.Lexeme); shape != nil {
			return &ast.StructPattern{Name: n.Name, Shape: shape}
		}

		return &ast.BindPattern{Binding: ast.Binding{Name: n.Name}}
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

		for _, f := range n.Fields {
			if n.Shape != nil && n.Shape.Index(f.Key.Lexeme) < 0 {
				p.error(f.Key, errors.NoField(n.Shape.Name, f.Key.Lexeme))
			}
		}

		p.resolveFields(n.Fields)
	case *ast.ObjectPattern:
		p.resolveFields(n.Fields)
	}

	return pat
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
