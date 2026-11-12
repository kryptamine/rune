package rune

import (
	"cmp"
	"fmt"
	"math"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/stdlib"
	"rune/pkg/value"
	"strings"
	"unicode/utf8"
)

// binaryOp applies a binary operator; operator also locates errors.
func binaryOp(operator *ast.Token, left, right value.Value) (value.Value, error) {
	op := operator.TokenType

	bothNumbers := left.IsNumber() && right.IsNumber()

	switch op {
	case ast.EQUAL_EQUAL:
		return value.NewBool(helpers.IsEqual(left, right)), nil
	case ast.BANG_EQUAL:
		return value.NewBool(!helpers.IsEqual(left, right)), nil
	case ast.PLUS, ast.LESS, ast.LESS_EQUAL, ast.GREATER, ast.GREATER_EQUAL:
		if left.IsString() && right.IsString() {
			if op == ast.PLUS {
				return value.Concat(left, right), nil
			}

			return value.NewBool(compare(op, left.Str(), right.Str())), nil
		}

		if !bothNumbers {
			return value.NilValue, errors.NewRuntimeError(errors.TypeError, *operator, "Operands must be two numbers or two strings.")
		}
	default:
		if !bothNumbers {
			return value.NilValue, errors.NewRuntimeError(errors.TypeError, *operator, "Operands must be numbers.")
		}
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

// compare applies one of < <= > >=.
func compare[T cmp.Ordered](op ast.TokenType, l, r T) bool {
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

func undefinedProperty(token *ast.Token, key string) error {
	return errors.NewRuntimeError(errors.KeyError, *token, fmt.Sprintf("Undefined property '%s'.", key))
}

func undefinedField(token *ast.Token, shape *value.Shape, key string) error {
	return errors.NewRuntimeError(errors.KeyError, *token, errors.NoField(shape.Name, key))
}

func immutableField(token *ast.Token, shape *value.Shape, key string) error {
	return errors.NewRuntimeError(errors.TypeError, *token, fmt.Sprintf(
		"Cannot set field '%s' of %s: structs are immutable.", key, shape.Name))
}

// with copies target, a struct, setting the fields keys to vals.
func with(token *ast.Token, target value.Value, keys []string, vals []value.Value) (value.Value, error) {
	if target.Kind() != value.Struct {
		return value.NilValue, errors.NewRuntimeError(errors.TypeError, *token, fmt.Sprintf(
			"Can only use 'with' on structs, got %s.", helpers.TypeName(target)))
	}

	shape := target.Shape()
	v, fields := value.NewStruct(shape)
	copy(fields, target.Fields())

	for i, k := range keys {
		slot := shape.Index(k)
		if slot < 0 {
			return value.NilValue, undefinedField(token, shape, k)
		}

		fields[slot] = vals[i]
	}

	return v, nil
}

// convert is a call of the struct type shape, which takes one value.
func convert(token *ast.Token, shape *value.Shape, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return value.NilValue, errors.NewRuntimeError(errors.ArgumentError, *token, fmt.Sprintf(
			"%s() expects 1 argument, got %d.", shape.Name, len(args)))
	}

	return stdlib.Convert(token, shape, args[0])
}

// propertyTarget fails unless target can have a property set, as only
// an object can.
func propertyTarget(token *ast.Token, target value.Value) error {
	switch target.Kind() {
	case value.Object:
		return nil
	case value.Struct:
		return immutableField(token, target.Shape(), token.Lexeme)
	default:
		return propertyError("set", token, target)
	}
}

// propertyError is the error of obj.name (verb "read") or obj.name = v
// ("set") on a target that is not an object.
func propertyError(verb string, token *ast.Token, target value.Value) error {
	return errors.NewRuntimeError(errors.TypeError, *token, fmt.Sprintf(
		"Cannot %s property '%s' of %s.", verb, token.Lexeme, helpers.TypeName(target)))
}

// getIndex reads target[index] for an array, string, object or struct target.
func getIndex(token *ast.Token, target, index value.Value) (value.Value, error) {
	v, found, err := lookup(token, target, index)
	if found || err != nil {
		return v, err
	}

	switch target.Kind() {
	case value.Object:
		return v, undefinedProperty(token, index.Str())
	case value.Struct:
		return v, undefinedField(token, target.Shape(), index.Str())
	default:
		return v, outOfBounds(token, target, index)
	}
}

// lookup is getIndex, but reports a missing element, property or field
// by found instead of an error.
func lookup(token *ast.Token, target, index value.Value) (value.Value, bool, error) {
	switch target.Kind() {
	case value.Array, value.String:
		idx, ok, err := elementIndex(token, target, index)
		switch {
		case !ok || err != nil:
			return value.NilValue, false, err
		case target.IsString():
			return value.NewString(helpers.CharAt(target.Str(), idx)), true, nil
		default:
			return target.Arr()[idx], true, nil
		}

	case value.Object:
		if !index.IsString() {
			return value.NilValue, false, errors.NewRuntimeError(errors.TypeError, *token, "Object keys must be strings.")
		}

		v, found := target.Obj()[index.Str()]

		return v, found, nil

	case value.Struct:
		if !index.IsString() {
			return value.NilValue, false, errors.NewRuntimeError(errors.TypeError, *token, "Struct fields must be accessed with string keys.")
		}

		i := target.Shape().Index(index.Str())
		if i < 0 {
			return value.NilValue, false, nil
		}

		return target.Fields()[i], true, nil
	}

	return value.NilValue, false, errors.NewRuntimeError(errors.TypeError, *token, fmt.Sprintf(
		"Can only index arrays, strings, objects and structs, got %s.", helpers.TypeName(target)))
}

func setIndex(token *ast.Token, target, index, v value.Value) error {
	switch target.Kind() {
	case value.Array:
		idx, ok, err := elementIndex(token, target, index)
		if !ok {
			return cmp.Or(err, outOfBounds(token, target, index))
		}

		target.Arr()[idx] = v

		return nil

	case value.Object:
		if !index.IsString() {
			return errors.NewRuntimeError(errors.TypeError, *token, "Object properties must be accessed with string keys.")
		}

		target.Obj()[index.Str()] = v

		return nil

	case value.String:
		return errors.NewRuntimeError(errors.TypeError, *token, "Strings are immutable.")

	case value.Struct:
		if !index.IsString() {
			return errors.NewRuntimeError(errors.TypeError, *token, "Struct fields must be accessed with string keys.")
		}

		return immutableField(token, target.Shape(), index.Str())

	default:
		return errors.NewRuntimeError(errors.TypeError, *token, fmt.Sprintf(
			"Can only index arrays, strings, objects and structs, got %s.", helpers.TypeName(target)))
	}
}

// elementIndex checks that v is an integer and reports whether it is in
// bounds of the array or string target.
func elementIndex(token *ast.Token, target, v value.Value) (int, bool, error) {
	f := v.Num()
	if !v.IsNumber() || f != math.Trunc(f) {
		return 0, false, notInteger(token, target)
	}

	if f < 0 || f >= float64(length(target)) {
		return 0, false, nil
	}

	return int(f), true, nil
}

func notInteger(token *ast.Token, target value.Value) error {
	kind := helpers.TypeName(target)
	return errors.NewRuntimeError(errors.TypeError, *token, strings.ToUpper(kind[:1])+kind[1:]+" index must be an integer.")
}

func outOfBounds(token *ast.Token, target, v value.Value) error {
	return errors.NewRuntimeError(errors.IndexError, *token, fmt.Sprintf("Index out of bounds: %v of %v", v.Num(), length(target)))
}

// length is the number of elements of an array, or characters of a string.
func length(target value.Value) int {
	if target.Kind() == value.Array {
		return len(target.Arr())
	}

	return utf8.RuneCountInString(target.Str())
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
