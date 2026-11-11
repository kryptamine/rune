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

var rangeParts = [...]string{"start", "end", "step"}

// rangeOf checks the start, end and step of a range.
func rangeOf(parts []value.Value, inclusive bool, tok *ast.Token) (numRange, error) {
	for i, v := range parts[:3] {
		if !v.IsNumber() {
			return numRange{}, errors.NewRuntimeError(*tok, fmt.Sprintf(
				"Range %s must be a number, got %s.", rangeParts[i], helpers.TypeName(v)))
		}
	}

	if parts[2].Num() == 0 {
		return numRange{}, errors.NewRuntimeError(*tok, "Range step must not be zero.")
	}

	return numRange{start: parts[0].Num(), end: parts[1].Num(), step: parts[2].Num(), inclusive: inclusive}, nil
}

func (r numRange) array() value.Value {
	var out []value.Value

	for i := 0.0; ; i++ {
		x, ok := r.at(i)
		if !ok {
			return value.NewArray(out)
		}

		out = append(out, value.NewNumber(x))
	}
}
