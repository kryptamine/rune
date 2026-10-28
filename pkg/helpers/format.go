package helpers

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// TypeName returns the language-level name of a runtime value's type,
// for use in error messages instead of Go type names.
func TypeName(val any) string {
	switch val.(type) {
	case nil:
		return "nil"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "function"
	}
}

// Stringify renders a runtime value the way the language prints it.
// A container that contains itself is printed as [...] or {...}.
func Stringify(val any) string {
	return stringify(val, map[uintptr]bool{})
}

// stringify tracks the containers on the current path in seen, so shared
// (but acyclic) values still print in full.
func stringify(val any, seen map[uintptr]bool) string {
	switch v := val.(type) {
	case nil:
		return "nil"
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case []any:
		if len(v) == 0 {
			return "[]"
		}

		ptr := reflect.ValueOf(v).Pointer()
		if seen[ptr] {
			return "[...]"
		}

		seen[ptr] = true
		defer delete(seen, ptr)

		items := make([]string, len(v))
		for i, item := range v {
			items[i] = stringify(item, seen)
		}

		return "[" + strings.Join(items, ", ") + "]"
	case map[string]any:
		ptr := reflect.ValueOf(v).Pointer()
		if seen[ptr] {
			return "{...}"
		}

		seen[ptr] = true
		defer delete(seen, ptr)

		pairs := make([]string, 0, len(v))
		for _, key := range slices.Sorted(maps.Keys(v)) {
			pairs = append(pairs, key+": "+stringify(v[key], seen))
		}

		return "{" + strings.Join(pairs, ", ") + "}"
	default:
		return fmt.Sprint(v)
	}
}
