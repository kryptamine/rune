package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/environment"
	"rune/pkg/errors"
	"rune/pkg/stdlib"
	"rune/pkg/value"
)

func compileCall(n *ast.CallExpr) evalFn {
	callee := compileExpr(n.Callee)
	args := make([]evalFn, len(n.Args))

	for i, arg := range n.Args {
		args[i] = compileExpr(arg)
	}

	token := &n.Token

	return func(p *Interpreter) (value.Value, error) {
		fn, err := callee(p)
		if err != nil {
			return value.NilValue, err
		}

		// Fast path: bind arguments straight into the callee's scope.
		if c, ok := fn.Fn().(*Closure); ok && fn.Kind() == value.Function && len(args) == c.fn.arity {
			env := p.newEnv(c.fn.pooled, c.env, c.fn.envSize)

			for i, arg := range args {
				v, err := arg(p)
				if err != nil {
					p.release(c.fn, env)
					return value.NilValue, err
				}

				env.Set(i, v)
			}

			return p.enter(c.fn, env, token)
		}

		base := len(p.args)

		for _, arg := range args {
			v, err := arg(p)
			if err != nil {
				p.popArgs(base)
				return value.NilValue, err
			}

			p.args = append(p.args, v)
		}

		v, err := p.CallValue(fn, p.args[base:len(p.args):len(p.args)], token)
		p.popArgs(base)

		return v, err
	}
}

func (p *Interpreter) popArgs(base int) {
	clear(p.args[base:])
	p.args = p.args[:base]
}

// CallValue calls fn with args; fn does not keep args.
func (p *Interpreter) CallValue(fn value.Value, args []value.Value, token *ast.Token) (value.Value, error) {
	if fn.Kind() == value.Function {
		switch f := fn.Fn().(type) {
		case *Closure:
			if len(args) != f.fn.arity {
				return value.NilValue, errors.NewRuntimeError(*token, fmt.Sprintf("Expected %d arguments but got %d.", f.fn.arity, len(args)))
			}

			env := p.newEnv(f.fn.pooled, f.env, f.fn.envSize)
			copy(env.Values(), args)

			return p.enter(f.fn, env, token)
		case *stdlib.Native:
			return p.frames.Call(f, p, args, token)
		}
	}

	return value.NilValue, errors.NewRuntimeError(*token, "Can only call functions.")
}

// enter runs fn's body in env, its freshly bound scope.
func (p *Interpreter) enter(fn *function, env *environment.Environment, token *ast.Token) (value.Value, error) {
	if p.recursionDepth >= p.maxRecursion {
		p.release(fn, env)
		return value.NilValue, errors.NewRuntimeError(*token, "Stack overflow.")
	}

	p.recursionDepth++
	err := p.runIn(fn.body, env, fn.pooled)
	p.recursionDepth--

	if err != errReturn {
		if err != nil {
			err = errors.WithFrame(err, errors.Frame{Name: fn.displayName(), Line: token.Line})
		}

		return value.NilValue, err
	}

	result := p.returnValue
	p.returnValue = value.NilValue

	return result, nil
}

func (p *Interpreter) release(fn *function, env *environment.Environment) {
	if fn.pooled {
		p.pool.Put(env)
	}
}
