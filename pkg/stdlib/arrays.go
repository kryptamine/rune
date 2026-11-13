package stdlib

import (
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
	"strings"
)

var arrayMethods = []*Native{
	// append copies, so the result never shares a backing array with the input.
	{"array.append", 2, -1, func(c *call) value.Value {
		return value.NewArray(slices.Concat(c.arr(0), c.args[1:]))
	}},
	{"array.join", 2, 2, func(c *call) value.Value {
		items, sep := c.arr(0), c.str(1)

		var buf []byte

		for i, item := range items {
			if i > 0 {
				buf = append(buf, sep...)
			}

			buf = helpers.AppendValue(buf, item)
		}

		return value.NewString(string(buf))
	}},
	{"array.map", 2, 2, func(c *call) value.Value {
		out := make([]value.Value, len(c.arr(0)))
		c.each(func(i int, _, r value.Value) bool {
			out[i] = r
			return true
		})

		return value.NewArray(out)
	}},
	{"array.filter", 2, 2, func(c *call) value.Value {
		var out []value.Value

		c.each(func(_ int, item, keep value.Value) bool {
			if helpers.IsTruthy(keep) {
				out = append(out, item)
			}

			return true
		})

		return value.NewArray(out)
	}},
	{"array.every", 2, 2, func(c *call) value.Value {
		return value.NewBool(firstWhere(c, false) < 0)
	}},
	{"array.some", 2, 2, func(c *call) value.Value {
		return value.NewBool(firstWhere(c, true) >= 0)
	}},
	{"array.find", 2, 2, func(c *call) value.Value {
		if i := firstWhere(c, true); i >= 0 {
			return c.arr(0)[i]
		}

		return value.NilValue
	}},
	{"array.findIndex", 2, 2, func(c *call) value.Value {
		return count(firstWhere(c, true))
	}},
	{"array.reduce", 2, 3, func(c *call) value.Value {
		items, f := c.arr(0), c.fn(1)

		var acc value.Value

		switch {
		case c.has(2):
			acc = c.args[2]
		case len(items) == 0:
			return c.fail(errors.ValueError, "of an empty array needs an initial value.")
		default:
			acc, items = items[0], items[1:]
		}

		buf := make([]value.Value, 2)

		for _, item := range items {
			buf[0], buf[1] = acc, item
			acc = c.invoke(f, buf)
		}

		return acc
	}},
}

// each calls args[1] on the elements of args[0] in order, passing the
// index too if it requires two arguments, and gives visit each element
// with the result until visit returns false or a call fails.
func (c *call) each(visit func(i int, item, r value.Value) bool) {
	items, f := c.arr(0), c.fn(1)
	if !c.ok() {
		return
	}

	buf := make([]value.Value, 1, 2)
	if c.caller.Required(f) >= 2 {
		buf = buf[:2]
	}

	for i, item := range items {
		buf[0] = item
		if len(buf) == 2 {
			buf[1] = value.NewInt(int64(i))
		}

		if r := c.invoke(f, buf); !c.ok() || !visit(i, item, r) {
			return
		}
	}
}

// firstWhere is the index of the first element for which args[1] gives
// a value as truthy as want, or -1.
func firstWhere(c *call, want bool) int {
	found := -1

	c.each(func(i int, _, r value.Value) bool {
		if helpers.IsTruthy(r) == want {
			found = i
		}

		return found < 0
	})

	return found
}

// arrayInPlace are the methods that change their array.
var arrayInPlace = []*Native{
	{"array.push", 2, -1, func(c *call) value.Value {
		if c.arr(0); !c.ok() {
			return value.NilValue
		}

		arr := c.args[0].ArrRef()
		*arr = append(*arr, c.args[1:]...)

		return value.NilValue
	}},
	{"array.pop", 1, 1, func(c *call) value.Value {
		if c.arr(0); !c.ok() {
			return value.NilValue
		}

		arr := c.args[0].ArrRef()
		n := len(*arr)

		if n == 0 {
			return c.fail(errors.ValueError, "got an empty array.")
		}

		last := (*arr)[n-1]
		(*arr)[n-1] = value.NilValue
		*arr = (*arr)[:n-1]

		return last
	}},
	{"array.insert", 3, 3, func(c *call) value.Value {
		if c.arr(0); !c.ok() {
			return value.NilValue
		}

		arr := c.args[0].ArrRef()

		i, ok := position(c, len(*arr), true)
		if !ok {
			return value.NilValue
		}

		*arr = slices.Insert(*arr, i, c.args[2])

		return value.NilValue
	}},
	{"array.remove", 2, 2, func(c *call) value.Value {
		if c.arr(0); !c.ok() {
			return value.NilValue
		}

		arr := c.args[0].ArrRef()

		i, ok := position(c, len(*arr), false)
		if !ok {
			return value.NilValue
		}

		x := (*arr)[i]
		*arr = slices.Delete(*arr, i, i+1)

		return x
	}},
	// sort sorts in place, stably.
	{"array.sort", 1, 2, func(c *call) value.Value {
		items := c.arr(0)

		compare := naturalOrder
		if c.has(1) {
			compare = comparator(c, c.fn(1))
		}

		if !c.ok() {
			return value.NilValue
		}

		if !c.has(1) && !sortable(items) {
			return c.fail(errors.TypeError, "without a comparator needs all numbers or all strings.")
		}

		slices.SortStableFunc(items, compare)

		return value.NilValue
	}},
}

// sortable reports whether items are all numbers or all strings.
func sortable(items []value.Value) bool {
	for _, item := range items {
		if item.Kind() != items[0].Kind() || !item.IsNumber() && !item.IsString() {
			return false
		}
	}

	return true
}

// naturalOrder compares two numbers or two strings.
func naturalOrder(a, b value.Value) int {
	if a.IsNumber() {
		c, _ := value.CompareNum(a, b)
		return c
	}

	return strings.Compare(a.Str(), b.Str())
}

// comparator orders by f(a, b) < 0; after a failure it treats all as equal.
func comparator(c *call, f value.Value) func(a, b value.Value) int {
	buf := make([]value.Value, 2)

	return func(a, b value.Value) int {
		buf[0], buf[1] = a, b

		r := c.invoke(f, buf)
		if !c.ok() {
			return 0
		}

		if !r.IsNumber() {
			c.fail(errors.TypeError, "comparator must return a number.")
			return 0
		}

		c, _ := value.CompareNum(r, value.NewInt(0))

		return c
	}
}

// position is args[1] as an index into n elements, negative counting from
// the end; atEnd allows n itself.
func position(c *call, n int, atEnd bool) (int, bool) {
	arg := c.int(1)
	if !c.ok() {
		return 0, false
	}

	i := arg
	if i < 0 {
		i += n
	}

	if i < 0 || i > n || i == n && !atEnd {
		c.fail(errors.IndexError, "index %d out of bounds of %s.", arg, helpers.Plural(n, "element"))
		return 0, false
	}

	return i, true
}
