package main

import (
	"fmt"
)

// PrintVisitor is a Visitor that prints each node.
type PrintVisitor struct{}

func (p *PrintVisitor) visitBinaryExpr(node *BinaryExpr) (any, error) {
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

	return nil, nil
}

func (p *PrintVisitor) visitVarExpr(node *VarExpr) (any, error) {
	return nil, nil
}

func (p *PrintVisitor) visitLiteralExpr(node *LiteralExpr) (any, error) {
	if node.value == nil {
		fmt.Print("nil")
		return nil, nil
	}

	if l, ok := node.value.(float64); ok {
		// Check if the float is an integer value
		if l == float64(int64(l)) {
			// Print with one decimal place (e.g., 10.0 instead of 10)
			fmt.Print(fmt.Sprintf("%.1f", l))
		} else {
			fmt.Print(fmt.Sprintf("%g", l))
		}

		return nil, nil
	}

	fmt.Print(node.value)

	return nil, nil
}

func (p *PrintVisitor) visitGroupingExpr(node *GroupingExpr) (any, error) {
	fmt.Print("(group ")
	node.expr.accept(p)
	fmt.Print(")")

	return nil, nil
}

func (p *PrintVisitor) visitUnaryExpr(node *UnaryExpr) (any, error) {
	fmt.Print("(")
	fmt.Print(node.operator.lexeme, " ")
	node.right.accept(p)
	fmt.Print(")")

	return nil, nil
}
