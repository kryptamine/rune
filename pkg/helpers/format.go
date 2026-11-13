package helpers

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/value"
	"strconv"
	"strings"
	"unsafe"
)

// TypeName returns the language-level name of a runtime value's type,
// for use in error messages instead of Go type names.
func TypeName(v value.Value) string {
	switch v.Kind() {
	case value.Nil:
		return "nil"
	case value.Bool:
		return "boolean"
	case value.Number:
		return "number"
	case value.String:
		return "string"
	case value.Array:
		return "array"
	case value.Object:
		return "object"
	case value.Struct:
		return v.Shape().Name
	case value.Type:
		return If(v.TypeShape().Enum, "enum", "struct")
	case value.Enum:
		return v.Variant().Shape.Name
	case value.Map:
		return "map"
	case value.Set:
		return "set"
	case value.Bytes:
		return "bytes"
	case value.Stream:
		return "stream"
	default:
		return "function"
	}
}

// Stringify renders a runtime value the way the language prints it.
// Strings inside containers are quoted, so ["1", 1] and [1, 1] differ.
// A container that contains itself is printed as [...] or {...}.
func Stringify(v value.Value) string {
	return stringify(v, nil)
}

var quoter = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`, "\r", `\r`)

// quote writes s as a string literal the scanner accepts.
func quote(s string) string { return `"` + quoter.Replace(s) + `"` }

// isIdentifier reports whether s can be an unquoted key in an object literal.
func isIdentifier(s string) bool {
	for i, r := range s {
		if !IsAlpha(r) && (i == 0 || !IsDigit(r)) {
			return false
		}
	}

	_, keyword := ast.Keywords[s]

	return s != "" && !keyword
}

// Quoted renders v as print does inside a container, strings quoted.
func Quoted(v value.Value) string { return element(v, nil) }

// element renders a value nested in a container.
func element(v value.Value, seen map[uintptr]bool) string {
	if v.IsString() {
		return quote(v.Str())
	}

	return stringify(v, seen)
}

// stringify tracks the containers on the current path in seen, so shared
// (but acyclic) values still print in full.
func stringify(v value.Value, seen map[uintptr]bool) string {
	switch v.Kind() {
	case value.Nil:
		return "nil"
	case value.Bool:
		return strconv.FormatBool(v.Bool())
	case value.Number:
		return string(AppendNumber(nil, v))
	case value.String:
		return v.Str()
	case value.Array:
		arr := v.Arr()
		if len(arr) == 0 {
			return "[]"
		}

		ptr := uintptr(unsafe.Pointer(v.ArrRef()))
		if seen[ptr] {
			return "[...]"
		}

		if seen == nil {
			seen = map[uintptr]bool{}
		}

		seen[ptr] = true
		defer delete(seen, ptr)

		items := make([]string, len(arr))
		for i, item := range arr {
			items[i] = element(item, seen)
		}

		return "[" + strings.Join(items, ", ") + "]"
	case value.Object:
		obj := v.Obj()
		ptr := uintptr(unsafe.Pointer(obj))
		if seen[ptr] {
			return "{...}"
		}

		if seen == nil {
			seen = map[uintptr]bool{}
		}

		seen[ptr] = true
		defer delete(seen, ptr)

		pairs := make([]string, 0, obj.Len())
		for key, x := range obj.All() {
			name := key
			if !isIdentifier(key) {
				name = quote(key)
			}

			pairs = append(pairs, name+": "+element(x, seen))
		}

		return "{" + strings.Join(pairs, ", ") + "}"
	case value.Struct:
		shape := v.Shape()
		ptr := uintptr(v.Ptr())
		if seen[ptr] {
			return shape.Name + " {...}"
		}

		if seen == nil {
			seen = map[uintptr]bool{}
		}

		seen[ptr] = true
		defer delete(seen, ptr)

		pairs := make([]string, len(shape.Fields))
		for i, f := range v.Fields() {
			pairs[i] = shape.Fields[i] + ": " + element(f, seen)
		}

		return shape.Name + " {" + strings.Join(pairs, ", ") + "}"
	case value.Type:
		return "<" + TypeName(v) + " " + v.TypeShape().Name + ">"
	case value.Enum:
		return v.Variant().Shape.Name + "." + v.Variant().Name
	case value.Map, value.Set:
		return stringifyTable(v, seen)
	case value.Bytes:
		return QuoteBytes(v.Bytes())
	case value.Stream:
		return fmt.Sprint(v.Internal())
	default:
		return fmt.Sprint(v.Fn())
	}
}

func stringifyTable(v value.Value, seen map[uintptr]bool) string {
	m := v.HashMap()
	name := TypeName(v)

	ptr := uintptr(v.Ptr())
	if seen[ptr] {
		return name + " {...}"
	}

	if m.Len() == 0 {
		return name + " {}"
	}

	if seen == nil {
		seen = map[uintptr]bool{}
	}

	seen[ptr] = true
	defer delete(seen, ptr)

	items := make([]string, 0, m.Len())
	m.Each(func(k, x value.Value) bool {
		if v.Kind() == value.Set {
			items = append(items, element(k, seen))
		} else {
			items = append(items, element(k, seen)+": "+element(x, seen))
		}

		return true
	})

	return name + " {" + strings.Join(items, ", ") + "}"
}

// QuoteBytes writes b as a bytes literal: printable ASCII as is, other
// bytes as \xNN.
func QuoteBytes(b []byte) string {
	buf := make([]byte, 0, len(b)+3)
	buf = append(buf, 'b', '"')

	for _, c := range b {
		switch {
		case c == '"' || c == '\\':
			buf = append(buf, '\\', c)
		case c == '\n':
			buf = append(buf, '\\', 'n')
		case c == '\t':
			buf = append(buf, '\\', 't')
		case c == '\r':
			buf = append(buf, '\\', 'r')
		case c == '{' || c == '}':
			buf = append(buf, '\\', c)
		case c >= 0x20 && c < 0x7f:
			buf = append(buf, c)
		default:
			buf = append(buf, '\\', 'x', hexDigits[c>>4], hexDigits[c&15])
		}
	}

	return string(append(buf, '"'))
}

const hexDigits = "0123456789abcdef"

func AppendValue(buf []byte, v value.Value) []byte {
	switch v.Kind() {
	case value.Number:
		return AppendNumber(buf, v)
	case value.String:
		return append(buf, v.Str()...)
	case value.Bool:
		return strconv.AppendBool(buf, v.Bool())
	case value.Nil:
		return append(buf, "nil"...)
	default:
		return append(buf, Stringify(v)...)
	}
}

// Plural is n and word, made plural unless n is 1.
func Plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}

	return fmt.Sprintf("%d %ss", n, word)
}
