package callable

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
)

// LenCallable is a callable that returns the length of an array or string.
type LenCallable struct{}

func NewLenCallable() Callable {
	return &LenCallable{}
}

func (c *LenCallable) Call(_ ExecuteBlockFn, args []any, token ast.Token) (any, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("len() requires one argument")
	}

	switch v := args[0].(type) {
	case []any:
		return float64(len(v)), nil
	case string:
		return float64(len(v)), nil
	default:
		return 0, errors.NewRuntimeError(token, fmt.Sprintf("len() can only be called on strings and arrays, got %T", args[0]))
	}
}

func (c *LenCallable) Arity() int {
	return 1
}

func (c *LenCallable) String() string {
	return "<native fn>"
}
