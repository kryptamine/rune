package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/value"
	"slices"
)

// The members every enum has: Enum.values() lists its variants, and
// v.name names a variant.
const (
	enumValues = "values"
	enumName   = "name"
)

// enumProperty reads key off a variant, its name, or off an enum type, a
// variant.
func enumProperty(t value.Value, key string) (value.Value, bool) {
	switch {
	case t.Kind() == value.Enum && key == enumName:
		return value.NewString(t.Variant().Name), true
	case t.Kind() == value.Type && t.TypeShape().Enum:
		return t.TypeShape().Variant(key)
	}

	return value.NilValue, false
}

func isValuesCall(t value.Value, name string) bool {
	return t.Kind() == value.Type && t.TypeShape().Enum && name == enumValues
}

// variants is enum.values() with n arguments.
func variants(tok *ast.Token, enum *value.Shape, n int) (value.Value, error) {
	if n > 0 {
		return value.NilValue, errors.NewRuntimeError(errors.ArgumentError, *tok, fmt.Sprintf(
			"%s.%s() expects 0 arguments, got %d.", enum.Name, enumValues, n))
	}

	return value.NewArray(slices.Clone(enum.Variants)), nil
}
