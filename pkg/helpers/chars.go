package helpers

import (
	"iter"
	"unicode/utf8"
)

// Strings are indexed by character (code point); an invalid byte counts as
// one character and reads as U+FFFD.

// Chars yields the characters of s with their indices.
func Chars(s string) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		for i, off := 0, 0; off < len(s); i++ {
			ch := char(s[off:])
			off += charLen(s[off:])

			if !yield(i, ch) {
				return
			}
		}
	}
}

// CharAt returns the i-th character of s; i must be in range.
func CharAt(s string, i int) string {
	return char(s[ByteOffset(s, i):])
}

// ByteOffset returns the byte offset of the n-th character of s.
func ByteOffset(s string, n int) int {
	off := 0

	for ; n > 0 && off < len(s); n-- {
		off += charLen(s[off:])
	}

	return off
}

func charLen(s string) int {
	_, n := utf8.DecodeRuneInString(s)
	return n
}

// char returns the first character of the non-empty s.
func char(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && n == 1 {
		return string(utf8.RuneError)
	}

	return s[:n]
}
