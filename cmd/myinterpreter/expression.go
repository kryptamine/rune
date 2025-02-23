package solus

import "fmt"

type ExprVisitor interface {
	visitBinaryExpr(binaryExpr *BinaryExpr) (any, error)
	visitLiteralExpr(literalExpr *LiteralExpr) (any, error)
	visitGroupingExpr(literalExpr *GroupingExpr) (any, error)
	visitUnaryExpr(UnaryExpr *UnaryExpr) (any, error)
	visitVarExpr(varExpr *VarExpr) (any, error)
	visitAssignExpr(assignExpr *AssignExpr) (any, error)
	visitLogicalExpr(logicalExpr *LogicalExpr) (any, error)
	visitCallExpr(callExpr *CallExpr) (any, error)
	visitArrayExpr(arrayExpr *ArrayExpr) (any, error)
	visitIndexExpr(indexExpr *IndexExpr) (any, error)
}

type Expr interface {
	accept(v ExprVisitor) (any, error)
}

type BinaryExpr struct {
	Expr
	left     Expr
	right    Expr
	operator Token
}

type AssignExpr struct {
	Expr
	name  Token
	value Expr
}

type UnaryExpr struct {
	Expr
	right    Expr
	operator Token
}

type LiteralExpr struct {
	tokenType TokenType
	value     any
}

type VarExpr struct {
	name Token
}

type LogicalExpr struct {
	left  Expr
	right Expr
	op    Token
}

type GroupingExpr struct {
	expr Expr
}

type CallExpr struct {
	token  Token
	callee Expr
	args   []Expr
}

type ArrayExpr struct {
	token Token
	items []Expr
}

type IndexExpr struct {
	token Token
	array Expr
	index Expr
}

func (n *ArrayExpr) accept(v ExprVisitor) (any, error) {
	return v.visitArrayExpr(n)
}

func (n *IndexExpr) accept(v ExprVisitor) (any, error) {
	return v.visitIndexExpr(n)
}

func (n *BinaryExpr) accept(v ExprVisitor) (any, error) {
	return v.visitBinaryExpr(n)
}

func (n *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %v %v)", n.operator.lexeme, n.left, n.right)
}

func (n *LiteralExpr) accept(v ExprVisitor) (any, error) {
	return v.visitLiteralExpr(n)
}

func (n *LiteralExpr) String() string {
	if n.value == nil {
		return "nil"
	}

	if l, ok := n.value.(float64); ok {
		// Check if the float is an integer value
		if l == float64(int64(l)) {
			// Print with one decimal place (e.g., 10.0 instead of 10)
			return fmt.Sprintf("%.1f", l)
		} else {
			return fmt.Sprintf("%g", l)
		}
	}

	return fmt.Sprintf("%v", n.value)
}

func (n *VarExpr) accept(v ExprVisitor) (any, error) {
	return v.visitVarExpr(n)
}

func (n *GroupingExpr) accept(v ExprVisitor) (any, error) {
	return v.visitGroupingExpr(n)
}

func (n *GroupingExpr) String() string {
	return fmt.Sprintf("(group %v)", n.expr)
}

func (n *UnaryExpr) accept(v ExprVisitor) (any, error) {
	return v.visitUnaryExpr(n)
}

func (n *UnaryExpr) String() string {
	return fmt.Sprintf("(%s %v)", n.operator.lexeme, n.right)
}

func (n *AssignExpr) accept(v ExprVisitor) (any, error) {
	return v.visitAssignExpr(n)
}

func (n *AssignExpr) String() string {
	return fmt.Sprintf("(assign %v %v)", n.name, n.value)
}

func (n *LogicalExpr) accept(v ExprVisitor) (any, error) {
	return v.visitLogicalExpr(n)
}

func (n *CallExpr) accept(v ExprVisitor) (any, error) {
	return v.visitCallExpr(n)
}
