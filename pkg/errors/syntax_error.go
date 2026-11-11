package errors

import (
	"fmt"
	"rune/pkg/ast"
)

// SyntaxError is a compile-time error (scanner, parser or resolver).
type SyntaxError struct {
	token  ast.Token
	errMsg string
	// bare errors have no token to point at: "Error: msg".
	bare  bool
	cause error
}

func (e SyntaxError) Error() string {
	if e.bare {
		return fmt.Sprintf("[line: %d] Error: %s", e.token.Line, e.errMsg)
	}

	where := fmt.Sprintf("at '%s'", e.token.Lexeme)
	if e.token.TokenType == ast.EOF {
		where = "at end"
	}

	return fmt.Sprintf("[line: %d] Error %s: %s", e.token.Line, where, e.errMsg)
}

func (e SyntaxError) Unwrap() error { return e.cause }

func (e SyntaxError) Pos() (line, col int) { return e.token.Line, e.token.Col }

func NewSyntaxError(token ast.Token, msg string) error {
	return SyntaxError{token: token, errMsg: msg}
}

// NewScanError is a scanner error at line:col; cause, if set, is what
// errors.Is matches.
func NewScanError(line, col int, msg string, cause error) error {
	return SyntaxError{token: ast.Token{Line: line, Col: col}, errMsg: msg, bare: true, cause: cause}
}
