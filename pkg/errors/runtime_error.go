package errors

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/value"
)

// Frame is a call the error unwound through: the function and the line it
// was called from (Line is 0 for a native's callback, which has no source).
type Frame struct {
	Name   string
	Line   int
	Native bool
}

// RuntimeError is an error raised while running a program.
type RuntimeError struct {
	token  ast.Token
	errMsg string
	value  value.Value
	thrown bool
	frames []Frame
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf("[line: %d] %s", e.token.Line, e.errMsg)
}

func (e RuntimeError) Pos() (line, col int) { return e.token.Line, e.token.Col }

// Value is what a catch clause binds: the thrown value for a throw, or
// {message, line} for an error raised by the language.
func (e RuntimeError) Value() value.Value {
	if e.thrown {
		return e.value
	}

	return value.NewObject(map[string]value.Value{
		"message": value.NewString(e.errMsg),
		"line":    value.NewNumber(float64(e.token.Line)),
	})
}

// Frames lists the calls unwound, innermost first.
func (e RuntimeError) Frames() []Frame { return e.frames }

// WithFrame records that the error left fn, called from line.
func WithFrame(err error, f Frame) error {
	e, ok := err.(RuntimeError)
	if !ok {
		return err
	}

	e.frames = append(e.frames, f)

	return e
}

func NewRuntimeError(token ast.Token, msg string) error {
	return RuntimeError{token: token, errMsg: msg}
}

// NewThrownError is the error of `throw v`; msg is v as the language prints it.
func NewThrownError(token ast.Token, v value.Value, msg string) error {
	return RuntimeError{token: token, errMsg: msg, value: v, thrown: true}
}
