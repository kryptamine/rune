package callable

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/environment"
)

// FunctionCallable is a user-defined function together with its closure.
type FunctionCallable struct {
	Declaration *ast.FunctionStmt
	Closure     *environment.Environment
}

func NewFunctionCallable(declaration *ast.FunctionStmt, closure *environment.Environment) *FunctionCallable {
	return &FunctionCallable{
		Declaration: declaration,
		Closure:     closure,
	}
}

func (f *FunctionCallable) Arity() int {
	return len(f.Declaration.Parameters)
}

func (f *FunctionCallable) String() string {
	return fmt.Sprintf("<fn %s>", f.Declaration.Name.Lexeme)
}
