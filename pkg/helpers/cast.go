package helpers

import "strconv"

func ToFloat(val any) float64 {
	switch i2 := val.(type) {
	case float64:
		return i2
	case string:
		val, _ := strconv.ParseFloat(i2, 64)
		return val
	default:
		return 0.0
	}
}
