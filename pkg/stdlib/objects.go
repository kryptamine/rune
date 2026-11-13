package stdlib

import "rune/pkg/value"

func (c *call) obj(i int) *value.Obj {
	if c.args[i].Kind() != value.Object {
		c.typeError(i, "an object")
		return nil
	}

	return c.args[i].Obj()
}

func objectMethod(name string, min, max int, fn func(c *call, o *value.Obj) value.Value) *Native {
	return method(name, min, max, (*call).obj, fn)
}

// objectEntries is what pick makes of each entry of o, in the order for-in
// walks them.
func objectEntries(o *value.Obj, pick func(k string, v value.Value) value.Value) value.Value {
	out := make([]value.Value, 0, o.Len())
	for k, v := range o.All() {
		out = append(out, pick(k, v))
	}

	return value.NewArray(out)
}

var objectMethods = []*Native{
	objectMethod("object.has", 2, 2, func(c *call, o *value.Obj) value.Value {
		return value.NewBool(o.Has(c.str(1)))
	}),
	objectMethod("object.get", 2, 3, func(c *call, o *value.Obj) value.Value {
		if v, found := o.Get(c.str(1)); found {
			return v
		}

		if c.has(2) {
			return c.args[2]
		}

		return value.NilValue
	}),
	objectMethod("object.keys", 1, 1, func(_ *call, o *value.Obj) value.Value {
		return objectEntries(o, func(k string, _ value.Value) value.Value { return value.NewString(k) })
	}),
	objectMethod("object.values", 1, 1, func(_ *call, o *value.Obj) value.Value {
		return objectEntries(o, func(_ string, v value.Value) value.Value { return v })
	}),
	objectMethod("object.entries", 1, 1, func(_ *call, o *value.Obj) value.Value {
		return objectEntries(o, func(k string, v value.Value) value.Value {
			return value.NewArray([]value.Value{value.NewString(k), v})
		})
	}),
}

var objectInPlace = []*Native{
	objectMethod("object.delete", 2, 2, func(c *call, o *value.Obj) value.Value {
		return value.NewBool(o.Delete(c.str(1)))
	}),
}
