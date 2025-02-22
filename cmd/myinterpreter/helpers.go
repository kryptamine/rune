package main

import "strconv"

func isTruthy(val any) bool {
	if val == nil {
		return false
	}

	switch i2 := val.(type) {
	case bool:
		return i2
	case string:
		return true
	case float64:
		return i2 != 0.0
	default:
		return false
	}
}

func toFloat(val any) float64 {
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

func isEqual(left any, right any) bool {
	if left == nil && right == nil {
		return true
	}

	if left == nil {
		return false
	}

	return left == right
}

func isString(val any) bool {
	_, ok := val.(string)
	return ok
}

func isFloat(val any) bool {
	_, ok := val.(float64)
	return ok
}
