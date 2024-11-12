package main

import (
	"strconv"
)

type Interpreter struct{}

func (p *Interpreter) visitBinaryExpr(node *BinaryExpr) any {
	left := node.left.accept(p)
	right := node.right.accept(p)

	switch node.operator.tokenType {
	case EQUAL_EQUAL:
		return p.isEqual(left, right)
	case BANG_EQUAL:
		return !p.isEqual(left, right)
	case PLUS:
		if _, ok := left.(string); ok {
			return left.(string) + right.(string)
		}
		return p.toFloat(left) + p.toFloat(right)
	case MINUS:
		return p.toFloat(left) - p.toFloat(right)
	case SLASH:
		return p.toFloat(left) / p.toFloat(right)
	case STAR:
		return p.toFloat(left) * p.toFloat(right)
	case LESS:
		return p.toFloat(left) < p.toFloat(right)
	case LESS_EQUAL:
		return p.toFloat(left) <= p.toFloat(right)
	case GREATER:
		return p.toFloat(left) > p.toFloat(right)
	case GREATER_EQUAL:
		return p.toFloat(left) >= p.toFloat(right)
	}

	return nil
}

func (p *Interpreter) visitLiteralExpr(node *LiteralExpr) any {
	return node.value
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
		return -1 * p.toFloat(right)
	}

	return nil
}

func (p *Interpreter) toFloat(val any) float64 {
	switch i2 := val.(type) {
	case bool:
		return 0.0
	case float64:
		return i2
	case string:
		val, _ := strconv.ParseFloat(i2, 64)
		return val
	default:
		return 0.0
	}
}

func (p *Interpreter) isTruthy(val any) bool {
	if val == nil {
		return false
	}

	switch i2 := val.(type) {
	case bool:
		return i2
	case string:
		return i2 != ""
	case float64:
		return i2 != 0.0
	default:
		return false
	}
}

func (p *Interpreter) isEqual(left any, right any) bool {
	if left == nil && right == nil {
		return true
	}

	if left == nil {
		return false
	}

	return left == right
}
