package solus

import (
	"fmt"
)

// FunctionCallable is a callable that represents a function.
type FunctionCallable struct {
	Callable
	declaration *FunctionStmt
	environment *Environment
}

func (f *FunctionCallable) Call(interpreter *Interpreter, args []any, _ Token) (any, error) {
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

func (f *FunctionCallable) Arity() int {
	return len(f.declaration.parameters)
}

func (f *FunctionCallable) String() string {
	return fmt.Sprintf("<fn %s>", f.declaration.name.lexeme)
}
