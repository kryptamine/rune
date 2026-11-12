package stdlib

import (
	"math"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"strings"
	"unicode/utf8"
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

var stringMethods = []*Native{
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
	{"string.charCode", 1, 2, func(c *call) value.Value {
		s, i := c.str(0), 0
		if c.has(1) {
			i = c.int(1)
		}

		if !c.ok() {
			return value.NilValue
		}

		if n := utf8.RuneCountInString(s); i < 0 || i >= n {
			return c.fail(errors.IndexError, "index %d out of bounds of %d characters.", i, n)
		}

		r, _ := utf8.DecodeRuneInString(s[helpers.ByteOffset(s, i):])

		return count(int(r))
	}},
	pad("string.padStart", true),
	pad("string.padEnd", false),
	{"string.format", 1, -1, func(c *call) value.Value {
		f := c.str(0)
		if !c.ok() {
			return value.NilValue
		}

		return format(c, f, c.args[1:])
	}},
}

var stringNatives = []*Native{
	{"string.fromCharCode", 1, -1, func(c *call) value.Value {
		b := make([]byte, 0, len(c.args))

		for i := range c.args {
			code := c.int(i)
			if !c.ok() {
				return value.NilValue
			}

			if code < 0 || code > utf8.MaxRune || (code >= 0xD800 && code <= 0xDFFF) {
				return c.fail(errors.ValueError, "%d is not a valid character code.", code)
			}

			b = utf8.AppendRune(b, rune(code))
		}

		return value.NewString(string(b))
	}},
}

// pad fills s with pad, repeated and cut, to width characters.
func pad(name string, start bool) *Native {
	return &Native{name, 2, 3, func(c *call) value.Value {
		s, width, fill := c.str(0), c.int(1), " "
		if c.has(2) {
			fill = c.str(2)
		}

		switch {
		case !c.ok():
			return value.NilValue
		case fill == "":
			return c.fail(errors.ValueError, "pad must not be empty.")
		case width > math.MaxInt32/utf8.UTFMax:
			return c.fail(errors.ValueError, "result is too large.")
		}

		return value.NewString(padded(s, width, fill, start))
	}}
}

func padded(s string, width int, fill string, start bool) string {
	n := width - utf8.RuneCountInString(s)
	if n <= 0 {
		return s
	}

	var b strings.Builder

	if !start {
		b.WriteString(s)
	}

	for n > 0 {
		for _, r := range fill {
			if n == 0 {
				break
			}

			b.WriteRune(r)
			n--
		}
	}

	if start {
		b.WriteString(s)
	}

	return b.String()
}
