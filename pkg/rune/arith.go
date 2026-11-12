package rune

import (
	"cmp"
	"math"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
)

var arithOps = map[ast.TokenType]value.Arith{
	ast.PLUS: value.OpAdd, ast.MINUS: value.OpSub, ast.STAR: value.OpMul, ast.SLASH: value.OpDiv, ast.PERCENT: value.OpMod,
}

var bitOps = map[ast.TokenType]value.Bit{
	ast.AMP: value.BitAnd, ast.PIPE: value.BitOr, ast.CARET: value.BitXor,
	ast.LESS_LESS: value.BitShl, ast.GREATER_GREATER: value.BitShr,
}

// binaryIns is the instruction applying the arithmetic or bit operator
// op to registers b and c.
func binaryIns(op ast.TokenType, a, b, c int32) ins {
	if x, ok := bitOps[op]; ok {
		return ins{op: opBitwise, x: uint8(x), a: a, b: b, c: c}
	}

	return ins{op: arithOp(arithOps[op], false), a: a, b: b, c: c}
}

// binaryOp applies a binary operator; operator also locates errors.
func binaryOp(operator *ast.Token, l, r value.Value) (value.Value, error) {
	op := operator.TokenType

	if a, ok := arithOps[op]; ok {
		return arith(operator, a, l, r)
	}

	if b, ok := bitOps[op]; ok {
		return bitwise(operator, b, l, r)
	}

	switch op {
	case ast.EQUAL_EQUAL:
		return value.NewBool(helpers.IsEqual(l, r)), nil
	case ast.BANG_EQUAL:
		return value.NewBool(!helpers.IsEqual(l, r)), nil
	}

	switch {
	case l.IsNumber() && r.IsNumber():
		c, ok := value.CompareNum(l, r)
		return value.NewBool(ok && compare(op, c, 0)), nil
	case l.IsString() && r.IsString():
		return value.NewBool(compare(op, l.Str(), r.Str())), nil
	case l.Kind() == value.Bytes && r.Kind() == value.Bytes:
		return value.NewBool(compare(op, string(l.Bytes()), string(r.Bytes()))), nil
	}

	return value.NilValue, notOrdered(operator)
}

// arith applies op: integers give integers, except by /, and an integer
// with a float gives a float; + also joins strings and bytes.
func arith(operator *ast.Token, op value.Arith, l, r value.Value) (value.Value, error) {
	switch {
	case l.IsInt() && r.IsInt():
		if v, ok := value.IntArith(op, l.Int(), r.Int()); ok {
			return v, nil
		}

		if op == value.OpMod {
			return value.NilValue, errors.NewRuntimeError(errors.ValueError, *operator, "Integer modulo by zero.")
		}

		return value.NilValue, overflow(operator)
	case l.IsNumber() && r.IsNumber():
		return value.FloatArith(op, l.Num(), r.Num()), nil
	case op != value.OpAdd:
		return value.NilValue, errors.NewRuntimeError(errors.TypeError, *operator, "Operands must be numbers.")
	case l.IsString() && r.IsString():
		return value.Concat(l, r), nil
	case l.Kind() == value.Bytes && r.Kind() == value.Bytes:
		b := l.Bytes()
		return value.NewBytes(append(b[:len(b):len(b)], r.Bytes()...)), nil
	}

	return value.NilValue, notOrdered(operator)
}

func notOrdered(operator *ast.Token) error {
	return errors.NewRuntimeError(errors.TypeError, *operator, "Operands must be two numbers, two strings or two bytes.")
}

// bitwise applies op to integers, or floats holding one.
func bitwise(operator *ast.Token, op value.Bit, l, r value.Value) (value.Value, error) {
	if !l.IsNumber() || !r.IsNumber() {
		return value.NilValue, errors.NewRuntimeError(errors.TypeError, *operator, "Operands must be numbers.")
	}

	a, aok := value.ToInt(l)
	b, bok := value.ToInt(r)

	if !aok || !bok {
		return value.NilValue, errors.NewRuntimeError(errors.TypeError, *operator, "Operands must be integers.")
	}

	v, ok := value.Bitwise(op, a, b)
	if !ok {
		return value.NilValue, errors.NewRuntimeError(errors.ValueError, *operator, "Shift count must not be negative.")
	}

	return v, nil
}

func overflow(tok *ast.Token) error {
	return errors.NewRuntimeError(errors.ValueError, *tok, "Integer overflow.")
}

// negate is -v; increment v + delta, for ++ and --.
func negate(tok *ast.Token, v value.Value) (value.Value, error) {
	switch {
	case v.IsInt():
		if v.Int() == math.MinInt64 {
			return value.NilValue, overflow(tok)
		}

		return value.NewInt(-v.Int()), nil
	case v.IsFloat():
		return value.NewNumber(-v.Float()), nil
	}

	return value.NilValue, errors.NewRuntimeError(errors.TypeError, *tok, "Operand must be a number.")
}

func increment(tok *ast.Token, v value.Value, delta int64) (value.Value, error) {
	if v.IsNumber() {
		return arith(tok, value.OpAdd, v, value.NewInt(delta))
	}

	return value.NilValue, errors.NewRuntimeError(errors.TypeError, *tok, "Operand must be a number.")
}

func bitNot(tok *ast.Token, v value.Value) (value.Value, error) {
	if n, ok := value.ToInt(v); ok {
		return value.NewInt(^n), nil
	}

	return value.NilValue, errors.NewRuntimeError(errors.TypeError, *tok, "Operand must be an integer.")
}

// compare applies one of < <= > >=.
func compare[T cmp.Ordered](op ast.TokenType, l, r T) bool {
	switch op {
	case ast.LESS:
		return l < r
	case ast.LESS_EQUAL:
		return l <= r
	case ast.GREATER:
		return l > r
	default:
		return l >= r
	}
}
