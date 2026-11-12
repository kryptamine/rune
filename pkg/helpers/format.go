package helpers

import (
	"fmt"
	"maps"
	"reflect"
	"rune/pkg/ast"
	"rune/pkg/value"
	"slices"
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
		return "struct"
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

func formatNumber(buf []byte, n float64) []byte {
	if n == 0 {
		n = 0 // -0 prints as 0
	}

	return strconv.AppendFloat(buf, n, 'f', -1, 64)
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
		return string(formatNumber(nil, v.Num()))
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
		ptr := reflect.ValueOf(obj).Pointer()
		if seen[ptr] {
			return "{...}"
		}

		if seen == nil {
			seen = map[uintptr]bool{}
		}

		seen[ptr] = true
		defer delete(seen, ptr)

		pairs := make([]string, 0, len(obj))
		for _, key := range slices.Sorted(maps.Keys(obj)) {
			name := key
			if !isIdentifier(key) {
				name = quote(key)
			}

			pairs = append(pairs, name+": "+element(obj[key], seen))
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
		return "<struct " + v.TypeShape().Name + ">"
	default:
		return fmt.Sprint(v.Fn())
	}
}

func AppendValue(buf []byte, v value.Value) []byte {
	switch v.Kind() {
	case value.Number:
		return formatNumber(buf, v.Num())
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
