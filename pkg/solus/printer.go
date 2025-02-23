package solus

import (
	"fmt"
	"github.com/codecrafters-io/interpreter-starter-go/pkg/ast"
)

type PrintVisitor struct{}

func PrintExpr(expr ast.Expr) {
	expr.Accept(&PrintVisitor{})
}

func (p *PrintVisitor) printNode(node ast.Expr) (any, error) {
	fmt.Print(node)
	return nil, nil
}

func (p *PrintVisitor) VisitBinaryExpr(node *ast.BinaryExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) VisitCallExpr(node *ast.CallExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) VisitVarExpr(node *ast.VarExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) VisitLogicalExpr(node *ast.LogicalExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) VisitAssignExpr(node *ast.AssignExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) VisitLiteralExpr(node *ast.LiteralExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) VisitGroupingExpr(node *ast.GroupingExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) VisitUnaryExpr(node *ast.UnaryExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) VisitArrayExpr(node *ast.ArrayExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) VisitIndexExpr(node *ast.IndexExpr) (any, error) {
	return p.printNode(node)
}

func (p *PrintVisitor) VisitSetIndexExpr(node *ast.SetIndexExpr) (any, error) {
	return p.printNode(node)
}
