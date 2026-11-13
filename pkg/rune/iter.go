package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/stdlib"
	"rune/pkg/value"
	"slices"
)

// cursor walks what a for-in iterates, as (index or key, element) pairs:
// an array by itself, which the dispatch loop reads, anything else by next.
type cursor struct {
	arr  []value.Value
	i    int
	next func() (key, val value.Value, ok bool, err error)
	// own is set when arr is the cursor's own snapshot, so a loop that
	// runs to its end can hand the cursor on to the next.
	own bool
}

// maxCursors bounds the cursors kept for reuse.
const maxCursors = 64

// arrayCursor walks a snapshot of items, in a cursor a finished loop left.
func (p *Interpreter) arrayCursor(items []value.Value) *cursor {
	n := len(p.cursors)
	if n == 0 {
		return &cursor{arr: slices.Clone(items), own: true}
	}

	c := p.cursors[n-1]
	p.cursors = p.cursors[:n-1]
	c.arr, c.i = append(c.arr[:0], items...), 0

	return c
}

// release keeps the cursor of a loop that ran to its end for reuse.
func (p *Interpreter) release(c *cursor) {
	if c.own && len(p.cursors) < maxCursors {
		p.cursors = append(p.cursors, c)
	}
}

func (c *cursor) advance() (key, val value.Value, ok bool, err error) {
	if c.next != nil {
		return c.next()
	}

	if i := c.i; i < len(c.arr) {
		c.i++
		return value.NewInt(int64(i)), c.arr[i], true, nil
	}

	return key, val, false, nil
}

// indexed yields the elements at returns, with their indices, until it
// reports no more.
func indexed(at func(i int) (value.Value, bool)) func() (value.Value, value.Value, bool, error) {
	i := 0

	return func() (value.Value, value.Value, bool, error) {
		v, ok := at(i)
		i++

		return value.NewInt(int64(i - 1)), v, ok, nil
	}
}

// cursorOf snapshots arrays and walks maps and sets in place. Objects
// yield their keys in sorted order and structs in declaration order; a
// loop that binds only one variable walks them like an array of keys.
func (p *Interpreter) cursorOf(v value.Value, hasKey bool, keyword *ast.Token) (*cursor, error) {
	switch v.Kind() {
	case value.Array:
		return p.arrayCursor(v.Arr()), nil
	case value.String:
		s := v.Str()

		return &cursor{next: indexed(func(int) (value.Value, bool) {
			if s == "" {
				return value.NilValue, false
			}

			ch, n := helpers.NextChar(s)
			s = s[n:]

			return value.NewString(ch), true
		})}, nil
	case value.Bytes:
		b := v.Bytes()

		return &cursor{next: indexed(func(i int) (value.Value, bool) {
			if i >= len(b) {
				return value.NilValue, false
			}

			return value.NewInt(int64(b[i])), true
		})}, nil
	case value.Object:
		o := v.Obj()
		keys := make([]value.Value, 0, o.Len())
		for k := range o.All() {
			keys = append(keys, value.NewString(k))
		}

		return pairs(keys, hasKey, func(_ int, k value.Value) value.Value { x, _ := o.Get(k.Str()); return x }), nil
	case value.Struct:
		// Keys and fields never change, so the cursor shares them.
		fields := v.Fields()
		return pairs(v.Shape().Keys, hasKey, func(i int, _ value.Value) value.Value { return fields[i] }), nil
	case value.Map, value.Set:
		return tableCursor(v, hasKey && v.Kind() == value.Map, *keyword), nil
	case value.Stream:
		return streamCursor(v.Internal().(*stdlib.Stream), *keyword), nil
	}

	return nil, errors.NewRuntimeError(errors.TypeError, *keyword, fmt.Sprintf(
		"Can only iterate over arrays, strings, objects, structs, maps, sets, bytes and streams, got %s.", helpers.TypeName(v)))
}

// pairs walks keys, with their values if hasKey.
func pairs(keys []value.Value, hasKey bool, val func(i int, k value.Value) value.Value) *cursor {
	if !hasKey {
		return &cursor{arr: keys}
	}

	i := 0

	return &cursor{next: func() (value.Value, value.Value, bool, error) {
		if i >= len(keys) {
			return value.NilValue, value.NilValue, false, nil
		}

		i++

		return keys[i-1], val(i-1, keys[i-1]), true, nil
	}}
}

// tableCursor walks a Map or a Set in place, by key with keys, else by
// index; adding a key while it walks is an error.
func tableCursor(v value.Value, keys bool, keyword ast.Token) *cursor {
	walk, n := v.HashMap().Walk(), 0

	return &cursor{next: func() (value.Value, value.Value, bool, error) {
		k, x, ok, stale := walk()

		switch {
		case stale:
			return k, x, false, errors.NewRuntimeError(errors.ValueError, keyword,
				"Cannot add to a "+helpers.TypeName(v)+" while iterating over it.")
		case !ok:
			return k, x, false, nil
		case keys:
			return k, x, true, nil
		}

		n++

		return value.NewInt(int64(n - 1)), k, true, nil
	}}
}

// streamCursor reads lines up to the end.
func streamCursor(s *stdlib.Stream, keyword ast.Token) *cursor {
	n := 0

	return &cursor{next: func() (value.Value, value.Value, bool, error) {
		line, ok, err := s.ReadLine()
		if err != nil {
			return value.NilValue, value.NilValue, false, errors.NewIOError(keyword, s.String()+": "+err.Error()+".", "", "")
		}

		n++

		return value.NewInt(int64(n - 1)), value.NewString(line), ok, nil
	}}
}
