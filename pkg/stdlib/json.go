package stdlib

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"math"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"strconv"
	"strings"
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

		v, err := parseJSON(s)
		if err != nil {
			return c.fail(errors.ValueError, "could not parse JSON: %s", err)
		}

		return v
	}},
	{"json.stringify", 1, 1, func(c *call) value.Value {
		buf := appendJSON(c, nil, c.args[0], 0)
		if !c.ok() {
			return value.NilValue
		}

		return value.NewString(string(buf))
	}},
}

// parseJSON reads numbers without a fraction or an exponent as integers,
// when they fit, and keeps the order of object keys.
func parseJSON(s string) (value.Value, error) {
	d := json.NewDecoder(strings.NewReader(s))
	d.UseNumber()

	v, err := readJSON(d)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return value.NilValue, fmt.Errorf("unexpected end of JSON input")
	} else if err != nil {
		return value.NilValue, err
	}

	if _, err := d.Token(); err != io.EOF {
		return value.NilValue, fmt.Errorf("invalid data after the top-level value")
	}

	return v, nil
}

func readJSON(d *json.Decoder) (value.Value, error) {
	t, err := d.Token()
	if err != nil {
		return value.NilValue, err
	}

	switch t := t.(type) {
	case json.Delim:
		if t == '[' {
			items := []value.Value{}
			for d.More() {
				item, err := readJSON(d)
				if err != nil {
					return value.NilValue, err
				}

				items = append(items, item)
			}

			_, err := d.Token()

			return value.NewArray(items), err
		}

		obj := value.NewObj(0)
		for d.More() {
			k, err := d.Token()
			if err != nil {
				return value.NilValue, err
			}

			item, err := readJSON(d)
			if err != nil {
				return value.NilValue, err
			}

			obj.Set(k.(string), item)
		}

		_, err := d.Token()

		return value.NewObject(obj), err
	case json.Number:
		if n, err := strconv.ParseInt(string(t), 10, 64); err == nil {
			return value.NewInt(n), nil
		}

		f, err := strconv.ParseFloat(string(t), 64)
		if err != nil {
			return value.NilValue, fmt.Errorf("number %s is out of range", t)
		}

		return value.NewNumber(f), nil
	case nil:
		return value.NilValue, nil
	}

	return value.FromGo(t), nil
}

// appendJSON encodes objects in key order and structs in field order.
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
		if v.IsInt() {
			return strconv.AppendInt(buf, v.Int(), 10)
		}

		return appendJSONNumber(c, buf, v.Float())
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
		return appendJSONPairs(c, buf, v.Obj().All(), depth)
	case value.Struct:
		return appendJSONPairs(c, buf, v.FieldPairs(), depth)
	case value.Enum:
		return appendJSON(c, buf, value.NewString(v.Variant().Name), depth)
	case value.Set:
		return appendJSON(c, buf, keysOf(v.HashMap()), depth)
	case value.Map:
		obj := value.NewObj(v.HashMap().Len())

		ok := true
		v.HashMap().Each(func(k, x value.Value) bool {
			if ok = k.IsString(); ok {
				obj.Set(k.Str(), x)
			}

			return ok
		})

		if !ok {
			c.fail(errors.TypeError, "cannot encode a map with keys other than strings.")
			return buf
		}

		return appendJSONPairs(c, buf, obj.All(), depth)
	}

	c.fail(errors.TypeError, "cannot encode %s.", helpers.TypeName(v))

	return buf
}

func appendJSONPairs(c *call, buf []byte, pairs iter.Seq2[string, value.Value], depth int) []byte {
	buf = append(buf, '{')
	first := true

	for k, v := range pairs {
		if !first {
			buf = append(buf, ',')
		}

		first = false

		buf = append(appendJSONString(buf, k), ':')
		if buf = appendJSON(c, buf, v, depth+1); !c.ok() {
			return buf
		}
	}

	return append(buf, '}')
}

func appendJSONNumber(c *call, buf []byte, f float64) []byte {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		c.fail(errors.ValueError, "cannot encode a non-finite number.")
		return buf
	}

	return helpers.AppendFloat(buf, f)
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
