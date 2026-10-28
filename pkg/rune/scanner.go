package rune

import (
	"errors"
	"fmt"
	"math"
	"rune/pkg/ast"
	"rune/pkg/helpers"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Scanner struct {
	source string
	tokens []ast.Token
	errors []error

	start   int
	current int
	line    int
}

func Scan(source []byte) ([]ast.Token, []error) {
	scanner := &Scanner{
		source:  string(source),
		tokens:  []ast.Token{},
		errors:  []error{},
		current: 0,
		line:    1,
		start:   0,
	}

	return scanner.scanTokens(), scanner.errors
}

func (s *Scanner) scanTokens() []ast.Token {
	for !s.isAtEnd() {
		s.start = s.current
		s.scanToken()
	}

	s.tokens = append(
		s.tokens,
		ast.NewToken(ast.EOF, "", "", s.line),
	)

	return s.tokens
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
		s.addToken(ast.DOT)
		break
	case '-':
		s.addToken(ast.MINUS)
		break
	case '+':
		s.addToken(ast.PLUS)
		break
	case '*':
		s.addToken(ast.STAR)
		break
		// Operators
	case '=':
		s.addToken(helpers.If(s.match('='), ast.EQUAL_EQUAL, ast.EQUAL))
		break
	case '!':
		s.addToken(helpers.If(s.match('='), ast.BANG_EQUAL, ast.BANG))
		break
	case '<':
		s.addToken(helpers.If(s.match('='), ast.LESS_EQUAL, ast.LESS))
		break
	case '>':
		s.addToken(helpers.If(s.match('='), ast.GREATER_EQUAL, ast.GREATER))
		break
	case '/':
		if s.match('/') {
			// A comment goes until the end of the line.
			for s.peek() != '\n' && !s.isAtEnd() {
				s.advance()
			}
		} else {
			s.addToken(ast.SLASH)
		}
	case '"':
		if err := s.string(); err != nil {
			s.errors = append(s.errors, fmt.Errorf("[line: %d] Error: %w", s.line, err))
		}

		break
		// Ignore whitespace.
	case ' ':
	case '\r':
	case '\t':
		break
	case '\n':
		s.line++
		break
	default:
		if helpers.IsAlpha(char) {
			s.identifier()
		} else if helpers.IsDigit(char) {
			s.number()
		} else {
			// Report the whole (possibly multi-byte) character, not a single byte of it.
			r, size := utf8.DecodeRuneInString(s.source[s.start:])
			s.current = s.start + size
			s.errors = append(s.errors, fmt.Errorf("[line: %d] Error: Unexpected character: %c", s.line, r))
		}
	}
}

func (s *Scanner) addToken(tokenType ast.TokenType) {
	lexeme := s.source[s.start:s.current]

	s.tokens = append(
		s.tokens,
		ast.NewToken(tokenType, lexeme, "", s.line),
	)
}

func (s *Scanner) addTokenWithLiteral(tokenType ast.TokenType, literal string) {
	lexeme := s.source[s.start:s.current]

	s.tokens = append(
		s.tokens,
		ast.NewToken(tokenType, lexeme, literal, s.line),
	)
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
}

func (s *Scanner) string() error {
	var value strings.Builder
	var escapeErr error

	for s.peek() != '"' && !s.isAtEnd() {
		c := s.source[s.current]
		s.current++

		if c == '\n' {
			s.line++
		}

		if c != '\\' {
			value.WriteByte(c)
			continue
		}

		if s.isAtEnd() {
			break
		}

		// Keep scanning after a bad escape so the string still becomes a token.
		r, size := utf8.DecodeRuneInString(s.source[s.current:])
		s.current += size

		if e, ok := escapes[byte(r)]; ok && size == 1 {
			value.WriteByte(e)
		} else if escapeErr == nil {
			escapeErr = fmt.Errorf("Invalid escape sequence '\\%c'.", r)
		}

		if r == '\n' {
			s.line++
		}
	}

	if s.isAtEnd() {
		return errors.New("Unterminated string.")
	}

	s.advance()
	s.addTokenWithLiteral(ast.STRING, value.String())

	return escapeErr
}

func (s *Scanner) number() {
	for helpers.IsDigit(s.peek()) {
		s.advance()
	}
	// Look for a fractional part.
	if s.peek() == '.' && helpers.IsDigit(s.peekNext()) {
		// Consume the "."
		s.advance()

		for helpers.IsDigit(s.peek()) {
			s.advance()
		}
	}

	// Look for an exponent: e10, E-3, e+2.
	if s.peek() == 'e' || s.peek() == 'E' {
		digitAt := s.current + 1
		if digitAt < len(s.source) && (s.source[digitAt] == '+' || s.source[digitAt] == '-') {
			digitAt++
		}

		if digitAt < len(s.source) && helpers.IsDigit(rune(s.source[digitAt])) {
			s.current = digitAt

			for helpers.IsDigit(s.peek()) {
				s.advance()
			}
		}
	}

	val, _ := strconv.ParseFloat(s.source[s.start:s.current], 64)
	literal := ""

	if val == math.Trunc(val) {
		literal = fmt.Sprintf("%.1f", val) // Ensures 1234.0 for whole numbers
	} else {
		literal = fmt.Sprintf("%g", val) // Keeps the precision for non-whole numbers
	}

	s.addTokenWithLiteral(ast.NUMBER, literal)
}

func (s *Scanner) identifier() {
	for s.isAlphaNumeric(s.peek()) {
		s.advance()
	}

	text := s.source[s.start:s.current]

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
