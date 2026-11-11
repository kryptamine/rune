package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
	"strings"
)

type cursorMode uint8

const (
	arrayCursor cursorMode = iota
	stringCursor
	objectCursor
)

// cursor walks what a for-in iterates: (index or key, element) pairs. An
// object cursor walks the keys in arr, reading each value from obj.
type cursor struct {
	mode cursorMode
	i    int
	arr  []value.Value
	str  string
	obj  map[string]value.Value
}

func (c *cursor) next() (key, val value.Value, ok bool) {
	i := c.i
	c.i++

	switch c.mode {
	case arrayCursor:
		if i < len(c.arr) {
			return value.NewNumber(float64(i)), c.arr[i], true
		}
	case stringCursor:
		if c.str != "" {
			ch, n := helpers.NextChar(c.str)
			c.str = c.str[n:]

			return value.NewNumber(float64(i)), value.NewString(ch), true
		}
	case objectCursor:
		if i < len(c.arr) {
			return c.arr[i], c.obj[c.arr[i].Str()], true
		}
	}

	return key, val, false
}

// cursorOf snapshots arrays. Objects yield their keys in sorted order; a
// loop that binds only one variable walks them like an array of keys.
func cursorOf(v value.Value, hasKey bool, keyword *ast.Token) (*cursor, error) {
	switch v.Kind() {
	case value.Array:
		return &cursor{mode: arrayCursor, arr: slices.Clone(v.Arr())}, nil
	case value.String:
		return &cursor{mode: stringCursor, str: v.Str()}, nil
	case value.Object:
		obj := v.Obj()
		keys := sortedKeys(obj)

		if !hasKey {
			return &cursor{mode: arrayCursor, arr: keys}, nil
		}

		return &cursor{mode: objectCursor, arr: keys, obj: obj}, nil
	}

	return nil, errors.NewRuntimeError(*keyword, fmt.Sprintf(
		"Can only iterate over arrays, strings and objects, got %s.", helpers.TypeName(v)))
}

func sortedKeys(obj map[string]value.Value) []value.Value {
	keys := make([]value.Value, 0, len(obj))
	for k := range obj {
		keys = append(keys, value.NewString(k))
	}

	slices.SortFunc(keys, func(a, b value.Value) int { return strings.Compare(a.Str(), b.Str()) })

	return keys
}
