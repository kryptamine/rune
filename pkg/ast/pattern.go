package ast

import "rune/pkg/value"

type Pattern interface {
	patternNode()
}

// WildcardPattern is _, matching anything.
type WildcardPattern struct {
	Token Token
}

// BindPattern binds the value to a variable.
type BindPattern struct {
	Binding
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

// FieldPattern matches the property or field Key (by its Lexeme) with
// Pattern; Shorthand is a bare key, binding the value to its name.
type FieldPattern struct {
	Key       Token
	Pattern   Pattern
	Shorthand bool
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

// TypePattern is name is T | U, or _ is T: Inner, a BindPattern or a
// WildcardPattern, after testing the value is one of Types; the resolver
// sets Shapes.
type TypePattern struct {
	Inner  Pattern
	Types  []*VarExpr
	Shapes []*value.Shape
}

// ObjectPattern is { key: pattern, ... }, matching an object or a struct
// that has every key; other keys are allowed.
type ObjectPattern struct {
	Brace  Token
	Fields []FieldPattern
}

// ArrayPattern is [p, q], matching an array of exactly as many elements,
// or, if Open, [p, q, ...rest] or [p, q, ..._], at least as many; Rest
// binds the others, unless nil.
type ArrayPattern struct {
	Bracket Token
	Elems   []Pattern
	Open    bool
	Rest    *BindPattern
}

func (*WildcardPattern) patternNode() {}
func (*BindPattern) patternNode()     {}
func (*LiteralPattern) patternNode()  {}
func (*RangePattern) patternNode()    {}
func (*OrPattern) patternNode()       {}
func (*StructPattern) patternNode()   {}
func (*ObjectPattern) patternNode()   {}
func (*TypePattern) patternNode()     {}
func (*VariantPattern) patternNode()  {}
func (*ArrayPattern) patternNode()    {}

// Bindings returns the variables p binds; alternatives bind none.
func Bindings(p Pattern) []*Binding {
	var out []*Binding

	var walk func(p Pattern)
	walk = func(p Pattern) {
		switch n := p.(type) {
		case *BindPattern:
			out = append(out, &n.Binding)
		case *TypePattern:
			walk(n.Inner)
		case *StructPattern:
			for _, f := range n.Fields {
				walk(f.Pattern)
			}
		case *ObjectPattern:
			for _, f := range n.Fields {
				walk(f.Pattern)
			}
		case *ArrayPattern:
			for _, e := range n.Elems {
				walk(e)
			}

			if n.Rest != nil {
				out = append(out, &n.Rest.Binding)
			}
		}
	}

	walk(p)

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

// MatchExpr is match subject { arms }; no arm matching is an error.
type MatchExpr struct {
	Keyword Token
	Subject Expr
	Arms    []MatchArm
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
