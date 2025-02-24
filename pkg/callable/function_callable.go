package callable

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/environment"
)

// FunctionCallable is a callable that represents a function.
type FunctionCallable struct {
	Declaration *ast.FunctionStmt
	environment *environment.Environment
}

func NewFunctionCallable(declaration *ast.FunctionStmt, environment *environment.Environment) *FunctionCallable {
	return &FunctionCallable{
		Declaration: declaration,
		environment: environment,
	}
}

func (f *FunctionCallable) Call(executeBlock ExecuteBlockFn, args []any, _ ast.Token) (any, error) {
	env := environment.NewEnvironment(f.environment)

	for i, param := range f.Declaration.Parameters {
		if len(args) <= i {
			continue
		}

		env.Define(param.Lexeme, args[i])
	}

	err := executeBlock(f.Declaration.Body, env)

	if ret, isReturn := err.(*Return); isReturn {
		return ret.value, nil
	}

	return nil, err
}

func (f *FunctionCallable) Arity() int {
	return len(f.Declaration.Parameters)
}

func (f *FunctionCallable) String() string {
	return fmt.Sprintf("<fn %s>", f.Declaration.Name.Lexeme)
}
