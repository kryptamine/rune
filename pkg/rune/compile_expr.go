package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
)

func compileExpr(expr ast.Expr) evalFn {
	switch n := expr.(type) {
	case *ast.LiteralExpr:
		return constant(n.Value)
	case *ast.GroupingExpr:
		return compileExpr(n.Expr)
	case *ast.VarExpr:
		return compileVarExpr(n)
	case *ast.AssignExpr:
		return compileAssign(n)
	case *ast.BinaryExpr:
		return compileBinary(n)
	case *ast.LogicalExpr:
		return compileLogical(n)
	case *ast.UnaryExpr:
		return compileUnary(n)
	case *ast.CallExpr:
		return compileCall(n)
	case *ast.ArrayExpr:
		return compileArray(n)
	case *ast.ObjectExpr:
		return compileObject(n)
	case *ast.IndexExpr:
		return compileIndex(n)
	case *ast.SetIndexExpr:
		return compileSetIndex(n)
	case *ast.FunctionExpr:
		fn := compileFunction(n.Decl)

		return func(p *Interpreter) (value.Value, error) {
			return value.NewFunction(&Closure{fn: fn, env: p.environment}), nil
		}
	}

	panic(fmt.Sprintf("compile: unknown expression %T", expr))
}

func compileVarExpr(n *ast.VarExpr) evalFn {
	slot := n.Slot

	switch n.Depth {
	case ast.Unresolved:
		slot := globalSlot(n.Slot, n.Name.Lexeme)
		name := &n.Name

		return func(p *Interpreter) (value.Value, error) {
			return p.globals.Get(slot, name)
		}
	case 0:
		return func(p *Interpreter) (value.Value, error) {
			return p.environment.Get(slot), nil
		}
	case 1:
		return func(p *Interpreter) (value.Value, error) {
			return p.environment.Enclosing().Get(slot), nil
		}
	}

	depth := n.Depth

	return func(p *Interpreter) (value.Value, error) {
		return p.environment.GetAt(depth, slot), nil
	}
}

func compileAssign(n *ast.AssignExpr) evalFn {
	if f := compileIncrement(n); f != nil {
		return f
	}

	val := compileExpr(n.Value)
	depth, slot := n.Depth, n.Slot

	switch depth {
	case ast.Unresolved:
		slot := globalSlot(n.Slot, n.Name.Lexeme)
		name := &n.Name

		return func(p *Interpreter) (value.Value, error) {
			v, err := val(p)
			if err != nil {
				return value.NilValue, err
			}

			return v, p.globals.Assign(slot, name, v)
		}
	case 0:
		return func(p *Interpreter) (value.Value, error) {
			v, err := val(p)
			if err != nil {
				return value.NilValue, err
			}

			p.environment.Set(slot, v)

			return v, nil
		}
	}

	return func(p *Interpreter) (value.Value, error) {
		v, err := val(p)
		if err != nil {
			return value.NilValue, err
		}

		p.environment.AssignAt(depth, slot, v)

		return v, nil
	}
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

	switch n.Depth {
	case ast.Unresolved:
		slot := globalSlot(n.Slot, n.Name.Lexeme)
		name := &n.Name

		return func(p *Interpreter) (value.Value, error) {
			old, err := p.globals.Get(slot, name)
			if err != nil {
				return value.NilValue, err
			}

			if !old.IsNumber() {
				_, err := binaryOp(op, old, c)
				return value.NilValue, err
			}

			v := value.NewNumber(old.Num() + delta)

			return v, p.globals.Assign(slot, name, v)
		}
	case 0:
		slot := n.Slot

		return func(p *Interpreter) (value.Value, error) {
			env := p.environment

			old := env.Get(slot)
			if !old.IsNumber() {
				_, err := binaryOp(op, old, c)
				return value.NilValue, err
			}

			v := value.NewNumber(old.Num() + delta)
			env.Set(slot, v)

			return v, nil
		}
	}

	return nil
}

func compileBinary(n *ast.BinaryExpr) evalFn {
	op := &n.Operator
	opType := op.TokenType

	lc, lConst := literalOf(n.Left)
	rc, rConst := literalOf(n.Right)

	// A fold that fails is left to fail at run time.
	if lConst && rConst {
		if v, err := binaryOp(op, lc, rc); err == nil {
			return constant(v)
		}
	}

	left := compileExpr(n.Left)

	if rConst && rc.IsNumber() {
		r := rc.Num()

		return func(p *Interpreter) (value.Value, error) {
			l, err := left(p)
			if err != nil {
				return value.NilValue, err
			}

			if l.IsNumber() {
				return numberOp(opType, l.Num(), r), nil
			}

			return binaryOp(op, l, rc)
		}
	}

	right := compileExpr(n.Right)

	return func(p *Interpreter) (value.Value, error) {
		l, err := left(p)
		if err != nil {
			return value.NilValue, err
		}

		r, err := right(p)
		if err != nil {
			return value.NilValue, err
		}

		if l.IsNumber() && r.IsNumber() {
			return numberOp(opType, l.Num(), r.Num()), nil
		}

		return binaryOp(op, l, r)
	}
}

func compileLogical(n *ast.LogicalExpr) evalFn {
	left := compileExpr(n.Left)
	right := compileExpr(n.Right)
	isOr := n.Op.TokenType == ast.OR

	return func(p *Interpreter) (value.Value, error) {
		l, err := left(p)
		if err != nil {
			return value.NilValue, err
		}

		if helpers.IsTruthy(l) == isOr {
			return l, nil
		}

		return right(p)
	}
}

func compileUnary(n *ast.UnaryExpr) evalFn {
	op := n.Operator

	if c, ok := literalOf(n.Right); ok && op.TokenType == ast.MINUS && c.IsNumber() {
		return constant(value.NewNumber(-c.Num()))
	}

	right := compileExpr(n.Right)

	if op.TokenType == ast.BANG {
		return func(p *Interpreter) (value.Value, error) {
			v, err := right(p)
			if err != nil {
				return value.NilValue, err
			}

			return value.NewBool(!helpers.IsTruthy(v)), nil
		}
	}

	return func(p *Interpreter) (value.Value, error) {
		v, err := right(p)
		if err != nil {
			return value.NilValue, err
		}

		if !v.IsNumber() {
			return value.NilValue, errors.NewRuntimeError(op, "Operand must be a number.")
		}

		return value.NewNumber(-v.Num()), nil
	}
}

func compileCond(expr ast.Expr) condFn {
	switch n := expr.(type) {
	case *ast.GroupingExpr:
		return compileCond(n.Expr)

	case *ast.BinaryExpr:
		if !isComparison(n.Operator.TokenType) {
			break
		}

		op := &n.Operator
		opType := op.TokenType
		left := compileExpr(n.Left)

		if rc, ok := literalOf(n.Right); ok && rc.IsNumber() {
			r := rc.Num()

			return func(p *Interpreter) (bool, error) {
				l, err := left(p)
				if err != nil {
					return false, err
				}

				if l.IsNumber() {
					return compare(opType, l.Num(), r), nil
				}

				v, err := binaryOp(op, l, rc)

				return v.Bool(), err
			}
		}

		right := compileExpr(n.Right)

		return func(p *Interpreter) (bool, error) {
			l, err := left(p)
			if err != nil {
				return false, err
			}

			r, err := right(p)
			if err != nil {
				return false, err
			}

			if l.IsNumber() && r.IsNumber() {
				return compare(opType, l.Num(), r.Num()), nil
			}

			v, err := binaryOp(op, l, r)

			return v.Bool(), err
		}

	case *ast.LogicalExpr:
		left, right := compileCond(n.Left), compileCond(n.Right)

		if n.Op.TokenType == ast.OR {
			return func(p *Interpreter) (bool, error) {
				if l, err := left(p); err != nil || l {
					return l, err
				}

				return right(p)
			}
		}

		return func(p *Interpreter) (bool, error) {
			if l, err := left(p); err != nil || !l {
				return l, err
			}

			return right(p)
		}

	case *ast.UnaryExpr:
		if n.Operator.TokenType != ast.BANG {
			break
		}

		inner := compileCond(n.Right)

		return func(p *Interpreter) (bool, error) {
			c, err := inner(p)
			return !c, err
		}
	}

	e := compileExpr(expr)

	return func(p *Interpreter) (bool, error) {
		v, err := e(p)
		if err != nil {
			return false, err
		}

		return helpers.IsTruthy(v), nil
	}
}
