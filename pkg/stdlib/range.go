package stdlib

import (
	"iter"
	"rune/pkg/value"
)

// Range is the sequence range(start, end, step) yields.
type Range struct{ Start, End, Step float64 }

// All yields (index, number) pairs.
func (r Range) All() iter.Seq2[value.Value, value.Value] {
	return func(yield func(value.Value, value.Value) bool) {
		i := 0.0

		for x := r.Start; r.Step > 0 && x < r.End || r.Step < 0 && x > r.End; x += r.Step {
			if !yield(value.NewNumber(i), value.NewNumber(x)) {
				return
			}

			i++
		}
	}
}

// AsRange reports the Range that fn(args...) would build, if fn is range and
// the call would succeed.
func AsRange(fn value.Value, args []value.Value) (Range, bool) {
	if fn.Kind() != value.Function || fn.Fn() != rangeNative || len(args) < rangeNative.min || len(args) > rangeNative.max {
		return Range{}, false
	}

	for _, a := range args {
		if !a.IsNumber() {
			return Range{}, false
		}
	}

	r := rangeOf(args)

	return r, r.Step != 0
}

// rangeOf expects 1 to 3 numbers: end, or start, end[, step].
func rangeOf(args []value.Value) Range {
	r := Range{End: args[0].Num(), Step: 1}

	if len(args) > 1 {
		r.Start, r.End = r.End, args[1].Num()
	}

	if len(args) > 2 {
		r.Step = args[2].Num()
	}

	return r
}

var rangeNative = &Native{"range", 1, 3, func(c *call) value.Value {
	for i := range c.args {
		c.num(i)
	}

	if !c.ok() {
		return value.NilValue
	}

	r := rangeOf(c.args)
	if r.Step == 0 {
		return c.fail("step must not be zero.")
	}

	var out []value.Value

	for _, x := range r.All() {
		out = append(out, x)
	}

	return value.NewArray(out)
}}
