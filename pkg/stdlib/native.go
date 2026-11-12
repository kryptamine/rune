// Package stdlib holds Rune's built-in functions and constants.
package stdlib

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
	"strings"
)

// Caller lets a native call back into a Rune function value.
type Caller interface {
	CallValue(fn value.Value, args []value.Value, token *ast.Token) (value.Value, error)
}

// Native is a built-in taking between min and max arguments (max -1: no limit).
type Native struct {
	name     string
	min, max int
	fn       func(c *call) value.Value
}

// Host is what natives run with; each interpreter owns one. It holds the
// program's stdio and a context Cancel ends, and reuses call frames.
type Host struct {
	Stdio  *Stdio
	ctx    context.Context
	cancel context.CancelFunc
	free   []*call
	// regexes caches the patterns compiled so far.
	regexes map[string]*regexp.Regexp
	streams map[*Stream]struct{}
}

func NewHost(stdio *Stdio) *Host {
	ctx, cancel := context.WithCancel(context.Background())
	return &Host{Stdio: stdio, ctx: ctx, cancel: cancel}
}

// Cancel stops what natives are waiting on, such as a running command;
// safe from any goroutine.
func (f *Host) Cancel() { f.cancel() }

// Call runs n; it must not keep args.
func (f *Host) Call(n *Native, caller Caller, args []value.Value, token *ast.Token) (value.Value, error) {
	return f.call(n, caller, args, token, 0)
}

// CallMethod runs the method n with its receiver as args[0]; errors count
// the arguments after it.
func (f *Host) CallMethod(n *Native, caller Caller, args []value.Value, token *ast.Token) (value.Value, error) {
	return f.call(n, caller, args, token, 1)
}

func (f *Host) call(n *Native, caller Caller, args []value.Value, token *ast.Token, self int) (value.Value, error) {
	c := f.get()
	*c = call{caller: caller, host: f, args: args, token: token, name: n.name, self: self}

	var v value.Value

	if len(args) < n.min || (n.max >= 0 && len(args) > n.max) {
		c.fail(errors.ArgumentError, "expects %s, got %d.", n.arityText(self), len(args)-self)
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

func (f *Host) get() *call {
	n := len(f.free)
	if n == 0 {
		return &call{}
	}

	c := f.free[n-1]
	f.free = f.free[:n-1]

	return c
}

func (n *Native) arityText(self int) string {
	lo, hi := n.min-self, n.max-self

	switch {
	case n.max < 0:
		return "at least " + helpers.Plural(lo, "argument")
	case lo == hi:
		return helpers.Plural(lo, "argument")
	default:
		return fmt.Sprintf("%d to %s", lo, helpers.Plural(hi, "argument"))
	}
}

func (n *Native) String() string { return "<native fn " + n.name + ">" }

// call is one invocation of a native. The first failure sticks: later
// accessors return zero values and the native's result is discarded.
type call struct {
	caller Caller
	host   *Host
	args   []value.Value
	token  *ast.Token
	name   string
	// self is 1 for a method, whose receiver is args[0].
	self int
	err  error
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
	return c.fail(errors.TypeError, "expects %s as argument %d, got %s.", want, i+1-c.self, helpers.TypeName(c.args[i]))
}

func (c *call) num(i int) float64 { return c.number(i).Num() }

// number is argument i, which must be a number, as is.
func (c *call) number(i int) value.Value {
	if !c.args[i].IsNumber() {
		c.typeError(i, "a number")
		return value.NewInt(0)
	}

	return c.args[i]
}

func (c *call) int(i int) int {
	if n, ok := value.ToInt(c.args[i]); ok {
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

func count(n int) value.Value { return value.NewInt(int64(n)) }

// Builtin is a global defined before a program runs.
type Builtin struct {
	Name  string
	Value value.Value
}

// Types are the built-in struct types, which programs see by name.
var Types = slices.Concat(errors.Kinds, []*value.Shape{FileInfo, ExecOptions, ExecResult, Match, HttpOptions, HttpResponse, Date})

// FieldNames returns each field name of Types as its one string.
func FieldNames() map[string]string {
	names := map[string]string{}
	for _, t := range Types {
		for _, f := range t.Fields {
			names[f] = f
		}
	}

	return names
}

// namespace groups built-ins under one global, an immutable struct of them.
// Its natives are named "<namespace>.<member>".
type namespace struct {
	name    string
	natives []*Native
	consts  []Builtin
}

var namespaces = []namespace{
	{"string", stringNatives, nil},
	{"math", mathNatives, []Builtin{
		{"PI", value.NewNumber(math.Pi)},
		{"INF", value.NewNumber(math.Inf(1))},
		{"MAX_INT", value.NewInt(math.MaxInt64)},
		{"MIN_INT", value.NewInt(math.MinInt64)},
	}},
	{"regex", regexNatives, nil},
	{"bytes", bytesNatives, nil},
	{"json", jsonNatives, nil},
	{"http", httpNatives, nil},
	{"time", timeNatives, nil},
	{"fs", fsNatives, nil},
	{"io", ioNatives, nil},
	{"os", osNatives, osConsts},
}

func (ns namespace) value() value.Value {
	members := make([]Builtin, 0, len(ns.natives)+len(ns.consts))

	for _, n := range ns.natives {
		prefix, member, _ := strings.Cut(n.name, ".")
		if prefix != ns.name {
			panic("stdlib: " + n.name + " is not in namespace " + ns.name)
		}

		members = append(members, Builtin{member, value.NewFunction(n)})
	}

	return NewNamespace(append(members, ns.consts...))
}

// NewNamespace makes a namespace of members, an immutable struct.
func NewNamespace(members []Builtin) value.Value {
	decls := make([]value.Field, len(members))

	for i, m := range members {
		decls[i] = value.Field{Name: m.Name, Required: true}
	}

	shape := value.NewShape("namespace", decls)
	shape.Namespace = true

	v, fields := value.NewStruct(shape)
	for i, m := range members {
		fields[i] = m.Value
	}

	return v
}

var namespaceValues = func() map[string]value.Value {
	m := make(map[string]value.Value, len(namespaces))
	for _, ns := range namespaces {
		m[ns.name] = ns.value()
	}

	return m
}()

// Member returns member key of the namespace called name; isNamespace is
// false when no namespace has that name.
func Member(name, key string) (v value.Value, found, isNamespace bool) {
	ns, ok := namespaceValues[name]
	if !ok {
		return value.NilValue, false, false
	}

	v, found = NamespaceMember(ns, key)

	return v, found, true
}

// NamespaceMember returns member key of the namespace ns.
func NamespaceMember(ns value.Value, key string) (value.Value, bool) {
	i := ns.Shape().Index(key)
	if i < 0 {
		return value.NilValue, false
	}

	return ns.Fields()[i], true
}

// Owner returns the first namespace that has member key, or "".
func Owner(key string) string {
	for _, ns := range namespaces {
		if namespaceValues[ns.name].Shape().Index(key) >= 0 {
			return ns.name
		}
	}

	return ""
}

// Method is a method of a built-in type, whose native takes the receiver
// first. An in-place one changes the receiver.
type Method struct {
	Kind    value.Kind
	Name    string
	Fn      value.Value
	InPlace bool
}

// Methods are the methods of the built-in types.
var Methods = func() (out []Method) {
	add := func(kind value.Kind, inPlace bool, natives ...[]*Native) {
		for _, n := range slices.Concat(natives...) {
			_, name, _ := strings.Cut(n.name, ".")
			out = append(out, Method{kind, name, value.NewFunction(n), inPlace})
		}
	}

	add(value.Array, false, arrayMethods, arraySequenceMethods, lenMethod("array"))
	add(value.Array, true, arrayInPlace)
	add(value.String, false, stringMethods, stringSequenceMethods, lenMethod("string"))
	add(value.Number, false, numberMethods)
	add(value.Object, false, objectMethods, lenMethod("object"))
	add(value.Object, true, objectInPlace)
	add(value.Map, false, mapMethods, lenMethod("Map"))
	add(value.Map, true, mapInPlace)
	add(value.Set, false, setMethods, lenMethod("Set"))
	add(value.Set, true, setInPlace)
	add(value.Bytes, false, bytesMethods, lenMethod("bytes"))
	add(value.Stream, false, streamMethods)

	return out
}()

var Builtins = builtins()

func builtins() []Builtin {
	var out []Builtin

	for _, n := range slices.Concat(preludeNatives, conversionNatives, collectionNatives) {
		out = append(out, Builtin{n.name, value.NewFunction(n)})
	}

	for _, ns := range namespaces {
		out = append(out, Builtin{ns.name, namespaceValues[ns.name]})
	}

	for _, t := range Types {
		out = append(out, Builtin{t.Name, value.NewType(t)})
	}

	return out
}
