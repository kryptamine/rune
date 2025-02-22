package main

import (
	"errors"
	"fmt"
	"strconv"
)

func Scan(source []byte) ([]Token, []error) {
	scanner := &Scanner{
		source:  string(source),
		tokens:  []Token{},
		errors:  []error{},
		current: 0,
		line:    1,
		start:   0,
	}

	return scanner.scanTokens(), scanner.errors
}

func (token Token) String() string {
	literal := "nil"

	if token.literal != "" {
		literal = token.literal
	}

	return fmt.Sprintf(
		"%s %s %s",
		token.tokenType,
		token.lexeme,
		literal,
	)
}

func (s *Scanner) scanTokens() []Token {
	for !s.isAtEnd() {
		s.start = s.current
		s.scanToken()
	}

	s.tokens = append(
		s.tokens,
		Token{
			tokenType: EOF,
			lexeme:    "",
			literal:   "",
			line:      s.line,
		},
	)

	return s.tokens
}

func (s *Scanner) isAtEnd() bool {
	return s.current >= len(s.source)
}

func (s *Scanner) scanToken() {
	char := s.advance()

	switch char {
	case ';':
		s.addToken(SEMICOLON)
		break
	case '(':
		s.addToken(LEFT_PAREN)
		break
	case ')':
		s.addToken(RIGHT_PAREN)
		break
	case '{':
		s.addToken(LEFT_BRACE)
		break
	case '}':
		s.addToken(RIGHT_BRACE)
		break
	case ',':
		s.addToken(COMMA)
		break
	case '.':
		s.addToken(DOT)
		break
	case '-':
		s.addToken(MINUS)
		break
	case '+':
		s.addToken(PLUS)
		break
	case '*':
		s.addToken(STAR)
		break
		// Operators
	case '=':
		s.addToken(If(s.match('='), EQUAL_EQUAL, EQUAL))
		break
	case '!':
		s.addToken(If(s.match('='), BANG_EQUAL, BANG))
		break
	case '<':
		s.addToken(If(s.match('='), LESS_EQUAL, LESS))
		break
	case '>':
		s.addToken(If(s.match('='), GREATER_EQUAL, GREATER))
		break
	case '/':
		if s.match('/') {
			// A comment goes until the end of the line.
			for s.peek() != '\n' && !s.isAtEnd() {
				s.advance()
			}
		} else {
			s.addToken(SLASH)
		}
	case '"':
		if err := s.string(); err != nil {
			s.errors = append(s.errors, fmt.Errorf("[line %d] Error: %w", s.line, err))
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
		if isAlpha(char) {
			s.identifier()
		} else if isDigit(char) {
			s.number()
		} else {
			s.errors = append(s.errors, fmt.Errorf("[line %d] Error: Unexpected character: %c", s.line, char))
		}
	}
}

func (s *Scanner) addToken(tokenType TokenType) {
	lexeme := s.source[s.start:s.current]

	s.tokens = append(
		s.tokens,
		Token{
			tokenType: tokenType,
			lexeme:    lexeme,
			line:      s.line,
		},
	)
}

func (s *Scanner) addTokenWithLiteral(tokenType TokenType, literal string) {
	lexeme := s.source[s.start:s.current]

	s.tokens = append(
		s.tokens,
		Token{
			tokenType: tokenType,
			lexeme:    lexeme,
			literal:   literal,
			line:      s.line,
		},
	)
}

func (s *Scanner) currentChar() rune {
	return rune(s.source[s.current])
}

func (s *Scanner) string() error {
	for s.peek() != '"' && !s.isAtEnd() {
		if s.peek() == '\n' {
			s.line++
		}

		s.advance()
	}

	if s.isAtEnd() {
		return errors.New("Unterminated string.")
	}

	s.advance()

	value := s.source[s.start+1 : s.current-1]

	s.addTokenWithLiteral(STRING, value)

	return nil
}

func (s *Scanner) number() {
	for isDigit(s.peek()) {
		s.advance()
	}
	// Look for a fractional part.
	if s.peek() == '.' && isDigit(s.peekNext()) {
		// Consume the "."
		s.advance()

		for isDigit(s.peek()) {
			s.advance()
		}
	}

	val, _ := strconv.ParseFloat(s.source[s.start:s.current], 64)
	literal := ""

	if val == float64(int(val)) {
		literal = fmt.Sprintf("%.1f", val) // Ensures 1234.0 for whole numbers
	} else {
		literal = fmt.Sprintf("%g", val) // Keeps the precision for non-whole numbers
	}

	s.addTokenWithLiteral(NUMBER, literal)
}

func (s *Scanner) identifier() {
	for s.isAlphaNumeric(s.peek()) {
		s.advance()
	}

	text := s.source[s.start:s.current]

	if val, ok := keywords[text]; ok {
		s.addToken(val)
	} else {
		s.addToken(IDENTIFIER)
	}
}

func (s *Scanner) isAlphaNumeric(c rune) bool {
	return isAlpha(c) || isDigit(c)
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
