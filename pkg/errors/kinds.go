package errors

import "rune/pkg/value"

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
	IOError       = value.NewShape("IOError", errorFields)
)

// Kinds lists the built-in error kinds, which programs see as struct types.
var Kinds = []*value.Shape{TypeError, ArgumentError, ValueError, IndexError, KeyError, NameError, IOError}
