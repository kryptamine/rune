package rune

import (
	"fmt"
	"math"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
	"unicode/utf8"
)

// Flags of opSlice.
const (
	sliceInclusive = 1 << iota
	sliceStart
	sliceEnd
)

// slice is target[start..end] of an array, a string, bytes or a buffer.
func slice(tok *ast.Token, target, start, end value.Value, flags uint8) (value.Value, error) {
	i, j := 0, math.MaxInt

	var err error
	if flags&sliceStart != 0 {
		if i, err = sliceBound(tok, start, "start"); err != nil {
			return value.NilValue, err
		}
	}

	if flags&sliceEnd != 0 {
		if j, err = sliceBound(tok, end, "end"); err != nil {
			return value.NilValue, err
		}
	}

	inclusive := flags&sliceInclusive != 0

	switch target.Kind() {
	case value.Array:
		xs := target.Arr()
		lo, hi := sliceRange(len(xs), i, j, inclusive)

		return value.NewArray(slices.Clone(xs[lo:hi])), nil
	case value.String:
		s := target.Str()
		n := utf8.RuneCountInString(s)
		lo, hi := sliceRange(n, i, j, inclusive)

		if n != len(s) {
			lo, hi = helpers.ByteOffset(s, lo), helpers.ByteOffset(s, hi)
		}

		return value.NewString(s[lo:hi]), nil
	case value.Bytes:
		b := target.Bytes()
		lo, hi := sliceRange(len(b), i, j, inclusive)

		return value.NewBytes(b[lo:hi:hi]), nil
	case value.Buffer:
		b := target.Buf().Data
		lo, hi := sliceRange(len(b), i, j, inclusive)

		return value.NewBytes(slices.Clone(b[lo:hi])), nil
	}

	return value.NilValue, errors.NewRuntimeError(errors.TypeError, *tok, fmt.Sprintf(
		"Cannot slice %s; only an array, a string, bytes or a buffer.", helpers.TypeName(target)))
}

// sliceRange resolves bounds i and j against length n: a negative one
// counts from the end, and one past either end is clamped, so the range
// is never backwards.
func sliceRange(n, i, j int, inclusive bool) (lo, hi int) {
	if i < 0 {
		i += n
	}

	if j < 0 {
		j += n
	}

	if inclusive && j < n {
		j++
	}

	lo = max(0, min(i, n))

	return lo, max(lo, min(j, n))
}

func sliceBound(tok *ast.Token, v value.Value, which string) (int, error) {
	if !v.IsInt() {
		return 0, errors.NewRuntimeError(errors.TypeError, *tok, fmt.Sprintf(
			"Slice %s must be an integer, got %s.", which, helpers.TypeName(v)))
	}

	return int(v.Int()), nil
}
