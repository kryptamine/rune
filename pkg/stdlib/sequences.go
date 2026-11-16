package stdlib

import (
	"math"
	"rune/pkg/errors"
	"rune/pkg/value"
	"slices"
	"strings"
	"unicode/utf8"
)

var arraySequenceMethods = []*Native{
	{"array.repeat", 2, 2, func(c *call) value.Value {
		items, n := c.arr(0), c.int(1)

		switch {
		case !c.ok():
			return value.NilValue
		case n < 0:
			return c.fail(errors.ValueError, "count must not be negative.")
		case len(items) > 0 && n > math.MaxInt32/len(items):
			return c.fail(errors.ValueError, "result is too large.")
		}

		return sharedArray(slices.Repeat(items, n))
	}},
}

// String positions count characters.
var stringSequenceMethods = []*Native{
	{"string.indexOf", 2, 2, func(c *call) value.Value {
		s, sub := c.str(0), c.str(1)
		if i := strings.Index(s, sub); i >= 0 {
			return count(utf8.RuneCountInString(s[:i]))
		}

		return value.NilValue
	}},
}
