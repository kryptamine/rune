package helpers

import (
	"reflect"
	"rune/pkg/value"
)

// IsTruthy follows JS-like semantics: nil, false, 0 and "" are falsy,
// everything else (including arrays, objects and functions) is truthy.
func IsTruthy(v value.Value) bool {
	switch v.Kind() {
	case value.Nil:
		return false
	case value.Bool:
		return v.Bool()
	case value.String:
		return len(v.Str()) != 0
	case value.Number:
		return v.Num() != 0.0
	default:
		return true
	}
}

// IsEqual compares arrays and objects structurally and everything else by value
// (functions by identity). Values of different kinds are never equal.
func IsEqual(left, right value.Value) bool {
	if left.Kind() != right.Kind() {
		return false
	}

	switch left.Kind() {
	case value.Nil:
		return true
	case value.Bool:
		return left.Bool() == right.Bool()
	case value.Number:
		return left.Num() == right.Num()
	case value.String:
		return left.Str() == right.Str()
	case value.Array, value.Object:
		return reflect.DeepEqual(left.Fn(), right.Fn())
	default:
		return left.Fn() == right.Fn()
	}
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
