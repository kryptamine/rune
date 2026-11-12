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
	unaryMath("math.floor", math.Floor),
	unaryMath("math.ceil", math.Ceil),
	unaryMath("math.round", math.Round),
	unaryMath("math.abs", math.Abs),
	unaryMath("math.sqrt", math.Sqrt),
	extremum("math.min", math.Min),
	extremum("math.max", math.Max),
	{"math.pow", 2, 2, func(c *call) value.Value {
		return value.NewNumber(math.Pow(c.num(0), c.num(1)))
	}},
	{"math.random", 0, 0, func(*call) value.Value {
		return value.NewNumber(rand.Float64())
	}},
}
