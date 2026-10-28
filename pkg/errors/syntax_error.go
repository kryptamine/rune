package errors

import (
	"fmt"
	"rune/pkg/ast"
)

// SyntaxError is a compile-time error (parser or resolver) pointing at a token.
type SyntaxError struct {
	token  ast.Token
	errMsg string
}

func (e SyntaxError) Error() string {
	where := fmt.Sprintf("at '%s'", e.token.Lexeme)
	if e.token.TokenType == ast.EOF {
		where = "at end"
	}

	return fmt.Sprintf("[line: %d] Error %s: %s", e.token.Line, where, e.errMsg)
}

func NewSyntaxError(token ast.Token, msg string) error {
	return SyntaxError{token: token, errMsg: msg}
}
