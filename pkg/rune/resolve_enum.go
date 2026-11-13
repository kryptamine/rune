package rune

import (
	"rune/pkg/ast"
	"rune/pkg/value"
)

// variantOf returns the variant Enum.Variant names, for an enum of this file.
func (p *Resolver) variantOf(n *ast.IndexExpr) (value.Value, bool) {
	enum, isVar := n.Array.(*ast.VarExpr)
	key, isLiteral := n.Index.(*ast.LiteralExpr)

	if !n.Property || !isVar || !isLiteral || enum.Module != nil || p.lookupScopes(enum.Name.Lexeme) != nil {
		return value.NilValue, false
	}

	shape := p.structs[p.mod.topLevel[enum.Name.Lexeme]]
	if shape == nil || !shape.Enum {
		return value.NilValue, false
	}

	return shape.Variant(key.Value.Str())
}

// resolveVariant binds Enum.Variant to the variant, a constant; any other
// name of an enum must be one of its methods.
func (p *Resolver) resolveVariant(expr *ast.IndexExpr) {
	shape := p.enumOf(expr.Array)
	key, isLiteral := expr.Index.(*ast.LiteralExpr)

	if shape == nil || !expr.Property || !isLiteral {
		return
	}

	name := key.Value.Str()
	if v, ok := shape.Variant(name); ok {
		expr.Member = &v
		return
	}

	if name != enumValues && value.FindMethod(shape.Methods, name) == nil {
		p.error(expr.Token, shape.Name+" has no variant '"+name+"'.")
	}
}

// enumOf returns the enum type e names, as Color or m.Color, or nil.
func (p *Resolver) enumOf(e ast.Expr) *value.Shape {
	var shape *value.Shape

	switch n := e.(type) {
	case *ast.VarExpr:
		shape = p.structs[n.Local]
	case *ast.IndexExpr:
		if n.Export != nil {
			shape = p.structs[n.Export.Local]
		}
	}

	if shape != nil && shape.Enum {
		return shape
	}

	return nil
}

func notStruct(enum *value.Shape) string {
	msg := "'" + enum.Name + "' is an enum, not a struct"
	if len(enum.Variants) > 0 {
		msg += "; its values are " + enum.Name + "." + enum.Variants[0].Variant().Name + " and so on"
	}

	return msg + "."
}

// resolveVariantPattern returns the literal test of the variant n names.
func (p *Resolver) resolveVariantPattern(n *ast.VariantPattern) ast.Pattern {
	// A refutable stand-in for a mistake, so no later arm is reported unreachable.
	missing := &ast.LiteralPattern{Token: n.Variant}
	t := &ast.VarExpr{Module: n.Module, Name: n.Enum}

	if n.Module == nil && p.isModule(t) {
		p.error(n.Variant, "Expect '{' after struct name; test the type with '_ is "+n.Enum.Lexeme+"."+n.Variant.Lexeme+"'.")
		return missing
	}

	p.resolveExpr(t)

	enum := p.enumOf(t)
	if enum == nil {
		if n.Module == nil || t.Local != nil {
			p.error(n.Enum, "'"+qualified(n.Module, n.Enum)+"' is not an enum.")
		}

		return missing
	}

	v, ok := enum.Variant(n.Variant.Lexeme)
	if !ok {
		p.error(n.Variant, enum.Name+" has no variant '"+n.Variant.Lexeme+"'.")
		return missing
	}

	return &ast.LiteralPattern{Token: n.Variant, Value: v}
}

func hasVariant(enum *value.Shape, name string) bool {
	_, ok := enum.Variant(name)
	return ok
}
