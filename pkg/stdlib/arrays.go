package stdlib

import (
	"cmp"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
	"strings"
)

var inPlace = map[string]bool{"array.push": true, "array.pop": true, "array.sort": true}

// InPlace reports whether v is a built-in that mutates its first argument.
func InPlace(v value.Value) bool {
	if v.Kind() != value.Function {
		return false
	}

	n, ok := v.Fn().(*Native)

	return ok && inPlace[n.name]
}

var arrayNatives = []*Native{
	// append copies, so the result never shares a backing array with the input.
	{"array.append", 2, -1, func(c *call) value.Value {
		return value.NewArray(slices.Concat(c.arr(0), c.args[1:]))
	}},
	{"array.push", 2, -1, func(c *call) value.Value {
		if c.arr(0); !c.ok() {
			return value.NilValue
		}

		arr := c.args[0].ArrRef()
		*arr = append(*arr, c.args[1:]...)

		return count(len(*arr))
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
		items, f := c.arr(0), c.fn(1)
		out := make([]value.Value, len(items))
		buf := make([]value.Value, 1)

		for i, item := range items {
			buf[0] = item
			if out[i] = c.invoke(f, buf); !c.ok() {
				return value.NilValue
			}
		}

		return value.NewArray(out)
	}},
	{"array.filter", 2, 2, func(c *call) value.Value {
		items, f := c.arr(0), c.fn(1)
		buf := make([]value.Value, 1)

		var out []value.Value

		for _, item := range items {
			buf[0] = item
			if keep := c.invoke(f, buf); helpers.IsTruthy(keep) && c.ok() {
				out = append(out, item)
			}
		}

		return value.NewArray(out)
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
	// sort sorts in place, stably, and returns the array.
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

		return c.args[0]
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
		return cmp.Compare(a.Num(), b.Num())
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

		return cmp.Compare(r.Num(), 0)
	}
}
