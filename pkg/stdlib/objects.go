package stdlib

import (
	"maps"
	"rune/pkg/value"
	"slices"
)

func (c *call) obj(i int) map[string]value.Value {
	if c.args[i].Kind() != value.Object {
		c.typeError(i, "an object")
		return nil
	}

	return c.args[i].Obj()
}

func objectMethod(name string, min, max int, fn func(c *call, o map[string]value.Value) value.Value) *Native {
	return method(name, min, max, (*call).obj, fn)
}

// objectEntries is what pick makes of each entry of o, in the order for-in
// walks them.
func objectEntries(o map[string]value.Value, pick func(k string, v value.Value) value.Value) value.Value {
	out := make([]value.Value, 0, len(o))
	for _, k := range slices.Sorted(maps.Keys(o)) {
		out = append(out, pick(k, o[k]))
	}

	return value.NewArray(out)
}

var objectMethods = []*Native{
	objectMethod("object.has", 2, 2, func(c *call, o map[string]value.Value) value.Value {
		_, found := o[c.str(1)]
		return value.NewBool(found)
	}),
	objectMethod("object.get", 2, 3, func(c *call, o map[string]value.Value) value.Value {
		if v, found := o[c.str(1)]; found {
			return v
		}

		if c.has(2) {
			return c.args[2]
		}

		return value.NilValue
	}),
	objectMethod("object.keys", 1, 1, func(_ *call, o map[string]value.Value) value.Value {
		return objectEntries(o, func(k string, _ value.Value) value.Value { return value.NewString(k) })
	}),
	objectMethod("object.values", 1, 1, func(_ *call, o map[string]value.Value) value.Value {
		return objectEntries(o, func(_ string, v value.Value) value.Value { return v })
	}),
	objectMethod("object.entries", 1, 1, func(_ *call, o map[string]value.Value) value.Value {
		return objectEntries(o, func(k string, v value.Value) value.Value {
			return value.NewArray([]value.Value{value.NewString(k), v})
		})
	}),
}

var objectInPlace = []*Native{
	objectMethod("object.delete", 2, 2, func(c *call, o map[string]value.Value) value.Value {
		k := c.str(1)
		_, found := o[k]
		delete(o, k)

		return value.NewBool(found)
	}),
}
