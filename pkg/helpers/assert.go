package helpers

import (
	"reflect"
	"rune/pkg/value"
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
		return left.Num() == right.Num()
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

		pair := refPair{reflect.ValueOf(l).Pointer(), reflect.ValueOf(r).Pointer()}
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
		if len(l) != len(r) {
			return false
		}

		pair := refPair{reflect.ValueOf(l).Pointer(), reflect.ValueOf(r).Pointer()}
		if pair.l == pair.r || seen[pair] {
			return true
		}

		if seen == nil {
			seen = map[refPair]bool{}
		}

		seen[pair] = true
		defer delete(seen, pair)

		for k, lv := range l {
			rv, ok := r[k]
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
