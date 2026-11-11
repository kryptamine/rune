package helpers

import "unicode/utf8"

// Strings are indexed by character (code point); an invalid byte counts as
// one character and reads as U+FFFD.

// NextChar returns the first character of the non-empty s and its byte length.
func NextChar(s string) (string, int) {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && n == 1 {
		return string(utf8.RuneError), 1
	}

	return s[:n], n
}

// CharAt returns the i-th character of s; i must be in range.
func CharAt(s string, i int) string {
	ch, _ := NextChar(s[ByteOffset(s, i):])
	return ch
}

// ByteOffset returns the byte offset of the n-th character of s.
func ByteOffset(s string, n int) int {
	off := 0

	for ; n > 0 && off < len(s); n-- {
		_, size := utf8.DecodeRuneInString(s[off:])
		off += size
	}

	return off
}
