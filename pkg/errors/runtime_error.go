package errors

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/value"
)

// RuntimeError is an error raised while running a program. Value is what a
// catch clause binds: the message string for built-in errors, or the thrown
// value for a throw statement.
type RuntimeError struct {
	token  ast.Token
	errMsg string
	value  value.Value
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf(
		"[line: %d] %s",
		e.token.Line,
		e.errMsg,
	)
}

// Value returns the value a catch clause receives for this error.
func (e RuntimeError) Value() value.Value {
	return e.value
}

func NewRuntimeError(token ast.Token, msg string) error {
	return RuntimeError{token: token, errMsg: msg, value: value.NewString(msg)}
}

// NewThrownError is the error of `throw v`; msg is v as the language prints it.
func NewThrownError(token ast.Token, v value.Value, msg string) error {
	return RuntimeError{token: token, errMsg: msg, value: v}
}
