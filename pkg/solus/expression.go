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
	left     Expr
	right    Expr
	operator Token
}

func NewBinaryExpr(left Expr, right Expr, operator Token) Expr {
	return &BinaryExpr{left: left, right: right, operator: operator}
}

type AssignExpr struct {
	name  Token
	value Expr
}

func NewAssignExpr(name Token, value Expr) Expr {
	return &AssignExpr{name: name, value: value}
}

type UnaryExpr struct {
	right    Expr
	operator Token
}

func NewUnaryExpr(right Expr, operator Token) Expr {
	return &UnaryExpr{right: right, operator: operator}
}

type LiteralExpr struct {
	tokenType TokenType
	value     any
}

func NewLiteralExpr(tokenType TokenType, value any) Expr {
	return &LiteralExpr{tokenType: tokenType, value: value}
}

type VarExpr struct {
	name Token
}

func NewVarExpr(name Token) Expr {
	return &VarExpr{name: name}
}

type LogicalExpr struct {
	left  Expr
	right Expr
	op    Token
}

func NewLogicalExpr(left Expr, right Expr, op Token) Expr {
	return &LogicalExpr{left: left, right: right, op: op}
}

type GroupingExpr struct {
	expr Expr
}

func NewGroupingExpr(expr Expr) Expr {
	return &GroupingExpr{expr: expr}
}

type CallExpr struct {
	token  Token
	callee Expr
	args   []Expr
}

func NewCallExpr(token Token, callee Expr, args []Expr) Expr {
	return &CallExpr{token: token, callee: callee, args: args}
}

type ArrayExpr struct {
	token Token
	items []Expr
}

func NewArrayExpr(token Token, items []Expr) Expr {
	return &ArrayExpr{token: token, items: items}
}

type IndexExpr struct {
	token Token
	array Expr
	index Expr
}

func NewIndexExpr(array Expr, index Expr, token Token) Expr {
	return &IndexExpr{array: array, index: index, token: token}
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
