package solus

import (
	"fmt"
)

type AppendCallable struct{}

func NewAppendCallable() Callable {
	return &AppendCallable{}
}

func (c *AppendCallable) Call(interpreter *Interpreter, args []any, token Token) (any, error) {
	if len(args) < 2 {
		return nil, NewRuntimeError(token, "Can't append to nothing, pass an array to append to. Example: append([1, 2, 3], 4)")
	}

	switch v := args[0].(type) {
	case []any:
		v = append(v, args[1:]...)
		return v, nil
	default:
		return 0, NewRuntimeError(token, fmt.Sprintf("Can only append to arrays, got %T", args[0]))
	}
}

func (c *AppendCallable) Arity() int {
	return -1
}

func (c *AppendCallable) String() string {
	return "<native fn>"
}
