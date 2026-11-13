package rune

import (
	"cmp"
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/stdlib"
	"rune/pkg/value"
	"strings"
	"unicode/utf8"
)

func undefinedProperty(token *ast.Token, key string) error {
	return errors.NewRuntimeError(errors.KeyError, *token, fmt.Sprintf("Undefined property '%s'.", key))
}

func undefinedField(token *ast.Token, shape *value.Shape, key string) error {
	return errors.NewRuntimeError(errors.KeyError, *token, errors.NoField(shape.Name, key))
}

func readOnlyField(token *ast.Token, shape *value.Shape, key string) error {
	if shape.Namespace {
		return errors.NewRuntimeError(errors.TypeError, *token, fmt.Sprintf("Cannot set member '%s' of a namespace.", key))
	}

	return errors.NewRuntimeError(errors.TypeError, *token, fmt.Sprintf(
		"Cannot set field '%s' of %s outside its mut self methods.", key, shape.Name))
}

// setField copies the struct target with the field key names set to v.
func setField(token *ast.Token, target, key, v value.Value) (value.Value, error) {
	if !key.IsString() {
		return value.NilValue, errors.NewRuntimeError(errors.TypeError, *token, "Struct fields must be accessed with string keys.")
	}

	slot := target.Shape().Index(key.Str())
	if slot < 0 {
		return value.NilValue, undefinedField(token, target.Shape(), key.Str())
	}

	return target.With(slot, v), nil
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
		return readOnlyField(token, target.Shape(), token.Lexeme)
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
	case value.Map:
		return v, errors.NewRuntimeError(errors.KeyError, *token, "Key not found: "+helpers.Quoted(index)+".")
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
	case value.Array, value.String, value.Bytes:
		idx, ok, err := elementIndex(token, target, index)
		switch {
		case !ok || err != nil:
			return value.NilValue, false, err
		case target.IsString():
			return value.NewString(helpers.CharAt(target.Str(), idx)), true, nil
		case target.Kind() == value.Bytes:
			return value.NewInt(int64(target.Bytes()[idx])), true, nil
		default:
			return target.Arr()[idx], true, nil
		}

	case value.Object:
		if !index.IsString() {
			return value.NilValue, false, errors.NewRuntimeError(errors.TypeError, *token, "Object keys must be strings.")
		}

		v, found := target.Obj().Get(index.Str())

		return v, found, nil

	case value.Map:
		v, found, err := target.HashMap().Get(index)
		if err != nil {
			return v, false, keyError(token, err)
		}

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
		"Can only index arrays, strings, objects, structs, maps and bytes, got %s.", helpers.TypeName(target)))
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

		target.Obj().Set(index.Str(), v)

		return nil

	case value.Map:
		if err := target.HashMap().Set(index, v); err != nil {
			return keyError(token, err)
		}

		return nil

	case value.String:
		return errors.NewRuntimeError(errors.TypeError, *token, "Strings are immutable.")

	case value.Bytes:
		return errors.NewRuntimeError(errors.TypeError, *token, "Bytes are immutable.")

	case value.Struct:
		if !index.IsString() {
			return errors.NewRuntimeError(errors.TypeError, *token, "Struct fields must be accessed with string keys.")
		}

		return readOnlyField(token, target.Shape(), index.Str())

	default:
		return errors.NewRuntimeError(errors.TypeError, *token, fmt.Sprintf(
			"Can only index arrays, strings, objects, structs, maps and bytes, got %s.", helpers.TypeName(target)))
	}
}

// elementIndex checks that v is an integer and reports whether it is in
// bounds of the array, string or bytes target; a negative one counts from
// the end.
func elementIndex(token *ast.Token, target, v value.Value) (int, bool, error) {
	if !v.IsInt() {
		return 0, false, notInteger(token, target)
	}

	i, n := v.Int(), int64(length(target))
	if i < 0 {
		i += n
	}

	if i < 0 || i >= n {
		return 0, false, nil
	}

	return int(i), true, nil
}

// keyError is a key a Map or a Set cannot hold.
func keyError(token *ast.Token, err error) error {
	return errors.NewRuntimeError(errors.TypeError, *token, err.Error())
}

func notInteger(token *ast.Token, target value.Value) error {
	kind := helpers.TypeName(target)
	return errors.NewRuntimeError(errors.TypeError, *token, strings.ToUpper(kind[:1])+kind[1:]+" index must be an integer.")
}

func outOfBounds(token *ast.Token, target, v value.Value) error {
	return errors.NewRuntimeError(errors.IndexError, *token, fmt.Sprintf("Index out of bounds: %s of %d", helpers.Stringify(v), length(target)))
}

// length is the number of elements of an array, or characters of a string.
func length(target value.Value) int {
	switch target.Kind() {
	case value.Array:
		return len(target.Arr())
	case value.Bytes:
		return len(target.Bytes())
	}

	return utf8.RuneCountInString(target.Str())
}
