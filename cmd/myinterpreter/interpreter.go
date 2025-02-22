package main

import (
	"fmt"
)

type Interpreter struct {
	environment *Environment
}

func EvaluateExpr(expr Expr) (any, error) {
	return expr.accept(&Interpreter{})
}

func EvaluateStmts(stmts []Stmt) error {
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
		return nil
	}

	// if v, ok := val.(float64); ok {
	// 	// Check if the float is an integer value
	// 	if v == float64(int64(v)) {
	// 		// Print with one decimal place (e.g., 10.0 instead of 10)
	// 		fmt.Println(fmt.Sprintf("%.0f", v))
	// 	} else {
	// 		fmt.Println(fmt.Sprintf("%g", v))
	// 	}
	//
	// 	return nil
	// }

	fmt.Println(val)

	return nil
}

func (p *Interpreter) visitLogicalExpr(node *LogicalExpr) (any, error) {
	left, err := node.left.accept(p)

	if err != nil {
		return nil, err
	}

	if node.op.tokenType == OR {
		if isTruthy(left) {
			return left, nil
		}
	} else {
		if !isTruthy(left) {
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

	for isTruthy(val) {
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
		if len(args) != callable.Arity() {
			return nil, NewRuntimeError(
				callExpr.token,
				fmt.Sprintf("Expected %d arguments but got %d.", callable.Arity(), len(args)),
			)
		}

		return callable.Call(p, args)
	}

	return nil, NewRuntimeError(callExpr.token, "Can only call functions and classes.")
}

func (p *Interpreter) visitFunctionStmt(functionStmt *FunctionStmt) error {
	function := &Function{
		declaration: functionStmt,
		environment: p.environment,
	}

	if len(function.declaration.name.lexeme) == 0 {
		return NewRuntimeError(functionStmt.name, "Function name is required.")
	}

	p.environment.define(functionStmt.name.lexeme, function)
	return nil
}

func (p *Interpreter) visitIfStmt(ifStmt *IfStmt) error {
	condition, err := ifStmt.condition.accept(p)
	if err != nil {
		return err
	}

	if isTruthy(condition) {
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
		return isEqual(left, right), nil
	case BANG_EQUAL:
		return !isEqual(left, right), nil
	case PLUS:
		if isString(left) && isString(right) {
			return left.(string) + right.(string), nil
		}

		if isFloat(left) && isFloat(right) {
			return left.(float64) + right.(float64), nil
		}

		return nil, NewRuntimeError(node.operator, "Operands must be two numbers or two strings.")
	case MINUS:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, NewRuntimeError(node.operator, err.Error())
		}
		return toFloat(left) - toFloat(right), nil
	case SLASH:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, NewRuntimeError(node.operator, err.Error())
		}
		return toFloat(left) / toFloat(right), nil
	case STAR:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, NewRuntimeError(node.operator, err.Error())
		}
		return toFloat(left) * toFloat(right), nil
	case LESS:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, NewRuntimeError(node.operator, err.Error())
		}
		return toFloat(left) < toFloat(right), nil
	case LESS_EQUAL:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, NewRuntimeError(node.operator, err.Error())
		}
		return toFloat(left) <= toFloat(right), nil
	case GREATER:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, NewRuntimeError(node.operator, err.Error())
		}
		return toFloat(left) > toFloat(right), nil
	case GREATER_EQUAL:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, NewRuntimeError(node.operator, err.Error())
		}
		return toFloat(left) >= toFloat(right), nil
	}

	return nil, nil
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
		return !isTruthy(right), nil
	case MINUS:
		if !isFloat(right) {
			return nil, NewRuntimeError(node.operator, "Operand must be a number.")
		}

		return -1 * toFloat(right), nil
	}

	return nil, nil
}

func (p *Interpreter) visitArrayExpr(node *ArrayExpr) (any, error) {
	return node.items, nil
}

func (p *Interpreter) checkNumberOperands(left any, right any) error {
	if isFloat(left) && isFloat(right) {
		return nil
	}

	return fmt.Errorf("Operands must be numbers.")
}
