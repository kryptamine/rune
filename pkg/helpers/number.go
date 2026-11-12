package helpers

import (
	"rune/pkg/value"
	"strconv"
	"strings"
)

// NumberEnd returns where the number literal whose digits start at s[i]
// ends: digits, then optionally "." and digits, then an exponent; or a
// 0x, 0o or 0b prefix and digits of that base. A "_" may separate digits.
func NumberEnd(s string, i int) int {
	digits := func(i int, isDigit func(byte) bool) int {
		start := i
		for i < len(s) {
			switch {
			case isDigit(s[i]):
				i++
			case s[i] == '_' && i > start && i+1 < len(s) && isDigit(s[i+1]):
				i++
			default:
				return i
			}
		}

		return i
	}

	if i+2 < len(s) && s[i] == '0' {
		if isDigit := basePrefix(s[i+1]); isDigit != nil && isDigit(s[i+2]) {
			return digits(i+2, isDigit)
		}
	}

	i = digits(i, isDecimal)

	if i+1 < len(s) && s[i] == '.' && isDecimal(s[i+1]) {
		i = digits(i+1, isDecimal)
	}

	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		at := i + 1
		if at < len(s) && (s[at] == '+' || s[at] == '-') {
			at++
		}

		if at < len(s) && isDecimal(s[at]) {
			i = digits(at, isDecimal)
		}
	}

	return i
}

func isDecimal(c byte) bool { return c >= '0' && c <= '9' }

func isHex(c byte) bool { return isDecimal(c) || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' }

func basePrefix(c byte) func(byte) bool {
	switch c {
	case 'x', 'X':
		return isHex
	case 'o', 'O':
		return func(c byte) bool { return c >= '0' && c <= '7' }
	case 'b', 'B':
		return func(c byte) bool { return c == '0' || c == '1' }
	}

	return nil
}

// ParseLiteral parses a number literal the scanner found: an integer
// unless it has a fraction or an exponent. ok is false for an integer
// int64 cannot hold.
func ParseLiteral(lexeme string) (value.Value, bool) {
	s := strings.ReplaceAll(lexeme, "_", "")

	if strings.ContainsAny(s, ".eE") && !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		f, err := strconv.ParseFloat(s, 64)
		return value.NewNumber(f), err == nil
	}

	// A leading zero is decimal, not octal.
	base := 0
	if len(s) > 1 && isDecimal(s[1]) {
		base = 10
	}

	n, err := strconv.ParseInt(s, base, 64)

	return value.NewInt(n), err == nil
}

// ParseNumber parses s as a decimal number literal without "_",
// optionally negated:
// an integer, or a float if it has a fraction or an exponent or is too
// large for one.
func ParseNumber(s string) (value.Value, bool) {
	digits := strings.TrimPrefix(s, "-")

	if digits == "" || !isDecimal(digits[0]) || NumberEnd(digits, 0) != len(digits) ||
		basePrefix(at(digits, 1)) != nil || strings.ContainsRune(digits, '_') {
		return value.NilValue, false
	}

	if !strings.ContainsAny(digits, ".eE") {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return value.NewInt(n), true
		}
	}

	f, err := strconv.ParseFloat(s, 64)

	return value.NewNumber(f), err == nil
}

func at(s string, i int) byte {
	if i < len(s) {
		return s[i]
	}

	return 0
}

// AppendNumber appends a number as print shows it.
func AppendNumber(buf []byte, v value.Value) []byte {
	if v.IsInt() {
		return strconv.AppendInt(buf, v.Int(), 10)
	}

	return AppendFixed(buf, v.Float(), -1)
}

// AppendFixed appends n with digits decimals, or as few as it needs if digits is -1.
func AppendFixed(buf []byte, n float64, digits int) []byte {
	if n == 0 {
		n = 0 // -0 prints as 0
	}

	return strconv.AppendFloat(buf, n, 'f', digits, 64)
}
