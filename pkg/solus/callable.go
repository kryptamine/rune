package solus

import "github.com/codecrafters-io/interpreter-starter-go/pkg/ast"

const MaxArity = 255

// Return is a special type of error that is returned by a function
type Return struct {
	value any
}

func (e *Return) Error() string {
	return "<fn return>"
}

type Callable interface {
	Call(interpreter *Interpreter, args []any, token ast.Token) (any, error)
	Arity() int
}
