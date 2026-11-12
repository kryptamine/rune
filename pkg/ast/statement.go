package ast

import "rune/pkg/value"

type StmtVisitor interface {
	VisitPrintStmt(printStmt *PrintStmt) error
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
	VisitThrowStmt(throwStmt *ThrowStmt) error
	VisitBreakStmt(breakStmt *BreakStmt) error
	VisitContinueStmt(continueStmt *ContinueStmt) error
	VisitForInStmt(forInStmt *ForInStmt) error
	VisitStructStmt(structStmt *StructStmt) error
	VisitMatchStmt(matchStmt *MatchStmt) error
}

// Binding is a declared name; Mutable is set for names declared with "mut".
// The resolver sets Local unless the name is a global.
type Binding struct {
	Name    Token
	Mutable bool
	Local   *Local
}

type VarStmt struct {
	Binding
	Initializer Expr
}

func NewVarStmt(initializer Expr, binding Binding) Stmt {
	return &VarStmt{Initializer: initializer, Binding: binding}
}

// DestructureField binds the property Key (by its Lexeme) of the value.
type DestructureField struct {
	Key Token
	Binding
}

// DestructureStmt is let { a, mut b, c: d } = value;
type DestructureStmt struct {
	Fields      []DestructureField
	Initializer Expr
}

// ArrayDestructureStmt is let [a, _, mut b, ...rest] = value; Elems
// binds the elements in order, nil for _, and Rest, if any, the others.
type ArrayDestructureStmt struct {
	Bracket     Token
	Elems       []*Binding
	Rest        *Binding
	Initializer Expr
}

// Declared returns the variables stmt declares, structs aside.
func Declared(stmt Stmt) []*Binding {
	switch s := stmt.(type) {
	case *VarStmt:
		return []*Binding{&s.Binding}
	case *FunctionStmt:
		return []*Binding{&s.Binding}
	case *DestructureStmt:
		bs := make([]*Binding, len(s.Fields))
		for i := range s.Fields {
			bs[i] = &s.Fields[i].Binding
		}

		return bs
	case *ArrayDestructureStmt:
		var bs []*Binding
		for _, b := range s.Elems {
			if b != nil {
				bs = append(bs, b)
			}
		}

		if s.Rest != nil {
			bs = append(bs, s.Rest)
		}

		return bs
	}

	return nil
}

type ReturnStmt struct {
	Value   Expr
	Keyword Token
}

func NewReturnStmt(value Expr, keyword Token) Stmt {
	return &ReturnStmt{Value: value, Keyword: keyword}
}

type PrintStmt struct {
	Expr Expr
}

func NewPrintStmt(expr Expr) Stmt {
	return &PrintStmt{Expr: expr}
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
type FunctionStmt struct {
	Binding
	Parameters []Binding
	Rest       bool
	Body       []Stmt
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
type CatchClause struct {
	Keyword    Token
	Param      *Token
	ParamLocal *Local
	Types      []*VarExpr
	Body       []Stmt
}

func NewTryStmt(body Stmt, catches []CatchClause) Stmt {
	return &TryStmt{Body: body, Catches: catches}
}

type ThrowStmt struct {
	Value   Expr
	Keyword Token
}

func NewThrowStmt(value Expr, keyword Token) Stmt {
	return &ThrowStmt{Value: value, Keyword: keyword}
}

// StructField is a field of a struct declaration; Default is a literal,
// or nil for a required field.
type StructField struct {
	Name    Token
	Default Expr
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

type Stmt interface {
	Accept(v StmtVisitor) error
}

func (n *FunctionStmt) Accept(v StmtVisitor) error {
	return v.VisitFunctionStmt(n)
}

func (n *PrintStmt) Accept(v StmtVisitor) error {
	return v.VisitPrintStmt(n)
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

func (n *ThrowStmt) Accept(v StmtVisitor) error {
	return v.VisitThrowStmt(n)
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
