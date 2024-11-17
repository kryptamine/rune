package main

import (
	"fmt"
	"strconv"
)

type Interpreter struct{}

func Interpret(stmts []Stmt) error {
	p := &Interpreter{}

	for _, stmt := range stmts {
		err := stmt.accept(p)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Interpreter) visitExprStmt(exprStmt *ExprStmt) error {
	_, err := exprStmt.expr.accept(p)

	if err != nil {
		return err
	}

	return nil
}

func (p *Interpreter) visitPrintStmt(exprStmt *PrintStmt) error {
	val, err := exprStmt.expr.accept(p)

	if err != nil {
		return err
	}

	fmt.Println(val)

	return nil
}

func (p *Interpreter) visitBinaryExpr(node *BinaryExpr) (any, error) {
	left, err := node.left.accept(p)
	right, err := node.right.accept(p)

	if err != nil {
		return nil, err
	}

	switch node.operator.tokenType {
	case EQUAL_EQUAL:
		return p.isEqual(left, right), nil
	case BANG_EQUAL:
		return !p.isEqual(left, right), nil
	case PLUS:
		if p.isString(left) && p.isString(right) {
			return left.(string) + right.(string), nil
		}

		if p.isFloat(left) && p.isFloat(right) {
			return left.(float64) + right.(float64), nil
		}

		return nil, fmt.Errorf("Operands must be two numbers or two strings.")
	case MINUS:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, err
		}
		return p.toFloat(left) - p.toFloat(right), nil
	case SLASH:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, err
		}
		return p.toFloat(left) / p.toFloat(right), nil
	case STAR:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, err
		}
		return p.toFloat(left) * p.toFloat(right), nil
	case LESS:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, err
		}
		return p.toFloat(left) < p.toFloat(right), nil
	case LESS_EQUAL:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, err
		}
		return p.toFloat(left) <= p.toFloat(right), nil
	case GREATER:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, err
		}
		return p.toFloat(left) > p.toFloat(right), nil
	case GREATER_EQUAL:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, err
		}
		return p.toFloat(left) >= p.toFloat(right), nil
	}

	return nil, nil
}

func (p *Interpreter) isString(val any) bool {
	_, ok := val.(string)
	return ok
}

func (p *Interpreter) isFloat(val any) bool {
	_, ok := val.(float64)
	return ok
}

func (p *Interpreter) visitLiteralExpr(node *LiteralExpr) (any, error) {
	return node.value, nil
}

func (p *Interpreter) visitGroupingExpr(node *GroupingExpr) (any, error) {
	return node.expr.accept(p)
}

func (p *Interpreter) visitUnaryExpr(node *UnaryExpr) (any, error) {
	right, err := node.right.accept(p)

	if err != nil {
		return nil, err
	}

	switch node.operator.tokenType {
	case BANG:
		return !p.isTruthy(right), nil
	case MINUS:
		if err := p.checkNumberOperand(right); err != nil {
			return nil, err
		}

		return -1 * p.toFloat(right), nil
	}

	return nil, nil
}

func (p *Interpreter) toFloat(val any) float64 {
	switch i2 := val.(type) {
	case float64:
		return i2
	case string:
		val, _ := strconv.ParseFloat(i2, 64)
		return val
	default:
		return 0.0
	}
}

func (p *Interpreter) checkNumberOperands(left any, right any) error {
	_, okLeft := left.(float64)
	_, okRight := right.(float64)

	if okLeft && okRight {
		return nil
	}

	return fmt.Errorf("Operands must be numbers.")
}

func (p *Interpreter) checkNumberOperand(val any) error {
	if _, ok := val.(float64); !ok {
		return fmt.Errorf("Operand must be a number.")
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
