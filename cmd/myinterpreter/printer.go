package main

import (
	"fmt"
)

// PrintVisitor is a Visitor that prints each node.
type PrintVisitor struct{}

func (p *PrintVisitor) visitBinaryExpr(node *BinaryExpr) {
	fmt.Print("(")

	fmt.Printf("%s", node.operator.lexeme)
	if node.left != nil {
		fmt.Print(" ")
		node.left.accept(p)
	}

	if node.right != nil {
		fmt.Print(" ")
		node.right.accept(p)
	}

	fmt.Print(")")
}

func (p *PrintVisitor) visitLiteralExpr(node *LiteralExpr) {
	fmt.Print(node.value)
}

func (p *PrintVisitor) visitGroupingExpr(node *GroupingExpr) {
	fmt.Print("(group ")
	node.expr.accept(p)
	fmt.Print(")")
}

func (p *PrintVisitor) visitUnaryExpr(node *UnaryExpr) {
	fmt.Print("(")
	fmt.Print(node.operator.lexeme, " ")
	node.right.accept(p)
	fmt.Print(")")
}
