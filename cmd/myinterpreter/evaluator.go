package main

import (
	"strings"
)

type Interpreter struct{}

func (p *Interpreter) visitBinaryExpr(node *BinaryExpr) any {
	left := node.left.accept(p)
	right := node.right.accept(p)

	switch node.operator.tokenType {
	case EQUAL_EQUAL:
		return p.isEqual(left, right)
	}

	return nil
}

func (p *Interpreter) visitLiteralExpr(node *LiteralExpr) any {
	switch node.tokenType {
	case NUMBER:
		if strings.HasSuffix(node.value, ".0") {
			return strings.TrimSuffix(node.value, ".0")
		}
		return node.value
	default:
		return node.value
	}
}

func (p *Interpreter) visitGroupingExpr(node *GroupingExpr) any {
	return node.expr.accept(p)
}

func (p *Interpreter) visitUnaryExpr(node *UnaryExpr) any {
	return node.right.accept(p)
}

func (p *Interpreter) isEqual(left any, right any) any {
	if left == nil && right == nil {
		return true
	}

	if left == nil {
		return false
	}

	return left == right
}
