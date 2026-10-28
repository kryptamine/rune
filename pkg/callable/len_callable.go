package callable

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"unicode/utf8"
)

// LenCallable is a callable that returns the length of an array or string.
type LenCallable struct{}

func NewLenCallable() Callable {
	return &LenCallable{}
}

func (c *LenCallable) Call(_ ExecuteBlockFn, args []any, token ast.Token) (any, error) {
	switch v := args[0].(type) {
	case []any:
		return float64(len(v)), nil
	case map[string]any:
		return float64(len(v)), nil
	case string:
		return float64(utf8.RuneCountInString(v)), nil
	default:
		return nil, errors.NewRuntimeError(token, fmt.Sprintf("len() can only be called on strings, arrays and objects, got %s", helpers.TypeName(args[0])))
	}
}

func (c *LenCallable) Arity() int {
	return 1
}

func (c *LenCallable) String() string {
	return "<native fn>"
}
