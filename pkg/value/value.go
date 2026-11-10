// Package value defines the runtime representation of Rune values.
//
// Value is a small tagged struct instead of a bare `any`, so numbers and
// booleans live inline and never allocate when produced or copied.
package value

import "fmt"

type Kind uint8

const (
	Nil Kind = iota
	Bool
	Number
	String
	Array
	Object
	Function
)

// Value is a Rune runtime value. The zero Value is nil.
//
// Numbers and booleans are stored in num (booleans as 0/1); strings,
// arrays ([]Value), objects (map[string]Value) and functions are stored in obj.
type Value struct {
	kind Kind
	num  float64
	obj  any
}

var (
	NilValue = Value{}
	True     = Value{kind: Bool, num: 1}
	False    = Value{kind: Bool, num: 0}
)

func NewNumber(n float64) Value { return Value{kind: Number, num: n} }

func NewBool(b bool) Value {
	if b {
		return True
	}
	return False
}

func NewString(s string) Value { return Value{kind: String, obj: s} }

func NewArray(items []Value) Value { return Value{kind: Array, obj: items} }

func NewObject(fields map[string]Value) Value { return Value{kind: Object, obj: fields} }

// NewFunction wraps a callable; fn must print via fmt.Stringer.
func NewFunction(fn any) Value { return Value{kind: Function, obj: fn} }

func (v Value) Kind() Kind { return v.kind }

func (v Value) IsNil() bool    { return v.kind == Nil }
func (v Value) IsNumber() bool { return v.kind == Number }
func (v Value) IsString() bool { return v.kind == String }

// Num returns the number payload; only meaningful when Kind() == Number.
func (v Value) Num() float64 { return v.num }

// Bool returns the boolean payload; only meaningful when Kind() == Bool.
func (v Value) Bool() bool { return v.num != 0 }

// Str returns the string payload; only meaningful when Kind() == String.
func (v Value) Str() string { return v.obj.(string) }

// Arr returns the array payload; only meaningful when Kind() == Array.
func (v Value) Arr() []Value { return v.obj.([]Value) }

// Obj returns the object payload; only meaningful when Kind() == Object.
func (v Value) Obj() map[string]Value { return v.obj.(map[string]Value) }

// Fn returns the function payload; only meaningful when Kind() == Function.
func (v Value) Fn() any { return v.obj }

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
		return NewNumber(float64(v))
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
