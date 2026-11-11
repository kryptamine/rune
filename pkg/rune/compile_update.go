package rune

import (
	"fmt"

	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/value"
)

// typeErrFn reports an update of a non-number.
type typeErrFn func(old value.Value) error

func compileUpdate(n *ast.UpdateExpr) evalFn {
	op := n.Operator
	postfix := !n.Prefix
	delta := 1.0

	if op.TokenType == ast.MINUS_MINUS {
		delta = -1
	}

	typeErr := func(value.Value) error {
		return errors.NewRuntimeError(op, "Operand must be a number.")
	}

	switch t := n.Target.(type) {
	case *ast.VarExpr:
		return compileUpdateVar(t.Name, t.Depth, t.Slot, delta, postfix, typeErr)
	case *ast.IndexExpr:
		return compileUpdateIndex(t, delta, postfix, typeErr)
	}

	panic(fmt.Sprintf("compile: invalid update target %T", n.Target))
}

// compileIncrement handles x = x ± c (and x += c); nil otherwise.
func compileIncrement(n *ast.AssignExpr) evalFn {
	bin, ok := n.Value.(*ast.BinaryExpr)
	if !ok || (bin.Operator.TokenType != ast.PLUS && bin.Operator.TokenType != ast.MINUS) {
		return nil
	}

	target, ok := bin.Left.(*ast.VarExpr)
	if !ok || target.Name.Lexeme != n.Name.Lexeme || target.Depth != n.Depth || target.Slot != n.Slot {
		return nil
	}

	c, ok := literalOf(bin.Right)
	if !ok || !c.IsNumber() {
		return nil
	}

	delta := c.Num()
	if bin.Operator.TokenType == ast.MINUS {
		delta = -delta
	}

	op := &bin.Operator

	return compileUpdateVar(n.Name, n.Depth, n.Slot, delta, false, func(old value.Value) error {
		_, err := binaryOp(op, old, c)
		return err
	})
}

// compileUpdateVar adds delta to a numeric variable, yielding the old value
// if postfix and the new one otherwise.
func compileUpdateVar(name ast.Token, depth, slot int, delta float64, postfix bool, typeErr typeErrFn) evalFn {
	switch depth {
	case ast.Unresolved:
		slot := globalSlot(slot, name.Lexeme)

		return func(p *Interpreter) (value.Value, error) {
			old, err := p.globals.Get(slot, &name)
			if err != nil {
				return value.NilValue, err
			}

			if !old.IsNumber() {
				return value.NilValue, typeErr(old)
			}

			v := value.NewNumber(old.Num() + delta)
			if err := p.globals.Assign(slot, &name, v); err != nil || !postfix {
				return v, err
			}

			return old, nil
		}
	case 0:
		return func(p *Interpreter) (value.Value, error) {
			env := p.environment

			old := env.Get(slot)
			if !old.IsNumber() {
				return value.NilValue, typeErr(old)
			}

			v := value.NewNumber(old.Num() + delta)
			env.Set(slot, v)

			if postfix {
				return old, nil
			}

			return v, nil
		}
	}

	return func(p *Interpreter) (value.Value, error) {
		old := p.environment.GetAt(depth, slot)
		if !old.IsNumber() {
			return value.NilValue, typeErr(old)
		}

		v := value.NewNumber(old.Num() + delta)
		p.environment.AssignAt(depth, slot, v)

		if postfix {
			return old, nil
		}

		return v, nil
	}
}

func compileUpdateIndex(t *ast.IndexExpr, delta float64, postfix bool, typeErr typeErrFn) evalFn {
	target := compileExpr(t.Array)
	index := compileExpr(t.Index)
	token, property := &t.Token, t.Property

	return func(p *Interpreter) (value.Value, error) {
		tv, i, err := evalPlace(p, target, index, token, property)
		if err != nil {
			return value.NilValue, err
		}

		old, err := getIndex(token, tv, i)
		if err != nil {
			return value.NilValue, err
		}

		if !old.IsNumber() {
			return value.NilValue, typeErr(old)
		}

		v := value.NewNumber(old.Num() + delta)
		if err := setIndex(token, tv, i, v); err != nil || !postfix {
			return v, err
		}

		return old, nil
	}
}
