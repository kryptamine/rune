package rune

import (
	rerrors "rune/pkg/errors"
	"strings"
)

// multiline dedents a """ string whose opening quotes end their line and
// whose closing ones start theirs: the newline after the opening quotes and
// the one before the closing quotes are dropped, and the whitespace before
// the closing quotes is taken off every line. A string scans into pieces,
// one token each; starts holds the offsets in a piece's text where a line
// of the source begins.
type multiline struct {
	active bool
	starts []int
	pieces []multilinePiece
}

type multilinePiece struct {
	tok    int
	starts []int
}

// newMultiline is called after the opening quotes; for a string that is
// not triple-quoted, or whose quotes do not end their line, it does nothing.
func newMultiline(s *Scanner, triple bool) *multiline {
	m := &multiline{}
	rest := s.source[s.current:]

	switch {
	case !triple:
		return m
	case strings.HasPrefix(rest, "\n"):
		s.current++
	case strings.HasPrefix(rest, "\r\n"):
		s.current += 2
	default:
		return m
	}

	s.newline()
	m.active = true
	m.starts = []int{0}

	return m
}

// line records that a line of the source begins at off in the current piece.
func (m *multiline) line(off int) {
	if m.active {
		m.starts = append(m.starts, off)
	}
}

// piece is called after each piece's token is added.
func (m *multiline) piece(s *Scanner) {
	if m.active {
		m.pieces = append(m.pieces, multilinePiece{tok: len(s.tokens) - 1, starts: m.starts})
		m.starts = nil
	}
}

func (m *multiline) dedent(s *Scanner, line, col int) {
	if !m.active {
		return
	}

	last := &m.pieces[len(m.pieces)-1]
	if len(last.starts) == 0 {
		return
	}

	lit := &s.tokens[last.tok].Literal
	o := last.starts[len(last.starts)-1]
	indent := (*lit)[o:]

	if strings.Trim(indent, " \t") != "" {
		return
	}

	*lit = strings.TrimSuffix((*lit)[:max(o-1, 0)], "\r")
	last.starts = last.starts[:len(last.starts)-1]

	for i, p := range m.pieces {
		lit := &s.tokens[p.tok].Literal
		v := *lit

		for j := len(p.starts) - 1; j >= 0; j-- {
			off := p.starts[j]
			rest := v[off:]

			if strings.HasPrefix(rest, indent) {
				v = v[:off] + rest[len(indent):]
				continue
			}

			text, _, ended := strings.Cut(rest, "\n")
			if strings.Trim(text, " \t\r") != "" || !ended && i < len(m.pieces)-1 {
				s.errors = append(s.errors, rerrors.NewScanError(s.file, line, col, "Insufficient indentation in multi-line string."))
				return
			}

			v = v[:off] + strings.TrimLeft(rest, " \t")
		}

		*lit = v
	}
}
