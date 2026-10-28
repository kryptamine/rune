package helpers

import "reflect"

// IsTruthy follows JS-like semantics: nil, false, 0 and "" are falsy,
// everything else (including arrays, objects and functions) is truthy.
func IsTruthy(val any) bool {
	switch v := val.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return len(v) != 0
	case float64:
		return v != 0.0
	default:
		return true
	}
}

// IsEqual compares arrays and objects structurally and everything else with ==.
// Plain == on slices or maps would panic, so containers go through DeepEqual.
func IsEqual(left any, right any) bool {
	switch left.(type) {
	case []any, map[string]any:
		return reflect.DeepEqual(left, right)
	}

	switch right.(type) {
	case []any, map[string]any:
		return false
	}

	return left == right
}

func IsString(val any) bool {
	_, ok := val.(string)
	return ok
}

func IsFloat(val any) bool {
	_, ok := val.(float64)
	return ok
}

func IsDigit(c rune) bool {
	return c >= '0' && c <= '9'
}

func IsAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		c == '_'
}

func If[T any](cond bool, vtrue, vfalse T) T {
	if cond {
		return vtrue
	}
	return vfalse
}
