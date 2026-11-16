package ast

import (
	"rune/pkg/value"
	"slices"
)

type StmtVisitor interface {
	VisitExprStmt(exprStmt *ExprStmt) error
	VisitVarStmt(VarStmt *VarStmt) error
	VisitPatternStmt(stmt *PatternStmt) error
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
	VisitDeferStmt(deferStmt *DeferStmt) error
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

// VarStmt is let name = value; or, if Const, const NAME = value; whose
// value the resolver computes.
type VarStmt struct {
	Binding
	Initializer Expr
	Const       bool
}

func NewVarStmt(initializer Expr, binding Binding) Stmt {
	return &VarStmt{Initializer: initializer, Binding: binding}
}

// PatternStmt is let pattern = value; and the destructuring of a pattern
// parameter or for-in variable: each part the pattern names must be there.
type PatternStmt struct {
	Pattern     Pattern
	Initializer Expr
}

func (n *PatternStmt) Accept(v StmtVisitor) error {
	return v.VisitPatternStmt(n)
}

// Declared returns the variables stmt declares, structs and discards aside.
func Declared(stmt Stmt) []*Binding {
	var bs []*Binding

	switch s := stmt.(type) {
	case *VarStmt:
		bs = []*Binding{&s.Binding}
	case *FunctionStmt:
		bs = []*Binding{&s.Binding}
	case *PatternStmt:
		bs = Bindings(s.Pattern)
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

// BlockStmt is { stmts }; if Valued, its last statement is an *ExprStmt
// written without ';', whose value is the block's.
type BlockStmt struct {
	Stmts  []Stmt
	Valued bool
}

// Value is the expression whose value the block gives, or nil.
func (b *BlockStmt) Value() Expr {
	if !b.Valued {
		return nil
	}

	return b.Stmts[len(b.Stmts)-1].(*ExprStmt).Expr
}

func NewBlockStmt(stmts []Stmt) Stmt {
	return &BlockStmt{Stmts: stmts}
}

// ValuedStmt is a statement that gives a value as the last of a block
// with one: if, match and try.
type ValuedStmt interface {
	Stmt
	// Values returns the value of each branch, nil for one without.
	Values() []Expr
	// AsExpr is the statement as an expression giving that value.
	AsExpr() Expr
}

// IfStmt is if cond { then } [else ...]; El is nil, a *BlockStmt or an
// *IfStmt.
type IfStmt struct {
	Condition Expr
	Then      *BlockStmt
	El        Stmt
}

func (n *IfStmt) AsExpr() Expr { return &IfExpr{If: n} }

// Values counts a missing else as a branch without a value.
func (n *IfStmt) Values() []Expr {
	vs := []Expr{n.Then.Value()}

	switch el := n.El.(type) {
	case *BlockStmt:
		return append(vs, el.Value())
	case *IfStmt:
		return append(vs, el.Values()...)
	}

	return append(vs, nil)
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
	// Alone is set on a for mut whose body never names the variable the
	// collection is read from, by the compiler's liveness walk.
	Alone bool
}

func NewForInStmt(keyword Token, key *Binding, val Binding, iterable Expr, body Stmt) Stmt {
	return &ForInStmt{Keyword: keyword, Key: key, Value: val, Iterable: iterable, Body: body}
}

// DeferStmt runs Body when the block it is in is left, however it is.
type DeferStmt struct {
	Keyword Token
	Body    Stmt
}

func (n *DeferStmt) Accept(v StmtVisitor) error {
	return v.VisitDeferStmt(n)
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

// Inout has a bit for each mut parameter, self and a rest one aside.
func (n *FunctionStmt) Inout() uint32 {
	var mask uint32
	for i, p := range n.Parameters {
		if p.Mutable && !(i == 0 && p.Name.Lexeme == "self") && !(n.Rest && i == len(n.Parameters)-1) {
			mask |= 1 << i
		}
	}

	return mask
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
	Body    *BlockStmt
	Catches []CatchClause
}

func (n *TryStmt) AsExpr() Expr { return &TryExpr{Try: n} }

// Values gives the value of the body, then of each catch.
func (n *TryStmt) Values() []Expr {
	vs := []Expr{n.Body.Value()}
	for _, c := range n.Catches {
		vs = append(vs, c.Body.Value())
	}

	return vs
}

// CatchClause is catch [name [is T | U]] { body }; without Types it
// catches any error. The resolver sets Shapes, one per type.
type CatchClause struct {
	Keyword    Token
	Param      *Token
	ParamLocal *Local
	Types      []*VarExpr
	Shapes     []*value.Shape
	Body       *BlockStmt
}

// StructField is a field of a struct declaration; Default, at DefaultAt,
// is nil for a required field.
type StructField struct {
	Name      Token
	Default   Expr
	DefaultAt Token
}

// StructStmt declares a struct type, or an enum type of Variants if
// Enum; the resolver sets Local and Shape.
type StructStmt struct {
	Name     Token
	Fields   []StructField
	Enum     bool
	Variants []EnumVariant
	Local    *Local
	Shape    *value.Shape
}

// EnumVariant is a variant of an enum declaration, with Fields if
// HasFields.
type EnumVariant struct {
	Name      Token
	Fields    []StructField
	HasFields bool
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
