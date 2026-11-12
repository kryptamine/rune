package stdlib

import (
	"rune/pkg/helpers"
	"rune/pkg/value"
	"time"
	"unicode/utf8"
)

// preludeNatives stay global; every other built-in lives in a namespace.
var preludeNatives = []*Native{
	{"len", 1, 1, func(c *call) value.Value {
		switch v := c.args[0]; v.Kind() {
		case value.Array:
			return count(len(v.Arr()))
		case value.Object:
			return count(len(v.Obj()))
		case value.String:
			return count(utf8.RuneCountInString(v.Str()))
		case value.Struct:
			return count(len(v.Shape().Fields))
		case value.Map, value.Set:
			return count(v.HashMap().Len())
		case value.Bytes:
			return count(len(v.Bytes()))
		}

		return c.typeError(0, "a string, an array, an object, a struct, a Map, a Set or bytes")
	}},
	{"type", 1, 1, func(c *call) value.Value {
		return value.NewString(helpers.TypeName(c.args[0]))
	}},
	{"object", 1, 1, func(c *call) value.Value {
		v := c.args[0]
		if v.Kind() != value.Struct {
			return c.typeError(0, "a struct")
		}

		names := v.Shape().Fields
		obj := make(map[string]value.Value, len(names))

		for i, f := range v.Fields() {
			obj[names[i]] = f
		}

		return value.NewObject(obj)
	}},
}

var timeNatives = []*Native{
	{"time.now", 0, 0, func(*call) value.Value {
		return value.NewNumber(float64(time.Now().UnixNano()) / 1e9)
	}},
}
