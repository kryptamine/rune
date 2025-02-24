package callable

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
)

type AppendCallable struct{}

func NewAppendCallable() Callable {
	return &AppendCallable{}
}

func (c *AppendCallable) Call(_ ExecuteBlockFn, args []any, token ast.Token) (any, error) {
	if len(args) < 2 {
		return nil, errors.NewRuntimeError(token, "Can't append to nothing, pass an array to append to. Example: append([1, 2, 3], 4)")
	}

	switch v := args[0].(type) {
	case []any:
		v = append(v, args[1:]...)
		return v, nil
	default:
		return 0, errors.NewRuntimeError(token, fmt.Sprintf("Can only append to arrays, got %T", args[0]))
	}
}

func (c *AppendCallable) Arity() int {
	return -1
}

func (c *AppendCallable) String() string {
	return "<native fn>"
}
