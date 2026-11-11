package callable

import (
	"rune/pkg/ast"
	"rune/pkg/value"
)

const MaxArity = 8

// Callable is a native (built-in) function. User-defined functions are
// closures, which the interpreter calls directly.
type Callable interface {
	Call(args []value.Value, token ast.Token) (value.Value, error)
	// Arity is the exact number of arguments, or -1 for variadic.
	Arity() int
}
