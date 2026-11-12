package stdlib

import (
	"maps"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
)

// method is a built-in method, whose receiver, argument 0, recv checks.
func method[T any](name string, min, max int, recv func(c *call, i int) T, fn func(c *call, r T) value.Value) *Native {
	return &Native{name, min, max, func(c *call) value.Value {
		r := recv(c, 0)
		if !c.ok() {
			return value.NilValue
		}

		return fn(c, r)
	}}
}

func (c *call) table(i int, kind value.Kind, want string) *value.HashMap {
	if c.args[i].Kind() != kind {
		c.typeError(i, want)
		return nil
	}

	return c.args[i].HashMap()
}

func (c *call) mapArg(i int) *value.HashMap { return c.table(i, value.Map, "a map") }
func (c *call) setArg(i int) *value.HashMap { return c.table(i, value.Set, "a set") }

// keyErr turns a failure of a table into the call's error.
func (c *call) keyErr(err error) bool {
	if u, ok := err.(*value.Unhashable); ok {
		c.fail(errors.TypeError, "cannot use %s as a key.", u.What)
	}

	return err == nil
}

func (c *call) pair(v value.Value) (k, x value.Value, ok bool) {
	if v.Kind() == value.Array && len(v.Arr()) == 2 {
		return v.Arr()[0], v.Arr()[1], true
	}

	c.fail(errors.TypeError, "expects [key, value] pairs, got %s.", helpers.Quoted(v))

	return k, x, false
}

var collectionNatives = []*Native{
	{"map", 0, 1, func(c *call) value.Value {
		if !c.has(0) {
			return value.NewMap(value.NewHashMap(0))
		}

		switch src := c.args[0]; src.Kind() {
		case value.Map:
			return value.NewMap(src.HashMap().Clone())
		case value.Object:
			m := value.NewHashMap(len(src.Obj()))
			for _, k := range slices.Sorted(maps.Keys(src.Obj())) {
				m.Set(value.NewString(k), src.Obj()[k])
			}

			return value.NewMap(m)
		case value.Array:
			m := value.NewHashMap(len(src.Arr()))
			for _, p := range src.Arr() {
				k, v, ok := c.pair(p)
				if !ok || !c.keyErr(m.Set(k, v)) {
					return value.NilValue
				}
			}

			return value.NewMap(m)
		}

		return c.typeError(0, "a map, an object or an array of pairs")
	}},
	{"set", 0, 1, func(c *call) value.Value {
		if !c.has(0) {
			return value.NewSet(value.NewHashMap(0))
		}

		switch src := c.args[0]; src.Kind() {
		case value.Set:
			return value.NewSet(src.HashMap().Clone())
		case value.Array:
			m := value.NewHashMap(len(src.Arr()))
			for _, x := range src.Arr() {
				if !c.keyErr(m.Set(x, value.NilValue)) {
					return value.NilValue
				}
			}

			return value.NewSet(m)
		}

		return c.typeError(0, "a set or an array")
	}},
}

// collect gathers what pick makes of each entry of m into an array.
func collect(m *value.HashMap, pick func(k, v value.Value) value.Value) value.Value {
	out := make([]value.Value, 0, m.Len())
	m.Each(func(k, v value.Value) bool {
		out = append(out, pick(k, v))
		return true
	})

	return value.NewArray(out)
}

func keysOf(m *value.HashMap) value.Value {
	return collect(m, func(k, _ value.Value) value.Value { return k })
}

// The methods both tables have.
func hasKey(c *call, m *value.HashMap) value.Value {
	_, found, err := m.Get(c.args[1])
	c.keyErr(err)

	return value.NewBool(found)
}

func deleteKey(c *call, m *value.HashMap) value.Value {
	found, err := m.Delete(c.args[1])
	c.keyErr(err)

	return value.NewBool(found)
}

func clearTable(_ *call, m *value.HashMap) value.Value {
	m.Clear()
	return value.NilValue
}

func mapMethod(name string, min, max int, fn func(c *call, m *value.HashMap) value.Value) *Native {
	return method(name, min, max, (*call).mapArg, fn)
}

func setMethod(name string, min, max int, fn func(c *call, m *value.HashMap) value.Value) *Native {
	return method(name, min, max, (*call).setArg, fn)
}

var mapMethods = []*Native{
	mapMethod("map.get", 2, 3, func(c *call, m *value.HashMap) value.Value {
		v, found, err := m.Get(c.args[1])
		if !c.keyErr(err) {
			return value.NilValue
		}

		if !found && c.has(2) {
			return c.args[2]
		}

		return v
	}),
	mapMethod("map.has", 2, 2, hasKey),
	mapMethod("map.keys", 1, 1, func(_ *call, m *value.HashMap) value.Value { return keysOf(m) }),
	mapMethod("map.values", 1, 1, func(_ *call, m *value.HashMap) value.Value {
		return collect(m, func(_, v value.Value) value.Value { return v })
	}),
	mapMethod("map.entries", 1, 1, func(_ *call, m *value.HashMap) value.Value {
		return collect(m, func(k, v value.Value) value.Value { return value.NewArray([]value.Value{k, v}) })
	}),
}

var mapInPlace = []*Native{
	mapMethod("map.set", 3, 3, func(c *call, m *value.HashMap) value.Value {
		c.keyErr(m.Set(c.args[1], c.args[2]))
		return value.NilValue
	}),
	mapMethod("map.delete", 2, 2, deleteKey),
	mapMethod("map.clear", 1, 1, clearTable),
}

// setOp builds a new set of the elements of the receiver that keep says
// to, given whether the other set has them, plus with union the other's.
func setOp(name string, keep func(inOther bool) bool, union bool) *Native {
	return setMethod(name, 2, 2, func(c *call, a *value.HashMap) value.Value {
		b := c.setArg(1)
		if b == nil {
			return value.NilValue
		}

		out := value.NewHashMap(a.Len())
		a.Each(func(k, _ value.Value) bool {
			if _, in, _ := b.Get(k); keep(in) {
				out.Set(k, value.NilValue)
			}

			return true
		})

		if union {
			b.Each(func(k, _ value.Value) bool {
				out.Set(k, value.NilValue)
				return true
			})
		}

		return value.NewSet(out)
	})
}

var setInPlace = []*Native{
	setMethod("set.add", 2, 2, func(c *call, m *value.HashMap) value.Value {
		c.keyErr(m.Set(c.args[1], value.NilValue))
		return value.NilValue
	}),
	setMethod("set.delete", 2, 2, deleteKey),
	setMethod("set.clear", 1, 1, clearTable),
}

var setMethods = []*Native{
	setMethod("set.contains", 2, 2, hasKey),
	setMethod("set.values", 1, 1, func(_ *call, m *value.HashMap) value.Value { return keysOf(m) }),
	setOp("set.union", func(bool) bool { return true }, true),
	setOp("set.intersection", func(in bool) bool { return in }, false),
	setOp("set.difference", func(in bool) bool { return !in }, false),
}
