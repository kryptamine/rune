// Package stdlib holds Rune's built-in functions and constants.
package stdlib

import (
	"fmt"
	"math"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
)

// Caller lets a native call back into a Rune function value.
type Caller interface {
	CallValue(fn value.Value, args []value.Value, token *ast.Token) (value.Value, error)
}

var inPlace = map[string]bool{"array.push": true, "array.pop": true, "array.sort": true}

// InPlace reports whether v is a built-in that mutates its first argument.
func InPlace(v value.Value) bool {
	if v.Kind() != value.Function {
		return false
	}

	n, ok := v.Fn().(*Native)

	return ok && inPlace[n.name]
}

// Native is a built-in taking between min and max arguments (max -1: no limit).
type Native struct {
	name     string
	min, max int
	fn       func(c *call) value.Value
}

// Frames reuses call frames across native calls; each interpreter owns one.
type Frames struct {
	free []*call
}

// Call runs n; it must not keep args.
func (f *Frames) Call(n *Native, caller Caller, args []value.Value, token *ast.Token) (value.Value, error) {
	c := f.get()
	*c = call{caller: caller, args: args, token: token, name: n.name}

	var v value.Value

	if len(args) < n.min || (n.max >= 0 && len(args) > n.max) {
		c.fail(errors.ArgumentError, "expects %s, got %d.", n.arityText(), len(args))
	} else {
		v = n.fn(c)
	}

	// Not zeroed: get overwrites it, and it only points at long-lived state.
	f.free = append(f.free, c)

	if c.err != nil {
		return value.NilValue, errors.WithFrame(c.err, errors.Frame{Name: n.name, File: token.File, Line: token.Line, Native: true})
	}

	return v, nil
}

func (f *Frames) get() *call {
	n := len(f.free)
	if n == 0 {
		return &call{}
	}

	c := f.free[n-1]
	f.free = f.free[:n-1]

	return c
}

func (n *Native) arityText() string {
	switch {
	case n.max < 0:
		return "at least " + plural(n.min, "argument")
	case n.min == n.max:
		return plural(n.min, "argument")
	default:
		return fmt.Sprintf("%d to %s", n.min, plural(n.max, "argument"))
	}
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}

	return fmt.Sprintf("%d %ss", n, word)
}

func (n *Native) String() string { return "<native fn " + n.name + ">" }

// call is one invocation of a native. The first failure sticks: later
// accessors return zero values and the native's result is discarded.
type call struct {
	caller Caller
	args   []value.Value
	token  *ast.Token
	name   string
	err    error
}

func (c *call) ok() bool { return c.err == nil }

func (c *call) has(i int) bool { return i < len(c.args) }

func (c *call) fail(kind *value.Shape, format string, a ...any) value.Value {
	if c.err == nil {
		c.err = errors.NewRuntimeError(kind, *c.token, c.name+"() "+fmt.Sprintf(format, a...))
	}

	return value.NilValue
}

func (c *call) typeError(i int, want string) value.Value {
	return c.fail(errors.TypeError, "expects %s as argument %d, got %s.", want, i+1, helpers.TypeName(c.args[i]))
}

func (c *call) num(i int) float64 {
	if !c.args[i].IsNumber() {
		c.typeError(i, "a number")
		return 0
	}

	return c.args[i].Num()
}

func (c *call) int(i int) int {
	if n := c.num(i); n == math.Trunc(n) && !math.IsInf(n, 0) {
		return int(n)
	}

	c.typeError(i, "an integer")

	return 0
}

func (c *call) str(i int) string {
	if !c.args[i].IsString() {
		c.typeError(i, "a string")
		return ""
	}

	return c.args[i].Str()
}

func (c *call) arr(i int) []value.Value {
	if c.args[i].Kind() != value.Array {
		c.typeError(i, "an array")
		return nil
	}

	return c.args[i].Arr()
}

func (c *call) fn(i int) value.Value {
	if c.args[i].Kind() != value.Function {
		return c.typeError(i, "a function")
	}

	return c.args[i]
}

// invoke calls fn with args, which may be reused once it returns.
func (c *call) invoke(fn value.Value, args []value.Value) value.Value {
	if c.err != nil {
		return value.NilValue
	}

	v, err := c.caller.CallValue(fn, args, c.token)
	if err != nil {
		c.err = err
	}

	return v
}

func count(n int) value.Value { return value.NewNumber(float64(n)) }

// Builtin is a global defined before a program runs.
type Builtin struct {
	Name  string
	Value value.Value
}

// namespace groups built-ins under one global, an immutable struct of them.
type namespace struct {
	name    string
	natives [][]*Native
	consts  []Builtin
}

var namespaces = []namespace{
	{"array", [][]*Native{arrayNatives, arraySequenceNatives}, nil},
	{"string", [][]*Native{stringNatives, stringSequenceNatives}, nil},
	{"math", [][]*Native{mathNatives}, []Builtin{
		{"PI", value.NewNumber(math.Pi)},
		{"INF", value.NewNumber(math.Inf(1))},
	}},
	{"json", [][]*Native{jsonNatives}, nil},
	{"http", [][]*Native{httpNatives}, nil},
	{"time", [][]*Native{timeNatives}, nil},
	{"fs", [][]*Native{fsNatives}, nil},
}

// value qualifies the natives' names with the namespace's, for errors.
func (ns namespace) value() value.Value {
	var decls []value.Field

	var members []value.Value

	for _, n := range slices.Concat(ns.natives...) {
		decls = append(decls, value.Field{Name: n.name, Required: true})
		members = append(members, value.NewFunction(n))
		n.name = ns.name + "." + n.name
	}

	for _, c := range ns.consts {
		decls = append(decls, value.Field{Name: c.Name, Required: true})
		members = append(members, c.Value)
	}

	v, fields := value.NewStruct(value.NewShape("namespace", decls))
	copy(fields, members)

	return v
}

var Builtins = builtins()

func builtins() []Builtin {
	var out []Builtin

	for _, n := range preludeNatives {
		out = append(out, Builtin{n.name, value.NewFunction(n)})
	}

	for _, ns := range namespaces {
		out = append(out, Builtin{ns.name, ns.value()})
	}

	for _, t := range Types {
		out = append(out, Builtin{t.Name, value.NewType(t)})
	}

	return out
}
