package solus

import (
	"fmt"
	"github.com/codecrafters-io/interpreter-starter-go/pkg/ast"
)

type Interpreter struct {
	environment *Environment
}

func EvaluateExpr(expr ast.Expr) (any, error) {
	return expr.Accept(&Interpreter{})
}

func EvaluateStmts(stmts []ast.Stmt) error {
	globals := NewEnvironment(nil)

	globals.RegisterGlobalCallable("clock", NewClockCallable())
	globals.RegisterGlobalCallable("len", NewLenCallable())
	globals.RegisterGlobalCallable("append", NewAppendCallable())

	p := &Interpreter{
		environment: globals,
	}

	for _, stmt := range stmts {
		err := stmt.Accept(p)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Interpreter) VisitReturnStmt(returnStmt *ast.ReturnStmt) error {
	if returnStmt.Value == nil {
		return &Return{nil}
	}

	value, err := returnStmt.Value.Accept(p)

	if err != nil {
		return err
	}

	return &Return{value}
}

func (p *Interpreter) VisitExprStmt(exprStmt *ast.ExprStmt) error {
	_, err := exprStmt.Expr.Accept(p)

	if err != nil {
		return err
	}

	return nil
}

func (p *Interpreter) VisitPrintStmt(exprStmt *ast.PrintStmt) error {
	val, err := exprStmt.Expr.Accept(p)

	if err != nil {
		return err
	}

	if val == nil {
		fmt.Println("nil")
		return nil
	}

	fmt.Println(val)

	return nil
}

func (p *Interpreter) VisitLogicalExpr(node *ast.LogicalExpr) (any, error) {
	left, err := node.Left.Accept(p)

	if err != nil {
		return nil, err
	}

	if node.Op.TokenType == ast.OR {
		if isTruthy(left) {
			return left, nil
		}
	} else {
		if !isTruthy(left) {
			return left, nil
		}
	}

	return node.Right.Accept(p)
}

func (p *Interpreter) VisitWhileStmt(whileStmt *ast.WhileStmt) error {
	val, err := whileStmt.Condition.Accept(p)
	if err != nil {
		return err
	}

	for isTruthy(val) {
		err := whileStmt.Body.Accept(p)
		if err != nil {
			return err
		}

		val, err = whileStmt.Condition.Accept(p)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Interpreter) VisitVarStmt(varStmt *ast.VarStmt) error {
	if varStmt.Initializer != nil {
		value, err := varStmt.Initializer.Accept(p)

		if err != nil {
			return err
		}

		p.environment.define(varStmt.Name.Lexeme, value)
	} else {
		p.environment.define(varStmt.Name.Lexeme, nil)
	}

	return nil
}

func (p *Interpreter) VisitCallExpr(callExpr *ast.CallExpr) (any, error) {
	callee, err := callExpr.Callee.Accept(p)
	if err != nil {
		return nil, err
	}

	args := []any{}

	for _, arg := range callExpr.Args {
		arg, err := arg.Accept(p)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	if callable, ok := callee.(Callable); ok {
		if callable.Arity() != -1 && len(args) != callable.Arity() {
			return nil, NewRuntimeError(
				callExpr.Token,
				fmt.Sprintf("Expected %d arguments but got %d.", callable.Arity(), len(args)),
			)
		}

		return callable.Call(p, args, callExpr.Token)
	}

	return nil, NewRuntimeError(callExpr.Token, "Can only call functions and classes.")
}

func (p *Interpreter) VisitFunctionStmt(functionStmt *ast.FunctionStmt) error {
	function := &FunctionCallable{
		declaration: functionStmt,
		environment: p.environment,
	}

	if len(function.declaration.Name.Lexeme) == 0 {
		return NewRuntimeError(functionStmt.Name, "Function name is required.")
	}

	p.environment.define(functionStmt.Name.Lexeme, function)
	return nil
}

func (p *Interpreter) VisitIfStmt(ifStmt *ast.IfStmt) error {
	condition, err := ifStmt.Condition.Accept(p)
	if err != nil {
		return err
	}

	if isTruthy(condition) {
		return ifStmt.Then.Accept(p)
	} else if ifStmt.El != nil {
		return ifStmt.El.Accept(p)
	}

	return nil
}

func (p *Interpreter) VisitBlockStmt(blockStmt *ast.BlockStmt) error {
	return p.executeBlock(blockStmt.Stmts, NewEnvironment(p.environment))
}

func (p *Interpreter) executeBlock(statements []ast.Stmt, env *Environment) error {
	prevEnv := p.environment
	p.environment = env

	defer func() {
		p.environment = prevEnv
	}()

	for _, stmt := range statements {
		err := stmt.Accept(p)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Interpreter) VisitVarExpr(node *ast.VarExpr) (any, error) {
	return p.environment.get(node.Name)
}

func (p *Interpreter) VisitAssignExpr(node *ast.AssignExpr) (any, error) {
	value, err := node.Value.Accept(p)
	if err != nil {
		return nil, err
	}

	p.environment.assign(node.Name, value)

	return value, nil
}

func (p *Interpreter) VisitBinaryExpr(node *ast.BinaryExpr) (any, error) {
	left, err := node.Left.Accept(p)
	right, err := node.Right.Accept(p)

	if err != nil {
		return nil, err
	}

	switch node.Operator.TokenType {
	case ast.EQUAL_EQUAL:
		return isEqual(left, right), nil
	case ast.BANG_EQUAL:
		return !isEqual(left, right), nil
	case ast.PLUS:
		if isString(left) && isString(right) {
			return left.(string) + right.(string), nil
		}

		if isFloat(left) && isFloat(right) {
			return left.(float64) + right.(float64), nil
		}

		return nil, NewRuntimeError(node.Operator, "Operands must be two numbers or two strings.")
	case ast.MINUS:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, NewRuntimeError(node.Operator, err.Error())
		}
		return toFloat(left) - toFloat(right), nil
	case ast.SLASH:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, NewRuntimeError(node.Operator, err.Error())
		}
		return toFloat(left) / toFloat(right), nil
	case ast.STAR:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, NewRuntimeError(node.Operator, err.Error())
		}
		return toFloat(left) * toFloat(right), nil
	case ast.LESS:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, NewRuntimeError(node.Operator, err.Error())
		}
		return toFloat(left) < toFloat(right), nil
	case ast.LESS_EQUAL:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, NewRuntimeError(node.Operator, err.Error())
		}
		return toFloat(left) <= toFloat(right), nil
	case ast.GREATER:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, NewRuntimeError(node.Operator, err.Error())
		}
		return toFloat(left) > toFloat(right), nil
	case ast.GREATER_EQUAL:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, NewRuntimeError(node.Operator, err.Error())
		}
		return toFloat(left) >= toFloat(right), nil
	}

	return nil, nil
}

func (p *Interpreter) VisitLiteralExpr(node *ast.LiteralExpr) (any, error) {
	return node.Value, nil
}

func (p *Interpreter) VisitGroupingExpr(node *ast.GroupingExpr) (any, error) {
	return node.Expr.Accept(p)
}

func (p *Interpreter) VisitUnaryExpr(node *ast.UnaryExpr) (any, error) {
	right, err := node.Right.Accept(p)

	if err != nil {
		return nil, err
	}

	switch node.Operator.TokenType {
	case ast.BANG:
		return !isTruthy(right), nil
	case ast.MINUS:
		if !isFloat(right) {
			return nil, NewRuntimeError(node.Operator, "Operand must be a number.")
		}

		return -1 * toFloat(right), nil
	}

	return nil, nil
}

func (p *Interpreter) VisitArrayExpr(node *ast.ArrayExpr) (any, error) {
	var result []any

	for _, item := range node.Items {
		item, err := item.Accept(p)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	return result, nil
}

func (p *Interpreter) VisitIndexExpr(node *ast.IndexExpr) (any, error) {
	arrayVal, err := node.Array.Accept(p)
	if err != nil {
		return nil, err
	}

	indexVal, err := node.Index.Accept(p)
	if err != nil {
		return nil, err
	}

	arr, ok := arrayVal.([]any)
	if !ok {
		return nil, NewRuntimeError(node.Token, "Indexing is only supported on arrays.")
	}

	idx, ok := indexVal.(float64)
	if !isFloat(indexVal) || int(idx) < 0 || int(idx) >= len(arr) {
		return nil, NewRuntimeError(node.Token, fmt.Sprintf("Index out of bounds: %v of %v", idx, len(arr)))
	}

	return arr[int(idx)], nil
}

func (p *Interpreter) checkNumberOperands(left any, right any) error {
	if isFloat(left) && isFloat(right) {
		return nil
	}

	return fmt.Errorf("Operands must be numbers.")
}
