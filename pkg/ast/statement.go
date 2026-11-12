package ast

import (
	"rune/pkg/value"
	"slices"
)

type StmtVisitor interface {
	VisitExprStmt(exprStmt *ExprStmt) error
	VisitVarStmt(VarStmt *VarStmt) error
	VisitDestructureStmt(stmt *DestructureStmt) error
	VisitArrayDestructureStmt(stmt *ArrayDestructureStmt) error
	VisitBlockStmt(blockStmt *BlockStmt) error
	VisitIfStmt(ifStmt *IfStmt) error
	VisitWhileStmt(whileStmt *WhileStmt) error
	VisitFunctionStmt(functionStmt *FunctionStmt) error
	VisitReturnStmt(returnStmt *ReturnStmt) error
	VisitTryStmt(tryStmt *TryStmt) error
	VisitBreakStmt(breakStmt *BreakStmt) error
	VisitContinueStmt(continueStmt *ContinueStmt) error
	VisitForInStmt(forInStmt *ForInStmt) error
	VisitStructStmt(structStmt *StructStmt) error
	VisitMatchStmt(matchStmt *MatchStmt) error
	VisitImplStmt(implStmt *ImplStmt) error
}

// Binding is a declared name; Mutable is set for names declared with "mut".
// The resolver sets Local unless the name is a global.
type Binding struct {
	Name    Token
	Mutable bool
	Local   *Local
}

// Discards reports whether b is _, which takes a value and declares nothing.
func (b Binding) Discards() bool { return b.Name.Lexeme == "_" }

type VarStmt struct {
	Binding
	Initializer Expr
}

func NewVarStmt(initializer Expr, binding Binding) Stmt {
	return &VarStmt{Initializer: initializer, Binding: binding}
}

// PatternVar is a variable a pattern binds, and its default, if any, taken
// when the value has no such property or element.
type PatternVar struct {
	Binding
	Default Expr
}

// DestructureField binds the property Key (by its Lexeme) of the value.
type DestructureField struct {
	Key Token
	PatternVar
}

// DestructureStmt is let { a, mut b = 1, c: d } = value;
type DestructureStmt struct {
	Fields      []DestructureField
	Initializer Expr
}

// ArrayDestructureStmt is let [a, _, mut b = 1, ...rest] = value; Elems
// binds the elements in order, nil for _, and Rest, if any, the others.
type ArrayDestructureStmt struct {
	Bracket     Token
	Elems       []*PatternVar
	Rest        *PatternVar
	Initializer Expr
}

// Required is the number of elements the value must have: those before
// the first with a default.
func (s *ArrayDestructureStmt) Required() int {
	for i, e := range s.Elems {
		if e != nil && e.Default != nil {
			return i
		}
	}

	return len(s.Elems)
}

// PatternVars returns the variables a destructuring stmt binds, in order.
func PatternVars(stmt Stmt) []*PatternVar {
	var vs []*PatternVar

	switch s := stmt.(type) {
	case *DestructureStmt:
		for i := range s.Fields {
			vs = append(vs, &s.Fields[i].PatternVar)
		}
	case *ArrayDestructureStmt:
		for _, e := range s.Elems {
			if e != nil {
				vs = append(vs, e)
			}
		}

		if s.Rest != nil {
			vs = append(vs, s.Rest)
		}
	}

	return vs
}

// Declared returns the variables stmt declares, structs and discards aside.
func Declared(stmt Stmt) []*Binding {
	var bs []*Binding

	switch s := stmt.(type) {
	case *VarStmt:
		bs = []*Binding{&s.Binding}
	case *FunctionStmt:
		bs = []*Binding{&s.Binding}
	case *DestructureStmt, *ArrayDestructureStmt:
		for _, v := range PatternVars(s) {
			bs = append(bs, &v.Binding)
		}
	}

	return slices.DeleteFunc(bs, func(b *Binding) bool { return b.Discards() })
}

type ReturnStmt struct {
	Value   Expr
	Keyword Token
}

func NewReturnStmt(value Expr, keyword Token) Stmt {
	return &ReturnStmt{Value: value, Keyword: keyword}
}

type ExprStmt struct {
	Expr Expr
}

func NewExprStmt(expr Expr) Stmt {
	return &ExprStmt{Expr: expr}
}

type BlockStmt struct {
	Stmts []Stmt
}

func NewBlockStmt(stmts []Stmt) Stmt {
	return &BlockStmt{Stmts: stmts}
}

type IfStmt struct {
	Condition Expr
	Then      Stmt
	El        Stmt
}

func NewIfStmt(condition Expr, then Stmt, el Stmt) Stmt {
	return &IfStmt{Condition: condition, Then: then, El: el}
}

type WhileStmt struct {
	Condition Expr
	Body      Stmt
}

func NewWhileStmt(condition Expr, body Stmt) Stmt {
	return &WhileStmt{Condition: condition, Body: body}
}

// ForInStmt iterates over an array (elements), a string (characters) or an
// object (keys, in sorted order). With two variables, Key gets the index
// (or the object key) and Value the element. Each iteration binds them anew.
type ForInStmt struct {
	Keyword  Token
	Key      *Binding
	Value    Binding
	Iterable Expr
	Body     Stmt
}

func NewForInStmt(keyword Token, key *Binding, val Binding, iterable Expr, body Stmt) Stmt {
	return &ForInStmt{Keyword: keyword, Key: key, Value: val, Iterable: iterable, Body: body}
}

type BreakStmt struct {
	Keyword Token
}

func NewBreakStmt(keyword Token) Stmt {
	return &BreakStmt{Keyword: keyword}
}

type ContinueStmt struct {
	Keyword Token
}

func NewContinueStmt(keyword Token) Stmt {
	return &ContinueStmt{Keyword: keyword}
}

// FunctionStmt declares a function; its Binding is never mutable. With
// Rest, the last parameter takes the arguments after the others as an array.
// Defaults holds each parameter's default, nil for a required one.
type FunctionStmt struct {
	Binding
	Parameters []Binding
	Defaults   []Expr
	Rest       bool
	Body       []Stmt
}

// Required counts the parameters before the first with a default, a rest
// one excluded.
func (n *FunctionStmt) Required() int {
	for i, d := range n.Defaults {
		if d != nil {
			return i
		}
	}

	if n.Rest {
		return len(n.Parameters) - 1
	}

	return len(n.Parameters)
}

func NewFunctionStmt(name Token, parameters []Binding, body []Stmt) *FunctionStmt {
	return &FunctionStmt{Binding: Binding{Name: name}, Parameters: parameters, Body: body}
}

// TryStmt runs Body and, if it fails with a runtime error or a throw, runs
// the first catch clause that matches.
type TryStmt struct {
	Body    Stmt
	Catches []CatchClause
}

// CatchClause catches errors that are one of Types, or any without Types.
// Param is the optional catch variable (nil without one).
// CatchClause is catch [name [is T | U]] { body }; the resolver sets
// Shapes, one per type.
type CatchClause struct {
	Keyword    Token
	Param      *Token
	ParamLocal *Local
	Types      []*VarExpr
	Shapes     []*value.Shape
	Body       []Stmt
}

func NewTryStmt(body Stmt, catches []CatchClause) Stmt {
	return &TryStmt{Body: body, Catches: catches}
}

// StructField is a field of a struct declaration; Default, at DefaultAt,
// is nil for a required field.
type StructField struct {
	Name      Token
	Default   Expr
	DefaultAt Token
}

// StructStmt declares a struct type; the resolver sets Local and Shape.
type StructStmt struct {
	Name   Token
	Fields []StructField
	Local  *Local
	Shape  *value.Shape
}

func NewStructStmt(name Token, fields []StructField) Stmt {
	return &StructStmt{Name: name, Fields: fields}
}

// ImplStmt adds methods to a struct of its file or a built-in type.
type ImplStmt struct {
	Type    Token
	Methods []ImplMethod
}

// ImplMethod is a method declaration; the resolver sets Method.
type ImplMethod struct {
	Fn *FunctionStmt
	value.Method
}

type Stmt interface {
	Accept(v StmtVisitor) error
}

func (n *FunctionStmt) Accept(v StmtVisitor) error {
	return v.VisitFunctionStmt(n)
}

func (n *ExprStmt) Accept(v StmtVisitor) error {
	return v.VisitExprStmt(n)
}

func (n *VarStmt) Accept(v StmtVisitor) error {
	return v.VisitVarStmt(n)
}

func (n *DestructureStmt) Accept(v StmtVisitor) error {
	return v.VisitDestructureStmt(n)
}

func (n *ArrayDestructureStmt) Accept(v StmtVisitor) error {
	return v.VisitArrayDestructureStmt(n)
}

func (n *BlockStmt) Accept(v StmtVisitor) error {
	return v.VisitBlockStmt(n)
}

func (n *IfStmt) Accept(v StmtVisitor) error {
	return v.VisitIfStmt(n)
}

func (n *WhileStmt) Accept(v StmtVisitor) error {
	return v.VisitWhileStmt(n)
}

func (n *ReturnStmt) Accept(v StmtVisitor) error {
	return v.VisitReturnStmt(n)
}

func (n *TryStmt) Accept(v StmtVisitor) error {
	return v.VisitTryStmt(n)
}

func (n *BreakStmt) Accept(v StmtVisitor) error {
	return v.VisitBreakStmt(n)
}

func (n *ContinueStmt) Accept(v StmtVisitor) error {
	return v.VisitContinueStmt(n)
}

func (n *ForInStmt) Accept(v StmtVisitor) error {
	return v.VisitForInStmt(n)
}

func (n *StructStmt) Accept(v StmtVisitor) error {
	return v.VisitStructStmt(n)
}

func (n *ImplStmt) Accept(v StmtVisitor) error {
	return v.VisitImplStmt(n)
}
