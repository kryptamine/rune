package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
)

// BoundMethod is a method read off its receiver, as in p.len.
type BoundMethod struct {
	recv value.Value
	fn   *Closure
}

func (b *BoundMethod) String() string { return b.fn.String() }

// methodsOf returns the methods of v's type, and the type's name.
func (p *Interpreter) methodsOf(v value.Value) ([]value.Method, string) {
	switch v.Kind() {
	case value.Struct:
		return v.Shape().Methods, v.Shape().Name
	case value.Type:
		return v.TypeShape().Methods, v.TypeShape().Name
	}

	return p.kinds[v.Kind()], helpers.TypeName(v)
}

// own returns the field or property of t that key names, which comes
// before any method of that name.
func own(tok *ast.Token, t, key value.Value) (value.Value, bool) {
	if k := t.Kind(); k != value.Object && k != value.Struct {
		return value.NilValue, false
	}

	v, found, _ := lookup(tok, t, key)

	return v, found
}

// property reads t.key: its own field or property, else a method, else
// nil if optional and t has properties at all.
func (p *Interpreter) property(tok *ast.Token, t, key value.Value, optional bool) (value.Value, error) {
	if v, ok := own(tok, t, key); ok {
		return v, nil
	}

	ms, owner := p.methodsOf(t)
	if m := value.FindMethod(ms, key.Str()); m != nil {
		return p.methodValue(tok, t, m, owner)
	}

	if k := t.Kind(); optional && (k == value.Object || k == value.Struct) {
		return value.NilValue, nil
	}

	return value.NilValue, missingMember(tok, t, key.Str())
}

// methodValue is m read off t: bound to an instance, or off a struct type
// the plain function, which takes the instance first.
func (p *Interpreter) methodValue(tok *ast.Token, t value.Value, m *value.Method, owner string) (value.Value, error) {
	switch {
	case t.Kind() == value.Type:
		return p.methods[m.ID], nil
	case !m.Self:
		return value.NilValue, staticMethod(tok, owner, m.Name)
	case m.Mut:
		return value.NilValue, errors.NewRuntimeError(errors.TypeError, *tok, fmt.Sprintf(
			"Cannot take mutating method '%s' of %s as a value.", m.Name, owner))
	}

	return value.NewFunction(&BoundMethod{recv: t, fn: (*Closure)(p.methods[m.ID].Ptr())}), nil
}

// invoke calls t.K[C](args) for the receiver t at R[A+1]: a function in
// its own field or property, or off a struct type, without t; a method
// with it.
func (p *Interpreter) invoke(fr *frame, in Instr, tok *ast.Token) error {
	at, n := fr.base+in.a(), in.b()
	t, key := p.stack[at+1], fr.cl.proto.consts[in.c()]

	if v, ok := own(tok, t, key); ok {
		p.stack[at] = v
		return p.callWithout(at, n, tok)
	}

	ms, owner := p.methodsOf(t)

	m := value.FindMethod(ms, key.Str())
	switch {
	case m == nil:
		return missingMember(tok, t, key.Str())
	case t.Kind() == value.Type:
		p.stack[at] = p.methods[m.ID]
		return p.callWithout(at, n, tok)
	case !m.Self:
		return staticMethod(tok, owner, m.Name)
	case m.Mut && !in.y():
		return errors.NewRuntimeError(errors.TypeError, *tok, fmt.Sprintf(
			"Cannot call mutating method '%s' of %s: the receiver is not a mutable variable, element or property.", m.Name, owner))
	}

	fn := p.methods[m.ID]
	if f := (*Closure)(fn.Ptr()); !f.proto.accepts(n + 1) {
		return argumentError(f.proto, n+1, 1, tok)
	}

	// A mutating method resumes past the jump over its write-back.
	if m.Mut {
		fr.pc++
	}

	p.stack[at] = fn

	return p.callAt(at, n+1, tok)
}

// callWithout calls the function at stack[at] with the n arguments after
// the receiver.
func (p *Interpreter) callWithout(at, n int, tok *ast.Token) error {
	copy(p.stack[at+1:], p.stack[at+2:at+2+n])
	return p.callAt(at, n, tok)
}

func missingMember(tok *ast.Token, t value.Value, name string) error {
	switch t.Kind() {
	case value.Object:
		return undefinedProperty(tok, name)
	case value.Struct:
		return undefinedField(tok, t.Shape(), name)
	case value.Type:
		return errors.NewRuntimeError(errors.KeyError, *tok, fmt.Sprintf("%s has no method '%s'.", t.TypeShape().Name, name))
	}

	return propertyError("read", tok, t)
}

func staticMethod(tok *ast.Token, owner, name string) error {
	return errors.NewRuntimeError(errors.TypeError, *tok, fmt.Sprintf(
		"'%s' is a static method of %s; call it as %s.%s().", name, owner, owner, name))
}
