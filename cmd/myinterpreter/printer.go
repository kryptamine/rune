package main

import (
	"fmt"
)

// PrintVisitor is a Visitor that prints each node.
type PrintVisitor struct{}

func (p *PrintVisitor) visitBinaryExpr(node *BinaryExpr) any {
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

	return nil
}

func (p *PrintVisitor) visitLiteralExpr(node *LiteralExpr) any {
	fmt.Print(node.value)

	return nil
}

func (p *PrintVisitor) visitGroupingExpr(node *GroupingExpr) any {
	fmt.Print("(group ")
	node.expr.accept(p)
	fmt.Print(")")

	return nil
}

func (p *PrintVisitor) visitUnaryExpr(node *UnaryExpr) any {
	fmt.Print("(")
	fmt.Print(node.operator.lexeme, " ")
	node.right.accept(p)
	fmt.Print(")")

	return nil
}
