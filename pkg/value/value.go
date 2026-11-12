// Package value defines the runtime representation of Rune values.
package value

import (
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

type Kind uint8

const (
	Nil Kind = iota
	Bool
	Number
	String
	Array
	Object
	Function
	// Undefined marks a variable not yet defined; never a language value.
	Undefined
	// Internal carries interpreter state (cells, iterators) in a register.
	Internal
	Struct
	// Type is a struct type, the value of its name.
	Type
	Map
	Set
	Bytes
	// Stream is an open file or a body being read.
	Stream
	NumKinds
)

// Value is a Rune runtime value in two words; the zero Value is nil.
//
// p tells the kind apart: nil for nil, a tag address for booleans,
// floats and integers (whose payload is n), and otherwise the heap object, with the
// kind in the top byte of n. A string's p is its data and n holds its
// length; with bufFlag set, p is the *strBuf it was concatenated into.
// Functions and internal values keep the data word of an interface in p
// and its type's index in the low byte of n.
type Value struct {
	p unsafe.Pointer
	n uint64
}

var tags [4]byte

// The tags are functions so they compile to constant addresses.
func boolTag() unsafe.Pointer  { return unsafe.Pointer(&tags[0]) }
func numTag() unsafe.Pointer   { return unsafe.Pointer(&tags[1]) }
func otherTag() unsafe.Pointer { return unsafe.Pointer(&tags[2]) }
func intTag() unsafe.Pointer   { return unsafe.Pointer(&tags[3]) }

const (
	kindShift = 56
	bufFlag   = 1 << 55
	lenMask   = bufFlag - 1
)

var (
	NilValue       = Value{}
	True           = Value{p: boolTag(), n: 1}
	False          = Value{p: boolTag(), n: 0}
	UndefinedValue = Value{p: otherTag(), n: uint64(Undefined) << kindShift}
)

// NewNumber returns a float; NewInt an integer. Both are of kind Number.
func NewNumber(n float64) Value { return Value{p: numTag(), n: *(*uint64)(unsafe.Pointer(&n))} }

func NewInt(i int64) Value { return Value{p: intTag(), n: uint64(i)} }

func NewBool(b bool) Value {
	if b {
		return True
	}

	return False
}

func NewString(s string) Value {
	p := unsafe.Pointer(unsafe.StringData(s))
	if p == nil {
		p = otherTag()
	}

	return Value{p: p, n: uint64(String)<<kindShift | uint64(len(s))}
}

func NewArray(items []Value) Value {
	return Value{p: unsafe.Pointer(&items), n: uint64(Array) << kindShift}
}

func NewObject(fields map[string]Value) Value {
	if fields == nil {
		fields = map[string]Value{}
	}

	return Value{p: *(*unsafe.Pointer)(unsafe.Pointer(&fields)), n: uint64(Object) << kindShift}
}

// Field declares a struct field; Default is unused when Required.
type Field struct {
	Name     string
	Default  Value
	Required bool
}

// Shape describes a struct type: its name and fields in declaration order.
type Shape struct {
	Name   string
	Fields []string
	// Keys holds Fields as values, for iteration.
	Keys    []Value
	Methods []Method
	decls   []Field
}

// Method is a method of an impl block or a built-in type; ID indexes the
// program's methods. Without Self it is static; Mut takes mut self;
// InPlace, only built-in, changes its receiver.
type Method struct {
	Name               string
	ID                 int
	Self, Mut, InPlace bool
}

// FindMethod returns the method of ms named name, or nil.
func FindMethod(ms []Method, name string) *Method {
	for i := range ms {
		if ms[i].Name == name {
			return &ms[i]
		}
	}

	return nil
}

func NewShape(name string, decls []Field) *Shape {
	fields := make([]string, len(decls))
	keys := make([]Value, len(decls))

	for i, f := range decls {
		fields[i] = f.Name
		keys[i] = NewString(f.Name)
	}

	return &Shape{Name: name, Fields: fields, Keys: keys, decls: decls}
}

// Default returns the default of field i, if it has one; one that is
// Mutable is taken through Fresh.
func (s *Shape) Default(i int) (Value, bool) {
	f := s.decls[i]
	return f.Default, !f.Required
}

// Fresh deep-copies the arrays and objects of a constant; its maps and
// sets, being empty, are copied as they are.
func Fresh(v Value) Value {
	switch v.Kind() {
	case Array:
		items := make([]Value, len(v.Arr()))
		for i, x := range v.Arr() {
			items[i] = Fresh(x)
		}
		return NewArray(items)
	case Object:
		fields := make(map[string]Value, len(v.Obj()))
		for k, x := range v.Obj() {
			fields[k] = Fresh(x)
		}
		return NewObject(fields)
	case Map:
		return NewMap(v.HashMap().Clone())
	case Set:
		return NewSet(v.HashMap().Clone())
	}

	return v
}

// Mutable reports whether v is an array, object, map or set.
func Mutable(v Value) bool {
	switch v.Kind() {
	case Array, Object, Map, Set:
		return true
	}

	return false
}

// Index returns the slot of field name, or -1.
func (s *Shape) Index(name string) int {
	for i, f := range s.Fields {
		if f == name {
			return i
		}
	}

	return -1
}

// Assign copies into fields the keys of obj that name fields, walking the
// fields so no map order shows; extra is the least key naming none.
func (s *Shape) Assign(fields []Value, obj map[string]Value) (extra string, ok bool) {
	given := 0

	for i, name := range s.Fields {
		if v, found := obj[name]; found {
			fields[i] = v
			given++
		}
	}

	if given == len(obj) {
		return "", true
	}

	first := true
	for k := range obj {
		if s.Index(k) < 0 && (first || k < extra) {
			extra, first = k, false
		}
	}

	return extra, false
}

// Fill gives each undefined field a fresh copy of its default; missing is
// the first that has none.
func (s *Shape) Fill(fields []Value) (missing string, ok bool) {
	for i, f := range fields {
		switch {
		case !f.IsUndefined():
		case s.decls[i].Required:
			return s.Fields[i], false
		default:
			fields[i] = Fresh(s.decls[i].Default)
		}
	}

	return "", true
}

// InternedMethod is FindMethod for a key sharing the name's storage, as
// the compiler arranges; a miss does not mean there is no such method.
func InternedMethod(ms []Method, key Value) *Method {
	for i := range ms {
		if unsafe.Pointer(unsafe.StringData(ms[i].Name)) == key.p {
			return &ms[i]
		}
	}

	return nil
}

// NewStruct allocates an instance of s and returns it with its fields to
// fill. Slot 0 of the allocation holds the shape, the fields follow it.
func NewStruct(s *Shape) (Value, []Value) {
	slots := make([]Value, 1+len(s.Fields))
	slots[0].p = unsafe.Pointer(s)

	return Value{p: unsafe.Pointer(&slots[0]), n: uint64(Struct) << kindShift}, slots[1:]
}

func NewType(s *Shape) Value { return Value{p: unsafe.Pointer(s), n: uint64(Type) << kindShift} }

// NewFunction wraps a callable; fn must print via fmt.Stringer.
func NewFunction(fn any) Value { return boxed(Function, fn) }

func NewInternal(x any) Value { return boxed(Internal, x) }

func (v Value) Kind() Kind {
	switch v.p {
	case nil:
		return Nil
	case boolTag():
		return Bool
	case numTag(), intTag():
		return Number
	}

	return Kind(v.n >> kindShift)
}

func (v Value) IsNil() bool    { return v.p == nil }
func (v Value) IsNumber() bool { return v.p == numTag() || v.p == intTag() }
func (v Value) IsInt() bool    { return v.p == intTag() }
func (v Value) IsFloat() bool  { return v.p == numTag() }
func (v Value) IsString() bool { return v.Kind() == String }

func (v Value) IsUndefined() bool {
	return v.p == otherTag() && v.n == uint64(Undefined)<<kindShift
}

// Truthy follows JS-like semantics: nil, false, 0 and "" are falsy,
// everything else is truthy.
func (v Value) Truthy() bool {
	switch v.p {
	case nil:
		return false
	case boolTag():
		return v.n != 0
	case numTag():
		return v.n<<1 != 0
	case intTag():
		return v.n != 0
	}

	return Kind(v.n>>kindShift) != String || v.n&lenMask != 0
}

// Num returns a number as a float; only meaningful when Kind() == Number.
func (v Value) Num() float64 {
	if v.p == intTag() {
		return float64(int64(v.n))
	}

	return v.Float()
}

// Float returns the payload of a float; Int that of an integer.
func (v Value) Float() float64 { return *(*float64)(unsafe.Pointer(&v.n)) }
func (v Value) Int() int64     { return int64(v.n) }

// Bool returns the boolean payload; only meaningful when Kind() == Bool.
func (v Value) Bool() bool { return v.n != 0 }

// Str returns the string payload; only meaningful when Kind() == String.
func (v Value) Str() string {
	n := int(v.n & lenMask)
	if v.n&bufFlag != 0 {
		return unsafe.String(unsafe.SliceData((*strBuf)(v.p).data), n)
	}

	return unsafe.String((*byte)(v.p), n)
}

// strBuf backs strings built by concatenation. Bytes below any string's
// length never change, so a string ending at len(data) may append in place.
type strBuf struct {
	data []byte
}

// minBuffered is the shortest concatenation result that gets a strBuf.
const minBuffered = 64

func buffered(b *strBuf, n int) Value {
	return Value{p: unsafe.Pointer(b), n: uint64(String)<<kindShift | bufFlag | uint64(n)}
}

// Concat joins two strings, appending in place when l ends its buffer, so
// building a string in a loop is linear instead of quadratic.
func Concat(l, r Value) Value {
	ls, rs := l.Str(), r.Str()
	n := len(ls) + len(rs)

	if l.n&bufFlag != 0 {
		if b := (*strBuf)(l.p); len(b.data) == len(ls) {
			b.data = append(b.data, rs...)
			return buffered(b, n)
		}
	}

	if n < minBuffered {
		return NewString(ls + rs)
	}

	b := &strBuf{data: make([]byte, 0, 2*n)}
	b.data = append(append(b.data, ls...), rs...)

	return buffered(b, n)
}

// Detach returns v without a concat buffer, so it is safe to share
// between interpreters.
func (v Value) Detach() Value {
	if v.Kind() == String && v.n&bufFlag != 0 {
		return NewString(v.Str())
	}

	return v
}

func NewMap(m *HashMap) Value { return Value{p: unsafe.Pointer(m), n: uint64(Map) << kindShift} }
func NewSet(m *HashMap) Value { return Value{p: unsafe.Pointer(m), n: uint64(Set) << kindShift} }

// HashMap returns the table of a map or a set.
func (v Value) HashMap() *HashMap { return (*HashMap)(v.p) }

// NewBytes wraps b, which must not change afterwards.
func NewBytes(b []byte) Value {
	p := unsafe.Pointer(unsafe.SliceData(b))
	if len(b) == 0 {
		p = otherTag()
	}

	return Value{p: p, n: uint64(Bytes)<<kindShift | uint64(len(b))}
}

// Bytes returns the payload of bytes, which must not be changed.
func (v Value) Bytes() []byte { return unsafe.Slice((*byte)(v.p), int(v.n&lenMask)) }

// NewStream wraps x, a stream of package stdlib.
func NewStream(x any) Value { return boxed(Stream, x) }

// Arr returns the array payload; only meaningful when Kind() == Array.
func (v Value) Arr() []Value { return *(*[]Value)(v.p) }

func (v Value) ArrRef() *[]Value { return (*[]Value)(v.p) }

// Obj returns the object payload; only meaningful when Kind() == Object.
func (v Value) Obj() map[string]Value { return *(*map[string]Value)(unsafe.Pointer(&v.p)) }

// Shape returns a struct's shape; only meaningful when Kind() == Struct.
func (v Value) Shape() *Shape { return (*Shape)((*Value)(v.p).p) }

// Fields returns a struct's fields in declaration order.
func (v Value) Fields() []Value {
	n := len(v.Shape().Fields)
	if n == 0 {
		return nil
	}

	return unsafe.Slice((*Value)(unsafe.Add(v.p, unsafe.Sizeof(Value{}))), n)
}

// InternedSlot returns the slot of the field named by key, matching the
// string data pointer: it finds the field only if key shares the name's
// storage, as the compiler arranges, so -1 does not mean there is none.
func (v Value) InternedSlot(key Value) int {
	for i, f := range v.Shape().Fields {
		if unsafe.Pointer(unsafe.StringData(f)) == key.p {
			return i
		}
	}

	return -1
}

// InternedField returns the field InternedSlot finds.
func (v Value) InternedField(key Value) (Value, bool) {
	if i := v.InternedSlot(key); i >= 0 {
		return v.Fields()[i], true
	}

	return NilValue, false
}

// With returns a copy of v with field slot set to x.
func (v Value) With(slot int, x Value) Value {
	c, fields := NewStruct(v.Shape())
	copy(fields, v.Fields())
	fields[slot] = x

	return c
}

// TypeShape returns the shape a struct type describes; only meaningful when Kind() == Type.
func (v Value) TypeShape() *Shape { return (*Shape)(v.p) }

// Fn returns the function payload; only meaningful when Kind() == Function.
func (v Value) Fn() any { return v.unbox() }

func (v Value) Internal() any { return v.unbox() }

// Ptr returns the pointer a function or internal value boxes, for a
// caller that knows its type from TypeID.
func (v Value) Ptr() unsafe.Pointer { return v.p }

// TypeID identifies the Go type a function or internal value boxes.
func (v Value) TypeID() uint8 { return uint8(v.n) }

// eface is the layout of an empty interface.
type eface struct {
	typ, data unsafe.Pointer
}

// boxTypes lists the types boxed so far; an index is never reused.
var (
	boxTypes   atomic.Pointer[[]unsafe.Pointer]
	boxTypesMu sync.Mutex
)

// TypeIDOf returns the TypeID of values boxing x's type.
func TypeIDOf(x any) uint8 { return typeID((*eface)(unsafe.Pointer(&x)).typ) }

func typeID(typ unsafe.Pointer) uint8 {
	if types := boxTypes.Load(); types != nil {
		for i, t := range *types {
			if t == typ {
				return uint8(i)
			}
		}
	}

	boxTypesMu.Lock()
	defer boxTypesMu.Unlock()

	var types []unsafe.Pointer
	if old := boxTypes.Load(); old != nil {
		types = *old
	}

	for i, t := range types {
		if t == typ {
			return uint8(i)
		}
	}

	if len(types) == 256 {
		panic("value: too many boxed types")
	}

	types = append(types[:len(types):len(types)], typ)
	boxTypes.Store(&types)

	return uint8(len(types) - 1)
}

func boxed(k Kind, x any) Value {
	e := (*eface)(unsafe.Pointer(&x))
	return Value{p: e.data, n: uint64(k)<<kindShift | uint64(typeID(e.typ))}
}

func (v Value) unbox() any {
	var x any

	e := (*eface)(unsafe.Pointer(&x))
	e.typ, e.data = (*boxTypes.Load())[uint8(v.n)], v.p

	return x
}

// FromGo converts a plain Go value (as produced by literals or encoding/json)
// into a Value, recursively converting slices and maps.
func FromGo(x any) Value {
	switch v := x.(type) {
	case nil:
		return NilValue
	case Value:
		return v
	case bool:
		return NewBool(v)
	case float64:
		return NewNumber(v)
	case int:
		return NewInt(int64(v))
	case int64:
		return NewInt(v)
	case string:
		return NewString(v)
	case []any:
		items := make([]Value, len(v))
		for i, item := range v {
			items[i] = FromGo(item)
		}
		return NewArray(items)
	case map[string]any:
		fields := make(map[string]Value, len(v))
		for k, item := range v {
			fields[k] = FromGo(item)
		}
		return NewObject(fields)
	default:
		panic(fmt.Sprintf("value.FromGo: unsupported type %T", x))
	}
}
