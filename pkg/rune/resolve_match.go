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

		p.resolveExpr(arm.Guard)

		p.resolveStmt(arm.Body)
		p.endScope()
	}

	p.checkExhaustive(expr)

	return value.NilValue, nil
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
		case *ast.TypePattern:
			full = full && ast.Irrefutable(n.Inner)

			for _, t := range n.Shapes {
				switch {
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

// resolvePattern returns pat with each bare name of a constant replaced by
// a test of its value; any other bare name binds.
func (p *Resolver) resolvePattern(pat ast.Pattern) ast.Pattern {
	switch n := pat.(type) {
	case *ast.BindPattern:
		if l := p.constNamed(n.Name.Lexeme); l != nil && !n.Mutable {
			return p.constPattern(n.Name, l)
		}

		p.checkBind(n)
	case *ast.TypePattern:
		n.Shapes = p.resolveTypes(n.Types)

		if b, ok := n.Inner.(*ast.BindPattern); ok {
			p.checkBind(b)
		}
	case *ast.RangePattern:
		n.Lo, n.Hi = p.rangeBound(n.Op, n.Lo), p.rangeBound(n.Op, n.Hi)

		lo, _ := n.Lo.(*ast.LiteralPattern)
		hi, _ := n.Hi.(*ast.LiteralPattern)

		if lo != nil && hi != nil && lo.Value.IsNumber() && hi.Value.IsNumber() {
			if c, _ := value.CompareNum(lo.Value, hi.Value); c > 0 || c == 0 && !n.Inclusive {
				p.error(n.Op, "Range pattern matches no number.")
			}
		}
	case *ast.OrPattern:
		for i, alt := range n.Alts {
			n.Alts[i] = p.resolvePattern(alt)

			if bs := ast.Bindings(n.Alts[i]); len(bs) > 0 {
				p.error(bs[0].Name, "Cannot bind variables in '|' patterns.")
			}
		}
	case *ast.StructPattern:
		if t := n.Type; t.Module != nil {
			p.resolveExpr(t)
			n.Shape = p.structType(t)
		} else if n.Shape = p.structNamed(t.Name.Lexeme); n.Shape == nil {
			p.error(t.Name, "'"+t.Name.Lexeme+"' is not a struct.")
		}

		if n.Shape != nil && n.Shape.Enum {
			p.error(n.Type.Name, notStruct(n.Shape))
		}

		for _, f := range n.Fields {
			if n.Shape != nil && n.Shape.Index(f.Key.Lexeme) < 0 {
				p.noField(f.Key, n.Shape)
			}
		}

		p.resolveFields(n.Fields)
	case *ast.ObjectPattern:
		p.resolveFields(n.Fields)
	case *ast.ArrayPattern:
		for i, e := range n.Elems {
			n.Elems[i] = p.resolvePattern(e)
		}

		if n.Rest != nil {
			p.checkBind(n.Rest)
		}
	case *ast.VariantPattern:
		return p.resolveVariantPattern(n)
	}

	return pat
}

// rangeBound returns the bound b of the range pattern at op, if it has
// one, as the test of its number.
func (p *Resolver) rangeBound(op ast.Token, b ast.Pattern) ast.Pattern {
	if b == nil {
		return nil
	}

	errs := len(p.errors)
	b = p.resolvePattern(b)

	// A name that is no constant is reported once, by what it is not.
	if lit, ok := b.(*ast.LiteralPattern); (!ok || !lit.Value.IsNumber()) && len(p.errors) == errs {
		p.error(op, "A bound of a range pattern must be a number or a constant holding one.")
	}

	return b
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
		if f := &fields[i]; f.Shorthand {
			p.checkBind(f.Pattern.(*ast.BindPattern))
		} else {
			f.Pattern = p.resolvePattern(f.Pattern)
		}
	}
}

func (p *Resolver) checkBind(n *ast.BindPattern) {
	if name := n.Name.Lexeme; p.structNamed(name) != nil {
		p.error(n.Name, "'"+name+"' is a struct type; test it with '_ is "+name+"'.")
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
