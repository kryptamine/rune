package main

type ExprVisitor interface {
	visitBinaryExpr(binaryExpr *BinaryExpr) (any, error)
	visitLiteralExpr(literalExpr *LiteralExpr) (any, error)
	visitGroupingExpr(literalExpr *GroupingExpr) (any, error)
	visitUnaryExpr(UnaryExpr *UnaryExpr) (any, error)
	visitVarExpr(varExpr *VarExpr) (any, error)
	visitAssignExpr(assignExpr *AssignExpr) (any, error)
}

type Expr interface {
	accept(v ExprVisitor) (any, error)
}

type BinaryExpr struct {
	left     Expr
	right    Expr
	operator Token
}

type AssignExpr struct {
	name  Token
	value Expr
}

type UnaryExpr struct {
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

type GroupingExpr struct {
	expr Expr
}

func (n *BinaryExpr) accept(v ExprVisitor) (any, error) {
	return v.visitBinaryExpr(n)
}

func (n *LiteralExpr) accept(v ExprVisitor) (any, error) {
	return v.visitLiteralExpr(n)
}

func (n *VarExpr) accept(v ExprVisitor) (any, error) {
	return v.visitVarExpr(n)
}

func (n *GroupingExpr) accept(v ExprVisitor) (any, error) {
	return v.visitGroupingExpr(n)
}

func (n *UnaryExpr) accept(v ExprVisitor) (any, error) {
	return v.visitUnaryExpr(n)
}

func (n *AssignExpr) accept(v ExprVisitor) (any, error) {
	return v.visitAssignExpr(n)
}
