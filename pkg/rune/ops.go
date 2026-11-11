package rune

import (
	"fmt"
	"math"
	"rune/pkg/ast"
	"rune/pkg/environment"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"unicode/utf8"
)

// globalSlot returns slot, looking it up by name when the resolver has not
// run (as for a lone expression evaluated with EvaluateExpr).
func globalSlot(slot int, name string) int {
	if slot == ast.Unresolved {
		return environment.GlobalSlot(name)
	}

	return slot
}

// binaryOp applies a binary operator; operator also locates errors.
func binaryOp(operator *ast.Token, left, right value.Value) (value.Value, error) {
	op := operator.TokenType

	switch op {
	case ast.EQUAL_EQUAL:
		return value.NewBool(helpers.IsEqual(left, right)), nil
	case ast.BANG_EQUAL:
		return value.NewBool(!helpers.IsEqual(left, right)), nil
	case ast.PLUS:
		if left.IsNumber() && right.IsNumber() {
			return value.NewNumber(left.Num() + right.Num()), nil
		}

		if left.IsString() && right.IsString() {
			return value.NewString(left.Str() + right.Str()), nil
		}

		return value.NilValue, errors.NewRuntimeError(*operator, "Operands must be two numbers or two strings.")
	}

	if isComparison(op) && left.IsString() && right.IsString() {
		return value.NewBool(compareStrings(op, left.Str(), right.Str())), nil
	}

	if isComparison(op) && (!left.IsNumber() || !right.IsNumber()) {
		return value.NilValue, errors.NewRuntimeError(*operator, "Operands must be two numbers or two strings.")
	}

	if !left.IsNumber() || !right.IsNumber() {
		return value.NilValue, errors.NewRuntimeError(*operator, "Operands must be numbers.")
	}

	return numberOp(op, left.Num(), right.Num()), nil
}

func numberOp(op ast.TokenType, l, r float64) value.Value {
	switch op {
	case ast.PLUS:
		return value.NewNumber(l + r)
	case ast.MINUS:
		return value.NewNumber(l - r)
	case ast.SLASH:
		return value.NewNumber(l / r)
	case ast.STAR:
		return value.NewNumber(l * r)
	case ast.PERCENT:
		return value.NewNumber(mod(l, r))
	case ast.LESS:
		return value.NewBool(l < r)
	case ast.LESS_EQUAL:
		return value.NewBool(l <= r)
	case ast.GREATER:
		return value.NewBool(l > r)
	case ast.GREATER_EQUAL:
		return value.NewBool(l >= r)
	case ast.EQUAL_EQUAL:
		return value.NewBool(l == r)
	case ast.BANG_EQUAL:
		return value.NewBool(l != r)
	}

	return value.NilValue
}

func isComparison(op ast.TokenType) bool {
	switch op {
	case ast.LESS, ast.LESS_EQUAL, ast.GREATER, ast.GREATER_EQUAL:
		return true
	}

	return false
}

func compareNumbers(op ast.TokenType, l, r float64) bool {
	switch op {
	case ast.LESS:
		return l < r
	case ast.LESS_EQUAL:
		return l <= r
	case ast.GREATER:
		return l > r
	default:
		return l >= r
	}
}

func compareStrings(op ast.TokenType, l, r string) bool {
	switch op {
	case ast.LESS:
		return l < r
	case ast.LESS_EQUAL:
		return l <= r
	case ast.GREATER:
		return l > r
	default:
		return l >= r
	}
}

// isProperty reports whether token comes from obj.name rather than obj[i].
func isProperty(token *ast.Token) bool { return token.TokenType == ast.IDENTIFIER }

// getIndex reads target[index] for an array, string or object target.
func getIndex(token *ast.Token, target, index value.Value) (value.Value, error) {
	if isProperty(token) && target.Kind() != value.Object {
		return value.NilValue, errors.NewRuntimeError(*token, fmt.Sprintf(
			"Cannot read property '%s' of %s.", token.Lexeme, helpers.TypeName(target)))
	}

	switch target.Kind() {
	case value.Array:
		arr := target.Arr()

		idx, err := arrayIndex(token, value.Array, index, len(arr))
		if err != nil {
			return value.NilValue, err
		}

		return arr[idx], nil

	case value.String:
		str := target.Str()

		idx, err := arrayIndex(token, value.String, index, utf8.RuneCountInString(str))
		if err != nil {
			return value.NilValue, err
		}

		return value.NewString(charAt(str, idx)), nil

	case value.Object:
		if !index.IsString() {
			return value.NilValue, errors.NewRuntimeError(*token, "Object keys must be strings.")
		}

		key := index.Str()

		v, exists := target.Obj()[key]
		if !exists {
			return value.NilValue, errors.NewRuntimeError(*token, fmt.Sprintf("Undefined property '%s'.", key))
		}

		return v, nil
	}

	return value.NilValue, errors.NewRuntimeError(*token, fmt.Sprintf(
		"Can only index arrays, strings and objects, got %s.", helpers.TypeName(target)))
}

// charAt returns the idx-th character of s; idx is in range.
func charAt(s string, idx int) string {
	for off := range s {
		if idx == 0 {
			r, n := utf8.DecodeRuneInString(s[off:])
			if r == utf8.RuneError && n == 1 {
				return string(utf8.RuneError)
			}

			return s[off : off+n]
		}

		idx--
	}

	return ""
}

func setIndex(token *ast.Token, target, index, v value.Value) error {
	if isProperty(token) && target.Kind() != value.Object {
		return errors.NewRuntimeError(*token, fmt.Sprintf(
			"Cannot set property '%s' of %s.", token.Lexeme, helpers.TypeName(target)))
	}

	switch target.Kind() {
	case value.Array:
		arr := target.Arr()

		idx, err := arrayIndex(token, value.Array, index, len(arr))
		if err != nil {
			return err
		}

		arr[idx] = v

		return nil

	case value.Object:
		if !index.IsString() {
			return errors.NewRuntimeError(*token, "Object properties must be accessed with string keys.")
		}

		target.Obj()[index.Str()] = v

		return nil

	case value.String:
		return errors.NewRuntimeError(*token, "Strings are immutable.")

	default:
		return errors.NewRuntimeError(*token, fmt.Sprintf(
			"Can only index arrays, strings and objects, got %s.", helpers.TypeName(target)))
	}
}

// arrayIndex validates that v is an integer index within [0, length).
func arrayIndex(token *ast.Token, kind value.Kind, v value.Value, length int) (int, error) {
	if !v.IsNumber() || v.Num() != math.Trunc(v.Num()) {
		return 0, errors.NewRuntimeError(*token, helpers.If(kind == value.String, "String", "Array")+" index must be an integer.")
	}

	f := v.Num()
	if f < 0 || f >= float64(length) {
		return 0, errors.NewRuntimeError(*token, fmt.Sprintf("Index out of bounds: %v of %v", f, length))
	}

	return int(f), nil
}

const maxExactInt = 1 << 53

// mod matches math.Mod, including the sign of zero results.
func mod(l, r float64) float64 {
	if l > -maxExactInt && l < maxExactInt && r > -maxExactInt && r < maxExactInt && r != 0 {
		li, ri := int64(l), int64(r)

		if float64(li) == l && float64(ri) == r {
			return math.Copysign(float64(li%ri), l)
		}
	}

	return math.Mod(l, r)
}
