package stdlib

import (
	"cmp"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
	"strings"
)

var arrayNatives = []*Native{
	{"map", 2, 2, func(c *call) value.Value {
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
	{"filter", 2, 2, func(c *call) value.Value {
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
	{"reduce", 2, 3, func(c *call) value.Value {
		items, f := c.arr(0), c.fn(1)

		var acc value.Value

		switch {
		case c.has(2):
			acc = c.args[2]
		case len(items) == 0:
			return c.fail("of an empty array needs an initial value.")
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
	{"sort", 1, 2, func(c *call) value.Value {
		items := c.arr(0)

		compare := naturalOrder
		if c.has(1) {
			compare = comparator(c, c.fn(1))
		}

		if !c.ok() {
			return value.NilValue
		}

		if !c.has(1) && !sortable(items) {
			return c.fail("without a comparator needs all numbers or all strings.")
		}

		slices.SortStableFunc(items, compare)

		return c.args[0]
	}},
	{"range", 1, 3, func(c *call) value.Value {
		start, end, step := 0.0, c.num(0), 1.0

		if c.has(1) {
			start, end = end, c.num(1)
		}

		if c.has(2) {
			step = c.num(2)
		}

		if c.ok() && step == 0 {
			return c.fail("step must not be zero.")
		}

		var out []value.Value

		for x := start; c.ok() && (step > 0 && x < end || step < 0 && x > end); x += step {
			out = append(out, value.NewNumber(x))
		}

		return value.NewArray(out)
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
			c.fail("comparator must return a number.")
			return 0
		}

		return cmp.Compare(r.Num(), 0)
	}
}
