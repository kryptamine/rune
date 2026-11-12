package value

import (
	"encoding/binary"
	"math"
	"unsafe"
)

// HashMap is the table of a Map or a Set, which keeps its keys in the
// order they were added; a Set's values are nil.
//
// A deleted entry stays as a hole, keys[i] undefined, until an insert
// compacts the table; inserting a key bumps version, so a walk can tell
// its positions went stale.
type HashMap struct {
	keys    []Value
	vals    []Value
	index   map[hashKey]int
	holes   int
	version int
}

// hashKey identifies a key: equal keys, 1 and 1.0 included, have one.
// Floats that are not integers keep their bits in n; structs and bytes
// are encoded in s.
type hashKey struct {
	tag uint8
	n   uint64
	s   string
	p   unsafe.Pointer
}

const (
	tagNil uint8 = iota
	tagBool
	tagInt
	tagFloat
	tagString
	tagBytes
	tagRef
	tagStruct
)

func NewHashMap(size int) *HashMap {
	return &HashMap{index: make(map[hashKey]int, size)}
}

// Unhashable is the error of a key no table can hold, What.
type Unhashable struct{ What string }

func (e *Unhashable) Error() string { return "Cannot use " + e.What + " as a key." }

func keyOf(k Value) (hashKey, error) {
	switch k.Kind() {
	case Nil:
		return hashKey{tag: tagNil}, nil
	case Bool:
		return hashKey{tag: tagBool, n: k.n}, nil
	case Number:
		if k.IsInt() {
			return hashKey{tag: tagInt, n: k.n}, nil
		}

		f := k.Float()
		if f != f {
			return hashKey{}, &Unhashable{"NaN"}
		}

		if i, ok := FloatToInt(f); ok {
			return hashKey{tag: tagInt, n: uint64(i)}, nil
		}

		return hashKey{tag: tagFloat, n: math.Float64bits(f)}, nil
	case String:
		return hashKey{tag: tagString, s: k.Str()}, nil
	case Bytes:
		return hashKey{tag: tagBytes, s: string(k.Bytes())}, nil
	case Function, Type:
		return hashKey{tag: tagRef, p: k.p, n: k.n}, nil
	case Struct:
		buf, err := appendStructKey(nil, k, 0)
		return hashKey{tag: tagStruct, s: string(buf)}, err
	}

	return hashKey{}, unhashable(k)
}

func unhashable(k Value) error { return &Unhashable{kindNames[k.Kind()]} }

var kindNames = [NumKinds]string{
	Array: "an array", Object: "an object", Map: "a map", Set: "a set", Stream: "a stream",
	Internal: "an internal value", Undefined: "an undefined value",
}

// maxKeyDepth bounds the nesting of a struct key.
const maxKeyDepth = 100

// appendStructKey encodes a struct as its shape and the keys of its
// fields; shapes are compared by address, which the keys keep alive.
func appendStructKey(buf []byte, v Value, depth int) ([]byte, error) {
	if depth > maxKeyDepth {
		return buf, &Unhashable{"a struct nested so deeply"}
	}

	buf = binary.LittleEndian.AppendUint64(buf, uint64(uintptr(unsafe.Pointer(v.Shape()))))

	for _, f := range v.Fields() {
		if f.Kind() == Struct {
			var err error
			if buf, err = appendStructKey(append(buf, tagStruct), f, depth+1); err != nil {
				return buf, err
			}

			continue
		}

		k, err := keyOf(f)
		if err != nil {
			return buf, &Unhashable{"a struct holding " + err.(*Unhashable).What}
		}

		buf = append(buf, k.tag)
		buf = binary.LittleEndian.AppendUint64(buf, k.n)
		buf = binary.LittleEndian.AppendUint64(buf, uint64(uintptr(k.p)))
		buf = binary.LittleEndian.AppendUint64(buf, uint64(len(k.s)))
		buf = append(buf, k.s...)
	}

	return buf, nil
}

func (m *HashMap) Len() int { return len(m.index) }

// Get returns the value of key k, and whether there is one.
func (m *HashMap) Get(k Value) (Value, bool, error) {
	hk, err := keyOf(k)
	if err != nil {
		return NilValue, false, err
	}

	i, ok := m.index[hk]
	if !ok {
		return NilValue, false, nil
	}

	return m.vals[i], true, nil
}

// Set sets key k to v.
func (m *HashMap) Set(k, v Value) error {
	hk, err := keyOf(k)
	if err != nil {
		return err
	}

	if i, ok := m.index[hk]; ok {
		m.vals[i] = v
		return nil
	}

	if m.holes > 16 && m.holes > len(m.index) {
		m.compact()
	}

	m.version++
	m.index[hk] = len(m.keys)
	m.keys = append(m.keys, k)
	m.vals = append(m.vals, v)

	return nil
}

// Delete removes key k and reports whether it was there.
func (m *HashMap) Delete(k Value) (bool, error) {
	hk, err := keyOf(k)
	if err != nil {
		return false, err
	}

	i, ok := m.index[hk]
	if !ok {
		return false, nil
	}

	delete(m.index, hk)

	m.keys[i], m.vals[i] = UndefinedValue, NilValue
	m.holes++

	return true, nil
}

func (m *HashMap) Clear() {
	m.version++
	clear(m.index)
	clear(m.keys)
	clear(m.vals)
	m.keys, m.vals, m.holes = m.keys[:0], m.vals[:0], 0
}

func (m *HashMap) compact() {
	keys, vals := m.keys[:0], m.vals[:0]

	for i, k := range m.keys {
		if k.IsUndefined() {
			continue
		}

		hk, _ := keyOf(k)
		m.index[hk] = len(keys)
		keys, vals = append(keys, k), append(vals, m.vals[i])
	}

	clear(m.keys[len(keys):])
	clear(m.vals[len(vals):])

	m.keys, m.vals, m.holes = keys, vals, 0
}

// Walk returns a function yielding the entries in order, which says stale
// once a key was added since the walk began.
func (m *HashMap) Walk() func() (k, v Value, ok, stale bool) {
	i, version := 0, m.version

	return func() (k, v Value, ok, stale bool) {
		if m.version != version {
			return k, v, false, true
		}

		for ; i < len(m.keys); i++ {
			if k := m.keys[i]; !k.IsUndefined() {
				i++
				return k, m.vals[i-1], true, false
			}
		}

		return k, v, false, false
	}
}

// Each calls f on each entry in order until it returns false.
func (m *HashMap) Each(f func(k, v Value) bool) {
	for i, k := range m.keys {
		if !k.IsUndefined() && !f(k, m.vals[i]) {
			return
		}
	}
}

// Clone copies the table, without its holes.
func (m *HashMap) Clone() *HashMap {
	c := NewHashMap(m.Len())
	m.Each(func(k, v Value) bool {
		c.Set(k, v)
		return true
	})

	return c
}
