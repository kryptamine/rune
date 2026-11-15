package errors

import (
	"fmt"
	"rune/pkg/ast"
	"strings"
	"unicode/utf8"
)

// Span is the place an error points at: Width characters from column Col
// of a line.
type Span struct {
	File, Line, Col, Width int
}

func tokenSpan(t ast.Token) Span {
	lexeme, _, _ := strings.Cut(t.Lexeme, "\n")
	return Span{File: t.File, Line: t.Line, Col: t.Col, Width: max(utf8.RuneCountInString(lexeme), 1)}
}

// SyntaxError is a parser or resolver error pointing at a token, or at
// mark when set.
type SyntaxError struct {
	token  ast.Token
	errMsg string
	fix    string
	mark   *Span
}

func (e SyntaxError) Error() string {
	where := fmt.Sprintf("at '%s'", e.token.Lexeme)
	if e.token.TokenType == ast.EOF {
		where = "at end"
	}

	return fmt.Sprintf("[line: %d] Error %s: %s", e.token.Line, where, e.errMsg)
}

func (e SyntaxError) heading() (string, string) { return "", e.errMsg }

func (e SyntaxError) hint() string { return e.fix }

func (e SyntaxError) span() Span {
	if e.mark != nil {
		return *e.mark
	}

	return tokenSpan(e.token)
}

func NewSyntaxError(token ast.Token, msg string) error {
	return SyntaxError{token: token, errMsg: msg}
}

// NewMissingError is the error of a token missing after prev, found
// before next; it points just past prev when next starts a later line.
func NewMissingError(prev, next ast.Token, msg string) error {
	err := SyntaxError{token: next, errMsg: msg}
	if prev.Line < next.Line && prev.Col > 0 && !strings.Contains(prev.Lexeme, "\n") {
		err.mark = &Span{File: prev.File, Line: prev.Line, Col: prev.Col + utf8.RuneCountInString(prev.Lexeme), Width: 1}
	}

	return err
}

// NewRedeclaredError is the error of a declaration of name, which its
// scope already declares at first.
func NewRedeclaredError(name, first ast.Token) error {
	return SyntaxError{
		token:  name,
		errMsg: "Variable with this name already declared in this scope.",
		fix:    fmt.Sprintf("'%s' is first declared on line %d; rename one of them", name.Lexeme, first.Line),
	}
}

// ScanError is a scanner error; there is no token yet to name.
type ScanError struct {
	file      int
	line, col int
	errMsg    string
}

func (e ScanError) Error() string {
	return fmt.Sprintf("[line: %d] Error: %s", e.line, e.errMsg)
}

func (e ScanError) heading() (string, string) { return "", e.errMsg }

func (e ScanError) hint() string { return "" }

func (e ScanError) span() Span { return Span{File: e.file, Line: e.line, Col: e.col, Width: 1} }

func NewScanError(file, line, col int, msg string) error {
	return ScanError{file: file, line: line, col: col, errMsg: msg}
}

// WithHint attaches a fix to suggest, for an error sure of its cause.
func WithHint(err error, hint string) error {
	switch e := err.(type) {
	case SyntaxError:
		e.fix = hint
		return e
	case RuntimeError:
		e.fix = hint
		return e
	}

	return err
}
