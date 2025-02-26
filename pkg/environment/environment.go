package environment

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
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

func (e *Environment) String() string {
	return fmt.Sprintf("<env %v>", e.values)
}

func (e *Environment) Define(name string, value any) {
	e.values[name] = value
}

func (e *Environment) Get(token ast.Token) (any, error) {
	if val, ok := e.values[token.Lexeme]; ok {
		return val, nil
	}

	if e.enclosing != nil {
		return e.enclosing.Get(token)
	}

	return nil, errors.NewRuntimeError(token, fmt.Sprintf("Undefined variable '%s'.", token.Lexeme))
}

func (e *Environment) Assign(token ast.Token, value any) error {
	if _, ok := e.values[token.Lexeme]; ok {
		e.values[token.Lexeme] = value
		return nil
	}

	if e.enclosing != nil {
		return e.enclosing.Assign(token, value)
	}

	return errors.NewRuntimeError(token, fmt.Sprintf("Undefined variable '%s'.", token.Lexeme))
}

func (e *Environment) AssignAt(distance int, name string, value any) {
	e.ancestor(distance).values[name] = value
}

func (e *Environment) GetAt(distance int, name string) any {
	return e.ancestor(distance).values[name]
}

func (e *Environment) ancestor(distance int) *Environment {
	env := e

	for range distance {
		env = env.enclosing
	}

	return env
}
