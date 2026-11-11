package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/environment"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
	"strings"
)

func compileForIn(s *ast.ForInStmt) execFn {
	iterable := compileIterable(s)
	body := compileStmt(s.Body)
	size, pooled, hasKey := s.EnvSize, s.Pooled, s.Key != nil

	return func(p *Interpreter) error {
		c, err := iterable(p)
		if err != nil {
			return err
		}

		prev := p.environment

		var shared *environment.Environment
		if pooled {
			shared = p.pool.Get(prev, size)
		}

		for {
			key, val, ok := c.next()
			if !ok {
				break
			}

			env := shared
			if env == nil {
				env = environment.NewEnvironment(prev, size)
			}

			if hasKey {
				env.Set(0, key)
				env.Set(1, val)
			} else {
				env.Set(0, val)
			}

			p.environment = env

			if err = body(p); err == errContinue {
				err = nil
			} else if err != nil {
				break
			}
		}

		p.environment = prev

		if pooled {
			p.pool.Put(shared)
		}

		if err == errBreak {
			return nil
		}

		return err
	}
}

type cursorMode uint8

const (
	rangeCursor cursorMode = iota
	arrayCursor
	stringCursor
	objectCursor
)

// cursor walks what a for-in iterates: (index or key, element) pairs. An
// object cursor walks the keys in arr, reading each value from obj.
type cursor struct {
	mode cursorMode
	i    int
	r    numRange
	arr  []value.Value
	str  string
	obj  map[string]value.Value
}

func (c *cursor) next() (key, val value.Value, ok bool) {
	i := c.i
	c.i++

	switch c.mode {
	case rangeCursor:
		x, ok := c.r.at(float64(i))
		return value.NewNumber(float64(i)), value.NewNumber(x), ok
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

// compileIterable counts through a range instead of building its array.
func compileIterable(s *ast.ForInStmt) func(*Interpreter) (cursor, error) {
	if r, ok := s.Iterable.(*ast.RangeExpr); ok {
		bounds := compileRange(r)

		return func(p *Interpreter) (cursor, error) {
			r, err := bounds(p)
			return cursor{mode: rangeCursor, r: r}, err
		}
	}

	e := compileExpr(s.Iterable)

	return func(p *Interpreter) (cursor, error) {
		v, err := e(p)
		if err != nil {
			return cursor{}, err
		}

		return cursorOf(s, v)
	}
}

// cursorOf snapshots arrays. Objects yield their keys in sorted order; a
// loop that binds only one variable walks them like an array of keys.
func cursorOf(s *ast.ForInStmt, v value.Value) (cursor, error) {
	switch v.Kind() {
	case value.Array:
		return cursor{mode: arrayCursor, arr: slices.Clone(v.Arr())}, nil
	case value.String:
		return cursor{mode: stringCursor, str: v.Str()}, nil
	case value.Object:
		obj := v.Obj()
		keys := sortedKeys(obj)

		if s.Key == nil {
			return cursor{mode: arrayCursor, arr: keys}, nil
		}

		return cursor{mode: objectCursor, arr: keys, obj: obj}, nil
	}

	return cursor{}, errors.NewRuntimeError(s.Keyword, fmt.Sprintf(
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
