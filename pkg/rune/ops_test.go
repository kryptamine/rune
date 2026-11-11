package rune

import (
	"math"
	"testing"
)

func TestModMatchesMathMod(t *testing.T) {
	vals := []float64{0, math.Copysign(0, -1), 1, -1, 2, -2, 3, 7, -7, 10, -10, 2.5, -2.5,
		1e15, -1e15, 1 << 53, 1e300, math.Inf(1), math.Inf(-1), math.NaN()}

	for _, l := range vals {
		for _, r := range vals {
			got, want := mod(l, r), math.Mod(l, r)
			same := got == want && math.Signbit(got) == math.Signbit(want) || math.IsNaN(got) && math.IsNaN(want)

			if !same {
				t.Errorf("mod(%v, %v) = %v, want %v", l, r, got, want)
			}
		}
	}
}
