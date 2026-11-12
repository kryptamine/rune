package stdlib

import (
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
	"strings"
	"unicode/utf8"
)

var arraySequenceNatives = []*Native{
	{"array.slice", 2, 3, func(c *call) value.Value {
		items := c.arr(0)
		start, end := sliceBounds(c, len(items))

		return value.NewArray(slices.Clone(items[start:end]))
	}},
	{"array.reverse", 1, 1, func(c *call) value.Value {
		out := slices.Clone(c.arr(0))
		slices.Reverse(out)

		return value.NewArray(out)
	}},
	{"array.contains", 2, 2, func(c *call) value.Value {
		return value.NewBool(arrayIndex(c) >= 0)
	}},
	{"array.indexOf", 2, 2, func(c *call) value.Value {
		return count(arrayIndex(c))
	}},
}

// String positions count characters.
var stringSequenceNatives = []*Native{
	{"string.slice", 2, 3, func(c *call) value.Value {
		s := c.str(0)
		start, end := sliceBounds(c, utf8.RuneCountInString(s))

		return value.NewString(s[helpers.ByteOffset(s, start):helpers.ByteOffset(s, end)])
	}},
	{"string.reverse", 1, 1, func(c *call) value.Value {
		r := []rune(c.str(0))
		slices.Reverse(r)

		return value.NewString(string(r))
	}},
	{"string.contains", 2, 2, func(c *call) value.Value {
		return value.NewBool(strings.Contains(c.str(0), c.str(1)))
	}},
	{"string.indexOf", 2, 2, func(c *call) value.Value {
		s, sub := c.str(0), c.str(1)
		if i := strings.Index(s, sub); i >= 0 {
			return count(utf8.RuneCountInString(s[:i]))
		}

		return count(-1)
	}},
}

// sliceBounds resolves slice(x, start[, end]) for a sequence of length n;
// negative bounds count from the end, and the result never runs backwards.
func sliceBounds(c *call, n int) (start, end int) {
	start, end = clampIndex(c.int(1), n), n

	if c.has(2) {
		end = clampIndex(c.int(2), n)
	}

	return start, max(start, end)
}

func clampIndex(i, n int) int {
	if i < 0 {
		i += n
	}

	return max(0, min(i, n))
}

// arrayIndex finds args[1] in the array args[0]; -1 if absent.
func arrayIndex(c *call) int {
	return slices.IndexFunc(c.arr(0), func(x value.Value) bool { return helpers.IsEqual(x, c.args[1]) })
}
