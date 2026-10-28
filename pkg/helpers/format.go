package helpers

import (
	"fmt"
	"maps"
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
func Stringify(val any) string {
	switch v := val.(type) {
	case nil:
		return "nil"
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case []any:
		items := make([]string, len(v))
		for i, item := range v {
			items[i] = Stringify(item)
		}

		return "[" + strings.Join(items, ", ") + "]"
	case map[string]any:
		pairs := make([]string, 0, len(v))
		for _, key := range slices.Sorted(maps.Keys(v)) {
			pairs = append(pairs, key+": "+Stringify(v[key]))
		}

		return "{" + strings.Join(pairs, ", ") + "}"
	default:
		return fmt.Sprint(v)
	}
}
