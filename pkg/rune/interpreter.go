package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/callable"
	"rune/pkg/environment"
	"rune/pkg/errors"
	"rune/pkg/helpers"
)

const maxRecursionDepth = 999

type Interpreter struct {
	environment    *environment.Environment
	globals        *environment.Environment
	locals         map[ast.Expr]int
	recursionDepth int
	maxRecursion   int
}

func NewInterpreter() *Interpreter {
	globals := environment.NewEnvironment(nil)

	p := &Interpreter{
		environment:    globals,
		globals:        globals,
		locals:         make(map[ast.Expr]int),
		recursionDepth: 0,
		maxRecursion:   maxRecursionDepth,
	}

	// Global functions.
	p.registerGlobalCallable("clock", callable.NewClockCallable())
	p.registerGlobalCallable("len", callable.NewLenCallable())
	p.registerGlobalCallable("append", callable.NewAppendCallable())
	p.registerGlobalCallable("json", callable.NewJsonCallable())

	return p
}

func EvaluateExpr(expr ast.Expr) (any, error) {
	return expr.Accept(&Interpreter{})
}

func (p *Interpreter) EvaluateStmts(stmts []ast.Stmt) error {
	for _, stmt := range stmts {
		err := stmt.Accept(p)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Interpreter) registerGlobalCallable(name string, value callable.Callable) {
	p.environment.Define(name, value)
}

func (p *Interpreter) VisitReturnStmt(returnStmt *ast.ReturnStmt) error {
	if returnStmt.Value == nil {
		return callable.NewReturn(nil)
	}

	value, err := returnStmt.Value.Accept(p)

	if err != nil {
		return err
	}

	return callable.NewReturn(value)
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

	if v, ok := val.(float64); ok {
		if v == float64(int64(v)) {
			fmt.Println(fmt.Sprintf("%.0f", v))
			return nil
		}
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
		if helpers.IsTruthy(left) {
			return left, nil
		}
	} else {
		if !helpers.IsTruthy(left) {
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

	for helpers.IsTruthy(val) {
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

		p.environment.Define(varStmt.Name.Lexeme, value)
	} else {
		p.environment.Define(varStmt.Name.Lexeme, nil)
	}

	return nil
}

func (p *Interpreter) VisitCallExpr(callExpr *ast.CallExpr) (any, error) {
	if p.recursionDepth >= p.maxRecursion {
		return nil, errors.NewRuntimeError(callExpr.Token, "Stack overflow.")
	}

	p.recursionDepth++
	defer func() { p.recursionDepth-- }()

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

	if callable, ok := callee.(callable.Callable); ok {
		if callable.Arity() != -1 && len(args) != callable.Arity() {
			return nil, errors.NewRuntimeError(
				callExpr.Token,
				fmt.Sprintf("Expected %d arguments but got %d.", callable.Arity(), len(args)),
			)
		}

		return callable.Call(p.executeBlock, args, callExpr.Token)
	}

	return nil, errors.NewRuntimeError(callExpr.Token, "Can only call functions.")
}

func (p *Interpreter) VisitFunctionStmt(functionStmt *ast.FunctionStmt) error {
	function := callable.NewFunctionCallable(functionStmt, p.environment)

	if len(function.Declaration.Name.Lexeme) == 0 {
		return errors.NewRuntimeError(functionStmt.Name, "Function name is required.")
	}

	p.environment.Define(functionStmt.Name.Lexeme, function)
	return nil
}

func (p *Interpreter) VisitIfStmt(ifStmt *ast.IfStmt) error {
	condition, err := ifStmt.Condition.Accept(p)
	if err != nil {
		return err
	}

	if helpers.IsTruthy(condition) {
		return ifStmt.Then.Accept(p)
	} else if ifStmt.El != nil {
		return ifStmt.El.Accept(p)
	}

	return nil
}

func (p *Interpreter) VisitBlockStmt(blockStmt *ast.BlockStmt) error {
	return p.executeBlock(blockStmt.Stmts, environment.NewEnvironment(p.environment))
}

func (p *Interpreter) executeBlock(statements []ast.Stmt, env *environment.Environment) error {
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
	return p.lookupVariable(node.Name, node)
}

func (p *Interpreter) lookupVariable(name ast.Token, expr ast.Expr) (any, error) {
	if distance, ok := p.GetLocalDistance(expr); ok {
		return p.environment.GetAt(distance, name.Lexeme), nil
	}

	return p.globals.Get(name)
}

func (p *Interpreter) GetLocalDistance(expr ast.Expr) (int, bool) {
	distance, ok := p.locals[expr]

	return distance, ok
}

func (p *Interpreter) VisitAssignExpr(node *ast.AssignExpr) (any, error) {
	value, err := node.Value.Accept(p)
	if err != nil {
		return nil, err
	}

	distance, ok := p.GetLocalDistance(node)
	if ok {
		p.environment.AssignAt(distance, node.Name.Lexeme, value)
	} else {
		if err := p.globals.Assign(node.Name, value); err != nil {
			return nil, err
		}
	}

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
		return helpers.IsEqual(left, right), nil
	case ast.BANG_EQUAL:
		return !helpers.IsEqual(left, right), nil
	case ast.PLUS:
		if helpers.IsString(left) && helpers.IsString(right) {
			return left.(string) + right.(string), nil
		}

		if helpers.IsFloat(left) && helpers.IsFloat(right) {
			return left.(float64) + right.(float64), nil
		}

		return nil, errors.NewRuntimeError(node.Operator, "Operands must be two numbers or two strings.")
	case ast.MINUS:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, errors.NewRuntimeError(node.Operator, err.Error())
		}
		return helpers.ToFloat(left) - helpers.ToFloat(right), nil
	case ast.SLASH:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, errors.NewRuntimeError(node.Operator, err.Error())
		}
		return helpers.ToFloat(left) / helpers.ToFloat(right), nil
	case ast.STAR:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, errors.NewRuntimeError(node.Operator, err.Error())
		}
		return helpers.ToFloat(left) * helpers.ToFloat(right), nil
	case ast.LESS:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, errors.NewRuntimeError(node.Operator, err.Error())
		}
		return helpers.ToFloat(left) < helpers.ToFloat(right), nil
	case ast.LESS_EQUAL:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, errors.NewRuntimeError(node.Operator, err.Error())
		}
		return helpers.ToFloat(left) <= helpers.ToFloat(right), nil
	case ast.GREATER:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, errors.NewRuntimeError(node.Operator, err.Error())
		}
		return helpers.ToFloat(left) > helpers.ToFloat(right), nil
	case ast.GREATER_EQUAL:
		if err := p.checkNumberOperands(left, right); err != nil {
			return nil, errors.NewRuntimeError(node.Operator, err.Error())
		}
		return helpers.ToFloat(left) >= helpers.ToFloat(right), nil
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
		return !helpers.IsTruthy(right), nil
	case ast.MINUS:
		if !helpers.IsFloat(right) {
			return nil, errors.NewRuntimeError(node.Operator, "Operand must be a number.")
		}

		return -1 * helpers.ToFloat(right), nil
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
	// Evaluate the object (could be an array or an object)
	targetVal, err := node.Array.Accept(p)
	if err != nil {
		return nil, err
	}

	indexVal, err := node.Index.Accept(p)
	if err != nil {
		return nil, err
	}

	// Handle Array Indexing
	if arr, ok := targetVal.([]any); ok {
		if !helpers.IsFloat(indexVal) {
			return nil, errors.NewRuntimeError(node.Token, "Array index must be a number.")
		}

		idx := int(indexVal.(float64))
		if idx < 0 || idx >= len(arr) {
			return nil, errors.NewRuntimeError(node.Token, fmt.Sprintf("Index out of bounds: %v of %v", idx, len(arr)))
		}

		return arr[idx], nil
	}

	// Handle Object Property Access
	if obj, ok := targetVal.(map[string]any); ok {
		key, ok := indexVal.(string)
		if !ok {
			return nil, errors.NewRuntimeError(node.Token, "Object keys must be strings.")
		}

		value, exists := obj[key]
		if !exists {
			return nil, errors.NewRuntimeError(node.Token, fmt.Sprintf("Undefined property '%s'.", key))
		}

		return value, nil
	}

	return nil, errors.NewRuntimeError(node.Token, "Indexing is only supported on arrays and objects.")
}

func (p *Interpreter) VisitSetIndexExpr(node *ast.SetIndexExpr) (any, error) {
	targetVal, err := node.Array.Accept(p)
	if err != nil {
		return nil, err
	}

	indexVal, err := node.Index.Accept(p)
	if err != nil {
		return nil, err
	}

	value, err := node.Value.Accept(p)
	if err != nil {
		return nil, err
	}

	switch target := targetVal.(type) {
	case []any:
		idx, ok := indexVal.(float64)
		if !ok || int(idx) < 0 || int(idx) >= len(target) {
			return nil, errors.NewRuntimeError(
				node.Token,
				fmt.Sprintf("Index out of bounds: %v of %v", idx, len(target)),
			)
		}
		target[int(idx)] = value
		return value, nil

	case map[string]any:
		key, ok := indexVal.(string)
		if !ok {
			return nil, errors.NewRuntimeError(
				node.Token,
				"Object properties must be accessed with string keys.",
			)
		}
		target[key] = value
		return value, nil

	default:
		return nil, errors.NewRuntimeError(
			node.Token,
			"Indexing is only supported on arrays and objects.",
		)
	}
}

func (p *Interpreter) VisitObjectExpr(node *ast.ObjectExpr) (any, error) {
	obj := make(map[string]any)

	for key, value := range node.Pairs {
		v, err := value.Accept(p)
		if err != nil {
			return nil, err
		}

		obj[key] = v
	}

	return obj, nil
}

func (p *Interpreter) checkNumberOperands(left any, right any) error {
	if helpers.IsFloat(left) && helpers.IsFloat(right) {
		return nil
	}

	return fmt.Errorf("Operands must be numbers.")
}

func (p *Interpreter) Resolve(expr ast.Expr, depth int) {
	p.locals[expr] = depth
}
