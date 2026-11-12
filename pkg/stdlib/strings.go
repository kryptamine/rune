package stdlib

import (
	"math"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"strings"
)

func unaryString(name string, f func(string) string) *Native {
	return &Native{name, 1, 1, func(c *call) value.Value {
		return value.NewString(f(c.str(0)))
	}}
}

func stringPredicate(name string, f func(s, x string) bool) *Native {
	return &Native{name, 2, 2, func(c *call) value.Value {
		return value.NewBool(f(c.str(0), c.str(1)))
	}}
}

var stringNatives = []*Native{
	{"string.fixed", 2, 2, func(c *call) value.Value {
		n, digits := c.num(0), c.int(1)
		if !c.ok() {
			return value.NilValue
		}

		if digits < 0 || digits > 100 {
			return c.fail(errors.ValueError, "expects 0 to 100 digits, got %d.", digits)
		}

		return value.NewString(string(helpers.AppendFixed(nil, n, digits)))
	}},
	unaryString("string.upper", strings.ToUpper),
	unaryString("string.lower", strings.ToLower),
	unaryString("string.trim", strings.TrimSpace),
	stringPredicate("string.startsWith", strings.HasPrefix),
	stringPredicate("string.endsWith", strings.HasSuffix),
	{"string.split", 2, 2, func(c *call) value.Value {
		parts := strings.Split(c.str(0), c.str(1))
		items := make([]value.Value, len(parts))

		for i, part := range parts {
			items[i] = value.NewString(part)
		}

		return value.NewArray(items)
	}},
	{"string.replace", 3, 3, func(c *call) value.Value {
		return value.NewString(strings.ReplaceAll(c.str(0), c.str(1), c.str(2)))
	}},
	{"string.repeat", 2, 2, func(c *call) value.Value {
		s, n := c.str(0), c.int(1)

		switch {
		case !c.ok():
			return value.NilValue
		case n < 0:
			return c.fail(errors.ValueError, "count must not be negative.")
		case len(s) > 0 && n > math.MaxInt32/len(s):
			return c.fail(errors.ValueError, "result is too large.")
		}

		return value.NewString(strings.Repeat(s, n))
	}},
}
