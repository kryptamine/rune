package rune

import (
	"fmt"
	"iter"
	"maps"
	"rune/pkg/ast"
	"rune/pkg/environment"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/stdlib"
	"rune/pkg/value"
	"slices"
)

// items is what a for-in walks: (index or key, element) pairs.
type items = iter.Seq2[value.Value, value.Value]

func compileForIn(s *ast.ForInStmt) execFn {
	iterable := compileIterable(s)
	body := compileStmt(s.Body)
	size, pooled, hasKey := s.EnvSize, s.Pooled, s.Key != nil

	return func(p *Interpreter) error {
		seq, err := iterable(p)
		if err != nil {
			return err
		}

		var shared *environment.Environment
		if pooled {
			shared = p.pool.Get(p.environment, size)
			defer p.pool.Put(shared)
		}

		prev := p.environment
		defer func() { p.environment = prev }()

		for key, val := range seq {
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

			switch err := body(p); err {
			case nil, errContinue:
			case errBreak:
				return nil
			default:
				return err
			}
		}

		return nil
	}
}

// compileIterable counts through range(...) instead of building its array.
func compileIterable(s *ast.ForInStmt) func(*Interpreter) (items, error) {
	if call, ok := s.Iterable.(*ast.CallExpr); ok {
		if v, ok := call.Callee.(*ast.VarExpr); ok && v.Name.Lexeme == "range" {
			return compileRangeIterable(s, call)
		}
	}

	e := compileExpr(s.Iterable)

	return func(p *Interpreter) (items, error) {
		v, err := e(p)
		if err != nil {
			return nil, err
		}

		return itemsOf(s, v)
	}
}

// compileRangeIterable falls back to a plain call when range is shadowed or
// the call would fail, so errors stay those of range itself.
func compileRangeIterable(s *ast.ForInStmt, call *ast.CallExpr) func(*Interpreter) (items, error) {
	callee := compileExpr(call.Callee)
	args := make([]evalFn, len(call.Args))

	for i, arg := range call.Args {
		args[i] = compileExpr(arg)
	}

	return func(p *Interpreter) (items, error) {
		fn, err := callee(p)
		if err != nil {
			return nil, err
		}

		vals := make([]value.Value, len(args))

		for i, arg := range args {
			if vals[i], err = arg(p); err != nil {
				return nil, err
			}
		}

		if r, ok := stdlib.AsRange(fn, vals); ok {
			return r.All(), nil
		}

		v, err := p.CallValue(fn, vals, &call.Token)
		if err != nil {
			return nil, err
		}

		return itemsOf(s, v)
	}
}

// itemsOf snapshots arrays; objects yield their keys in sorted
// order, paired with values only when the loop binds both.
func itemsOf(s *ast.ForInStmt, v value.Value) (items, error) {
	switch v.Kind() {
	case value.Array:
		return indexed(slices.Clone(v.Arr())), nil
	case value.String:
		str := v.Str()

		return func(yield func(value.Value, value.Value) bool) {
			for i, ch := range helpers.Chars(str) {
				if !yield(value.NewNumber(float64(i)), value.NewString(ch)) {
					return
				}
			}
		}, nil
	case value.Object:
		obj, withValues := v.Obj(), s.Key != nil

		return func(yield func(value.Value, value.Value) bool) {
			for _, k := range slices.Sorted(maps.Keys(obj)) {
				key := value.NewString(k)

				val := key
				if withValues {
					val = obj[k]
				}

				if !yield(key, val) {
					return
				}
			}
		}, nil
	}

	return nil, errors.NewRuntimeError(s.Keyword, fmt.Sprintf(
		"Can only iterate over arrays, strings and objects, got %s.", helpers.TypeName(v)))
}

func indexed(vals []value.Value) items {
	return func(yield func(value.Value, value.Value) bool) {
		for i, v := range vals {
			if !yield(value.NewNumber(float64(i)), v) {
				return
			}
		}
	}
}
