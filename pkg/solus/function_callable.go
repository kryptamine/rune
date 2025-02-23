package solus

import (
	"fmt"
	"github.com/codecrafters-io/interpreter-starter-go/pkg/ast"
	"github.com/codecrafters-io/interpreter-starter-go/pkg/environment"
)

// FunctionCallable is a callable that represents a function.
type FunctionCallable struct {
	declaration *ast.FunctionStmt
	environment *environment.Environment
}

func (f *FunctionCallable) Call(interpreter *Interpreter, args []any, _ ast.Token) (any, error) {
	env := environment.NewEnvironment(f.environment)

	for i, param := range f.declaration.Parameters {
		if len(args) <= i {
			continue
		}

		env.Define(param.Lexeme, args[i])
	}

	err := interpreter.executeBlock(f.declaration.Body, env)

	if ret, isReturn := err.(*Return); isReturn {
		return ret.value, nil
	}

	return nil, err
}

func (f *FunctionCallable) Arity() int {
	return len(f.declaration.Parameters)
}

func (f *FunctionCallable) String() string {
	return fmt.Sprintf("<fn %s>", f.declaration.Name.Lexeme)
}
