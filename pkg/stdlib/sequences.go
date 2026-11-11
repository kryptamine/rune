package stdlib

import (
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
	"strings"
	"unicode/utf8"
)

// Sequence natives take an array or a string; string positions count characters.
var sequenceNatives = []*Native{
	{"slice", 2, 3, func(c *call) value.Value {
		switch v := c.args[0]; v.Kind() {
		case value.Array:
			start, end := sliceBounds(c, len(v.Arr()))
			return value.NewArray(slices.Clone(v.Arr()[start:end]))
		case value.String:
			s := v.Str()
			start, end := sliceBounds(c, utf8.RuneCountInString(s))

			return value.NewString(s[helpers.ByteOffset(s, start):helpers.ByteOffset(s, end)])
		}

		return c.typeError(0, "an array or a string")
	}},
	{"reverse", 1, 1, func(c *call) value.Value {
		switch v := c.args[0]; v.Kind() {
		case value.Array:
			out := slices.Clone(v.Arr())
			slices.Reverse(out)

			return value.NewArray(out)
		case value.String:
			r := []rune(v.Str())
			slices.Reverse(r)

			return value.NewString(string(r))
		}

		return c.typeError(0, "an array or a string")
	}},
	{"contains", 2, 2, func(c *call) value.Value {
		return value.NewBool(indexOf(c) >= 0)
	}},
	{"indexOf", 2, 2, func(c *call) value.Value {
		return count(indexOf(c))
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

// indexOf finds args[1] in args[0]; -1 if absent.
func indexOf(c *call) int {
	switch v := c.args[0]; v.Kind() {
	case value.Array:
		return slices.IndexFunc(v.Arr(), func(x value.Value) bool { return helpers.IsEqual(x, c.args[1]) })
	case value.String:
		s := v.Str()

		if i := strings.Index(s, c.str(1)); i >= 0 {
			return utf8.RuneCountInString(s[:i])
		}

		return -1
	}

	c.typeError(0, "an array or a string")

	return -1
}
