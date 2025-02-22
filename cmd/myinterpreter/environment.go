package main

import "fmt"

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

func (e *Environment) define(name string, value any) {
	e.values[name] = value
}

func (e *Environment) get(token Token) (any, error) {
	if val, ok := e.values[token.lexeme]; ok {
		return val, nil
	}

	if e.enclosing != nil {
		return e.enclosing.get(token)
	}

	return nil, NewRuntimeError(token, fmt.Sprintf("Undefined variable '%s'.", token.lexeme))
}

func (e *Environment) assign(token Token, value any) error {
	if _, ok := e.values[token.lexeme]; ok {
		e.values[token.lexeme] = value
		return nil
	}

	if e.enclosing != nil {
		return e.enclosing.assign(token, value)
	}

	return NewRuntimeError(token, fmt.Sprintf("Undefined variable '%s'.", token.lexeme))
}
