package main

import "fmt"

type Environment struct {
	values map[string]any
}

func NewEnvironment() Environment {
	return Environment{
		values: map[string]any{},
	}
}

func (e *Environment) define(name string, value any) {
	e.values[name] = value
}

func (e *Environment) get(token Token) (any, error) {
	if val, ok := e.values[token.lexeme]; ok {
		return val, nil
	}

	return nil, fmt.Errorf("Undefined variable '%s'.", token.lexeme)
}

func (e *Environment) assign(token Token, value any) error {
	if _, ok := e.values[token.lexeme]; ok {
		e.values[token.lexeme] = value
		return nil
	}

	return fmt.Errorf("Undefined variable '%s'.", token.lexeme)
}
