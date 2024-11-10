package main

import (
	"errors"
	"fmt"
	"strconv"
)

type TokenType int

type Token struct {
	tokenType TokenType
	lexeme    string
	literal   string
	line      int
}

type Scanner struct {
	source string
	tokens []Token
	errors []error

	start   int
	current int
	line    int
}

const (
	IDENTIFIER TokenType = iota

	// Keywords
	VAR
	AND
	OR
	CLASS
	ELSE
	FALSE
	FUN
	FOR
	IF
	NIL
	PRINT
	RETURN
	SUPER
	THIS
	TRUE
	WHILE

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

	// Types
	STRING
	NUMBER
)

var keywords = map[string]TokenType{
	"var":    VAR,
	"and":    AND,
	"or":     OR,
	"class":  CLASS,
	"else":   ELSE,
	"false":  FALSE,
	"fun":    FUN,
	"for":    FOR,
	"if":     IF,
	"nil":    NIL,
	"print":  PRINT,
	"return": RETURN,
	"super":  SUPER,
	"this":   THIS,
	"true":   TRUE,
	"while":  WHILE,
}

func Tokenize(source []byte) ([]Token, []error) {
	scanner := &Scanner{
		source:  string(source),
		tokens:  []Token{},
		errors:  []error{},
		current: 0,
		line:    1,
		start:   0,
	}

	tokens := scanner.scanTokens()

	return tokens, scanner.errors
}

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

		// types
	case STRING:
		return "STRING"
	case NUMBER:
		return "NUMBER"

		// keywords
	case VAR:
		return "VAR"
	case AND:
		return "AND"

	case CLASS:
		return "CLASS"
	case ELSE:
		return "ELSE"
	case FALSE:
		return "FALSE"
	case FUN:
		return "FUN"
	case FOR:
		return "FOR"
	case IF:
		return "IF"
	case NIL:
		return "NIL"
	case PRINT:
		return "PRINT"
	case RETURN:
		return "RETURN"
	case SUPER:
		return "SUPER"
	case THIS:
		return "THIS"
	case TRUE:
		return "TRUE"
	case WHILE:
		return "WHILE"
	case OR:
		return "OR"
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
	if token.literal == "" {
		return fmt.Sprintf(
			"%s %s null",
			token.tokenType,
			token.lexeme,
		)
	}

	return fmt.Sprintf(
		"%s %s %s",
		token.tokenType,
		token.lexeme,
		token.literal,
	)
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
		if s.isAlpha(char) {
			s.identifier()
		} else if s.isDigit(char) {
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
	for s.isDigit(s.peek()) {
		s.advance()
	}
	// Look for a fractional part.
	if s.peek() == '.' && s.isDigit(s.peekNext()) {
		// Consume the "."
		s.advance()

		for s.isDigit(s.peek()) {
			s.advance()
		}
	}

	val, _ := strconv.ParseFloat(s.source[s.start:s.current], 64)
	formattedVal := strconv.FormatFloat(val, 'f', -1, 64) // Automatically trims trailing zeroes
	s.addTokenWithLiteral(NUMBER, formattedVal)
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
	return s.isAlpha(c) || s.isDigit(c)
}

func (s *Scanner) isAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		c == '_'
}

func (s *Scanner) isDigit(c rune) bool {
	return c >= '0' && c <= '9'
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
