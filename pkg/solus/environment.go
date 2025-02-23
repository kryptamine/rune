package solus

import (
	"fmt"
	"github.com/codecrafters-io/interpreter-starter-go/pkg/ast"
)

type Environment struct {
	values    map[string]any
	enclosing *Environment
}

func NewEnvironment(enclosing *Environment) *Environment {
	return &Environment{
		values:    map[string]any{},
		enclosing: enclosing,
	}
}

func (e *Environment) RegisterGlobalCallable(name string, value Callable) {
	e.define(name, value)
}

func (e *Environment) String() string {
	return fmt.Sprintf("<env %v>", e.values)
}

func (e *Environment) define(name string, value any) {
	e.values[name] = value
}

func (e *Environment) get(token ast.Token) (any, error) {
	if val, ok := e.values[token.Lexeme]; ok {
		return val, nil
	}

	if e.enclosing != nil {
		return e.enclosing.get(token)
	}

	return nil, NewRuntimeError(token, fmt.Sprintf("Undefined variable '%s'.", token.Lexeme))
}

func (e *Environment) assign(token ast.Token, value any) error {
	if _, ok := e.values[token.Lexeme]; ok {
		e.values[token.Lexeme] = value
		return nil
	}

	if e.enclosing != nil {
		return e.enclosing.assign(token, value)
	}

	return NewRuntimeError(token, fmt.Sprintf("Undefined variable '%s'.", token.Lexeme))
}
