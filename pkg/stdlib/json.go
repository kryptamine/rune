package stdlib

import (
	"encoding/json"
	"maps"
	"math"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
	"strconv"
	"unicode/utf8"
)

// maxJSONDepth bounds stringify's nesting, which also stops it on a cycle.
const maxJSONDepth = 1000

var jsonNatives = []*Native{
	{"json.parse", 1, 1, func(c *call) value.Value {
		s := c.str(0)
		if !c.ok() {
			return value.NilValue
		}

		var v any
		if err := json.Unmarshal([]byte(s), &v); err != nil {
			return c.fail(errors.ValueError, "could not parse JSON: %s", err)
		}

		return value.FromGo(v)
	}},
	{"json.stringify", 1, 1, func(c *call) value.Value {
		buf := appendJSON(c, nil, c.args[0], 0)
		if !c.ok() {
			return value.NilValue
		}

		return value.NewString(string(buf))
	}},
}

// appendJSON encodes objects with sorted keys and structs in field order.
func appendJSON(c *call, buf []byte, v value.Value, depth int) []byte {
	if depth > maxJSONDepth {
		c.fail(errors.ValueError, "cannot encode a cyclic or too deeply nested value.")
		return buf
	}

	switch v.Kind() {
	case value.Nil:
		return append(buf, "null"...)
	case value.Bool:
		return strconv.AppendBool(buf, v.Bool())
	case value.Number:
		return appendJSONNumber(c, buf, v.Num())
	case value.String:
		return appendJSONString(buf, v.Str())
	case value.Array:
		buf = append(buf, '[')
		for i, item := range v.Arr() {
			if i > 0 {
				buf = append(buf, ',')
			}

			if buf = appendJSON(c, buf, item, depth+1); !c.ok() {
				return buf
			}
		}

		return append(buf, ']')
	case value.Object:
		obj := v.Obj()
		keys := slices.Sorted(maps.Keys(obj))

		return appendJSONPairs(c, buf, keys, func(i int) value.Value { return obj[keys[i]] }, depth)
	case value.Struct:
		return appendJSONPairs(c, buf, v.Shape().Fields, func(i int) value.Value { return v.Fields()[i] }, depth)
	}

	c.fail(errors.TypeError, "cannot encode %s.", helpers.TypeName(v))

	return buf
}

func appendJSONPairs(c *call, buf []byte, keys []string, val func(i int) value.Value, depth int) []byte {
	buf = append(buf, '{')

	for i, k := range keys {
		if i > 0 {
			buf = append(buf, ',')
		}

		buf = append(appendJSONString(buf, k), ':')
		if buf = appendJSON(c, buf, val(i), depth+1); !c.ok() {
			return buf
		}
	}

	return append(buf, '}')
}

// appendJSONNumber formats like JavaScript: exponents only outside [1e-6, 1e21).
func appendJSONNumber(c *call, buf []byte, f float64) []byte {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		c.fail(errors.ValueError, "cannot encode a non-finite number.")
		return buf
	}

	format := byte('f')
	if a := math.Abs(f); a != 0 && (a < 1e-6 || a >= 1e21) {
		format = 'e'
	}

	buf = strconv.AppendFloat(buf, f, format, -1, 64)

	if format == 'e' {
		// e-07 becomes e-7.
		if n := len(buf); n >= 4 && buf[n-4] == 'e' && buf[n-3] == '-' && buf[n-2] == '0' {
			buf[n-2] = buf[n-1]
			buf = buf[:n-1]
		}
	}

	return buf
}

const hexDigits = "0123456789abcdef"

func appendJSONString(buf []byte, s string) []byte {
	buf = append(buf, '"')

	for i := 0; i < len(s); {
		b := s[i]
		if b < utf8.RuneSelf {
			switch {
			case b == '"' || b == '\\':
				buf = append(buf, '\\', b)
			case b == '\n':
				buf = append(buf, '\\', 'n')
			case b == '\r':
				buf = append(buf, '\\', 'r')
			case b == '\t':
				buf = append(buf, '\\', 't')
			case b < 0x20:
				buf = append(buf, '\\', 'u', '0', '0', hexDigits[b>>4], hexDigits[b&0xf])
			default:
				buf = append(buf, b)
			}

			i++

			continue
		}

		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			buf = append(buf, `�`...)
		} else {
			buf = append(buf, s[i:i+size]...)
		}

		i += size
	}

	return append(buf, '"')
}
