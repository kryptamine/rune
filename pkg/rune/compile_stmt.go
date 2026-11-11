package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
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
		}
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
