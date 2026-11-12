package errors

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/helpers"
	"rune/pkg/value"
)

// Frame is a call the error unwound through: the function and the place
// it was called from.
type Frame struct {
	Name   string
	File   int
	Line   int
	Native bool
}

// RuntimeError is an error raised while running a program: a throw of the
// struct value, an error of kind raised by the language, or with neither a
// fatal error, which try does not catch.
type RuntimeError struct {
	token  ast.Token
	kind   *value.Shape
	errMsg string
	value  value.Value
	frames []Frame
	// extra fills the fields the kind has beyond the common ones.
	extra []value.Value
}

func (e RuntimeError) Error() string {
	if e.thrown() {
		return fmt.Sprintf("[line: %d] %s", e.token.Line, thrownMessage(e.value))
	}

	if e.kind != nil {
		return fmt.Sprintf("[line: %d] %s: %s", e.token.Line, e.kind.Name, e.errMsg)
	}

	return fmt.Sprintf("[line: %d] %s", e.token.Line, e.errMsg)
}

func (e RuntimeError) Fatal() bool { return e.kind == nil && !e.thrown() }

func (e RuntimeError) thrown() bool { return e.value.Kind() == value.Struct }

func (e RuntimeError) Pos() (file, line, col int) {
	return e.token.File, e.token.Line, e.token.Col
}

// Value is what a catch clause binds: the thrown value for a throw, or a
// struct of the error's kind for an error raised by the language.
func (e RuntimeError) Value() value.Value {
	if e.thrown() {
		return e.value
	}

	v, fields := value.NewStruct(e.kind)
	fields[0] = value.NewString(e.errMsg)
	fields[1] = value.NewNumber(float64(e.token.Line))
	copy(fields[len(errorFields):], e.extra)

	return v
}

// WithFrame records that the error left a call.
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

// NewIOError is an IOError about path with code; empty ones are nil.
func NewIOError(token ast.Token, msg, path, code string) error {
	return RuntimeError{token: token, kind: IOError, errMsg: msg, extra: []value.Value{optString(path), optString(code)}}
}

func optString(s string) value.Value {
	if s == "" {
		return value.NilValue
	}

	return value.NewString(s)
}

func NewFatalError(token ast.Token, msg string) error {
	return RuntimeError{token: token, errMsg: msg}
}

// NewThrownError is the error of `throw v`; v must be a struct.
func NewThrownError(token ast.Token, v value.Value) error {
	return RuntimeError{token: token, value: v}
}

// thrownMessage is "Name: message" when v has a string message, v as
// printed otherwise.
func thrownMessage(v value.Value) string {
	s := v.Shape()
	if i := s.Index("message"); i >= 0 {
		if m := v.Fields()[i]; m.Kind() == value.String {
			return s.Name + ": " + m.Str()
		}
	}

	return helpers.Stringify(v)
}
