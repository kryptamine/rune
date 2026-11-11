package environment

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/value"
	"sync"
)

// Environment is one local scope. The resolver assigns every local a fixed
// slot, so variables are addressed by (depth, slot) instead of by name.
type Environment struct {
	values    []value.Value
	enclosing *Environment
	inline    [inlineSlots]value.Value
}

const inlineSlots = 4

func NewEnvironment(enclosing *Environment, size int) *Environment {
	env := &Environment{enclosing: enclosing}

	if size <= inlineSlots {
		env.values = env.inline[:size]
	} else {
		env.values = make([]value.Value, size)
	}

	return env
}

func (e *Environment) Values() []value.Value { return e.values }

type Pool struct {
	free []*Environment
}

func (p *Pool) Get(enclosing *Environment, size int) *Environment {
	n := len(p.free)
	if n == 0 {
		return NewEnvironment(enclosing, size)
	}

	env := p.free[n-1]
	p.free = p.free[:n-1]

	if cap(env.values) >= size {
		env.values = env.values[:size]
	} else {
		env.values = make([]value.Value, size)
	}

	env.enclosing = enclosing

	return env
}

func (p *Pool) Put(env *Environment) {
	clear(env.values)
	env.enclosing = nil
	p.free = append(p.free, env)
}

func (e *Environment) String() string {
	return fmt.Sprintf("<env %v>", e.values)
}

func (e *Environment) Get(slot int) value.Value { return e.values[slot] }

func (e *Environment) Enclosing() *Environment { return e.enclosing }

func (e *Environment) Set(slot int, v value.Value) {
	e.values[slot] = v
}

func (e *Environment) GetAt(distance, slot int) value.Value {
	return e.ancestor(distance).values[slot]
}

func (e *Environment) AssignAt(distance, slot int, v value.Value) {
	e.ancestor(distance).values[slot] = v
}

func (e *Environment) ancestor(distance int) *Environment {
	env := e

	for range distance {
		env = env.enclosing
	}

	return env
}

// globalSlots interns global names process-wide, so a name has the same
// slot in every interpreter and slots stored in a shared AST stay valid.
var (
	globalSlotsMu sync.Mutex
	globalSlots   = map[string]int{}
)

// GlobalSlot returns the slot of the global named name.
func GlobalSlot(name string) int {
	globalSlotsMu.Lock()
	defer globalSlotsMu.Unlock()

	slot, ok := globalSlots[name]
	if !ok {
		slot = len(globalSlots)
		globalSlots[name] = slot
	}

	return slot
}

// Globals holds one interpreter's top-level variables, indexed by GlobalSlot.
type Globals struct {
	values  []value.Value
	defined []bool
}

func NewGlobals() *Globals {
	return &Globals{}
}

func (g *Globals) Define(slot int, v value.Value) {
	if slot >= len(g.values) {
		g.values = append(g.values, make([]value.Value, slot+1-len(g.values))...)
		g.defined = append(g.defined, make([]bool, slot+1-len(g.defined))...)
	}

	g.values[slot] = v
	g.defined[slot] = true
}

func (g *Globals) isDefined(slot int) bool {
	return slot < len(g.defined) && g.defined[slot]
}

func (g *Globals) Get(slot int, token *ast.Token) (value.Value, error) {
	if slot < len(g.defined) && g.defined[slot] {
		return g.values[slot], nil
	}

	return value.NilValue, undefined(*token)
}

func (g *Globals) Assign(slot int, token *ast.Token, v value.Value) error {
	if !g.isDefined(slot) {
		return undefined(*token)
	}

	g.values[slot] = v

	return nil
}

func undefined(token ast.Token) error {
	return errors.NewRuntimeError(token, fmt.Sprintf("Undefined variable '%s'.", token.Lexeme))
}
