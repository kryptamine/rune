package main

import "fmt"

// PrintVisitor is a Visitor that prints each node.
type PrintVisitor struct{}

func PrintExpr(expr Expr) (any, error) {
	return expr.accept(&Interpreter{})
}

func (p *PrintVisitor) visitBinaryExpr(node *BinaryExpr) (any, error) {
	fmt.Print(node)
	return nil, nil
}

func (p *PrintVisitor) visitCallExpr(node *CallExpr) (any, error) {
	fmt.Print(node)
	return nil, nil
}

func (p *PrintVisitor) visitVarExpr(node *VarExpr) (any, error) {
	fmt.Print(node)
	return nil, nil
}

func (p *PrintVisitor) visitLogicalExpr(node *LogicalExpr) (any, error) {
	fmt.Print(node)
	return nil, nil
}

func (p *PrintVisitor) visitAssignExpr(node *AssignExpr) (any, error) {
	fmt.Print(node)
	return nil, nil
}

func (p *PrintVisitor) visitLiteralExpr(node *LiteralExpr) (any, error) {
	fmt.Print(node)
	return nil, nil
}

func (p *PrintVisitor) visitGroupingExpr(node *GroupingExpr) (any, error) {
	fmt.Print(node)
	return nil, nil
}

func (p *PrintVisitor) visitUnaryExpr(node *UnaryExpr) (any, error) {
	fmt.Print(node)
	return nil, nil
}
