package callable

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
)

type PushCallable struct{}

func NewPushCallable() Callable {
	return &PushCallable{}
}

func (c *PushCallable) Call(args []value.Value, token ast.Token) (value.Value, error) {
	if len(args) < 2 {
		return value.NilValue, errors.NewRuntimeError(token, "push() expects an array and at least one item. Example: push(arr, 4)")
	}

	if args[0].Kind() != value.Array {
		return value.NilValue, errors.NewRuntimeError(token, fmt.Sprintf("Can only push to arrays, got %s", helpers.TypeName(args[0])))
	}

	arr := args[0].ArrRef()
	*arr = append(*arr, args[1:]...)

	return value.NewNumber(float64(len(*arr))), nil
}

func (c *PushCallable) Arity() int {
	return -1
}

func (c *PushCallable) String() string {
	return "<native fn>"
}

type PopCallable struct{}

func NewPopCallable() Callable {
	return &PopCallable{}
}

func (c *PopCallable) Call(args []value.Value, token ast.Token) (value.Value, error) {
	if args[0].Kind() != value.Array {
		return value.NilValue, errors.NewRuntimeError(token, fmt.Sprintf("Can only pop from arrays, got %s", helpers.TypeName(args[0])))
	}

	arr := args[0].ArrRef()
	n := len(*arr)

	if n == 0 {
		return value.NilValue, errors.NewRuntimeError(token, "Can't pop from an empty array.")
	}

	last := (*arr)[n-1]
	(*arr)[n-1] = value.NilValue
	*arr = (*arr)[:n-1]

	return last, nil
}

func (c *PopCallable) Arity() int {
	return 1
}

func (c *PopCallable) String() string {
	return "<native fn>"
}
