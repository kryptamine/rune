package callable

import (
	"rune/pkg/ast"
	"rune/pkg/environment"
)

const MaxArity = 8

// Return is a special type of error that is returned by a function
type Return struct {
	value any
}

func NewReturn(value any) *Return {
	return &Return{value}
}

func (e *Return) Error() string {
	return "<fn return>"
}

type ExecuteBlockFn func(statements []ast.Stmt, env *environment.Environment) error

type Callable interface {
	Call(executeBlock ExecuteBlockFn, args []any, token ast.Token) (any, error)
	Arity() int
}
