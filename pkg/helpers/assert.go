package helpers

import "rune/pkg/value"

// IsTruthy is false for nil, false, 0 and empty values; see Value.Truthy.
func IsTruthy(v value.Value) bool { return v.Truthy() }

// IsEqual compares arrays, objects and structs structurally and everything else by value
// (functions by identity). Values of different kinds are never equal.
func IsEqual(left, right value.Value) bool {
	if left.Kind() != right.Kind() {
		return false
	}

	// A copy not yet written is the same container; NaN is no number's equal.
	if left == right && left.Kind() != value.Number {
		return true
	}

	switch left.Kind() {
	case value.Nil:
		return true
	case value.Bool:
		return left.Bool() == right.Bool()
	case value.Number:
		return value.NumEqual(left, right)
	case value.String:
		return left.Str() == right.Str()
	case value.Array:
		l, r := left.Arr(), right.Arr()
		if len(l) != len(r) {
			return false
		}

		for i := range l {
			if !IsEqual(l[i], r[i]) {
				return false
			}
		}

		return true
	case value.Object:
		l, r := left.Obj(), right.Obj()
		if l.Len() != r.Len() {
			return false
		}

		for k, lv := range l.All() {
			rv, ok := r.Get(k)
			if !ok || !IsEqual(lv, rv) {
				return false
			}
		}

		return true
	case value.Struct:
		if left.Shape() != right.Shape() {
			return false
		}

		r := right.Fields()
		for i, lv := range left.Fields() {
			if !IsEqual(lv, r[i]) {
				return false
			}
		}

		return true
	case value.Type, value.Enum:
		return left.Ptr() == right.Ptr()
	case value.Bytes, value.Buffer:
		l, _ := left.Binary()
		r, _ := right.Binary()

		return string(l) == string(r)
	case value.Map, value.Set:
		return tablesEqual(left.HashMap(), right.HashMap())
	default:
		return left.Fn() == right.Fn()
	}
}

// tablesEqual compares maps by keys and values, sets by elements, in any
// order.
func tablesEqual(l, r *value.HashMap) bool {
	if l.Len() != r.Len() {
		return false
	}

	equal := true

	l.Each(func(k, lv value.Value) bool {
		rv, ok, _ := r.Get(k)
		equal = ok && IsEqual(lv, rv)

		return equal
	})

	return equal
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
