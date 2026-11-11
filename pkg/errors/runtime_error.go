package errors

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/value"
)

// Frame is a call the error unwound through: the function and the line it
// was called from.
type Frame struct {
	Name   string
	Line   int
	Native bool
}

// RuntimeError is an error raised while running a program: a throw of
// value, an error of kind raised by the language, or with neither a fatal
// error, which try does not catch.
type RuntimeError struct {
	token  ast.Token
	kind   *value.Shape
	errMsg string
	value  value.Value
	thrown bool
	frames []Frame
}

func (e RuntimeError) Error() string {
	if e.kind != nil {
		return fmt.Sprintf("[line: %d] %s: %s", e.token.Line, e.kind.Name, e.errMsg)
	}

	return fmt.Sprintf("[line: %d] %s", e.token.Line, e.errMsg)
}

func (e RuntimeError) Fatal() bool { return e.kind == nil && !e.thrown }

func (e RuntimeError) Pos() (line, col int) { return e.token.Line, e.token.Col }

// Value is what a catch clause binds: the thrown value for a throw, or a
// struct of the error's kind for an error raised by the language.
func (e RuntimeError) Value() value.Value {
	if e.thrown {
		return e.value
	}

	v, fields := value.NewStruct(e.kind)
	fields[0] = value.NewString(e.errMsg)
	fields[1] = value.NewNumber(float64(e.token.Line))

	return v
}

// WithFrame records that the error left fn, called from line.
func (e RuntimeError) WithFrame(f Frame) RuntimeError {
	e.frames = append(e.frames, f)
	return e
}

// WithFrame is RuntimeError.WithFrame for any error; others pass through.
func WithFrame(err error, f Frame) error {
	if e, ok := err.(RuntimeError); ok {
		return e.WithFrame(f)
	}

	return err
}

func NewRuntimeError(kind *value.Shape, token ast.Token, msg string) error {
	return RuntimeError{token: token, kind: kind, errMsg: msg}
}

func NewFatalError(token ast.Token, msg string) error {
	return RuntimeError{token: token, errMsg: msg}
}

// NewThrownError is the error of `throw v`; msg is v as the language prints it.
func NewThrownError(token ast.Token, v value.Value, msg string) error {
	return RuntimeError{token: token, errMsg: msg, value: v, thrown: true}
}
