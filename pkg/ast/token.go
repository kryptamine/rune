package ast

import "fmt"

type TokenType int

type Token struct {
	TokenType TokenType
	Lexeme    string
	Literal   string
	Line      int
	// Col is the 1-based character column of the token's start; 0 if unknown.
	Col int
}

func NewToken(tokenType TokenType, lexeme string, literal string, line int) Token {
	return Token{TokenType: tokenType, Lexeme: lexeme, Literal: literal, Line: line}
}

func (token Token) String() string {
	literal := "null"

	if token.Literal != "" {
		literal = token.Literal
	}

	return fmt.Sprintf(
		"%s %s %s",
		token.TokenType,
		token.Lexeme,
		literal,
	)
}

const (
	IDENTIFIER TokenType = iota

	// Keywords
	VAR
	AND
	OR
	ELSE
	FALSE
	FUN
	FOR
	IF
	NIL
	PRINT
	RETURN
	TRUE
	WHILE
	TRY
	CATCH
	THROW
	BREAK
	CONTINUE
	IN

	// Single tokens
	COLON
	SEMICOLON
	LEFT_PAREN
	RIGHT_PAREN
	LEFT_BRACKET
	RIGHT_BRACKET
	LEFT_BRACE
	RIGHT_BRACE
	COMMA
	DOT
	MINUS
	PLUS
	SLASH
	STAR
	PERCENT
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
	PLUS_EQUAL
	MINUS_EQUAL
	STAR_EQUAL
	SLASH_EQUAL
	PERCENT_EQUAL
	PLUS_PLUS
	MINUS_MINUS

	// Types
	OBJECT
	ARRAY
	STRING
	NUMBER
)

var Keywords = map[string]TokenType{
	"var":      VAR,
	"and":      AND,
	"or":       OR,
	"else":     ELSE,
	"false":    FALSE,
	"fun":      FUN,
	"for":      FOR,
	"if":       IF,
	"nil":      NIL,
	"print":    PRINT,
	"return":   RETURN,
	"true":     TRUE,
	"while":    WHILE,
	"try":      TRY,
	"catch":    CATCH,
	"throw":    THROW,
	"break":    BREAK,
	"continue": CONTINUE,
	"in":       IN,
}

func (tokenType TokenType) String() string {
	switch tokenType {
	case COLON:
		return "COLON"
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
	case PERCENT:
		return "PERCENT"
	case EOF:
		return "EOF"

		// types
	case STRING:
		return "STRING"
	case NUMBER:
		return "NUMBER"
	case ARRAY:
		return "ARRAY"
	case OBJECT:
		return "OBJECT"

		// keywords
	case VAR:
		return "VAR"
	case AND:
		return "AND"

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
	case TRUE:
		return "TRUE"
	case WHILE:
		return "WHILE"
	case TRY:
		return "TRY"
	case CATCH:
		return "CATCH"
	case THROW:
		return "THROW"
	case BREAK:
		return "BREAK"
	case CONTINUE:
		return "CONTINUE"
	case IN:
		return "IN"
	case OR:
		return "OR"
	case LEFT_PAREN:
		return "LEFT_PAREN"
	case RIGHT_PAREN:
		return "RIGHT_PAREN"
	case LEFT_BRACKET:
		return "LEFT_BRACKET"
	case RIGHT_BRACKET:
		return "RIGHT_BRACKET"
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
	case PLUS_EQUAL:
		return "PLUS_EQUAL"
	case MINUS_EQUAL:
		return "MINUS_EQUAL"
	case STAR_EQUAL:
		return "STAR_EQUAL"
	case SLASH_EQUAL:
		return "SLASH_EQUAL"
	case PERCENT_EQUAL:
		return "PERCENT_EQUAL"
	case PLUS_PLUS:
		return "PLUS_PLUS"
	case MINUS_MINUS:
		return "MINUS_MINUS"
	default:
		return "Undefined token."
	}
}
