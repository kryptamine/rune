package rune

import (
	"fmt"
	"math"
	"rune/pkg/ast"
	rerrors "rune/pkg/errors"
	"rune/pkg/helpers"
	"strconv"
	"unicode/utf8"
)

type Scanner struct {
	source string
	file   int
	tokens []ast.Token
	errors []error

	start   int
	current int
	line    int

	lineStart int
	startLine int
	startCol  int

	// truncated is set when an unterminated string swallowed the rest of the source.
	truncated bool
}

func Scan(source []byte) ([]ast.Token, []error) {
	s := scan(source, 0)

	return s.tokens, s.errors
}

func scan(source []byte, file int) *Scanner {
	scanner := &Scanner{
		source:  string(source),
		file:    file,
		tokens:  []ast.Token{},
		errors:  []error{},
		current: 0,
		line:    1,
		start:   0,
	}

	scanner.scanTokens()

	return scanner
}

func (s *Scanner) scanTokens() {
	for !s.isAtEnd() {
		s.next()
	}

	eof := ast.NewToken(ast.EOF, "", "", s.line)
	eof.Col, eof.File = s.col(s.current), s.file
	s.tokens = append(s.tokens, eof)
}

func (s *Scanner) next() {
	s.start = s.current
	s.startLine, s.startCol = s.line, s.col(s.current)
	s.scanToken()
}

func (s *Scanner) col(offset int) int {
	return utf8.RuneCountInString(s.source[s.lineStart:offset]) + 1
}

func (s *Scanner) newline() {
	s.line++
	s.lineStart = s.current
}

func (s *Scanner) errorAt(msg string) {
	s.errors = append(s.errors, rerrors.NewScanError(s.file, s.startLine, s.startCol, msg))
}

func (s *Scanner) isAtEnd() bool {
	return s.current >= len(s.source)
}

func (s *Scanner) scanToken() {
	char := s.advance()

	switch char {
	case ':':
		s.addToken(ast.COLON)
		break
	case ';':
		s.addToken(ast.SEMICOLON)
		break
	case '(':
		s.addToken(ast.LEFT_PAREN)
		break
	case ')':
		s.addToken(ast.RIGHT_PAREN)
		break
	case '[':
		s.addToken(ast.LEFT_BRACKET)
		break
	case ']':
		s.addToken(ast.RIGHT_BRACKET)
		break
	case '{':
		s.addToken(ast.LEFT_BRACE)
		break
	case '}':
		s.addToken(ast.RIGHT_BRACE)
		break
	case ',':
		s.addToken(ast.COMMA)
		break
	case '.':
		switch {
		case !s.match('.'):
			s.addToken(ast.DOT)
		case s.match('.'):
			s.addToken(ast.DOT_DOT_DOT)
		case s.match('='):
			s.addToken(ast.DOT_DOT_EQUAL)
		default:
			s.addToken(ast.DOT_DOT)
		}
	case '?':
		switch {
		case s.match('.'):
			s.addToken(ast.QUESTION_DOT)
		case s.match('?'):
			s.addToken(ast.QUESTION_QUESTION)
		default:
			s.addToken(ast.QUESTION)
		}
	case '-':
		switch {
		case s.match('-'):
			s.addToken(ast.MINUS_MINUS)
		case s.match('='):
			s.addToken(ast.MINUS_EQUAL)
		default:
			s.addToken(ast.MINUS)
		}
	case '+':
		switch {
		case s.match('+'):
			s.addToken(ast.PLUS_PLUS)
		case s.match('='):
			s.addToken(ast.PLUS_EQUAL)
		default:
			s.addToken(ast.PLUS)
		}
	case '*':
		s.addToken(helpers.If(s.match('='), ast.STAR_EQUAL, ast.STAR))
	case '%':
		s.addToken(helpers.If(s.match('='), ast.PERCENT_EQUAL, ast.PERCENT))
		// Operators
	case '=':
		switch {
		case s.match('='):
			s.addToken(ast.EQUAL_EQUAL)
		case s.match('>'):
			s.addToken(ast.ARROW)
		default:
			s.addToken(ast.EQUAL)
		}
	case '!':
		s.addToken(helpers.If(s.match('='), ast.BANG_EQUAL, ast.BANG))
		break
	case '<':
		switch {
		case s.match('<'):
			s.addToken(helpers.If(s.match('='), ast.LESS_LESS_EQUAL, ast.LESS_LESS))
		case s.match('='):
			s.addToken(ast.LESS_EQUAL)
		default:
			s.addToken(ast.LESS)
		}
	case '>':
		switch {
		case s.match('>'):
			s.addToken(helpers.If(s.match('='), ast.GREATER_GREATER_EQUAL, ast.GREATER_GREATER))
		case s.match('='):
			s.addToken(ast.GREATER_EQUAL)
		default:
			s.addToken(ast.GREATER)
		}
	case '^':
		s.addToken(helpers.If(s.match('='), ast.CARET_EQUAL, ast.CARET))
	case '~':
		s.addToken(ast.TILDE)
	case '/':
		if s.match('/') {
			// A comment goes until the end of the line.
			for s.peek() != '\n' && !s.isAtEnd() {
				s.advance()
			}
		} else {
			s.addToken(helpers.If(s.match('='), ast.SLASH_EQUAL, ast.SLASH))
		}
	case '"':
		s.string(false)

		break
		// Ignore whitespace.
	case ' ':
	case '\r':
	case '\t':
		break
	case '\n':
		s.newline()
		break
	case '&', '|':
		// A doubled one is most likely meant as "and"/"or": say so, and
		// emit that operator so the parser has nothing to trip over.
		if s.match(char) {
			word := helpers.If(char == '&', "and", "or")
			s.errorAt(fmt.Sprintf("Unexpected '%c%c'; use '%s'.", char, char, word))
			s.addToken(helpers.If(char == '&', ast.AND, ast.OR))

			break
		}

		if char == '|' {
			s.addToken(helpers.If(s.match('='), ast.PIPE_EQUAL, ast.PIPE))
		} else {
			s.addToken(helpers.If(s.match('='), ast.AMP_EQUAL, ast.AMP))
		}
	default:
		if helpers.IsAlpha(char) {
			s.identifier()
		} else if helpers.IsDigit(char) {
			s.number()
		} else {
			// Report the whole (possibly multi-byte) character, not a single byte of it.
			r, size := utf8.DecodeRuneInString(s.source[s.start:])
			s.current = s.start + size
			s.errorAt(fmt.Sprintf("Unexpected character: %c", r))
		}
	}
}

func (s *Scanner) addToken(tokenType ast.TokenType) {
	s.addTokenWithLiteral(tokenType, "")
}

func (s *Scanner) addTokenWithLiteral(tokenType ast.TokenType, literal string) {
	lexeme := s.source[s.start:s.current]

	token := ast.NewToken(tokenType, lexeme, literal, s.startLine)
	token.Col, token.File = s.startCol, s.file
	s.tokens = append(s.tokens, token)
}

func (s *Scanner) currentChar() rune {
	return rune(s.source[s.current])
}

var escapes = map[byte]byte{
	'n':  '\n',
	't':  '\t',
	'r':  '\r',
	'"':  '"',
	'\\': '\\',
	'{':  '{',
	'}':  '}',
}

// string scans a string literal. Each "{expr}" in it ends a piece as an
// INTERPOLATION token, followed by the tokens of expr; the piece after
// the last one is a STRING_END. A bytes literal, after its b, takes no
// interpolation and takes \xNN escapes.
func (s *Scanner) string(isBytes bool) {
	var value []byte
	var badEscape string
	end := helpers.If(isBytes, ast.BYTES, ast.STRING)
	line, col := s.startLine, s.startCol

	piece := func(tokenType ast.TokenType) {
		s.addTokenWithLiteral(tokenType, string(value))
		value = value[:0]

		if badEscape != "" {
			s.errorAt(badEscape)
			badEscape = ""
		}
	}

	for s.peek() != '"' && !s.isAtEnd() {
		c := s.source[s.current]
		s.current++

		if c == '{' && !isBytes {
			piece(ast.INTERPOLATION)
			if !s.interpolated() {
				break
			}

			s.start, s.startLine, s.startCol = s.current, s.line, s.col(s.current)
			s.current++
			end = ast.STRING_END

			continue
		}

		if c == '\n' {
			s.newline()
		}

		if c != '\\' {
			value = append(value, c)
			continue
		}

		if s.isAtEnd() {
			break
		}

		// Keep scanning after a bad escape so the string still becomes a token.
		r, size := utf8.DecodeRuneInString(s.source[s.current:])
		s.current += size

		if e, ok := escapes[byte(r)]; ok && size == 1 {
			value = append(value, e)
		} else if b, ok := hexByte(s.source[s.current:]); ok && r == 'x' && isBytes {
			value = append(value, b)
			s.current += 2
		} else if badEscape == "" {
			badEscape = fmt.Sprintf("Invalid escape sequence '\\%c'.", r)
		}

		if r == '\n' {
			s.newline()
		}
	}

	if s.isAtEnd() {
		s.unterminated(line, col, helpers.If(isBytes, "Unterminated bytes.", "Unterminated string."))
		return
	}

	s.advance()
	piece(end)
}

// rawString scans r"...", taken as written: no escapes, no interpolation.
func (s *Scanner) rawString() {
	from := s.current

	for s.peek() != '"' && !s.isAtEnd() {
		c := s.source[s.current]
		s.current++

		if c == '\n' {
			s.newline()
		}
	}

	if s.isAtEnd() {
		s.unterminated(s.startLine, s.startCol, "Unterminated string.")
		return
	}

	text := s.source[from:s.current]
	s.advance()
	s.addTokenWithLiteral(ast.STRING, text)
}

// unterminated reports a literal that started at line, col and ran to the end.
func (s *Scanner) unterminated(line, col int, msg string) {
	s.truncated = true
	s.startLine, s.startCol = line, col
	s.errorAt(msg)
}

// hexByte reads the two hex digits that start s.
func hexByte(s string) (byte, bool) {
	if len(s) < 2 {
		return 0, false
	}

	n, err := strconv.ParseUint(s[:2], 16, 8)

	return byte(n), err == nil
}

// interpolated scans the tokens of an interpolated expression up to its
// closing '}', and reports whether there is one.
func (s *Scanner) interpolated() bool {
	depth := 0

	for !s.isAtEnd() {
		switch s.peek() {
		case '{':
			depth++
		case '}':
			if depth == 0 {
				return true
			}

			depth--
		}

		s.next()
	}

	return false
}

func (s *Scanner) number() {
	s.current = helpers.NumberEnd(s.source, s.start)
	lexeme := s.source[s.start:s.current]

	v, ok := helpers.ParseLiteral(lexeme)
	if !ok {
		s.errorAt("Integer literal is too large.")
	}

	if s.current < len(s.source) && s.isAlphaNumeric(rune(s.source[s.current])) {
		for !s.isAtEnd() && s.isAlphaNumeric(s.peek()) {
			s.advance()
		}

		s.errorAt("Invalid number literal '" + s.source[s.start:s.current] + "'.")
	}

	val := v.Num()
	literal := fmt.Sprintf("%g", val) // Keeps the precision for non-whole numbers

	if val == math.Trunc(val) {
		literal = fmt.Sprintf("%.1f", val) // Ensures 1234.0 for whole numbers
	}

	s.addTokenWithLiteral(ast.NUMBER, literal)
}

func (s *Scanner) identifier() {
	for s.isAlphaNumeric(s.peek()) {
		s.advance()
	}

	text := s.source[s.start:s.current]

	if text == "b" && s.peek() == '"' {
		s.advance()
		s.string(true)

		return
	}

	if text == "r" && s.peek() == '"' {
		s.advance()
		s.rawString()

		return
	}

	if val, ok := ast.Keywords[text]; ok {
		s.addToken(val)
	} else {
		s.addToken(ast.IDENTIFIER)
	}
}

func (s *Scanner) isAlphaNumeric(c rune) bool {
	return helpers.IsAlpha(c) || helpers.IsDigit(c)
}

func (s *Scanner) peek() rune {
	if s.isAtEnd() {
		return rune(0)
	}

	return s.currentChar()
}

func (s *Scanner) peekNext() rune {
	if s.current+1 >= len(s.source) {
		return rune(0)
	}

	return rune(s.source[s.current+1])
}

func (s *Scanner) advance() rune {
	char := s.currentChar()

	s.current++

	return char
}

func (s *Scanner) match(expected rune) bool {
	if s.isAtEnd() {
		return false
	}

	if s.currentChar() != expected {
		return false
	}

	s.current++

	return true
}
