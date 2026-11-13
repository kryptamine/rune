package helpers

import (
	"rune/pkg/value"
	"unsafe"
)

type refPair struct{ l, r uintptr }

// IsTruthy follows JS-like semantics: nil, false, 0 and "" are falsy,
// everything else (including arrays, objects and functions) is truthy.
func IsTruthy(v value.Value) bool { return v.Truthy() }

// IsEqual compares arrays, objects and structs structurally and everything else by value
// (functions by identity). Values of different kinds are never equal.
func IsEqual(left, right value.Value) bool {
	return isEqual(left, right, nil)
}

func isEqual(left, right value.Value, seen map[refPair]bool) bool {
	if left.Kind() != right.Kind() {
		return false
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
		l, r := left.ArrRef(), right.ArrRef()
		if l == r {
			return true
		}

		if len(*l) != len(*r) {
			return false
		}

		pair := refPair{uintptr(unsafe.Pointer(l)), uintptr(unsafe.Pointer(r))}
		if seen[pair] {
			return true
		}

		if seen == nil {
			seen = map[refPair]bool{}
		}

		seen[pair] = true
		defer delete(seen, pair)

		for i := range *l {
			if !isEqual((*l)[i], (*r)[i], seen) {
				return false
			}
		}

		return true
	case value.Object:
		l, r := left.Obj(), right.Obj()
		if l.Len() != r.Len() {
			return false
		}

		pair := refPair{uintptr(unsafe.Pointer(l)), uintptr(unsafe.Pointer(r))}
		if pair.l == pair.r || seen[pair] {
			return true
		}

		if seen == nil {
			seen = map[refPair]bool{}
		}

		seen[pair] = true
		defer delete(seen, pair)

		for k, lv := range l.All() {
			rv, ok := r.Get(k)
			if !ok || !isEqual(lv, rv, seen) {
				return false
			}
		}

		return true
	case value.Struct:
		if left.Shape() != right.Shape() {
			return false
		}

		pair := refPair{uintptr(left.Ptr()), uintptr(right.Ptr())}
		if pair.l == pair.r || seen[pair] {
			return true
		}

		if seen == nil {
			seen = map[refPair]bool{}
		}

		seen[pair] = true
		defer delete(seen, pair)

		r := right.Fields()
		for i, lv := range left.Fields() {
			if !isEqual(lv, r[i], seen) {
				return false
			}
		}

		return true
	case value.Type:
		return left.Ptr() == right.Ptr()
	case value.Bytes:
		return string(left.Bytes()) == string(right.Bytes())
	case value.Map, value.Set:
		return tablesEqual(left, right, seen)
	default:
		return left.Fn() == right.Fn()
	}
}

// tablesEqual compares maps by keys and values, sets by elements, in any
// order.
func tablesEqual(left, right value.Value, seen map[refPair]bool) bool {
	l, r := left.HashMap(), right.HashMap()
	if l == r {
		return true
	}

	if l.Len() != r.Len() {
		return false
	}

	pair := refPair{uintptr(left.Ptr()), uintptr(right.Ptr())}
	if seen[pair] {
		return true
	}

	if seen == nil {
		seen = map[refPair]bool{}
	}

	seen[pair] = true
	defer delete(seen, pair)

	equal := true

	l.Each(func(k, lv value.Value) bool {
		rv, ok, _ := r.Get(k)
		equal = ok && isEqual(lv, rv, seen)

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
