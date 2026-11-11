package stdlib

import (
	"math"
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
	unaryString("upper", strings.ToUpper),
	unaryString("lower", strings.ToLower),
	unaryString("trim", strings.TrimSpace),
	stringPredicate("startsWith", strings.HasPrefix),
	stringPredicate("endsWith", strings.HasSuffix),
	{"split", 2, 2, func(c *call) value.Value {
		parts := strings.Split(c.str(0), c.str(1))
		items := make([]value.Value, len(parts))

		for i, part := range parts {
			items[i] = value.NewString(part)
		}

		return value.NewArray(items)
	}},
	{"join", 2, 2, func(c *call) value.Value {
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
	{"replace", 3, 3, func(c *call) value.Value {
		return value.NewString(strings.ReplaceAll(c.str(0), c.str(1), c.str(2)))
	}},
	{"repeat", 2, 2, func(c *call) value.Value {
		s, n := c.str(0), c.int(1)

		switch {
		case !c.ok():
			return value.NilValue
		case n < 0:
			return c.fail("count must not be negative.")
		case len(s) > 0 && n > math.MaxInt32/len(s):
			return c.fail("result is too large.")
		}

		return value.NewString(strings.Repeat(s, n))
	}},
}
