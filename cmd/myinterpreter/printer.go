package main

import "fmt"

type PrintVisitor struct{}

func PrintExpr(expr Expr) {
	expr.accept(&PrintVisitor{})
}

func (p *PrintVisitor) printNode(node Expr) (any, error) {
	fmt.Print(node)
	return nil, nil
}

func (p *PrintVisitor) visitBinaryExpr(node *BinaryExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) visitCallExpr(node *CallExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) visitVarExpr(node *VarExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) visitLogicalExpr(node *LogicalExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) visitAssignExpr(node *AssignExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) visitLiteralExpr(node *LiteralExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) visitGroupingExpr(node *GroupingExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) visitUnaryExpr(node *UnaryExpr) (any, error) {
	return p.printNode(node)
}
