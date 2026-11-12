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

// OrPattern matches if any of Alts does; they bind nothing.
type OrPattern struct {
	Alts []Pattern
}

// FieldPattern matches the property or field Key (by its Lexeme) with Pattern.
type FieldPattern struct {
	Key     Token
	Pattern Pattern
}

// StructPattern tests the value is a Name and matches its Fields; the
// resolver sets Shape.
type StructPattern struct {
	Module *Token
	Name   Token
	Fields []FieldPattern
	Shape  *value.Shape
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

func (*WildcardPattern) patternNode() {}
func (*BindPattern) patternNode()     {}
func (*LiteralPattern) patternNode()  {}
func (*OrPattern) patternNode()       {}
func (*StructPattern) patternNode()   {}
func (*ObjectPattern) patternNode()   {}
func (*TypePattern) patternNode()     {}

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

// MatchArm is pattern [if guard] => body. An expression body is an
// *ExprStmt, whose value is the arm's; any other body gives nil.
type MatchArm struct {
	Token   Token
	Pattern Pattern
	Guard   Expr
	Body    Stmt
}

// MatchExpr is match subject { arms }; no arm matching is an error.
type MatchExpr struct {
	Keyword Token
	Subject Expr
	Arms    []MatchArm
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
