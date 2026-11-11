package rune

import (
	"fmt"
	"math"
	"rune/pkg/ast"
	"rune/pkg/environment"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
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

// getIndex reads target[index] for an array or object target.
func getIndex(token *ast.Token, target, index value.Value) (value.Value, error) {
	switch target.Kind() {
	case value.Array:
		arr := target.Arr()

		idx, err := arrayIndex(token, index, len(arr))
		if err != nil {
			return value.NilValue, err
		}

		return arr[idx], nil

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

	return value.NilValue, errors.NewRuntimeError(*token, "Indexing is only supported on arrays and objects.")
}

func setIndex(token *ast.Token, target, index, v value.Value) error {
	switch target.Kind() {
	case value.Array:
		arr := target.Arr()

		idx, err := arrayIndex(token, index, len(arr))
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

	default:
		return errors.NewRuntimeError(*token, "Indexing is only supported on arrays and objects.")
	}
}

// arrayIndex validates that v is an integer index within [0, length).
func arrayIndex(token *ast.Token, v value.Value, length int) (int, error) {
	if !v.IsNumber() || v.Num() != math.Trunc(v.Num()) {
		return 0, errors.NewRuntimeError(*token, "Array index must be an integer.")
	}

	f := v.Num()
	if f < 0 || f >= float64(length) {
		return 0, errors.NewRuntimeError(*token, fmt.Sprintf("Index out of bounds: %v of %v", f, length))
	}

	return int(f), nil
}

func arityError(token *ast.Token, arity, got int) error {
	return errors.NewRuntimeError(*token, fmt.Sprintf("Expected %d arguments but got %d.", arity, got))
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
