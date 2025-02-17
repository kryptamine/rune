package main

import (
	"fmt"
	"strconv"
)

type Interpreter struct {
	environment *Environment
}

type Return struct {
	value any
}

func (e *Return) Error() string {
	return "<fn return>"
}

func Interpret(stmts []Stmt) error {
	globals := NewEnvironment(nil)

	globals.define("clock", &ClockCallable{})

	p := &Interpreter{
		environment: globals,
	}

	for _, stmt := range stmts {
		err := stmt.accept(p)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Interpreter) visitReturnStmt(returnStmt *ReturnStmt) error {
	if returnStmt.value == nil {
		return &Return{nil}
	}

	value, err := returnStmt.value.accept(p)

	if err != nil {
		return err
	}

	return &Return{value}
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

	if val == nil {
		fmt.Println("nil")
	} else {
		fmt.Println(val)
	}

	return nil
}

func (p *Interpreter) visitLogicalExpr(node *LogicalExpr) (any, error) {
	left, err := node.left.accept(p)

	if err != nil {
		return nil, err
	}

	if node.op.tokenType == OR {
		if p.isTruthy(left) {
			return left, nil
		}
	} else {
		if !p.isTruthy(left) {
			return left, nil
		}
	}

	return node.right.accept(p)
}

func (p *Interpreter) visitWhileStmt(whileStmt *WhileStmt) error {
	val, err := whileStmt.condition.accept(p)
	if err != nil {
		return err
	}

	for p.isTruthy(val) {
		err := whileStmt.body.accept(p)
		if err != nil {
			return err
		}

		val, err = whileStmt.condition.accept(p)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Interpreter) visitVarStmt(varStmt *VarStmt) error {
	if varStmt.initializer != nil {
		value, err := varStmt.initializer.accept(p)

		if err != nil {
			return err
		}

		p.environment.define(varStmt.name.lexeme, value)
	} else {
		p.environment.define(varStmt.name.lexeme, nil)
	}

	return nil
}

func (p *Interpreter) visitCallExpr(callExpr *CallExpr) (any, error) {
	callee, err := callExpr.callee.accept(p)
	if err != nil {
		return nil, err
	}

	args := []any{}

	for _, arg := range callExpr.args {
		arg, err := arg.accept(p)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	if callable, ok := callee.(Callable); ok {
		return callable.Call(p, args)
	}

	return nil, fmt.Errorf("Can only call functions and classes.")
}

func (p *Interpreter) visitFunctionStmt(functionStmt *FunctionStmt) error {
	function := &Function{
		declaration: functionStmt,
	}

	if len(function.declaration.name.lexeme) == 0 {
		return fmt.Errorf("Function name is required.")
	}

	p.environment.define(functionStmt.name.lexeme, function)
	return nil
}

func (p *Interpreter) visitIfStmt(ifStmt *IfStmt) error {
	condition, err := ifStmt.condition.accept(p)
	if err != nil {
		return err
	}

	if p.isTruthy(condition) {
		return ifStmt.then.accept(p)
	} else if ifStmt.el != nil {
		return ifStmt.el.accept(p)
	}

	return nil
}

func (p *Interpreter) visitBlockStmt(blockStmt *BlockStmt) error {
	return p.executeBlock(blockStmt.stmts, NewEnvironment(p.environment))
}

func (p *Interpreter) executeBlock(statements []Stmt, env *Environment) error {
	prevEnv := p.environment
	p.environment = env

	defer func() {
		p.environment = prevEnv
	}()

	for _, stmt := range statements {
		err := stmt.accept(p)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Interpreter) visitVarExpr(node *VarExpr) (any, error) {
	return p.environment.get(node.name)
}

func (p *Interpreter) visitAssignExpr(node *AssignExpr) (any, error) {
	value, err := node.value.accept(p)
	if err != nil {
		return nil, err
	}

	p.environment.assign(node.name, value)

	return value, nil
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
		return true
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
