package stdlib

import (
	"rune/pkg/value"
	"unicode/utf8"
)

// preludeNatives stay global; every other built-in lives in a namespace.
var preludeNatives = []*Native{
	writer("print", true, (*Stdio).Out),
}

func length(c *call) value.Value {
	if v, ok := lengthFast(c.args[:1]); ok {
		return v
	}

	return c.typeError(0, "a string, an array, an object, a map, a set, bytes or a buffer")
}

func lengthFast(args []value.Value) (value.Value, bool) {
	if len(args) != 1 {
		return value.NilValue, false
	}

	switch v := args[0]; v.Kind() {
	case value.Array:
		return count(len(v.Arr())), true
	case value.Object:
		return count(v.Obj().Len()), true
	case value.String:
		return count(utf8.RuneCountInString(v.Str())), true
	case value.Map, value.Set:
		return count(v.HashMap().Len()), true
	case value.Bytes, value.Buffer:
		b, _ := v.Binary()
		return count(len(b)), true
	}

	return value.NilValue, false
}

// lenMethod is the len method of the type called name.
func lenMethod(name string) []*Native {
	return []*Native{{name + ".len", 1, 1, length}}
}
