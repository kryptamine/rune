package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/stdlib"
	"rune/pkg/value"
)

// BoundMethod is a method read off its receiver, as in p.len.
type BoundMethod struct {
	recv value.Value
	fn   *Closure
}

func (b *BoundMethod) String() string { return b.fn.String() }

// BoundNative is a built-in method read off its receiver, as in m.keys.
type BoundNative struct {
	recv value.Value
	fn   *stdlib.Native
}

func (b *BoundNative) String() string { return b.fn.String() }

// methodsOf returns the methods of v's type, and the type's name.
func (p *Interpreter) methodsOf(v value.Value) ([]value.Method, string) {
	switch v.Kind() {
	case value.Struct:
		return v.Shape().Methods, v.Shape().Name
	case value.Type:
		return v.TypeShape().Methods, v.TypeShape().Name
	case value.Enum:
		return v.Variant().Shape.Methods, v.Variant().Shape.Name
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

// property reads t.key: its own field or property, else a method; one an
// object or struct does not have is missing, unless that is nil.
func (p *Interpreter) property(tok *ast.Token, t, key value.Value, missing *value.Value) (value.Value, error) {
	if v, ok := own(tok, t, key); ok {
		return v, nil
	}

	if v, ok := enumProperty(t, key.Str()); ok {
		return v, nil
	}

	ms, owner := p.methodsOf(t)
	if m := value.FindMethod(ms, key.Str()); m != nil {
		return p.methodValue(tok, t, m, owner)
	}

	if k := t.Kind(); missing != nil && (k == value.Object || k == value.Struct) {
		return *missing, nil
	}

	return value.NilValue, missingMember(tok, t, key.Str(), ms)
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
	case m.InPlace:
		return value.NilValue, errors.NewRuntimeError(errors.TypeError, *tok, fmt.Sprintf(
			"Cannot take in-place method '%s' of %s as a value.", m.Name, owner))
	}

	value.Share(t)

	if nat, ok := p.methods[m.ID].Fn().(*stdlib.Native); ok {
		return value.NewFunction(&BoundNative{recv: t, fn: nat}), nil
	}

	return value.NewFunction(&BoundMethod{recv: t, fn: (*Closure)(p.methods[m.ID].Ptr())}), nil
}

// invoke calls the method t.K[C](args) for the receiver t at R[A+1]:
// off a struct type without t, else with it. A function in a field or
// property is called as (t.k)(args).
func (p *Interpreter) invoke(fr *frame, in Instr, tok *ast.Token) error {
	at, n := fr.base+in.a(), in.b()
	t, key := p.stack[at+1], fr.cl.proto.consts[in.c()]

	ms, owner := p.methodsOf(t)

	m := value.FindMethod(ms, key.Str())
	switch {
	case m == nil && isValuesCall(t, key.Str()):
		v, err := variants(tok, t.TypeShape(), n)
		if err == nil {
			p.stack[at] = v
		}

		return err
	case m == nil:
		if _, ok := own(tok, t, key); ok {
			return notMethod(tok, t, owner, key.Str())
		}

		return noMethod(tok, t, owner, key.Str(), ms)
	case t.Kind() == value.Type:
		p.stack[at] = p.methods[m.ID]
		return p.callWithout(at, n, tok)
	case !m.Self:
		return staticMethod(tok, owner, m.Name)
	case m.Mut && !in.y():
		return errors.NewRuntimeError(errors.TypeError, *tok, mutatingCallError(m.Name, owner))
	case m.InPlace && in.x()&1 != 0:
		return errors.NewRuntimeError(errors.TypeError, *tok, fmt.Sprintf(
			"Cannot call in-place method '%s' of %s: its receiver is in an immutable variable.", m.Name, owner))
	}

	fn := p.methods[m.ID]
	if nat, ok := fn.Fn().(*stdlib.Native); ok {
		// An in-place method on a shared receiver changes a copy of its
		// own, which then goes back where the receiver was read from.
		copied := m.InPlace && t.Shared()
		if copied {
			p.stack[at+1] = value.Unshare(t)
		}

		caller := len(p.frames) - 1
		if err := p.callNativeMethod(at, n, nat, tok); err != nil {
			return err
		}

		if copied && in.y() {
			p.frames[caller].skipToWriteBacks(1)
		}

		return nil
	}

	cl := (*Closure)(fn.Ptr())
	if !cl.proto.accepts(n + 1) {
		return argumentError(cl.proto, n+1, 1, tok)
	}

	p.stack[at] = fn

	return p.callAt(at, n+1, tok, fr.siteInout(in))
}

// callWithout calls the function at stack[at] with the n arguments after
// the receiver.
func (p *Interpreter) callWithout(at, n int, tok *ast.Token) error {
	copy(p.stack[at+1:], p.stack[at+2:at+2+n])
	return p.callAt(at, n, tok, 0)
}

func missingMember(tok *ast.Token, t value.Value, name string, ms []value.Method) error {
	switch t.Kind() {
	case value.Object:
		return undefinedProperty(tok, t, name)
	case value.Struct:
		return undefinedField(tok, t.Shape(), name)
	case value.Type:
		return noMethod(tok, t, t.TypeShape().Name, name, ms)
	}

	return errors.WithHint(propertyError("read", tok, t), memberHint(t, name, ms))
}

// mutatingCallError is the error of a mut self method called on a value
// no variable holds; owner is its type, if known.
func mutatingCallError(name, owner string) string {
	method := "'" + name + "'"
	if owner != "" {
		method += " of " + owner
	}

	return "Cannot call mutating method " + method + ": the receiver is not a mutable variable, element or property."
}

func noMethod(tok *ast.Token, t value.Value, owner, name string, ms []value.Method) error {
	err := errors.NewRuntimeError(errors.TypeError, *tok, fmt.Sprintf("%s has no method '%s'.", owner, name))
	return errors.WithHint(err, memberHint(t, name, ms))
}

// notMethod is the error of t.name(args) for a function in a field or property.
func notMethod(tok *ast.Token, t value.Value, owner, name string) error {
	holder := helpers.If(t.Kind() == value.Struct, "field", "property")

	return errors.NewRuntimeError(errors.TypeError, *tok, fmt.Sprintf(
		"%s has no method '%s'; call the function in its %s as (x.%s)().", owner, name, holder, name))
}

func staticMethod(tok *ast.Token, owner, name string) error {
	return errors.NewRuntimeError(errors.TypeError, *tok, fmt.Sprintf(
		"'%s' is a static method of %s; call it as %s.%s().", name, owner, owner, name))
}
