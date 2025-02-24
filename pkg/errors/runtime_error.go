package errors

import (
	"fmt"
	"rune/pkg/ast"
)

type RuntimeError struct {
	token  ast.Token
	errMsg string
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf(
		"[line: %d] %s",
		e.token.Line,
		e.errMsg,
	)
}

func NewRuntimeError(token ast.Token, msg string) error {
	return RuntimeError{token: token, errMsg: msg}
}
