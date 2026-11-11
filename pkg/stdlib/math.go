package stdlib

import (
	"math"
	"math/rand/v2"
	"rune/pkg/value"
)

func unaryMath(name string, f func(float64) float64) *Native {
	return &Native{name, 1, 1, func(c *call) value.Value {
		return value.NewNumber(f(c.num(0)))
	}}
}

func extremum(name string, pick func(a, b float64) float64) *Native {
	return &Native{name, 1, -1, func(c *call) value.Value {
		best := c.num(0)

		for i := 1; i < len(c.args); i++ {
			best = pick(best, c.num(i))
		}

		return value.NewNumber(best)
	}}
}

var mathNatives = []*Native{
	unaryMath("floor", math.Floor),
	unaryMath("ceil", math.Ceil),
	unaryMath("round", math.Round),
	unaryMath("abs", math.Abs),
	unaryMath("sqrt", math.Sqrt),
	extremum("min", math.Min),
	extremum("max", math.Max),
	{"pow", 2, 2, func(c *call) value.Value {
		return value.NewNumber(math.Pow(c.num(0), c.num(1)))
	}},
	{"random", 0, 0, func(*call) value.Value {
		return value.NewNumber(rand.Float64())
	}},
}
