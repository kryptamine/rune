package main

import (
	"fmt"
	"time"
)

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
