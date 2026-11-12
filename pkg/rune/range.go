package rune

import (
	"fmt"
	"math"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
)

// numRange is a range of integers if start, end and step all are, and
// otherwise of floats: rangeOf makes all three floats then.
type numRange struct {
	start, end, step value.Value
	inclusive        bool
}

// at returns the i-th number and whether it is in range.
func (r numRange) at(i int64) (value.Value, bool) {
	if r.start.IsInt() {
		x, ok := intRangeAt(r.start.Int(), r.end.Int(), r.step.Int(), i, r.inclusive)
		return value.NewInt(x), ok
	}

	x, ok := floatRangeAt(r.start.Float(), r.end.Float(), r.step.Float(), float64(i), r.inclusive)

	return value.NewNumber(x), ok
}

// intRangeAt is at for integers; past int64 is past the end.
func intRangeAt(start, end, step, i int64, inclusive bool) (int64, bool) {
	d, ok := value.MulInt(i, step)
	if !ok {
		return 0, false
	}

	x, ok := value.AddInt(start, d)

	switch {
	case !ok:
		return 0, false
	case x == end:
		return x, inclusive
	case step > 0:
		return x, x < end
	default:
		return x, x > end
	}
}

// floatRangeAt is at for floats: start + i*step avoids drift with
// fractional steps, and an inclusive end within rounding error is
// returned exactly.
func floatRangeAt(start, end, step, i float64, inclusive bool) (float64, bool) {
	x := start + i*step
	// remaining is how many steps are left to end; negative is past it.
	remaining := (end - x) / step

	if inclusive && math.Abs(remaining) < 1e-9 {
		return end, true
	}

	return x, remaining > 0
}

var rangeParts = [...]string{"start", "end", "step"}

// rangeOf checks the start, end and step of a range in parts, and leaves
// them there as the range has them, for opForRangeNext to read.
func rangeOf(parts []value.Value, inclusive bool, tok *ast.Token) (numRange, error) {
	for i, v := range parts[:3] {
		if !v.IsNumber() {
			return numRange{}, errors.NewRuntimeError(errors.TypeError, *tok, fmt.Sprintf(
				"Range %s must be a number, got %s.", rangeParts[i], helpers.TypeName(v)))
		}
	}

	if parts[2].Num() == 0 {
		return numRange{}, errors.NewRuntimeError(errors.ValueError, *tok, "Range step must not be zero.")
	}

	if !parts[0].IsInt() || !parts[1].IsInt() || !parts[2].IsInt() {
		for i := range 3 {
			parts[i] = value.NewNumber(parts[i].Num())
		}
	}

	return numRange{start: parts[0], end: parts[1], step: parts[2], inclusive: inclusive}, nil
}

// maxRangeArray caps the length of a range made into an array.
const maxRangeArray = 1 << 24

// size bounds how many numbers the range has: at most one more than the
// steps from start to end, and 0 for one that is empty or NaN.
func (r numRange) size() float64 {
	n := (r.end.Num() - r.start.Num()) / r.step.Num()
	if !(n >= 0) {
		return 0
	}

	return math.Floor(n) + 1
}

func (r numRange) array(tok *ast.Token) (value.Value, error) {
	n := r.size()
	if n > maxRangeArray {
		return value.NilValue, errors.NewRuntimeError(errors.ValueError, *tok, "Range is too large to make an array.")
	}

	out := make([]value.Value, 0, int(n))

	for i := int64(0); ; i++ {
		x, ok := r.at(i)
		if !ok {
			return value.NewArray(out), nil
		}

		out = append(out, x)
	}
}
