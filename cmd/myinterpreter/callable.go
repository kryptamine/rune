package main

import (
	"fmt"
	"time"
)

const MaxArity = 255

// Return is a special type of error that is returned by a function
type Return struct {
	value any
}

func (e *Return) Error() string {
	return "<fn return>"
}

type Callable interface {
	Call(interpreter *Interpreter, args []any) (any, error)
	Arity() int
}

type Function struct {
	declaration *FunctionStmt
	environment *Environment
}

func (f *Function) Call(interpreter *Interpreter, args []any) (any, error) {
	env := NewEnvironment(f.environment)

	for i, param := range f.declaration.parameters {
		if len(args) <= i {
			continue
		}

		env.define(param.lexeme, args[i])
	}

	err := interpreter.executeBlock(f.declaration.body, env)

	if ret, isReturn := err.(*Return); isReturn {
		return ret.value, nil
	}

	return nil, err
}

func (f *Function) Arity() int {
	return len(f.declaration.parameters)
}

func (f *Function) String() string {
	return fmt.Sprintf("<fn %s>", f.declaration.name.lexeme)
}

type ClockCallable struct{}

func (c *ClockCallable) Call(interpreter *Interpreter, args []any) (any, error) {
	return float64(time.Now().Unix()), nil
}

func (c *ClockCallable) Arity() int {
	return 0
}

func (c *ClockCallable) String() string {
	return "<native fn>"
}

type LenCallable struct{}

func (c *LenCallable) Call(interpreter *Interpreter, args []any) (any, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("len() requires one argument")
	}

	switch v := args[0].(type) {
	case []any:
		return float64(len(v)), nil
	case string:
		return float64(len(v)), nil
	default:
		return 0, fmt.Errorf("len() expects an array or string, got %T", args[0])
	}
}

func (c *LenCallable) Arity() int {
	return 1
}

func (c *LenCallable) String() string {
	return "<native fn>"
}

type AppendCallable struct{}

func (c *AppendCallable) Call(interpreter *Interpreter, args []any) (any, error) {
	switch v := args[0].(type) {
	case []any:
		v = append(v, args[1:]...)
		return v, nil
	default:
		return 0, fmt.Errorf("append() expects an array, got %T", args[0])
	}
}

func (c *AppendCallable) Arity() int {
	return MaxArity
}
