package helpers

import (
	"rune/pkg/callable"
)

func IsTruthy(val any) bool {
	if val == nil {
		return false
	}

	switch i2 := val.(type) {
	case bool:
		return i2
	case string:
		return len(i2) != 0
	case float64:
		return i2 != 0.0
	case *callable.FunctionCallable:
		return true
	default:
		return false
	}
}

func IsEqual(left any, right any) bool {
	if left == nil && right == nil {
		return true
	}

	if left == nil {
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
