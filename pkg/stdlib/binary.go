package stdlib

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"hash/crc32"
	"math"
	"rune/pkg/errors"
	"rune/pkg/value"
	"slices"
	"unicode/utf8"
)

// binaryData is argument i, bytes or a buffer, as what it holds now.
func (c *call) binaryData(i int) []byte {
	b, ok := c.args[i].Binary()
	if !ok {
		c.typeError(i, "bytes or a buffer")
	}

	return b
}

func (c *call) buffer(i int) *value.ByteBuf {
	if c.args[i].Kind() != value.Buffer {
		c.typeError(i, "a buffer")
		return nil
	}

	return c.args[i].Buf()
}

type byteOrder interface {
	binary.ByteOrder
	binary.AppendByteOrder
}

// byteOrder is the optional argument i: "le", the default, or "be".
func (c *call) byteOrder(i int) byteOrder {
	if !c.has(i) {
		return binary.LittleEndian
	}

	if s := c.str(i); s == "be" {
		return binary.BigEndian
	} else if s != "le" && c.ok() {
		c.fail(errors.ValueError, "expects byte order \"le\" or \"be\", got %q.", s)
	}

	return binary.LittleEndian
}

type sizes struct {
	valid []int
	text  string
}

var (
	intSizes   = sizes{[]int{1, 2, 4, 8}, "1, 2, 4 or 8"}
	floatSizes = sizes{[]int{4, 8}, "4 or 8"}
)

// size is argument i, a width in bytes, 8 when left out.
func (c *call) size(i int, s sizes) int {
	if !c.has(i) {
		return 8
	}

	n := c.int(i)
	if c.ok() && !slices.Contains(s.valid, n) {
		c.fail(errors.ValueError, "expects a size of %s, got %d.", s.text, n)
	}

	return n
}

// field is the n bytes of b at offset at, or nil after failing if they
// run past the end.
func (c *call) field(b []byte, at, n int) []byte {
	if !c.ok() {
		return nil
	}

	if at < 0 || at > len(b)-n {
		c.fail(errors.IndexError, "cannot read %d bytes at %d of %d.", n, at, len(b))
		return nil
	}

	return b[at : at+n]
}

func getUint(o byteOrder, b []byte) uint64 {
	switch len(b) {
	case 1:
		return uint64(b[0])
	case 2:
		return uint64(o.Uint16(b))
	case 4:
		return uint64(o.Uint32(b))
	}

	return o.Uint64(b)
}

func appendUint(o byteOrder, b []byte, n int, x uint64) []byte {
	switch n {
	case 1:
		return append(b, byte(x))
	case 2:
		return o.AppendUint16(b, uint16(x))
	case 4:
		return o.AppendUint32(b, uint32(x))
	}

	return o.AppendUint64(b, x)
}

// readInt reads an integer of argument 2's size at argument 1, extending
// the sign if signed.
func readInt(name string, signed bool) *Native {
	return &Native{name, 3, 4, func(c *call) value.Value {
		b, at, n, o := c.binaryData(0), c.int(1), c.size(2, intSizes), c.byteOrder(3)
		f := c.field(b, at, n)
		if !c.ok() {
			return value.NilValue
		}

		x := int64(getUint(o, f))
		if shift := 64 - 8*n; signed {
			x = x << shift >> shift
		}

		return value.NewInt(x)
	}}
}

// readVarint gives [value, next offset], or nil if the data ends first.
func readVarint(name string, read func([]byte) (int64, int)) *Native {
	return &Native{name, 2, 2, func(c *call) value.Value {
		b, at := c.binaryData(0), c.int(1)
		switch {
		case !c.ok():
			return value.NilValue
		case at < 0 || at > len(b):
			return c.fail(errors.IndexError, "offset %d is out of bounds of %d bytes.", at, len(b))
		}

		x, n := read(b[at:])
		switch {
		case n == 0:
			return value.NilValue
		case n < 0:
			return c.fail(errors.ValueError, "varint at %d overflows 64 bits.", at)
		}

		return value.NewArray([]value.Value{value.NewInt(x), count(at + n)})
	}}
}

func digest(name string, sum func([]byte) []byte) *Native {
	return &Native{name, 1, 1, func(c *call) value.Value { return value.NewBytes(sum(c.binaryData(0))) }}
}

// readers are the methods that read binary data, of bytes and buffers
// alike; prefix names the type.
func readers(prefix string) []*Native {
	return []*Native{
		{prefix + ".text", 1, 1, func(c *call) value.Value {
			b := c.binaryData(0)
			switch {
			case !c.ok():
				return value.NilValue
			case !utf8.Valid(b):
				return c.fail(errors.ValueError, "got bytes that are not valid UTF-8.")
			}

			return value.NewString(string(b))
		}},
		{prefix + ".indexOf", 2, 2, func(c *call) value.Value {
			return found(bytes.Index(c.binaryData(0), c.binaryData(1)))
		}},
		{prefix + ".hex", 1, 1, func(c *call) value.Value {
			return value.NewString(hex.EncodeToString(c.binaryData(0)))
		}},
		{prefix + ".base64", 1, 1, func(c *call) value.Value {
			return value.NewString(base64.StdEncoding.EncodeToString(c.binaryData(0)))
		}},
		readInt(prefix+".uint", false),
		readInt(prefix+".int", true),
		{prefix + ".float", 2, 4, func(c *call) value.Value {
			b, at, n, o := c.binaryData(0), c.int(1), c.size(2, floatSizes), c.byteOrder(3)
			f := c.field(b, at, n)
			switch {
			case !c.ok():
				return value.NilValue
			case n == 4:
				return value.NewNumber(float64(math.Float32frombits(o.Uint32(f))))
			}

			return value.NewNumber(math.Float64frombits(o.Uint64(f)))
		}},
		readVarint(prefix+".uvarint", func(b []byte) (int64, int) {
			x, n := binary.Uvarint(b)
			return int64(x), n
		}),
		readVarint(prefix+".varint", binary.Varint),
		{prefix + ".crc32", 1, 1, func(c *call) value.Value {
			return value.NewInt(int64(crc32.ChecksumIEEE(c.binaryData(0))))
		}},
		digest(prefix+".md5", func(b []byte) []byte { s := md5.Sum(b); return s[:] }),
		digest(prefix+".sha1", func(b []byte) []byte { s := sha1.Sum(b); return s[:] }),
		digest(prefix+".sha256", func(b []byte) []byte { s := sha256.Sum256(b); return s[:] }),
	}
}

// appender is an in-place buffer method that appends what fn makes of its
// arguments.
func appender(name string, min, max int, fn func(c *call, b []byte) []byte) *Native {
	return method(name, min, max, (*call).buffer, func(c *call, buf *value.ByteBuf) value.Value {
		if out := fn(c, buf.Data); c.ok() {
			buf.Data = out
		}

		return value.NilValue
	})
}

// appendInt appends argument 1 as an integer of argument 2's size, which it
// must fit unless that is 8.
func appendInt(name string, signed bool) *Native {
	return appender(name, 3, 4, func(c *call, b []byte) []byte {
		x, n, o := c.int(1), c.size(2, intSizes), c.byteOrder(3)
		if !c.ok() {
			return b
		}

		lo, hi, kind := 0, 1<<(8*n), "an unsigned"
		if signed {
			lo, hi, kind = -hi/2, hi/2, "a signed"
		}

		if n < 8 && (x < lo || x >= hi) {
			c.fail(errors.ValueError, "cannot write %d as %s %d-byte integer.", x, kind, n)
		}

		return appendUint(o, b, n, uint64(x))
	})
}

var bufferInPlace = []*Native{
	appender("buffer.write", 2, 2, func(c *call, b []byte) []byte { return append(b, c.data(1)...) }),
	appendInt("buffer.writeUint", false),
	appendInt("buffer.writeInt", true),
	appender("buffer.writeFloat", 2, 4, func(c *call, b []byte) []byte {
		x, n, o := c.num(1), c.size(2, floatSizes), c.byteOrder(3)
		if n == 4 {
			return o.AppendUint32(b, math.Float32bits(float32(x)))
		}

		return o.AppendUint64(b, math.Float64bits(x))
	}),
	appender("buffer.writeUvarint", 2, 2, func(c *call, b []byte) []byte {
		x := c.int(1)
		if c.ok() && x < 0 {
			c.fail(errors.ValueError, "cannot write %d as an unsigned varint.", x)
		}

		return binary.AppendUvarint(b, uint64(x))
	}),
	appender("buffer.writeVarint", 2, 2, func(c *call, b []byte) []byte {
		return binary.AppendVarint(b, int64(c.int(1)))
	}),
	method("buffer.clear", 1, 1, (*call).buffer, func(c *call, buf *value.ByteBuf) value.Value {
		buf.Data = buf.Data[:0]
		return value.NilValue
	}),
}

var bufferMethods = slices.Concat(readers("buffer"), lenMethod("buffer"))

func newBuffer(c *call) value.Value {
	buf := &value.ByteBuf{}
	if c.has(0) {
		buf.Data = slices.Clone(c.data(0))
	}

	return value.NewBuffer(buf)
}
