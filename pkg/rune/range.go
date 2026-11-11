package rune

import (
	"fmt"
	"math"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
)

type numRange struct {
	start, end, step float64
	inclusive        bool
}

// at returns the i-th number and whether it is in range. start + i*step
// avoids drift with fractional steps; an inclusive end within rounding
// error is returned exactly.
func (r numRange) at(i float64) (float64, bool) {
	x := r.start + i*r.step
	// remaining is how many steps are left to end; negative is past it.
	remaining := (r.end - x) / r.step

	if r.inclusive && math.Abs(remaining) < 1e-9 {
		return r.end, true
	}

	return x, remaining > 0
}

func compileRangeExpr(n *ast.RangeExpr) evalFn {
	bounds := compileRange(n)

	return func(p *Interpreter) (value.Value, error) {
		r, err := bounds(p)
		if err != nil {
			return value.NilValue, err
		}

		var out []value.Value

		c := cursor{mode: rangeCursor, r: r}

		for _, x, ok := c.next(); ok; _, x, ok = c.next() {
			out = append(out, x)
		}

		return value.NewArray(out), nil
	}
}

var rangeParts = [...]string{"start", "end", "step"}

func compileRange(n *ast.RangeExpr) func(*Interpreter) (numRange, error) {
	parts := [...]evalFn{compileExpr(n.Start), compileExpr(n.End), constant(value.NewNumber(1))}

	if n.Step != nil {
		parts[2] = compileExpr(n.Step)
	}

	return func(p *Interpreter) (numRange, error) {
		var nums [3]float64

		for i, part := range parts {
			v, err := part(p)
			if err != nil {
				return numRange{}, err
			}

			if !v.IsNumber() {
				return numRange{}, errors.NewRuntimeError(n.Operator, fmt.Sprintf(
					"Range %s must be a number, got %s.", rangeParts[i], helpers.TypeName(v)))
			}

			nums[i] = v.Num()
		}

		if nums[2] == 0 {
			return numRange{}, errors.NewRuntimeError(n.Operator, "Range step must not be zero.")
		}

		return numRange{start: nums[0], end: nums[1], step: nums[2], inclusive: n.Inclusive}, nil
	}
}
