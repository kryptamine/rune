package errors

import (
	"iter"
	"strings"
)

// DidYouMean suggests the candidate closest to name, a likely typo of it:
// one edit per three characters, ignoring case. It is empty when none is
// that close.
func DidYouMean(name string, candidates ...iter.Seq[string]) string {
	best, bestDist := "", len([]rune(name))/3+1

	for _, seq := range candidates {
		for c := range seq {
			if c == name {
				continue
			}

			d := distance(strings.ToLower(name), strings.ToLower(c))
			if d < bestDist || d == bestDist && c < best {
				best, bestDist = c, d
			}
		}
	}

	if best == "" {
		return ""
	}

	return "did you mean '" + best + "'?"
}

// distance is the Damerau–Levenshtein distance of a and b, which counts a
// swap of neighbours as one edit.
func distance(a, b string) int {
	s, t := []rune(a), []rune(b)
	rows := make([][]int, len(s)+1)

	for i := range rows {
		rows[i] = make([]int, len(t)+1)
		rows[i][0] = i
	}

	for j := range rows[0] {
		rows[0][j] = j
	}

	for i := 1; i <= len(s); i++ {
		for j := 1; j <= len(t); j++ {
			cost := 1
			if s[i-1] == t[j-1] {
				cost = 0
			}

			rows[i][j] = min(rows[i-1][j]+1, rows[i][j-1]+1, rows[i-1][j-1]+cost)

			if i > 1 && j > 1 && s[i-1] == t[j-2] && s[i-2] == t[j-1] {
				rows[i][j] = min(rows[i][j], rows[i-2][j-2]+1)
			}
		}
	}

	return rows[len(s)][len(t)]
}
