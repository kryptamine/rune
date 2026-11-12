package value

import (
	"cmp"
	"math"
)

// AddInt, SubInt and MulInt report false on overflow.
func AddInt(a, b int64) (int64, bool) {
	s := a + b
	return s, (s^a)&(s^b) >= 0
}

func SubInt(a, b int64) (int64, bool) {
	d := a - b
	return d, (a^b)&(a^d) >= 0
}

func MulInt(a, b int64) (int64, bool) {
	if a == int64(int32(a)) && b == int64(int32(b)) {
		return a * b, true
	}

	if (a == -1 && b == math.MinInt64) || (b == -1 && a == math.MinInt64) {
		return 0, false
	}

	p := a * b

	return p, a == 0 || p/a == b
}

// ToInt returns a number with an integer value as an int64.
func ToInt(v Value) (int64, bool) {
	if v.IsInt() {
		return v.Int(), true
	}

	if !v.IsFloat() {
		return 0, false
	}

	return FloatToInt(v.Float())
}

// FloatToInt converts f if it is an integer that int64 holds.
func FloatToInt(f float64) (int64, bool) {
	if f != math.Trunc(f) || f < -0x1p63 || f >= 0x1p63 {
		return 0, false
	}

	return int64(f), true
}

// CompareNum compares numbers exactly, an integer and a float included;
// ok is false if either is NaN.
func CompareNum(a, b Value) (c int, ok bool) {
	switch {
	case a.IsInt() && b.IsInt():
		return cmp.Compare(a.Int(), b.Int()), true
	case a.IsInt():
		c, ok = compareIntFloat(a.Int(), b.Float())
		return c, ok
	case b.IsInt():
		c, ok = compareIntFloat(b.Int(), a.Float())
		return -c, ok
	}

	l, r := a.Float(), b.Float()
	if l != l || r != r {
		return 0, false
	}

	return cmp.Compare(l, r), true
}

func compareIntFloat(i int64, f float64) (int, bool) {
	switch {
	case f != f:
		return 0, false
	case f >= 0x1p63:
		return -1, true
	case f < -0x1p63:
		return 1, true
	}

	t := math.Trunc(f)
	if c := cmp.Compare(i, int64(t)); c != 0 {
		return c, true
	}

	return cmp.Compare(t, f), true
}

// NumEqual is == on numbers.
func NumEqual(a, b Value) bool {
	if a.IsInt() && b.IsInt() {
		return a.n == b.n
	}

	c, ok := CompareNum(a, b)

	return ok && c == 0
}

// Arith is an arithmetic operator; Bit a bitwise one.
type Arith uint8

const (
	OpAdd Arith = iota
	OpSub
	OpMul
	OpDiv
	OpMod
)

type Bit uint8

const (
	BitAnd Bit = iota
	BitOr
	BitXor
	BitShl
	BitShr
)

// IntArith applies op to integers; ok is false on overflow and for % 0.
// / gives a float.
func IntArith(op Arith, a, b int64) (v Value, ok bool) {
	var n int64

	switch op {
	case OpAdd:
		n, ok = AddInt(a, b)
	case OpSub:
		n, ok = SubInt(a, b)
	case OpMul:
		n, ok = MulInt(a, b)
	case OpDiv:
		return NewNumber(float64(a) / float64(b)), true
	default:
		if b == 0 {
			return NilValue, false
		}

		n, ok = a%b, true
	}

	return NewInt(n), ok
}

func FloatArith(op Arith, a, b float64) Value {
	switch op {
	case OpAdd:
		return NewNumber(a + b)
	case OpSub:
		return NewNumber(a - b)
	case OpMul:
		return NewNumber(a * b)
	case OpDiv:
		return NewNumber(a / b)
	default:
		return NewNumber(FloatMod(a, b))
	}
}

// Bitwise applies op to integers; ok is false for a negative shift.
func Bitwise(op Bit, a, b int64) (Value, bool) {
	switch op {
	case BitAnd:
		return NewInt(a & b), true
	case BitOr:
		return NewInt(a | b), true
	case BitXor:
		return NewInt(a ^ b), true
	case BitShl:
		return NewInt(a << uint64(b)), b >= 0
	default:
		return NewInt(a >> uint64(b)), b >= 0
	}
}

const maxExactInt = 1 << 53

// FloatMod matches math.Mod, including the sign of zero results.
func FloatMod(l, r float64) float64 {
	if m, ok := ExactMod(l, r); ok {
		return m
	}

	return math.Mod(l, r)
}

// ExactMod is FloatMod for integers that floats hold exactly.
func ExactMod(l, r float64) (float64, bool) {
	if l <= -maxExactInt || l >= maxExactInt || r <= -maxExactInt || r >= maxExactInt || r == 0 {
		return 0, false
	}

	li, ri := int64(l), int64(r)
	if float64(li) != l || float64(ri) != r {
		return 0, false
	}

	return math.Copysign(float64(li%ri), l), true
}
