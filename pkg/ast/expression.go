package ast

import "fmt"

type ExprVisitor interface {
	VisitBinaryExpr(binaryExpr *BinaryExpr) (any, error)
	VisitLiteralExpr(literalExpr *LiteralExpr) (any, error)
	VisitGroupingExpr(literalExpr *GroupingExpr) (any, error)
	VisitUnaryExpr(UnaryExpr *UnaryExpr) (any, error)
	VisitVarExpr(varExpr *VarExpr) (any, error)
	VisitAssignExpr(assignExpr *AssignExpr) (any, error)
	VisitLogicalExpr(logicalExpr *LogicalExpr) (any, error)
	VisitCallExpr(callExpr *CallExpr) (any, error)
	VisitArrayExpr(arrayExpr *ArrayExpr) (any, error)
	VisitIndexExpr(indexExpr *IndexExpr) (any, error)
	VisitSetIndexExpr(setIndexExpr *SetIndexExpr) (any, error)
	VisitObjectExpr(objectExpr *ObjectExpr) (any, error)
}

type Expr interface {
	Accept(v ExprVisitor) (any, error)
}

type BinaryExpr struct {
	Left     Expr
	Right    Expr
	Operator Token
}

func NewBinaryExpr(left Expr, right Expr, operator Token) Expr {
	return &BinaryExpr{Left: left, Right: right, Operator: operator}
}

func (n *BinaryExpr) Accept(v ExprVisitor) (any, error) {
	return v.VisitBinaryExpr(n)
}

func (n *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %v %v)", n.Operator.Lexeme, n.Left, n.Right)
}

type AssignExpr struct {
	Name  Token
	Value Expr
}

func NewAssignExpr(name Token, value Expr) Expr {
	return &AssignExpr{Name: name, Value: value}
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
	Value     any
}

func NewLiteralExpr(tokenType TokenType, value any) Expr {
	return &LiteralExpr{TokenType: tokenType, Value: value}
}

type VarExpr struct {
	Name Token
}

func NewVarExpr(name Token) Expr {
	return &VarExpr{Name: name}
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

type SetIndexExpr struct {
	Token Token
	Array Expr
	Index Expr
	Value Expr
}

func NewSetIndexExpr(token Token, array Expr, index Expr, value Expr) Expr {
	return &SetIndexExpr{Array: array, Index: index, Value: value, Token: token}
}

type ObjectExpr struct {
	TokenType TokenType
	Pairs     map[string]Expr
}

func NewObjectExpr(tokenType TokenType, pairs map[string]Expr) Expr {
	return &ObjectExpr{TokenType: tokenType, Pairs: pairs}
}

func (n *ObjectExpr) Accept(v ExprVisitor) (any, error) {
	return v.VisitObjectExpr(n)
}

func (n *ArrayExpr) Accept(v ExprVisitor) (any, error) {
	return v.VisitArrayExpr(n)
}

func (n *IndexExpr) Accept(v ExprVisitor) (any, error) {
	return v.VisitIndexExpr(n)
}

func (n *LiteralExpr) Accept(v ExprVisitor) (any, error) {
	return v.VisitLiteralExpr(n)
}

func (n *LiteralExpr) String() string {
	if n.Value == nil {
		return "nil"
	}

	if l, ok := n.Value.(float64); ok {
		// Check if the float is an integer value
		if l == float64(int64(l)) {
			// Print with one decimal place (e.g., 10.0 instead of 10)
			return fmt.Sprintf("%.1f", l)
		} else {
			return fmt.Sprintf("%g", l)
		}
	}

	return fmt.Sprintf("%v", n.Value)
}

func (n *VarExpr) Accept(v ExprVisitor) (any, error) {
	return v.VisitVarExpr(n)
}

func (n *GroupingExpr) Accept(v ExprVisitor) (any, error) {
	return v.VisitGroupingExpr(n)
}

func (n *GroupingExpr) String() string {
	return fmt.Sprintf("(group %v)", n.Expr)
}

func (n *UnaryExpr) Accept(v ExprVisitor) (any, error) {
	return v.VisitUnaryExpr(n)
}

func (n *UnaryExpr) String() string {
	return fmt.Sprintf("(%s %v)", n.Operator.Lexeme, n.Right)
}

func (n *AssignExpr) Accept(v ExprVisitor) (any, error) {
	return v.VisitAssignExpr(n)
}

func (n *AssignExpr) String() string {
	return fmt.Sprintf("(assign %v %v)", n.Name, n.Value)
}

func (n *LogicalExpr) Accept(v ExprVisitor) (any, error) {
	return v.VisitLogicalExpr(n)
}

func (n *CallExpr) Accept(v ExprVisitor) (any, error) {
	return v.VisitCallExpr(n)
}

func (n *SetIndexExpr) Accept(v ExprVisitor) (any, error) {
	return v.VisitSetIndexExpr(n)
}
