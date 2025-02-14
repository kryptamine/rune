package main

import (
	"fmt"
	"time"
)

type Callable interface {
	Call(interpreter *Interpreter, args []any) (any, error)
}

type Function struct {
	declaration *FunctionStmt
}

func (f *Function) Call(interpreter *Interpreter, args []any) (any, error) {
	env := NewEnvironment(interpreter.environment)

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

func (f *Function) String() string {
	return fmt.Sprintf("<fn %s>", f.declaration.name.lexeme)
}

type ClockCallable struct{}

func (c *ClockCallable) Call(interpreter *Interpreter, args []any) (any, error) {
	return float64(time.Now().Unix()), nil
}

func (c *ClockCallable) String() string {
	return "<native fn>"
}
