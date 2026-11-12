package errors

import (
	"fmt"
	"rune/pkg/value"
	"slices"
)

// Every built-in error kind has these fields, in this order.
var errorFields = []value.Field{
	{Name: "message", Required: true},
	{Name: "line"},
	{Name: "cause"},
}

var (
	TypeError     = value.NewShape("TypeError", errorFields)
	ArgumentError = value.NewShape("ArgumentError", errorFields)
	ValueError    = value.NewShape("ValueError", errorFields)
	IndexError    = value.NewShape("IndexError", errorFields)
	KeyError      = value.NewShape("KeyError", errorFields)
	NameError     = value.NewShape("NameError", errorFields)
	// IOError also names the path and, for the common failures, a code:
	// "notFound", "exists", "permission", "isDir", "notDir" or "notEmpty".
	IOError = value.NewShape("IOError", append(slices.Clip(errorFields), value.Field{Name: "path"}, value.Field{Name: "code"}))
)

// Kinds lists the built-in error kinds, which programs see as struct types.
var Kinds = []*value.Shape{TypeError, ArgumentError, ValueError, IndexError, KeyError, NameError, IOError}

func NoField(structName, key string) string {
	return fmt.Sprintf("%s has no field '%s'.", structName, key)
}

func MissingField(structName, field string) string {
	return fmt.Sprintf("Missing field '%s' of %s.", field, structName)
}
