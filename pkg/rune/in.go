package rune

import (
	"bytes"
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
	"strings"
)

// contains is x in xs: an element of an array, a substring of a string,
// a key of a map or a set, a byte or bytes in bytes or a
// buffer.
func contains(tok *ast.Token, x, xs value.Value) (bool, error) {
	switch xs.Kind() {
	case value.Array:
		return slices.ContainsFunc(xs.Arr(), func(v value.Value) bool { return helpers.IsEqual(v, x) }), nil
	case value.String:
		if !x.IsString() {
			return false, inOperand(tok, "a string", x, xs)
		}

		return strings.Contains(xs.Str(), x.Str()), nil
	case value.Map, value.Set:
		_, found, err := xs.HashMap().Get(x)
		if err != nil {
			return false, keyError(tok, err)
		}

		return found, nil
	case value.Bytes, value.Buffer:
		b, _ := xs.Binary()
		if sub, ok := x.Binary(); ok {
			return bytes.Contains(b, sub), nil
		}

		if x.IsInt() {
			return x.Int() >= 0 && x.Int() <= 255 && bytes.IndexByte(b, byte(x.Int())) >= 0, nil
		}

		return false, inOperand(tok, "bytes or an integer", x, xs)
	}

	return false, errors.NewRuntimeError(errors.TypeError, *tok, fmt.Sprintf(
		"Right operand of 'in' must be an array, a string, a map, a set, bytes or a range, got %s.", helpers.TypeName(xs)))
}

func inOperand(tok *ast.Token, want string, x, xs value.Value) error {
	return errors.NewRuntimeError(errors.TypeError, *tok, fmt.Sprintf(
		"Left operand of 'in' must be %s for %s, got %s.", want, helpers.TypeName(xs), helpers.TypeName(x)))
}
