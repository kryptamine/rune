package callable

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"unicode/utf8"
)

// LenCallable is a callable that returns the length of an array or string.
type LenCallable struct{}

func NewLenCallable() Callable {
	return &LenCallable{}
}

func (c *LenCallable) Call(args []value.Value, token ast.Token) (value.Value, error) {
	switch v := args[0]; v.Kind() {
	case value.Array:
		return value.NewNumber(float64(len(v.Arr()))), nil
	case value.Object:
		return value.NewNumber(float64(len(v.Obj()))), nil
	case value.String:
		return value.NewNumber(float64(utf8.RuneCountInString(v.Str()))), nil
	default:
		return value.NilValue, errors.NewRuntimeError(token, fmt.Sprintf("len() can only be called on strings, arrays and objects, got %s", helpers.TypeName(args[0])))
	}
}

func (c *LenCallable) Arity() int {
	return 1
}

func (c *LenCallable) String() string {
	return "<native fn>"
}
