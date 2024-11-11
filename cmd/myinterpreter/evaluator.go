package main

import (
	"strconv"
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
	right := node.right.accept(p)

	switch node.operator.tokenType {
	case BANG:
		return !p.isTruthy(right)
	case MINUS:
		val, _ := strconv.ParseFloat(right.(string), 64)
		return -1 * val
	}

	return nil
}

func (p *Interpreter) isTruthy(val any) bool {
	if val == nil {
		return false
	}

	switch i2 := val.(type) {
	case bool:
		return i2
	case string:
		return i2 != "" && i2 != "nil" && i2 != "false"
	case int:
		return i2 != 0
	default:
		return false
	}
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
