package errors

import (
	"fmt"
	"rune/pkg/ast"
)

// Positioned is an error that points at a place in the source.
type Positioned interface {
	error
	Pos() (line, col int)
}

// SyntaxError is a parser or resolver error pointing at a token.
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

func (e SyntaxError) Pos() (line, col int) { return e.token.Line, e.token.Col }

func NewSyntaxError(token ast.Token, msg string) error {
	return SyntaxError{token: token, errMsg: msg}
}

// ScanError is a scanner error; there is no token yet to name.
type ScanError struct {
	line, col int
	errMsg    string
}

func (e ScanError) Error() string {
	return fmt.Sprintf("[line: %d] Error: %s", e.line, e.errMsg)
}

func (e ScanError) Pos() (line, col int) { return e.line, e.col }

func NewScanError(line, col int, msg string) error {
	return ScanError{line: line, col: col, errMsg: msg}
}
