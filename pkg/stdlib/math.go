package stdlib

import (
	"math"
	"math/rand/v2"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
)

func unaryMath(name string, f func(float64) float64) *Native {
	return &Native{name, 1, 1, func(c *call) value.Value {
		return value.NewNumber(f(c.num(0)))
	}}
}

// rounding returns an integer as is, and a float rounded by f as an
// integer if one holds it.
func rounding(name string, f func(float64) float64) *Native {
	return &Native{name, 1, 1, func(c *call) value.Value {
		if v := c.args[0]; v.IsInt() {
			return v
		}

		x := f(c.num(0))
		if n, ok := value.FloatToInt(x); ok {
			return value.NewInt(n)
		}

		return value.NewNumber(x)
	}}
}

func extremum(name string, better int) *Native {
	return &Native{name, 1, -1, func(c *call) value.Value {
		best := c.number(0)

		for i := 1; i < len(c.args) && c.ok(); i++ {
			v := c.number(i)

			cmp, ok := value.CompareNum(v, best)
			if !ok {
				return value.NewNumber(math.NaN())
			}

			if cmp == better {
				best = v
			}
		}

		return best
	}}
}

var mathNatives = []*Native{
	rounding("math.floor", math.Floor),
	rounding("math.ceil", math.Ceil),
	rounding("math.round", math.Round),
	{"math.abs", 1, 1, func(c *call) value.Value {
		v := c.args[0]
		if !v.IsInt() {
			return value.NewNumber(math.Abs(c.num(0)))
		}

		if n := v.Int(); n >= 0 {
			return v
		} else if n == math.MinInt64 {
			return c.fail(errors.ValueError, "integer overflow.")
		} else {
			return value.NewInt(-n)
		}
	}},
	unaryMath("math.sqrt", math.Sqrt),
	extremum("math.min", -1),
	extremum("math.max", 1),
	{"math.pow", 2, 2, func(c *call) value.Value {
		return value.NewNumber(math.Pow(c.num(0), c.num(1)))
	}},
	{"math.idiv", 2, 2, func(c *call) value.Value {
		a, b := c.int(0), c.int(1)
		switch {
		case !c.ok():
			return value.NilValue
		case b == 0:
			return c.fail(errors.ValueError, "integer division by zero.")
		case a == math.MinInt64 && b == -1:
			return c.fail(errors.ValueError, "integer overflow.")
		}

		return value.NewInt(int64(a / b))
	}},
	{"math.isInt", 1, 1, func(c *call) value.Value {
		return value.NewBool(c.args[0].IsInt())
	}},
	{"math.random", 0, 0, func(*call) value.Value {
		return value.NewNumber(rand.Float64())
	}},
}

var numberMethods = []*Native{
	{"number.fixed", 2, 2, func(c *call) value.Value {
		n, digits := c.num(0), c.int(1)
		if !c.ok() {
			return value.NilValue
		}

		if digits < 0 || digits > 100 {
			return c.fail(errors.ValueError, "expects 0 to 100 digits, got %d.", digits)
		}

		return value.NewString(string(helpers.AppendFixed(nil, n, digits)))
	}},
}
