// Package format lays out rune source: indentation, spacing between
// tokens and blank lines. Line breaks, comments and string literals stay
// as written.
package format

import (
	"errors"
	"rune/pkg/ast"
	"rune/pkg/rune"
	"strings"
	"unicode/utf8"
)

const indentUnit = "    "

// atom is a token, or a whole string literal with its interpolations.
type atom struct {
	tok    ast.Token
	text   string
	role   role
	before gap
	// indent is how far the source line of the atom is indented.
	indent int
}

// role is what an atom is to the atoms around it, which spacing goes by.
type role int

const (
	other role = iota
	operand
	prefix
	postfix
	// path is a part of an import, after the keyword.
	path
)

func (a *atom) ended() bool { return a.role == operand || a.role == postfix }

// gap is what precedes an atom: comments, and the newlines before each.
type gap struct {
	comments []comment
	breaks   int
	spaced   bool
}

type comment struct {
	text   string
	breaks int
}

type frame struct {
	open   ast.TokenType
	indent int
	// pending counts the ? of conditionals waiting for their :.
	pending int
}

type printer struct {
	out    strings.Builder
	frames []frame
	prev   *atom
	line   int
	opened bool
	// trailing holds where each comment after code starts in out.
	trailing []int
}

// Source formats src, or returns the scan errors that stop it.
func Source(src []byte) ([]byte, []error) {
	tokens, errs := rune.Scan(src)
	if len(errs) > 0 {
		return nil, errs
	}

	atoms, tail := split(string(src), tokens)

	p := &printer{frames: []frame{{}}}
	for i := range atoms {
		var next *atom
		if i+1 < len(atoms) {
			next = &atoms[i+1]
		}
		p.print(&atoms[i], next)
	}
	p.comments(tail)

	out := []byte(align(p.out.String(), p.trailing))
	if len(out) > 0 {
		out = append(out, '\n')
	}

	if !sameTokens(tokens, out) {
		return nil, []error{errors.New("format: output scans to different tokens")}
	}

	return out, nil
}

// split makes atoms of tokens, and returns them with the comments after
// the last one.
func split(s string, tokens []ast.Token) ([]atom, []comment) {
	var atoms []atom
	from, ended, importing := 0, false, false

	for i := 0; i < len(tokens)-1; i++ {
		t := tokens[i]
		start := t.Offset

		if t.TokenType == ast.INTERPOLATION {
			i = rune.StringEnd(tokens, i)
			t.Lexeme = s[start : tokens[i].Offset+len(tokens[i].Lexeme)]
			t.TokenType = ast.STRING
		}

		a := atom{tok: t, text: t.Lexeme, before: parseGap(s[from:start]), indent: lineIndent(s, start)}
		a.role = classify(&a, ended)
		if importing {
			a.role = path
		}

		atoms = append(atoms, a)
		from = start + len(t.Lexeme)
		ended = a.ended()
		importing = (importing || t.TokenType == ast.IMPORT) && t.TokenType != ast.SEMICOLON
	}

	return atoms, parseGap(s[from:]).comments
}

// classify finds the role of a, given whether an operand ends before it.
func classify(a *atom, ended bool) role {
	switch a.tok.TokenType {
	case ast.IDENTIFIER:
		if ended && a.text == "step" {
			return other
		}
		return operand
	case ast.NUMBER, ast.STRING, ast.BYTES, ast.TRUE, ast.FALSE, ast.NIL,
		ast.RIGHT_PAREN, ast.RIGHT_BRACKET, ast.RIGHT_BRACE:
		return operand
	case ast.PLUS_PLUS, ast.MINUS_MINUS:
		if ended && a.before.breaks == 0 {
			return postfix
		}
		return prefix
	case ast.MINUS:
		if ended {
			return other
		}
		return prefix
	case ast.BANG, ast.TILDE, ast.DOT_DOT_DOT:
		return prefix
	}
	return other
}

func lineIndent(s string, at int) int {
	line := s[strings.LastIndexByte(s[:at], '\n')+1:]
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

func parseGap(s string) gap {
	g := gap{spaced: s != ""}
	for s != "" {
		switch {
		case s[0] == '\n':
			g.breaks++
			s = s[1:]
		case strings.HasPrefix(s, "//"):
			text, rest, _ := strings.Cut(s, "\n")
			g.comments = append(g.comments, comment{text: strings.TrimRight(text, " \t\r"), breaks: g.breaks})
			g.breaks = 1
			s = rest
		default:
			s = s[1:]
		}
	}
	return g
}

func (p *printer) top() *frame { return &p.frames[len(p.frames)-1] }

// inList reports whether the innermost bracket is a ( or a [, whose
// lines take no continuation indent.
func (p *printer) inList() bool {
	return p.top().open == ast.LEFT_PAREN || p.top().open == ast.LEFT_BRACKET
}

func (p *printer) print(a, next *atom) {
	t := a.tok.TokenType

	p.comments(a.before.comments)

	switch {
	case a.before.breaks > 0:
		p.newline(a.before.breaks > 1 && !closer(t), p.indentOf(a))
	case p.spaceBefore(a):
		p.out.WriteByte(' ')
	}

	text := a.text
	if t == ast.STRING {
		text = shift(text, p.line*len(indentUnit)-a.indent)
	}
	p.out.WriteString(text)
	p.opened = false

	p.track(a, next)
}

func (p *printer) comments(cs []comment) {
	for _, c := range cs {
		if c.breaks == 0 && p.out.Len() > 0 {
			p.trailing = append(p.trailing, p.out.Len())
		} else {
			p.newline(c.breaks > 1, p.top().indent)
		}
		p.out.WriteString(c.text)
		p.opened = false
	}
}

// newline starts a line at indent, after a blank one if asked for and
// not at the start of a block or of the file.
func (p *printer) newline(blank bool, indent int) {
	if p.out.Len() > 0 {
		p.out.WriteByte('\n')
		if blank && !p.opened {
			p.out.WriteByte('\n')
		}
	}
	p.out.WriteString(strings.Repeat(indentUnit, indent))
	p.line = indent
}

// indentOf is the indent of the line that a starts.
func (p *printer) indentOf(a *atom) int {
	switch {
	case closer(a.tok.TokenType):
		return max(p.top().indent-1, 0)
	case p.continues():
		return p.top().indent + 1
	}
	return p.top().indent
}

// continues reports whether a line goes on with the statement or item
// of the line before.
func (p *printer) continues() bool {
	if p.prev == nil || p.inList() {
		return false
	}
	switch p.prev.tok.TokenType {
	case ast.SEMICOLON, ast.LEFT_BRACE, ast.RIGHT_BRACE, ast.COMMA:
		return false
	}
	return true
}

// track keeps the brackets and conditionals open after a.
func (p *printer) track(a, next *atom) {
	switch t := a.tok.TokenType; {
	case t == ast.QUESTION:
		p.top().pending++
	case t == ast.COLON && p.top().pending > 0:
		p.top().pending--
	case opener(t):
		base := p.line
		if p.bodyBrace(t, next) {
			base = p.top().indent
		}
		p.frames = append(p.frames, frame{open: t, indent: base + 1})
		p.opened = true
	case closer(t) && len(p.frames) > 1:
		p.frames = p.frames[:len(p.frames)-1]
	}

	p.prev = a
}

// bodyBrace reports whether t ends a continued line and opens a body,
// as the { of if a \n && b { does, which indents from its statement
// rather than from the continued line.
func (p *printer) bodyBrace(t ast.TokenType, next *atom) bool {
	return t == ast.LEFT_BRACE && next != nil && next.before.breaks > 0 &&
		p.line > p.top().indent && !p.inList()
}

func (p *printer) spaceBefore(a *atom) bool {
	prev := p.prev
	if prev == nil {
		return false
	}
	pt, t := prev.tok.TokenType, a.tok.TokenType

	switch {
	case a.role == path:
		return pt == ast.IMPORT || isWord(a, "as") || isWord(prev, "as")
	case prev.role == prefix:
		return pt == ast.MINUS && (t == ast.MINUS || t == ast.MINUS_MINUS)
	case a.role == postfix:
		return false
	case t == ast.COMMA, t == ast.SEMICOLON, t == ast.RIGHT_PAREN, t == ast.RIGHT_BRACKET:
		return false
	case t == ast.RIGHT_BRACE:
		return pt != ast.LEFT_BRACE
	case pt == ast.LEFT_PAREN, pt == ast.LEFT_BRACKET:
		return false
	case t == ast.DOT, t == ast.QUESTION_DOT, pt == ast.DOT, pt == ast.QUESTION_DOT:
		return false
	case t == ast.DOT_DOT, t == ast.DOT_DOT_EQUAL:
		return keyword(prev)
	case pt == ast.DOT_DOT, pt == ast.DOT_DOT_EQUAL:
		return false
	case t == ast.LEFT_PAREN, t == ast.LEFT_BRACKET:
		if pt == ast.RIGHT_BRACE {
			return a.before.spaced
		}
		return !prev.ended()
	case t == ast.COLON:
		return p.top().pending > 0
	}
	return true
}

// shift moves the lines of a dedented """ string by delta columns, which
// keeps its value. A tab in the source indent counts as one column.
func shift(text string, delta int) string {
	body, ok := strings.CutPrefix(strings.TrimPrefix(text, "r"), `"""`)
	if delta == 0 || !ok || !strings.HasPrefix(strings.TrimLeft(body, "\r"), "\n") {
		return text
	}
	lines := strings.Split(text, "\n")
	if strings.Trim(lines[len(lines)-1], " \t") != `"""` {
		return text
	}
	for i := 1; i < len(lines); i++ {
		l := lines[i]
		switch {
		case delta > 0 && l != "":
			lines[i] = strings.Repeat(" ", delta) + l
		case delta < 0 && strings.HasPrefix(l, strings.Repeat(" ", -delta)):
			lines[i] = l[-delta:]
		case delta < 0 && strings.Trim(l, " \t\r") == "":
			lines[i] = ""
		case delta < 0:
			return text
		}
	}
	return strings.Join(lines, "\n")
}

// align lines up the comments after code on consecutive lines, one space
// after the longest code among them.
func align(s string, trailing []int) string {
	var b strings.Builder
	from := 0
	for i := 0; i < len(trailing); {
		j := i + 1
		for j < len(trailing) && strings.Count(s[trailing[j-1]:trailing[j]], "\n") == 1 {
			j++
		}

		width := 0
		for _, at := range trailing[i:j] {
			width = max(width, codeWidth(s, at))
		}
		for _, at := range trailing[i:j] {
			b.WriteString(s[from:at])
			b.WriteString(strings.Repeat(" ", width-codeWidth(s, at)+1))
			from = at
		}
		i = j
	}
	b.WriteString(s[from:])
	return b.String()
}

func codeWidth(s string, at int) int {
	return utf8.RuneCountInString(s[strings.LastIndexByte(s[:at], '\n')+1 : at])
}

func opener(t ast.TokenType) bool {
	return t == ast.LEFT_PAREN || t == ast.LEFT_BRACKET || t == ast.LEFT_BRACE
}

func closer(t ast.TokenType) bool {
	return t == ast.RIGHT_PAREN || t == ast.RIGHT_BRACKET || t == ast.RIGHT_BRACE
}

func keyword(a *atom) bool {
	_, ok := ast.Keywords[a.text]
	return ok
}

func isWord(a *atom, w string) bool {
	return a.tok.TokenType == ast.IDENTIFIER && a.text == w
}

// sameTokens reports whether out scans to want, as a check that
// formatting kept the program. Strings compare by value, as shift moves
// their text.
func sameTokens(want []ast.Token, out []byte) bool {
	got, errs := rune.Scan(out)
	if len(errs) > 0 || len(got) != len(want) {
		return false
	}
	for i, w := range want {
		g := got[i]
		text := w.TokenType == ast.STRING || w.TokenType == ast.INTERPOLATION || w.TokenType == ast.STRING_END
		if g.TokenType != w.TokenType || g.Literal != w.Literal || !text && g.Lexeme != w.Lexeme {
			return false
		}
	}
	return true
}
