package ast

import (
	"fmt"
	"rune/pkg/value"
)

type ExprVisitor interface {
	VisitBinaryExpr(binaryExpr *BinaryExpr) (value.Value, error)
	VisitLiteralExpr(literalExpr *LiteralExpr) (value.Value, error)
	VisitGroupingExpr(literalExpr *GroupingExpr) (value.Value, error)
	VisitUnaryExpr(UnaryExpr *UnaryExpr) (value.Value, error)
	VisitVarExpr(varExpr *VarExpr) (value.Value, error)
	VisitAssignExpr(assignExpr *AssignExpr) (value.Value, error)
	VisitLogicalExpr(logicalExpr *LogicalExpr) (value.Value, error)
	VisitCallExpr(callExpr *CallExpr) (value.Value, error)
	VisitArrayExpr(arrayExpr *ArrayExpr) (value.Value, error)
	VisitIndexExpr(indexExpr *IndexExpr) (value.Value, error)
	VisitSetIndexExpr(setIndexExpr *SetIndexExpr) (value.Value, error)
	VisitObjectExpr(objectExpr *ObjectExpr) (value.Value, error)
	VisitFunctionExpr(functionExpr *FunctionExpr) (value.Value, error)
}

type Expr interface {
	Accept(v ExprVisitor) (value.Value, error)
}

type BinaryExpr struct {
	Left     Expr
	Right    Expr
	Operator Token
}

func NewBinaryExpr(left Expr, right Expr, operator Token) Expr {
	return &BinaryExpr{Left: left, Right: right, Operator: operator}
}

func (n *BinaryExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitBinaryExpr(n)
}

func (n *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %v %v)", n.Operator.Lexeme, n.Left, n.Right)
}

// Unresolved marks a variable reference whose slot is not known yet.
const Unresolved = -1

// AssignExpr assigns to a variable. Depth is the number of scopes between
// the use and the declaration, or Unresolved for a global; Slot is the
// variable's index inside that scope (for globals it is filled lazily).
type AssignExpr struct {
	Name  Token
	Value Expr
	Depth int
	Slot  int
}

func NewAssignExpr(name Token, value Expr) Expr {
	return &AssignExpr{Name: name, Value: value, Depth: Unresolved, Slot: Unresolved}
}

type UnaryExpr struct {
	Right    Expr
	Operator Token
}

func NewUnaryExpr(right Expr, operator Token) Expr {
	return &UnaryExpr{Right: right, Operator: operator}
}

type LiteralExpr struct {
	TokenType TokenType
	Value     value.Value
}

// NewLiteralExpr accepts a plain Go value (nil, bool, float64 or string).
func NewLiteralExpr(tokenType TokenType, v any) Expr {
	return &LiteralExpr{TokenType: tokenType, Value: value.FromGo(v)}
}

// VarExpr reads a variable; Depth and Slot are as in AssignExpr.
type VarExpr struct {
	Name  Token
	Depth int
	Slot  int
}

func NewVarExpr(name Token) Expr {
	return &VarExpr{Name: name, Depth: Unresolved, Slot: Unresolved}
}

type LogicalExpr struct {
	Left  Expr
	Right Expr
	Op    Token
}

func NewLogicalExpr(left Expr, right Expr, op Token) Expr {
	return &LogicalExpr{Left: left, Right: right, Op: op}
}

type GroupingExpr struct {
	Expr Expr
}

func NewGroupingExpr(expr Expr) Expr {
	return &GroupingExpr{Expr: expr}
}

type CallExpr struct {
	Token  Token
	Callee Expr
	Args   []Expr
}

func NewCallExpr(token Token, callee Expr, args []Expr) Expr {
	return &CallExpr{Token: token, Callee: callee, Args: args}
}

type ArrayExpr struct {
	TokenType TokenType
	Items     []Expr
}

func NewArrayExpr(tokenType TokenType, items []Expr) Expr {
	return &ArrayExpr{TokenType: tokenType, Items: items}
}

type IndexExpr struct {
	Token Token
	Array Expr
	Index Expr
}

func NewIndexExpr(array Expr, index Expr, token Token) Expr {
	return &IndexExpr{Array: array, Index: index, Token: token}
}

// SetIndexExpr assigns to an array element or object property. For a
// compound assignment (arr[i] += v) Op is the binary operator to apply to
// the old value; otherwise it is nil. Array and Index are evaluated once.
type SetIndexExpr struct {
	Token Token
	Array Expr
	Index Expr
	Value Expr
	Op    *Token
}

func NewSetIndexExpr(token Token, array Expr, index Expr, value Expr, op *Token) Expr {
	return &SetIndexExpr{Array: array, Index: index, Value: value, Token: token, Op: op}
}

// FunctionExpr is an anonymous function: fun (a, b) { ... }. Its Decl has
// an empty name and is never declared in any scope.
type FunctionExpr struct {
	Decl *FunctionStmt
}

func NewFunctionExpr(decl *FunctionStmt) Expr {
	return &FunctionExpr{Decl: decl}
}

func (n *FunctionExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitFunctionExpr(n)
}

// ObjectPair is a single "key: value" entry of an object literal.
type ObjectPair struct {
	Key   string
	Value Expr
}

type ObjectExpr struct {
	TokenType TokenType
	// Pairs keep source order so values are evaluated left to right.
	Pairs []ObjectPair
}

func NewObjectExpr(tokenType TokenType, pairs []ObjectPair) Expr {
	return &ObjectExpr{TokenType: tokenType, Pairs: pairs}
}

func (n *ObjectExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitObjectExpr(n)
}

func (n *ArrayExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitArrayExpr(n)
}

func (n *IndexExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitIndexExpr(n)
}

func (n *LiteralExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitLiteralExpr(n)
}

func (n *LiteralExpr) String() string {
	if n.Value.IsNil() {
		return "nil"
	}

	if n.Value.IsNumber() {
		l := n.Value.Num()
		// Check if the float is an integer value
		if l == float64(int64(l)) {
			// Print with one decimal place (e.g., 10.0 instead of 10)
			return fmt.Sprintf("%.1f", l)
		} else {
			return fmt.Sprintf("%g", l)
		}
	}

	return fmt.Sprintf("%v", n.Value.Fn())
}

func (n *VarExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitVarExpr(n)
}

func (n *GroupingExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitGroupingExpr(n)
}

func (n *GroupingExpr) String() string {
	return fmt.Sprintf("(group %v)", n.Expr)
}

func (n *UnaryExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitUnaryExpr(n)
}

func (n *UnaryExpr) String() string {
	return fmt.Sprintf("(%s %v)", n.Operator.Lexeme, n.Right)
}

func (n *AssignExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitAssignExpr(n)
}

func (n *AssignExpr) String() string {
	return fmt.Sprintf("(assign %v %v)", n.Name, n.Value)
}

func (n *LogicalExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitLogicalExpr(n)
}

func (n *CallExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitCallExpr(n)
}

func (n *SetIndexExpr) Accept(v ExprVisitor) (value.Value, error) {
	return v.VisitSetIndexExpr(n)
}
