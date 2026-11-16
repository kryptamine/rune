package rune

import (
	"rune/pkg/ast"
	"rune/pkg/value"
)

func (p *Resolver) VisitPatternStmt(stmt *ast.PatternStmt) error {
	for _, b := range ast.Bindings(stmt.Pattern) {
		b.Local = p.declare(*b)
	}

	p.resolveExpr(stmt.Initializer)
	p.resolveDestructuring(stmt.Pattern)

	return nil
}

// resolveDestructuring checks pat only takes a value apart, and defines
// its variables in order, each once its default is resolved.
func (p *Resolver) resolveDestructuring(pat ast.Pattern) {
	switch n := pat.(type) {
	case *ast.WildcardPattern:
	case *ast.BindPattern:
		p.define(n.Name)
	case *ast.DefaultPattern:
		p.resolveExpr(n.Default)
		p.resolveDestructuring(n.Pattern)
	case *ast.ObjectPattern:
		for _, f := range n.Fields {
			p.resolveDestructuring(f.Pattern)
		}

		if n.Rest != nil {
			p.define(n.Rest.Name)
		}
	case *ast.ArrayPattern:
		for _, e := range n.Elems {
			p.resolveDestructuring(e)
		}

		if n.Rest != nil {
			p.define(n.Rest.Name)
		}
	default:
		p.error(patternToken(pat), "This pattern only takes a value apart; test the value with match or 'is'.")
	}
}

// condScope resolves the condition cond and then body, in a scope of
// their own when an is test in cond may bind variables.
func (p *Resolver) condScope(cond ast.Expr, body func()) {
	tests := condTests(cond)

	binds := false
	for _, t := range tests {
		t.Cond = true
		binds = binds || len(ast.Bindings(t.Pattern)) > 0
	}

	if binds {
		p.beginScope()
		defer p.endScope()
	}

	p.resolveExpr(cond)
	body()
}

// condTests returns the is tests of the condition cond that hold when it
// does: those it is made of with &&.
func condTests(cond ast.Expr) []*ast.IsExpr {
	switch n := unwrap(cond).(type) {
	case *ast.IsExpr:
		if !n.Negate {
			return []*ast.IsExpr{n}
		}
	case *ast.LogicalExpr:
		if n.Op.TokenType == ast.AND {
			return append(condTests(n.Left), condTests(n.Right)...)
		}
	}

	return nil
}

func (p *Resolver) VisitIsExpr(expr *ast.IsExpr) (value.Value, error) {
	p.resolveExpr(expr.Value)
	expr.Pattern = p.resolveIsTest(expr.Pattern, true)

	bs := ast.Bindings(expr.Pattern)
	switch {
	case len(bs) == 0:
	case expr.Negate:
		p.error(bs[0].Name, "Cannot bind variables in a '!is' test.")
	case !expr.Cond:
		p.error(bs[0].Name, "Only an if or while condition can bind variables with 'is'.")
	default:
		for _, b := range bs {
			b.Local = p.declare(*b)
			p.define(b.Name)
		}
	}

	return value.NilValue, nil
}

// resolvePattern returns pat with each bare name of a constant replaced by
// a test of its value; any other bare name binds.
func (p *Resolver) resolvePattern(pat ast.Pattern) ast.Pattern {
	switch n := pat.(type) {
	case *ast.BindPattern:
		if l := p.constNamed(n.Name.Lexeme); l != nil && !n.Mutable && !n.Shorthand {
			return p.constPattern(n.Name, l)
		}

		p.checkBind(n)
	case *ast.IsPattern:
		if b, ok := n.Inner.(*ast.BindPattern); ok {
			p.checkBind(b)
		}

		n.Test = p.resolveIsTest(n.Test, false)
	case *ast.DefaultPattern:
		n.Pattern = p.resolvePattern(n.Pattern)
		p.resolveExpr(n.Default)
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

		if n.Rest != nil {
			p.checkBind(n.Rest)
		}
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

// resolveIsTest returns the pattern pat on the right of is, in which a
// bare name is a type; with dynamic, one that names none is evaluated
// when the test runs.
func (p *Resolver) resolveIsTest(pat ast.Pattern, dynamic bool) ast.Pattern {
	switch n := pat.(type) {
	case *ast.OrPattern:
		for i, alt := range n.Alts {
			n.Alts[i] = p.resolveIsTest(alt, dynamic)

			if bs := ast.Bindings(n.Alts[i]); len(bs) > 0 {
				p.error(bs[0].Name, "Cannot bind variables in '|' patterns.")
			}
		}

		return n
	case *ast.TypeTestPattern:
		return p.resolveTypeTest(n, dynamic)
	}

	return p.resolvePattern(pat)
}

// resolveTypeTest returns the test of the type n names, or of the value
// a constant or a variant without fields is.
func (p *Resolver) resolveTypeTest(n *ast.TypeTestPattern, dynamic bool) ast.Pattern {
	t, ok := n.Type.(*ast.VarExpr)
	if !ok {
		p.resolveExpr(n.Type)
		return n
	}

	if p.namesValue(t) {
		return p.resolvePattern(namePattern(t))
	}

	if n.Shape = p.typeNamed(t); n.Shape != nil {
		return n
	}

	if !dynamic && (t.Module == nil || t.Local != nil) {
		p.error(t.Name, "'"+qualified(t.Module, t.Name)+"' is not a type.")
	}

	return n
}

// namesValue reports whether t names a value, which a pattern compares
// with, rather than a type: a constant, or a variant without fields.
func (p *Resolver) namesValue(t *ast.VarExpr) bool {
	if t.Module == nil {
		return p.constNamed(t.Name.Lexeme) != nil
	}

	module := p.isModule(&ast.VarExpr{Name: *t.Module})
	if module && t.Case == nil {
		l := p.mod.imports[t.Module.Lexeme].exports[t.Name.Lexeme]
		return l != nil && l.IsConst
	}

	enum, variant := p.localEnum(*t.Module), t.Name
	if t.Case != nil {
		if !module {
			return false
		}

		enum, variant = p.structs[p.mod.imports[t.Module.Lexeme].exports[t.Name.Lexeme]], *t.Case
	}

	return enum != nil && enum.Enum && hasVariant(enum, variant.Lexeme) && !hasFields(enum, variant.Lexeme)
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

// patternToken returns the token a pattern starts at, for errors.
func patternToken(pat ast.Pattern) ast.Token {
	switch n := pat.(type) {
	case *ast.WildcardPattern:
		return n.Token
	case *ast.BindPattern:
		return n.Name
	case *ast.LiteralPattern:
		return n.Token
	case *ast.RangePattern:
		return n.Op
	case *ast.OrPattern:
		return patternToken(n.Alts[0])
	case *ast.StructPattern:
		return n.Type.Name
	case *ast.VariantPattern:
		return n.Variant
	case *ast.IsPattern:
		return patternToken(n.Inner)
	case *ast.DefaultPattern:
		return patternToken(n.Pattern)
	case *ast.ObjectPattern:
		return n.Brace
	case *ast.ArrayPattern:
		return n.Bracket
	}

	return ast.Token{}
}
