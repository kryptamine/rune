package rune

import (
	"fmt"
	"maps"
	"rune/pkg/ast"
	"rune/pkg/environment"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
)

func compileStmts(stmts []ast.Stmt) execFn {
	fns := make([]execFn, len(stmts))
	for i, stmt := range stmts {
		fns[i] = compileStmt(stmt)
	}

	switch len(fns) {
	case 0:
		return nop
	case 1:
		return fns[0]
	case 2:
		a, b := fns[0], fns[1]

		return func(p *Interpreter) error {
			if err := a(p); err != nil {
				return err
			}

			return b(p)
		}
	}

	return func(p *Interpreter) error {
		for _, f := range fns {
			if err := f(p); err != nil {
				return err
			}
		}

		return nil
	}
}

func compileStmt(stmt ast.Stmt) execFn {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		e := compileExpr(s.Expr)

		return func(p *Interpreter) error {
			_, err := e(p)
			return err
		}
	case *ast.PrintStmt:
		return compilePrint(s)
	case *ast.VarStmt:
		return compileVar(s)
	case *ast.BlockStmt:
		return compileScope(compileStmts(s.Stmts), s.EnvSize, s.Pooled)
	case *ast.IfStmt:
		return compileIf(s)
	case *ast.WhileStmt:
		return compileWhile(s)
	case *ast.ForInStmt:
		return compileForIn(s)
	case *ast.FunctionStmt:
		return compileFunctionStmt(s)
	case *ast.ReturnStmt:
		return compileReturn(s)
	case *ast.TryStmt:
		return compileTry(s)
	case *ast.ThrowStmt:
		return compileThrow(s)
	case *ast.BreakStmt:
		return func(*Interpreter) error { return errBreak }
	case *ast.ContinueStmt:
		return func(*Interpreter) error { return errContinue }
	}

	panic(fmt.Sprintf("compile: unknown statement %T", stmt))
}

func compilePrint(s *ast.PrintStmt) execFn {
	e := compileExpr(s.Expr)

	return func(p *Interpreter) error {
		v, err := e(p)
		if err != nil {
			return err
		}

		p.line = append(helpers.AppendValue(p.line[:0], v), '\n')
		p.write(p.line)

		return nil
	}
}

func compileVar(s *ast.VarStmt) execFn {
	init := func(*Interpreter) (value.Value, error) { return value.NilValue, nil }
	if s.Initializer != nil {
		init = compileExpr(s.Initializer)
	}

	if s.Local {
		slot := s.Slot

		return func(p *Interpreter) error {
			v, err := init(p)
			if err != nil {
				return err
			}

			p.environment.Set(slot, v)

			return nil
		}
	}

	slot := globalSlot(s.Slot, s.Name.Lexeme)

	return func(p *Interpreter) error {
		v, err := init(p)
		if err != nil {
			return err
		}

		p.globals.Define(slot, v)

		return nil
	}
}

func compileScope(body execFn, size int, pooled bool) execFn {
	if size == 0 {
		return body
	}

	return func(p *Interpreter) error {
		return p.runIn(body, p.newEnv(pooled, p.environment, size), pooled)
	}
}

func compileIf(s *ast.IfStmt) execFn {
	cond := compileCond(s.Condition)
	then := compileStmt(s.Then)

	if s.El == nil {
		return func(p *Interpreter) error {
			c, err := cond(p)
			if err != nil {
				return err
			}

			if c {
				return then(p)
			}

			return nil
		}
	}

	el := compileStmt(s.El)

	return func(p *Interpreter) error {
		c, err := cond(p)
		if err != nil {
			return err
		}

		if c {
			return then(p)
		}

		return el(p)
	}
}

func compileWhile(s *ast.WhileStmt) execFn {
	cond := compileCond(s.Condition)
	body := compileStmt(s.Body)

	var incr evalFn
	if s.Increment != nil {
		incr = compileExpr(s.Increment)
	}

	return func(p *Interpreter) error {
		for {
			c, err := cond(p)
			if err != nil {
				return err
			}

			if !c {
				return nil
			}

			if err := body(p); err != nil && err != errContinue {
				if err == errBreak {
					return nil
				}

				return err
			}

			if incr != nil {
				if _, err := incr(p); err != nil {
					return err
				}
			}
		}
	}
}

// Elements are snapshotted when the loop starts.
func compileForIn(s *ast.ForInStmt) execFn {
	iterable := compileExpr(s.Iterable)
	body := compileStmt(s.Body)
	size, pooled, hasKey, keyword := s.EnvSize, s.Pooled, s.Key != nil, s.Keyword

	visit := func(p *Interpreter, env *environment.Environment, key, val value.Value) (bool, error) {
		if hasKey {
			env.Set(0, key)
			env.Set(1, val)
		} else {
			env.Set(0, val)
		}

		prev := p.environment
		p.environment = env
		err := body(p)
		p.environment = prev

		switch err {
		case nil, errContinue:
			return false, nil
		case errBreak:
			return true, nil
		}

		return true, err
	}

	return func(p *Interpreter) error {
		it, err := iterable(p)
		if err != nil {
			return err
		}

		var shared *environment.Environment
		if pooled {
			shared = p.pool.Get(p.environment, size)
			defer p.pool.Put(shared)
		}

		scope := func() *environment.Environment {
			if shared != nil {
				return shared
			}

			return environment.NewEnvironment(p.environment, size)
		}

		switch it.Kind() {
		case value.Array:
			for i, v := range slices.Clone(it.Arr()) {
				if stop, err := visit(p, scope(), value.NewNumber(float64(i)), v); stop {
					return err
				}
			}
		case value.String:
			for i, ch := range helpers.Chars(it.Str()) {
				if stop, err := visit(p, scope(), value.NewNumber(float64(i)), value.NewString(ch)); stop {
					return err
				}
			}
		case value.Object:
			obj := it.Obj()

			for _, k := range slices.Sorted(maps.Keys(obj)) {
				key := value.NewString(k)

				val := key
				if hasKey {
					val = obj[k]
				}

				if stop, err := visit(p, scope(), key, val); stop {
					return err
				}
			}
		default:
			return errors.NewRuntimeError(keyword, fmt.Sprintf(
				"Can only iterate over arrays, strings and objects, got %s.", helpers.TypeName(it)))
		}

		return nil
	}
}

func compileFunctionStmt(s *ast.FunctionStmt) execFn {
	fn := compileFunction(s)

	if s.Local {
		slot := s.Slot

		return func(p *Interpreter) error {
			p.environment.Set(slot, value.NewFunction(&Closure{fn: fn, env: p.environment}))
			return nil
		}
	}

	slot := globalSlot(s.Slot, s.Name.Lexeme)

	return func(p *Interpreter) error {
		p.globals.Define(slot, value.NewFunction(&Closure{fn: fn, env: p.environment}))
		return nil
	}
}

func compileReturn(s *ast.ReturnStmt) execFn {
	if s.Value == nil {
		return func(p *Interpreter) error {
			p.returnValue = value.NilValue
			return errReturn
		}
	}

	e := compileExpr(s.Value)

	return func(p *Interpreter) error {
		v, err := e(p)
		if err != nil {
			return err
		}

		p.returnValue = v

		return errReturn
	}
}

func compileTry(s *ast.TryStmt) execFn {
	body := compileStmt(s.Body)
	handler := compileStmts(s.Handler)
	size, pooled, hasParam := s.HandlerEnvSize, s.HandlerPooled, s.Param != nil

	return func(p *Interpreter) error {
		err := body(p)

		rtErr, ok := err.(errors.RuntimeError)
		if !ok {
			return err
		}

		if size == 0 {
			return handler(p)
		}

		env := p.newEnv(pooled, p.environment, size)

		if hasParam {
			env.Set(0, rtErr.Value())
		}

		return p.runIn(handler, env, pooled)
	}
}

func compileThrow(s *ast.ThrowStmt) execFn {
	e := compileExpr(s.Value)
	keyword := s.Keyword

	return func(p *Interpreter) error {
		v, err := e(p)
		if err != nil {
			return err
		}

		return errors.NewThrownError(keyword, v, helpers.Stringify(v))
	}
}
