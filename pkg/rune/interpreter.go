package rune

import (
	"fmt"
	"maps"
	"math"
	"rune/pkg/ast"
	"rune/pkg/callable"
	"rune/pkg/environment"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
)

// maxRecursionDepth caps nested calls; 10k frames take ~40MB of memory.
const maxRecursionDepth = 10_000

// returnSignal unwinds statements up to the enclosing call. The value being
// returned travels in Interpreter.returnValue, so returning never allocates.
type returnSignal struct{}

func (returnSignal) Error() string { return "<fn return>" }

var errReturn error = returnSignal{}

// loopSignal unwinds statements up to the enclosing loop for break and
// continue. The resolver guarantees they only appear inside loops.
type loopSignal struct{ name string }

func (s loopSignal) Error() string { return "<" + s.name + ">" }

var (
	errBreak    error = loopSignal{"break"}
	errContinue error = loopSignal{"continue"}
)

type Interpreter struct {
	// environment is the innermost local scope, or nil at the top level.
	environment    *environment.Environment
	globals        *environment.Globals
	returnValue    value.Value
	recursionDepth int
	maxRecursion   int
}

func NewInterpreter() *Interpreter {
	p := &Interpreter{
		globals:      environment.NewGlobals(),
		maxRecursion: maxRecursionDepth,
	}

	// Global functions.
	p.registerGlobalCallable("clock", callable.NewClockCallable())
	p.registerGlobalCallable("len", callable.NewLenCallable())
	p.registerGlobalCallable("append", callable.NewAppendCallable())
	p.registerGlobalCallable("json", callable.NewJsonCallable())

	return p
}

func EvaluateExpr(expr ast.Expr) (value.Value, error) {
	return expr.Accept(NewInterpreter())
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

func (p *Interpreter) registerGlobalCallable(name string, fn callable.Callable) {
	p.globals.Define(environment.GlobalSlot(name), value.NewFunction(fn))
}

// globalSlot returns slot, looking it up by name when the resolver has not
// run (as for a lone expression evaluated with EvaluateExpr).
func globalSlot(slot int, name string) int {
	if slot == ast.Unresolved {
		return environment.GlobalSlot(name)
	}

	return slot
}

// declare binds a freshly declared variable or function name.
func (p *Interpreter) declare(local bool, slot int, name string, v value.Value) {
	if local {
		p.environment.Set(slot, v)
	} else {
		p.globals.Define(globalSlot(slot, name), v)
	}
}

func (p *Interpreter) VisitReturnStmt(returnStmt *ast.ReturnStmt) error {
	if returnStmt.Value != nil {
		v, err := returnStmt.Value.Accept(p)
		if err != nil {
			return err
		}

		p.returnValue = v
	} else {
		p.returnValue = value.NilValue
	}

	return errReturn
}

// VisitTryStmt runs the handler when the body fails with a runtime error,
// built-in or thrown. Return signals pass through untouched.
func (p *Interpreter) VisitTryStmt(tryStmt *ast.TryStmt) error {
	err := tryStmt.Body.Accept(p)

	rtErr, ok := err.(errors.RuntimeError)
	if !ok {
		return err
	}

	if tryStmt.HandlerEnvSize == 0 {
		return p.EvaluateStmts(tryStmt.Handler)
	}

	env := environment.NewEnvironment(p.environment, tryStmt.HandlerEnvSize)

	if tryStmt.Param != nil {
		env.Set(0, rtErr.Value())
	}

	return p.executeBlock(tryStmt.Handler, env)
}

func (p *Interpreter) VisitThrowStmt(throwStmt *ast.ThrowStmt) error {
	v, err := throwStmt.Value.Accept(p)
	if err != nil {
		return err
	}

	return errors.NewThrownError(throwStmt.Keyword, v, helpers.Stringify(v))
}

func (p *Interpreter) VisitExprStmt(exprStmt *ast.ExprStmt) error {
	_, err := exprStmt.Expr.Accept(p)

	return err
}

func (p *Interpreter) VisitPrintStmt(exprStmt *ast.PrintStmt) error {
	val, err := exprStmt.Expr.Accept(p)
	if err != nil {
		return err
	}

	fmt.Println(helpers.Stringify(val))

	return nil
}

func (p *Interpreter) VisitLogicalExpr(node *ast.LogicalExpr) (value.Value, error) {
	left, err := node.Left.Accept(p)
	if err != nil {
		return value.NilValue, err
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
	for {
		cond, err := whileStmt.Condition.Accept(p)
		if err != nil {
			return err
		}

		if !helpers.IsTruthy(cond) {
			return nil
		}

		if err := whileStmt.Body.Accept(p); err != nil && err != errContinue {
			if err == errBreak {
				return nil
			}

			return err
		}

		if whileStmt.Increment != nil {
			if _, err := whileStmt.Increment.Accept(p); err != nil {
				return err
			}
		}
	}
}

func (p *Interpreter) VisitBreakStmt(*ast.BreakStmt) error { return errBreak }

func (p *Interpreter) VisitContinueStmt(*ast.ContinueStmt) error { return errContinue }

// VisitForInStmt runs the body once per element, each time in a fresh scope,
// so closures created in the body capture that iteration's variables. The
// elements are taken when the loop starts; changes made to the iterable by
// the body do not affect which elements are visited.
func (p *Interpreter) VisitForInStmt(stmt *ast.ForInStmt) error {
	iterable, err := stmt.Iterable.Accept(p)
	if err != nil {
		return err
	}

	var keys, vals []value.Value

	switch iterable.Kind() {
	case value.Array:
		vals = slices.Clone(iterable.Arr())
		keys = make([]value.Value, len(vals))
		for i := range vals {
			keys[i] = value.NewNumber(float64(i))
		}
	case value.String:
		for i, r := range []rune(iterable.Str()) {
			keys = append(keys, value.NewNumber(float64(i)))
			vals = append(vals, value.NewString(string(r)))
		}
	case value.Object:
		obj := iterable.Obj()
		for _, k := range slices.Sorted(maps.Keys(obj)) {
			keys = append(keys, value.NewString(k))
			vals = append(vals, obj[k])
		}
		// With one variable, iterating an object yields its keys.
		if stmt.Key == nil {
			vals = keys
		}
	default:
		return errors.NewRuntimeError(stmt.Keyword, fmt.Sprintf(
			"Can only iterate over arrays, strings and objects, got %s.", helpers.TypeName(iterable)))
	}

	body := []ast.Stmt{stmt.Body}

	for i := range vals {
		env := environment.NewEnvironment(p.environment, stmt.EnvSize)
		slot := 0

		if stmt.Key != nil {
			env.Set(0, keys[i])
			slot = 1
		}

		env.Set(slot, vals[i])

		if err := p.executeBlock(body, env); err != nil && err != errContinue {
			if err == errBreak {
				return nil
			}

			return err
		}
	}

	return nil
}

func (p *Interpreter) VisitVarStmt(varStmt *ast.VarStmt) error {
	v := value.NilValue

	if varStmt.Initializer != nil {
		var err error
		if v, err = varStmt.Initializer.Accept(p); err != nil {
			return err
		}
	}

	p.declare(varStmt.Local, varStmt.Slot, varStmt.Name.Lexeme, v)

	return nil
}

func (p *Interpreter) VisitCallExpr(callExpr *ast.CallExpr) (value.Value, error) {
	if p.recursionDepth >= p.maxRecursion {
		return value.NilValue, errors.NewRuntimeError(callExpr.Token, "Stack overflow.")
	}

	p.recursionDepth++
	result, err := p.call(callExpr)
	p.recursionDepth--

	return result, err
}

func (p *Interpreter) call(callExpr *ast.CallExpr) (value.Value, error) {
	callee, err := callExpr.Callee.Accept(p)
	if err != nil {
		return value.NilValue, err
	}

	if callee.Kind() == value.Function {
		switch fn := callee.Fn().(type) {
		case *callable.FunctionCallable:
			return p.callFunction(fn, callExpr)
		case callable.Callable:
			return p.callNative(fn, callExpr)
		}
	}

	if _, err := p.evalArgs(callExpr.Args); err != nil {
		return value.NilValue, err
	}

	return value.NilValue, errors.NewRuntimeError(callExpr.Token, "Can only call functions.")
}

// callFunction evaluates the arguments straight into the parameter slots of
// the new environment, so a user-defined call needs no argument slice.
func (p *Interpreter) callFunction(fn *callable.FunctionCallable, callExpr *ast.CallExpr) (value.Value, error) {
	decl := fn.Declaration

	if len(callExpr.Args) != len(decl.Parameters) {
		if _, err := p.evalArgs(callExpr.Args); err != nil {
			return value.NilValue, err
		}

		return value.NilValue, arityError(callExpr, len(decl.Parameters))
	}

	env := environment.NewEnvironment(fn.Closure, decl.EnvSize)

	for i, arg := range callExpr.Args {
		v, err := arg.Accept(p)
		if err != nil {
			return value.NilValue, err
		}

		env.Set(i, v)
	}

	err := p.executeBlock(decl.Body, env)

	if err == errReturn {
		result := p.returnValue
		p.returnValue = value.NilValue

		return result, nil
	}

	return value.NilValue, err
}

func (p *Interpreter) callNative(fn callable.Callable, callExpr *ast.CallExpr) (value.Value, error) {
	args, err := p.evalArgs(callExpr.Args)
	if err != nil {
		return value.NilValue, err
	}

	if arity := fn.Arity(); arity != -1 && len(args) != arity {
		return value.NilValue, arityError(callExpr, arity)
	}

	return fn.Call(args, callExpr.Token)
}

func (p *Interpreter) evalArgs(exprs []ast.Expr) ([]value.Value, error) {
	args := make([]value.Value, len(exprs))

	for i, arg := range exprs {
		v, err := arg.Accept(p)
		if err != nil {
			return nil, err
		}

		args[i] = v
	}

	return args, nil
}

func arityError(callExpr *ast.CallExpr, arity int) error {
	return errors.NewRuntimeError(
		callExpr.Token,
		fmt.Sprintf("Expected %d arguments but got %d.", arity, len(callExpr.Args)),
	)
}

func (p *Interpreter) VisitFunctionExpr(expr *ast.FunctionExpr) (value.Value, error) {
	return value.NewFunction(callable.NewFunctionCallable(expr.Decl, p.environment)), nil
}

func (p *Interpreter) VisitFunctionStmt(functionStmt *ast.FunctionStmt) error {
	fn := value.NewFunction(callable.NewFunctionCallable(functionStmt, p.environment))
	p.declare(functionStmt.Local, functionStmt.Slot, functionStmt.Name.Lexeme, fn)

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
	// A block without declarations needs no scope of its own.
	if blockStmt.EnvSize == 0 {
		return p.EvaluateStmts(blockStmt.Stmts)
	}

	return p.executeBlock(blockStmt.Stmts, environment.NewEnvironment(p.environment, blockStmt.EnvSize))
}

func (p *Interpreter) executeBlock(statements []ast.Stmt, env *environment.Environment) error {
	prevEnv := p.environment
	p.environment = env

	err := p.EvaluateStmts(statements)

	p.environment = prevEnv

	return err
}

func (p *Interpreter) VisitVarExpr(node *ast.VarExpr) (value.Value, error) {
	if node.Depth != ast.Unresolved {
		return p.environment.GetAt(node.Depth, node.Slot), nil
	}

	return p.globals.Get(globalSlot(node.Slot, node.Name.Lexeme), node.Name)
}

func (p *Interpreter) VisitAssignExpr(node *ast.AssignExpr) (value.Value, error) {
	v, err := node.Value.Accept(p)
	if err != nil {
		return value.NilValue, err
	}

	if node.Depth != ast.Unresolved {
		p.environment.AssignAt(node.Depth, node.Slot, v)
	} else if err := p.globals.Assign(globalSlot(node.Slot, node.Name.Lexeme), node.Name, v); err != nil {
		return value.NilValue, err
	}

	return v, nil
}

func (p *Interpreter) VisitBinaryExpr(node *ast.BinaryExpr) (value.Value, error) {
	left, err := node.Left.Accept(p)
	if err != nil {
		return value.NilValue, err
	}

	right, err := node.Right.Accept(p)
	if err != nil {
		return value.NilValue, err
	}

	// Fast path for the common number-number case, kept inline here because
	// binaryOp is too big for the compiler to inline.
	if left.IsNumber() && right.IsNumber() {
		l, r := left.Num(), right.Num()

		switch node.Operator.TokenType {
		case ast.PLUS:
			return value.NewNumber(l + r), nil
		case ast.MINUS:
			return value.NewNumber(l - r), nil
		case ast.STAR:
			return value.NewNumber(l * r), nil
		case ast.LESS:
			return value.NewBool(l < r), nil
		case ast.GREATER:
			return value.NewBool(l > r), nil
		}
	}

	return binaryOp(&node.Operator, left, right)
}

// binaryOp applies a binary operator; operator also locates errors.
func binaryOp(operator *ast.Token, left, right value.Value) (value.Value, error) {
	op := operator.TokenType

	switch op {
	case ast.EQUAL_EQUAL:
		return value.NewBool(helpers.IsEqual(left, right)), nil
	case ast.BANG_EQUAL:
		return value.NewBool(!helpers.IsEqual(left, right)), nil
	case ast.PLUS:
		if left.IsNumber() && right.IsNumber() {
			return value.NewNumber(left.Num() + right.Num()), nil
		}

		if left.IsString() && right.IsString() {
			return value.NewString(left.Str() + right.Str()), nil
		}

		return value.NilValue, errors.NewRuntimeError(*operator, "Operands must be two numbers or two strings.")
	}

	if !left.IsNumber() || !right.IsNumber() {
		return value.NilValue, errors.NewRuntimeError(*operator, "Operands must be numbers.")
	}

	l, r := left.Num(), right.Num()

	switch op {
	case ast.MINUS:
		return value.NewNumber(l - r), nil
	case ast.SLASH:
		return value.NewNumber(l / r), nil
	case ast.STAR:
		return value.NewNumber(l * r), nil
	case ast.PERCENT:
		return value.NewNumber(math.Mod(l, r)), nil
	case ast.LESS:
		return value.NewBool(l < r), nil
	case ast.LESS_EQUAL:
		return value.NewBool(l <= r), nil
	case ast.GREATER:
		return value.NewBool(l > r), nil
	case ast.GREATER_EQUAL:
		return value.NewBool(l >= r), nil
	}

	return value.NilValue, nil
}

func (p *Interpreter) VisitLiteralExpr(node *ast.LiteralExpr) (value.Value, error) {
	return node.Value, nil
}

func (p *Interpreter) VisitGroupingExpr(node *ast.GroupingExpr) (value.Value, error) {
	return node.Expr.Accept(p)
}

func (p *Interpreter) VisitUnaryExpr(node *ast.UnaryExpr) (value.Value, error) {
	right, err := node.Right.Accept(p)
	if err != nil {
		return value.NilValue, err
	}

	switch node.Operator.TokenType {
	case ast.BANG:
		return value.NewBool(!helpers.IsTruthy(right)), nil
	case ast.MINUS:
		if !right.IsNumber() {
			return value.NilValue, errors.NewRuntimeError(node.Operator, "Operand must be a number.")
		}

		return value.NewNumber(-right.Num()), nil
	}

	return value.NilValue, nil
}

func (p *Interpreter) VisitArrayExpr(node *ast.ArrayExpr) (value.Value, error) {
	items, err := p.evalArgs(node.Items)
	if err != nil {
		return value.NilValue, err
	}

	return value.NewArray(items), nil
}

func (p *Interpreter) VisitIndexExpr(node *ast.IndexExpr) (value.Value, error) {
	// Evaluate the object (could be an array or an object)
	target, err := node.Array.Accept(p)
	if err != nil {
		return value.NilValue, err
	}

	index, err := node.Index.Accept(p)
	if err != nil {
		return value.NilValue, err
	}

	return p.getIndex(node.Token, target, index)
}

// getIndex reads target[index] for an array or object target.
func (p *Interpreter) getIndex(token ast.Token, target, index value.Value) (value.Value, error) {
	switch target.Kind() {
	case value.Array:
		arr := target.Arr()

		idx, err := arrayIndex(token, index, len(arr))
		if err != nil {
			return value.NilValue, err
		}

		return arr[idx], nil

	case value.Object:
		if !index.IsString() {
			return value.NilValue, errors.NewRuntimeError(token, "Object keys must be strings.")
		}

		key := index.Str()

		v, exists := target.Obj()[key]
		if !exists {
			return value.NilValue, errors.NewRuntimeError(token, fmt.Sprintf("Undefined property '%s'.", key))
		}

		return v, nil
	}

	return value.NilValue, errors.NewRuntimeError(token, "Indexing is only supported on arrays and objects.")
}

func (p *Interpreter) VisitSetIndexExpr(node *ast.SetIndexExpr) (value.Value, error) {
	target, err := node.Array.Accept(p)
	if err != nil {
		return value.NilValue, err
	}

	index, err := node.Index.Accept(p)
	if err != nil {
		return value.NilValue, err
	}

	// For a compound assignment, read the old value before evaluating the
	// right-hand side, as "a[i] = a[i] op v" would.
	var old value.Value

	if node.Op != nil {
		if old, err = p.getIndex(node.Token, target, index); err != nil {
			return value.NilValue, err
		}
	}

	v, err := node.Value.Accept(p)
	if err != nil {
		return value.NilValue, err
	}

	if node.Op != nil {
		if v, err = binaryOp(node.Op, old, v); err != nil {
			return value.NilValue, err
		}
	}

	switch target.Kind() {
	case value.Array:
		arr := target.Arr()

		idx, err := arrayIndex(node.Token, index, len(arr))
		if err != nil {
			return value.NilValue, err
		}

		arr[idx] = v
		return v, nil

	case value.Object:
		if !index.IsString() {
			return value.NilValue, errors.NewRuntimeError(
				node.Token,
				"Object properties must be accessed with string keys.",
			)
		}

		target.Obj()[index.Str()] = v
		return v, nil

	default:
		return value.NilValue, errors.NewRuntimeError(
			node.Token,
			"Indexing is only supported on arrays and objects.",
		)
	}
}

func (p *Interpreter) VisitObjectExpr(node *ast.ObjectExpr) (value.Value, error) {
	obj := make(map[string]value.Value, len(node.Pairs))

	for _, pair := range node.Pairs {
		v, err := pair.Value.Accept(p)
		if err != nil {
			return value.NilValue, err
		}

		obj[pair.Key] = v
	}

	return value.NewObject(obj), nil
}

// arrayIndex validates that v is an integer index within [0, length).
func arrayIndex(token ast.Token, v value.Value, length int) (int, error) {
	if !v.IsNumber() || v.Num() != math.Trunc(v.Num()) {
		return 0, errors.NewRuntimeError(token, "Array index must be an integer.")
	}

	f := v.Num()
	if f < 0 || f >= float64(length) {
		return 0, errors.NewRuntimeError(token, fmt.Sprintf("Index out of bounds: %v of %v", f, length))
	}

	return int(f), nil
}
