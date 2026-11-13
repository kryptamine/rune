package stdlib

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"unicode/utf8"
)

func (c *call) bytes(i int) []byte {
	if c.args[i].Kind() != value.Bytes {
		c.typeError(i, "bytes")
		return nil
	}

	return c.args[i].Bytes()
}

var bytesMethods = []*Native{
	{"bytes.text", 1, 1, func(c *call) value.Value {
		b := c.bytes(0)
		if !c.ok() {
			return value.NilValue
		}

		if !utf8.Valid(b) {
			return c.fail(errors.ValueError, "got bytes that are not valid UTF-8.")
		}

		return value.NewString(string(b))
	}},
	{"bytes.toArray", 1, 1, func(c *call) value.Value {
		b := c.bytes(0)
		out := make([]value.Value, len(b))

		for i, x := range b {
			out[i] = value.NewInt(int64(x))
		}

		return value.NewArray(out)
	}},
	// slice shares the bytes, which never change.
	{"bytes.slice", 2, 3, func(c *call) value.Value {
		b := c.bytes(0)
		start, end := sliceBounds(c, len(b))

		return value.NewBytes(b[start:end:end])
	}},
	{"bytes.indexOf", 2, 2, func(c *call) value.Value {
		return count(bytes.Index(c.bytes(0), c.bytes(1)))
	}},
	{"bytes.concat", 1, -1, func(c *call) value.Value {
		n := 0
		for i := range c.args {
			n += len(c.bytes(i))
		}

		out := make([]byte, 0, n)
		for i := range c.args {
			out = append(out, c.bytes(i)...)
		}

		return value.NewBytes(out)
	}},
	{"bytes.hex", 1, 1, func(c *call) value.Value {
		return value.NewString(hex.EncodeToString(c.bytes(0)))
	}},
	{"bytes.base64", 1, 1, func(c *call) value.Value {
		return value.NewString(base64.StdEncoding.EncodeToString(c.bytes(0)))
	}},
}

// toBytes is bytes(v): a string's UTF-8, an array of integers from 0 to
// 255, or bytes as they are.
var toBytes = &Native{"bytes", 1, 1, func(c *call) value.Value {
	switch v := c.args[0]; v.Kind() {
	case value.Bytes:
		return v
	case value.String:
		return value.NewBytes([]byte(v.Str()))
	case value.Array:
		out := make([]byte, len(v.Arr()))

		for i, x := range v.Arr() {
			n, ok := value.ToInt(x)
			if !ok || n < 0 || n > 255 {
				return c.fail(errors.ValueError, "expects bytes from 0 to 255, got %s.", helpers.Quoted(x))
			}

			out[i] = byte(n)
		}

		return value.NewBytes(out)
	}

	return c.typeError(0, "a string, an array or bytes")
}}

var bytesNatives = []*Native{
	{"bytes.alloc", 1, 2, func(c *call) value.Value {
		n, fill := c.int(0), 0
		if c.has(1) {
			fill = c.int(1)
		}

		switch {
		case !c.ok():
			return value.NilValue
		case n < 0 || n > maxAlloc:
			return c.fail(errors.ValueError, "expects a size from 0 to %d, got %d.", maxAlloc, n)
		case fill < 0 || fill > 255:
			return c.fail(errors.ValueError, "expects a fill from 0 to 255, got %d.", fill)
		}

		return value.NewBytes(bytes.Repeat([]byte{byte(fill)}, n))
	}},
	{"bytes.fromHex", 1, 1, func(c *call) value.Value {
		b, err := hex.DecodeString(c.str(0))
		if c.ok() && err != nil {
			return c.fail(errors.ValueError, "could not decode hex: %s.", err)
		}

		return value.NewBytes(b)
	}},
	{"bytes.fromBase64", 1, 1, func(c *call) value.Value {
		b, err := base64.StdEncoding.DecodeString(c.str(0))
		if c.ok() && err != nil {
			return c.fail(errors.ValueError, "could not decode base64: %s.", err)
		}

		return value.NewBytes(b)
	}},
}

// maxAlloc caps bytes.alloc.
const maxAlloc = 1 << 30
