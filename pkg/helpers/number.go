package helpers

import "strconv"

// NumberEnd returns where the number literal whose digits start at s[i]
// ends: digits, then optionally "." and digits, then an exponent.
func NumberEnd(s string, i int) int {
	digits := func(i int) int {
		for i < len(s) && IsDigit(rune(s[i])) {
			i++
		}

		return i
	}

	i = digits(i)

	if i+1 < len(s) && s[i] == '.' && IsDigit(rune(s[i+1])) {
		i = digits(i + 1)
	}

	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		at := i + 1
		if at < len(s) && (s[at] == '+' || s[at] == '-') {
			at++
		}

		if at < len(s) && IsDigit(rune(s[at])) {
			i = digits(at)
		}
	}

	return i
}

// ParseNumber parses s as a number literal, optionally negated.
func ParseNumber(s string) (float64, bool) {
	digits := s
	if len(digits) > 0 && digits[0] == '-' {
		digits = digits[1:]
	}

	if digits == "" || !IsDigit(rune(digits[0])) || NumberEnd(digits, 0) != len(digits) {
		return 0, false
	}

	n, err := strconv.ParseFloat(s, 64)

	return n, err == nil
}

// AppendFixed appends n with digits decimals, or as few as it needs if digits is -1.
func AppendFixed(buf []byte, n float64, digits int) []byte {
	if n == 0 {
		n = 0 // -0 prints as 0
	}

	return strconv.AppendFloat(buf, n, 'f', digits, 64)
}
