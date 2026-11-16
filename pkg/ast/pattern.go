package ast

import "rune/pkg/value"

type Pattern interface {
	patternNode()
}

// WildcardPattern is _, matching anything.
type WildcardPattern struct {
	Token Token
}

// BindPattern binds the value to a variable; Shorthand is a bare key of
// a field pattern, which binds even the name of a constant.
type BindPattern struct {
	Binding
	Shorthand bool
}

type LiteralPattern struct {
	Token Token
	Value value.Value
}

// RangePattern is lo..hi, lo..=hi, lo.. or ..hi, matching a number within
// the bounds; the resolver makes each bound a LiteralPattern.
type RangePattern struct {
	Op        Token
	Lo, Hi    Pattern
	Inclusive bool
}

// OrPattern matches if any of Alts does; they bind nothing.
type OrPattern struct {
	Alts []Pattern
}

// FieldPattern matches the property or field Key (by its Lexeme) with Pattern.
type FieldPattern struct {
	Key     Token
	Pattern Pattern
}

// StructPattern tests the value is a Type and matches its Fields; the
// resolver sets Shape.
type StructPattern struct {
	Type   *VarExpr
	Fields []FieldPattern
	Shape  *value.Shape
}

// VariantPattern is Enum.Variant, or m.Enum.Variant, matching that
// variant; the resolver replaces it with a LiteralPattern.
type VariantPattern struct {
	Module  *Token
	Enum    Token
	Variant Token
}

// IsPattern is name is P, or _ is P: Inner, a BindPattern or a
// WildcardPattern, takes the value once it matches Test.
type IsPattern struct {
	Inner Pattern
	Test  Pattern
}

// TypeTestPattern is a type on the right of is, matching a value of that
// type; the resolver sets Shape, or in an is expression leaves it nil for
// a Type evaluated when it runs.
type TypeTestPattern struct {
	Token Token
	Type  Expr
	Shape *value.Shape
}

// DefaultPattern is P = default, an element of an array pattern or the
// pattern of a field: a missing value is taken as Default.
type DefaultPattern struct {
	Pattern Pattern
	Default Expr
}

// ObjectPattern is { key: pattern, ..., ...rest }, matching an object or a
// struct that has every key without a default; other keys are allowed and
// Rest, unless nil, binds an object of them.
type ObjectPattern struct {
	Brace  Token
	Fields []FieldPattern
	Rest   *BindPattern
}

// ArrayPattern is [p, q], matching an array of exactly as many elements,
// or, if Open, [p, q, ...rest] or [p, q, ..._], at least as many; Rest
// binds the others, unless nil. Elements with a default may be missing.
type ArrayPattern struct {
	Bracket Token
	Elems   []Pattern
	Open    bool
	Rest    *BindPattern
}

// Required is the number of elements the value must have: those before
// the first with a default.
func (n *ArrayPattern) Required() int {
	for i, e := range n.Elems {
		if _, ok := e.(*DefaultPattern); ok {
			return i
		}
	}

	return len(n.Elems)
}

func (*WildcardPattern) patternNode() {}
func (*BindPattern) patternNode()     {}
func (*LiteralPattern) patternNode()  {}
func (*RangePattern) patternNode()    {}
func (*OrPattern) patternNode()       {}
func (*StructPattern) patternNode()   {}
func (*ObjectPattern) patternNode()   {}
func (*IsPattern) patternNode()       {}
func (*TypeTestPattern) patternNode() {}
func (*DefaultPattern) patternNode()  {}
func (*VariantPattern) patternNode()  {}
func (*ArrayPattern) patternNode()    {}

// Walk calls visit on p and, while visit returns true, on the patterns
// matched against parts of the value, in the order they match.
func Walk(p Pattern, visit func(Pattern) bool) {
	if !visit(p) {
		return
	}

	fields := func(fs []FieldPattern) {
		for _, f := range fs {
			Walk(f.Pattern, visit)
		}
	}

	switch n := p.(type) {
	case *OrPattern:
		for _, alt := range n.Alts {
			Walk(alt, visit)
		}
	case *IsPattern:
		Walk(n.Test, visit)
		Walk(n.Inner, visit)
	case *DefaultPattern:
		Walk(n.Pattern, visit)
	case *StructPattern:
		fields(n.Fields)
	case *ObjectPattern:
		fields(n.Fields)

		if n.Rest != nil {
			Walk(n.Rest, visit)
		}
	case *ArrayPattern:
		for _, e := range n.Elems {
			Walk(e, visit)
		}

		if n.Rest != nil {
			Walk(n.Rest, visit)
		}
	}
}

// Bindings returns the variables p binds; alternatives bind none.
func Bindings(p Pattern) []*Binding {
	var out []*Binding

	Walk(p, func(p Pattern) bool {
		switch n := p.(type) {
		case *OrPattern:
			return false
		case *BindPattern:
			out = append(out, &n.Binding)
		}

		return true
	})

	return out
}

// Exprs returns the expressions p evaluates, in order: defaults and types
// evaluated when it runs.
func Exprs(p Pattern) []Expr {
	var out []Expr

	Walk(p, func(p Pattern) bool {
		switch n := p.(type) {
		case *TypeTestPattern:
			if n.Shape == nil {
				out = append(out, n.Type)
			}
		case *DefaultPattern:
			out = append(out, n.Default)
		}

		return true
	})

	return out
}

// Irrefutable reports whether p matches every value.
func Irrefutable(p Pattern) bool {
	switch n := p.(type) {
	case *WildcardPattern, *BindPattern:
		return true
	case *OrPattern:
		for _, alt := range n.Alts {
			if Irrefutable(alt) {
				return true
			}
		}
	}

	return false
}

// MatchArm is pattern [if guard] => body; an expression body is a valued
// block of it alone.
type MatchArm struct {
	Token   Token
	Pattern Pattern
	Guard   Expr
	Body    *BlockStmt
}

// MatchExpr is match subject { arms }; no arm matching is an error. With
// Enum, as in match subject is Enum { arms }, the subject must be one of
// its variants, which the arms name bare; the resolver sets EnumShape.
type MatchExpr struct {
	Keyword   Token
	Subject   Expr
	Enum      *VarExpr
	EnumShape *value.Shape
	Arms      []MatchArm
}

// Values returns the value of each arm, nil for one without.
func (n *MatchExpr) Values() []Expr {
	vs := make([]Expr, len(n.Arms))
	for i := range n.Arms {
		vs[i] = n.Arms[i].Body.Value()
	}

	return vs
}

func (n *MatchExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitMatchExpr(n)
}

// MatchStmt is a match at the start of a statement: its value is dropped
// and no arm matching does nothing.
type MatchStmt struct {
	Match *MatchExpr
}

func (n *MatchStmt) Accept(v StmtVisitor) error {
	return v.VisitMatchStmt(n)
}
