package stdlib

import (
	"rune/pkg/helpers"
	"rune/pkg/value"
	"unicode/utf8"
)

// preludeNatives stay global; every other built-in lives in a namespace.
var preludeNatives = []*Native{
	writer("print", true, (*Stdio).Out),
	{"type", 1, 1, func(c *call) value.Value {
		return value.NewString(helpers.TypeName(c.args[0]))
	}},
}

func length(c *call) value.Value {
	switch v := c.args[0]; v.Kind() {
	case value.Array:
		return count(len(v.Arr()))
	case value.Object:
		return count(v.Obj().Len())
	case value.String:
		return count(utf8.RuneCountInString(v.Str()))
	case value.Map, value.Set:
		return count(v.HashMap().Len())
	case value.Bytes:
		return count(len(v.Bytes()))
	}

	return c.typeError(0, "a string, an array, an object, a map, a set or bytes")
}

// lenMethod is the len method of the type called name.
func lenMethod(name string) []*Native {
	return []*Native{{name + ".len", 1, 1, length}}
}
