package helpers

import (
	"fmt"
	"maps"
	"reflect"
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
	default:
		return "function"
	}
}

// Stringify renders a runtime value the way the language prints it.
// A container that contains itself is printed as [...] or {...}.
func Stringify(v value.Value) string {
	return stringify(v, map[uintptr]bool{})
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
		return strconv.FormatFloat(v.Num(), 'f', -1, 64)
	case value.String:
		return v.Str()
	case value.Array:
		arr := v.Arr()
		if len(arr) == 0 {
			return "[]"
		}

		ptr := uintptr(unsafe.Pointer(unsafe.SliceData(arr)))
		if seen[ptr] {
			return "[...]"
		}

		seen[ptr] = true
		defer delete(seen, ptr)

		items := make([]string, len(arr))
		for i, item := range arr {
			items[i] = stringify(item, seen)
		}

		return "[" + strings.Join(items, ", ") + "]"
	case value.Object:
		obj := v.Obj()
		ptr := reflect.ValueOf(obj).Pointer()
		if seen[ptr] {
			return "{...}"
		}

		seen[ptr] = true
		defer delete(seen, ptr)

		pairs := make([]string, 0, len(obj))
		for _, key := range slices.Sorted(maps.Keys(obj)) {
			pairs = append(pairs, key+": "+stringify(obj[key], seen))
		}

		return "{" + strings.Join(pairs, ", ") + "}"
	default:
		return fmt.Sprint(v.Fn())
	}
}
