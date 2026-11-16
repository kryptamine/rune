package rune

import (
	"rune/pkg/ast"
	"rune/pkg/value"
	"slices"
	"strings"
)

func (p *Resolver) VisitMatchStmt(stmt *ast.MatchStmt) error {
	p.resolveExpr(stmt.Match)
	return nil
}

func (p *Resolver) VisitMatchExpr(expr *ast.MatchExpr) (value.Value, error) {
	p.resolveExpr(expr.Subject)

	if expr.Enum != nil {
		expr.EnumShape = p.enumNamed(expr.Enum)
	}

	for i := range expr.Arms {
		arm := &expr.Arms[i]

		if expr.Enum != nil {
			arm.Pattern = variantArm(arm.Pattern, expr.Enum, expr.EnumShape)
		}

		if i > 0 && reachedAll(&expr.Arms[i-1]) {
			p.error(arm.Token, "Unreachable match arm.")
		}

		p.beginScope()
		arm.Pattern = p.resolvePattern(arm.Pattern)

		for _, b := range ast.Bindings(arm.Pattern) {
			b.Local = p.declare(*b)
			p.define(b.Name)
		}

		p.resolveExpr(arm.Guard)

		p.resolveStmt(arm.Body)
		p.endScope()
	}

	p.checkExhaustive(expr)

	return value.NilValue, nil
}

// enumNamed returns the enum t names, reporting it if it names none.
func (p *Resolver) enumNamed(t *ast.VarExpr) *value.Shape {
	s := p.typeNamed(t)
	if s == nil || !s.Enum || s.Builtin || s.Parent != nil {
		p.error(t.Name, "'"+qualified(t.Module, t.Name)+"' is not an enum; match x is Enum { ... } names its variants bare.")
		return nil
	}

	return s
}

// variantArm returns the pattern of an arm of a match on the enum e, of
// type enum unless nil, with the bare names of its variants qualified.
func variantArm(pat ast.Pattern, e *ast.VarExpr, enum *value.Shape) ast.Pattern {
	switch n := pat.(type) {
	case *ast.BindPattern:
		if enum != nil && hasFields(enum, n.Name.Lexeme) {
			return &ast.IsPattern{Inner: &ast.WildcardPattern{Token: n.Name}, Test: &ast.TypeTestPattern{Token: n.Name, Type: variantOf(e, n.Name)}}
		}

		return namePattern(variantOf(e, n.Name))
	case *ast.StructPattern:
		if n.Type.Module == nil {
			n.Type = variantOf(e, n.Type.Name)
		}
	case *ast.OrPattern:
		for i, alt := range n.Alts {
			n.Alts[i] = variantArm(alt, e, enum)
		}
	}

	return pat
}

func hasFields(enum *value.Shape, variant string) bool {
	v, ok := enum.Variant(variant)
	return ok && v.Kind() == value.Type
}

// variantOf is the name of the variant v of the enum e, as Enum.v or m.Enum.v.
func variantOf(e *ast.VarExpr, v ast.Token) *ast.VarExpr {
	if e.Module != nil {
		return &ast.VarExpr{Module: e.Module, Name: e.Name, Case: &v}
	}

	enum := e.Name

	return &ast.VarExpr{Module: &enum, Name: v}
}

// checkExhaustive reports a match without a catch-all arm that names
// variants of an enum but not all of them. A variant is covered by an arm
// without a guard that matches all its values.
func (p *Resolver) checkExhaustive(expr *ast.MatchExpr) {
	var enums []*value.Shape
	covered := map[value.Value]bool{}

	cover := func(enum *value.Shape, v value.Value, full bool) {
		if !slices.Contains(enums, enum) {
			enums = append(enums, enum)
		}

		covered[v] = covered[v] || full
	}

	var visit func(pat ast.Pattern, full bool)
	visit = func(pat ast.Pattern, full bool) {
		switch n := pat.(type) {
		case *ast.OrPattern:
			for _, alt := range n.Alts {
				visit(alt, full)
			}
		case *ast.LiteralPattern:
			if n.Value.Kind() == value.Enum {
				cover(n.Value.Variant().Shape, n.Value, full)
			}
		case *ast.StructPattern:
			if n.Shape != nil && n.Shape.Parent != nil {
				cover(n.Shape.Parent, value.NewType(n.Shape), full && fieldsIrrefutable(n.Fields))
			}
		case *ast.IsPattern:
			visit(n.Test, full && ast.Irrefutable(n.Inner))
		case *ast.TypeTestPattern:
			switch t := n.Shape; {
			case t == nil:
			case t.Parent != nil:
				cover(t.Parent, value.NewType(t), full)
			case t.Enum && !t.Builtin:
				for _, v := range t.Variants {
					cover(t, v, full)
				}
			}
		}
	}

	for i := range expr.Arms {
		arm := &expr.Arms[i]
		if reachedAll(arm) {
			return
		}

		visit(arm.Pattern, arm.Guard == nil)
	}

	for _, enum := range enums {
		var missing []string
		for _, v := range enum.Variants {
			if !covered[v] {
				missing = append(missing, enum.Name+"."+value.VariantName(v))
			}
		}

		if len(missing) > 0 {
			p.error(expr.Keyword, "Match does not cover "+strings.Join(missing, ", ")+"; add arms or '_ =>'.")
		}
	}
}

func fieldsIrrefutable(fields []ast.FieldPattern) bool {
	for _, f := range fields {
		if !ast.Irrefutable(f.Pattern) {
			return false
		}
	}

	return true
}

// reachedAll reports whether arm matches every value, so no arm after it runs.
func reachedAll(arm *ast.MatchArm) bool {
	return arm.Guard == nil && ast.Irrefutable(arm.Pattern)
}
