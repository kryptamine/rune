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
		msg += "; its variants are " + enum.Name + "." + value.VariantName(enum.Variants[0]) + " and so on"
	}

	return msg + "."
}

// resolveVariantPattern returns the literal test of the variant n names.
func (p *Resolver) resolveVariantPattern(n *ast.VariantPattern) ast.Pattern {
	// A refutable stand-in for a mistake, so no later arm is reported unreachable.
	missing := &ast.LiteralPattern{Token: n.Variant}
	t := &ast.VarExpr{Module: n.Module, Name: n.Enum}

	if n.Module == nil && p.isModule(t) {
		if l := p.mod.imports[n.Enum.Lexeme].exports[n.Variant.Lexeme]; l != nil && l.IsConst {
			return p.constPattern(n.Variant, l)
		}

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

	if v.Kind() == value.Type {
		name := enum.Name + "." + n.Variant.Lexeme
		p.error(n.Variant, "'"+name+"' has fields; match it as "+name+" { ... } or _ is "+name+".")
		return missing
	}

	return &ast.LiteralPattern{Token: n.Variant, Value: v}
}

func hasVariant(enum *value.Shape, name string) bool {
	_, ok := enum.Variant(name)
	return ok
}

// shapeEnum makes the enum s declares, and a constant of each case's type.
func (p *Resolver) shapeEnum(s *ast.StructStmt) *value.Shape {
	names := make([]string, len(s.Variants))
	fields := make([]bool, len(s.Variants))

	for i, v := range s.Variants {
		names[i], fields[i] = v.Name.Lexeme, v.HasFields
	}

	enum := value.NewEnum(s.Name.Lexeme, names, fields)

	for c := range enum.Cases() {
		l := &ast.Local{TopLevel: true, IsConst: true, Const: value.NewType(c)}
		p.cases[c] = l
		p.structs[l] = c
	}

	return enum
}

// defineCases gives the cases of the enum s their fields.
func (p *Resolver) defineCases(s *ast.StructStmt) {
	for i, v := range s.Variants {
		if v.HasFields {
			s.Shape.Variants[i].TypeShape().Define(p.fields(v.Fields))
		}
	}
}

// localEnum returns the enum of this file name names, or nil.
func (p *Resolver) localEnum(name ast.Token) *value.Shape {
	if p.lookupScopes(name.Lexeme) != nil || p.mod.imports[name.Lexeme] != nil {
		return nil
	}

	if shape := p.structs[p.mod.topLevel[name.Lexeme]]; shape != nil && shape.Enum {
		return shape
	}

	return nil
}

// caseNamed returns the constant of the case Enum.Case, reporting it if
// enum is no enum or has no such case.
func (p *Resolver) caseNamed(enum *value.Shape, name, variant ast.Token) *ast.Local {
	if enum == nil || !enum.Enum {
		p.error(name, "'"+name.Lexeme+"' is not an enum.")
		return nil
	}

	v, ok := enum.Variant(variant.Lexeme)
	switch {
	case !ok:
		p.error(variant, enum.Name+" has no variant '"+variant.Lexeme+"'.")
	case v.Kind() != value.Type:
		p.error(variant, "'"+enum.Name+"."+variant.Lexeme+"' has no fields; it is a value, not a type.")
	default:
		return p.cases[v.TypeShape()]
	}

	return nil
}

// fieldOwner returns the struct, or the case of the enum, shape that has
// a field name, or nil.
func fieldOwner(shape *value.Shape, name string) *value.Shape {
	if shape == nil {
		return nil
	}

	if shape.Index(name) >= 0 {
		return shape
	}

	for c := range shape.Cases() {
		if c.Index(name) >= 0 {
			return c
		}
	}

	return nil
}
