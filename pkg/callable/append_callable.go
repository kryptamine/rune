package callable

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
)

type AppendCallable struct{}

func NewAppendCallable() Callable {
	return &AppendCallable{}
}

func (c *AppendCallable) Call(args []value.Value, token ast.Token) (value.Value, error) {
	if len(args) < 2 {
		return value.NilValue, errors.NewRuntimeError(token, "Can't append to nothing, pass an array to append to. Example: append([1, 2, 3], 4)")
	}

	switch args[0].Kind() {
	case value.Array:
		// Always copy so the result never shares a backing array with the input.
		return value.NewArray(slices.Concat(args[0].Arr(), args[1:])), nil
	default:
		return value.NilValue, errors.NewRuntimeError(token, fmt.Sprintf("Can only append to arrays, got %s", helpers.TypeName(args[0])))
	}
}

func (c *AppendCallable) Arity() int {
	return -1
}

func (c *AppendCallable) String() string {
	return "<native fn>"
}
