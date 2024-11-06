package main

import (
	"errors"
	"fmt"
)

type TokenType int

type Token struct {
	tokenType TokenType
	lexeme    string
	literal   any
	line      int
}

type Scanner struct {
	source   string
	tokens   []Token
	keywords map[string]TokenType
	errors   []error

	start   int
	current int
	line    int
}

const (
	VAR TokenType = iota

	// Single tokens
	SEMICOLON
	LEFT_PAREN
	RIGHT_PAREN
	LEFT_BRACE
	RIGHT_BRACE
	COMMA
	DOT
	MINUS
	PLUS
	SLASH
	STAR
	EOF

	// Operators
	BANG
	BANG_EQUAL
	EQUAL
	EQUAL_EQUAL
	LESS
	LESS_EQUAL
	GREATER
	GREATER_EQUAL

	IDENTIFIER

	// Types
	STRING
)

func (tokenType TokenType) String() string {
	switch tokenType {
	case SEMICOLON:
		return "SEMICOLON"
	case COMMA:
		return "COMMA"
	case DOT:
		return "DOT"
	case MINUS:
		return "MINUS"
	case PLUS:
		return "PLUS"
	case SLASH:
		return "SLASH"
	case STAR:
		return "STAR"
	case EOF:
		return "EOF"
	case STRING:
		return "STRING"
	case VAR:
		return "VAR"
	case LEFT_PAREN:
		return "LEFT_PAREN"
	case RIGHT_PAREN:
		return "RIGHT_PAREN"
	case LEFT_BRACE:
		return "LEFT_BRACE"
	case RIGHT_BRACE:
		return "RIGHT_BRACE"
	case IDENTIFIER:
		return "IDENTIFIER"
	case EQUAL:
		return "EQUAL"
	case EQUAL_EQUAL:
		return "EQUAL_EQUAL"
	case BANG:
		return "BANG"
	case BANG_EQUAL:
		return "BANG_EQUAL"
	case LESS:
		return "LESS"
	case LESS_EQUAL:
		return "LESS_EQUAL"
	case GREATER:
		return "GREATER"
	case GREATER_EQUAL:
		return "GREATER_EQUAL"
	default:
		return "Undefined token."
	}
}

func (token Token) String() string {
	if token.literal == nil {
		return fmt.Sprintf("%s %s null", token.tokenType, token.lexeme)
	}

	return fmt.Sprintf("%s %s %v", token.tokenType, token.lexeme, token.literal)
}

func If[T any](cond bool, vtrue, vfalse T) T {
	if cond {
		return vtrue
	}
	return vfalse
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
			literal:   nil,
			line:      s.line,
		},
	)

	return s.tokens
}

func (s *Scanner) run() []error {
	tokens := s.scanTokens()

	for _, token := range tokens {
		fmt.Println(token)
	}

	return s.errors
}

func (s *Scanner) isAtEnd() bool {
	return s.current >= len(s.source)
}

func (s *Scanner) scanToken() {
	char := s.advance()

	switch char {
	case ';':
		s.addToken(SEMICOLON, nil)
		break
	case '(':
		s.addToken(LEFT_PAREN, nil)
		break
	case ')':
		s.addToken(RIGHT_PAREN, nil)
		break
	case '{':
		s.addToken(LEFT_BRACE, nil)
		break
	case '}':
		s.addToken(RIGHT_BRACE, nil)
		break
	case ',':
		s.addToken(COMMA, nil)
		break
	case '.':
		s.addToken(DOT, nil)
		break
	case '-':
		s.addToken(MINUS, nil)
		break
	case '+':
		s.addToken(PLUS, nil)
		break
	case '*':
		s.addToken(STAR, nil)
		break
		// Operators
	case '=':
		s.addToken(If(s.match('='), EQUAL_EQUAL, EQUAL), nil)
		break
	case '!':
		s.addToken(If(s.match('='), BANG_EQUAL, BANG), nil)
		break
	case '<':
		s.addToken(If(s.match('='), LESS_EQUAL, LESS), nil)
		break
	case '>':
		s.addToken(If(s.match('='), GREATER_EQUAL, GREATER), nil)
		break
	case '"':
		s.string()
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
		if s.isAlpha(char) {
			s.identifier()
		} else {
			s.errors = append(s.errors, fmt.Errorf("[line %d] Error: Unexpected character: %c", s.line, char))
		}
	}
}

func (s *Scanner) addToken(tokenType TokenType, literal any) {
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

	s.addToken(STRING, value)

	return nil
}

func (s *Scanner) identifier() {
	for s.isAlpha(s.peek()) {
		s.advance()
	}

	text := s.source[s.start:s.current]

	if val, ok := s.keywords[text]; ok {
		s.addToken(val, nil)
	} else {
		s.addToken(IDENTIFIER, nil)
	}
}

func (s *Scanner) isAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		c == '_'
}

func (s *Scanner) peek() rune {
	if s.isAtEnd() {
		return rune(0)
	}

	return s.currentChar()
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
