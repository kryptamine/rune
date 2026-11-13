package value

import (
	"hash/maphash"
	"math/bits"
	"unsafe"
)

// HashMap is the table of a Map or a Set, which keeps its keys in the
// order they were added; a Set's values are nil.
//
// slots is probed linearly from a key's hash and holds the index of its
// entry plus one, or zero. A deleted entry stays as a hole, keys[i]
// undefined, until an insert rebuilds the table; inserting a key bumps
// version, so a walk can tell its positions went stale.
type HashMap struct {
	keys    []Value
	vals    []Value
	hashes  []uint64
	slots   []uint32
	shift   uint8
	holes   int
	version int
}

const minSlots = 8

func NewHashMap(size int) *HashMap {
	m := &HashMap{}

	if size > 0 {
		m.keys = make([]Value, 0, size)
		m.vals = make([]Value, 0, size)
		m.hashes = make([]uint64, 0, size)
		m.rebuild(slotsFor(size))
	}

	return m
}

// Unhashable is the error of a key no table can hold, What.
type Unhashable struct{ What string }

func (e *Unhashable) Error() string { return "Cannot use " + e.What + " as a key." }

func unhashable(k Value) error { return &Unhashable{kindNames[k.Kind()]} }

var kindNames = [NumKinds]string{
	Array: "an array", Object: "an object", Map: "a map", Set: "a set", Stream: "a stream",
	Internal: "an internal value", Undefined: "an undefined value",
}

// maxKeyDepth bounds the nesting of a struct key.
const maxKeyDepth = 100

var hashSeed = maphash.MakeSeed()

const (
	hashMul   = 0x9e3779b97f4a7c15
	floatSalt = 0x51afd7ed558ccd
	bytesSalt = 0x2545f4914f6cdd1d
)

func mix(x uint64) uint64 {
	x *= hashMul
	return x ^ x>>32
}

// hashOf hashes a key: equal keys, 1 and 1.0 included, hash alike.
func hashOf(k Value) (uint64, error) {
	switch {
	case k.IsInt():
		return mix(k.n), nil
	case k.IsString():
		return maphash.String(hashSeed, k.Str()), nil
	}

	switch k.Kind() {
	case Nil:
		return 1, nil
	case Bool:
		return 2 + k.n, nil
	case Number:
		f := k.Float()
		if f != f {
			return 0, &Unhashable{"NaN"}
		}

		if i, ok := FloatToInt(f); ok {
			return mix(uint64(i)), nil
		}

		return mix(k.n ^ floatSalt), nil
	case Bytes:
		return maphash.Bytes(hashSeed, k.Bytes()) ^ bytesSalt, nil
	case Function, Type, Enum:
		return mix(uint64(uintptr(k.p)) ^ bits.RotateLeft64(k.n, 32)), nil
	case Struct:
		return hashStruct(k, 0)
	}

	return 0, unhashable(k)
}

// hashStruct hashes a struct by its shape and the hashes of its fields.
func hashStruct(v Value, depth int) (uint64, error) {
	if depth > maxKeyDepth {
		return 0, &Unhashable{"a struct nested so deeply"}
	}

	h := uint64(uintptr(unsafe.Pointer(v.Shape())))

	for _, f := range v.Fields() {
		var (
			x   uint64
			err error
		)

		if f.Kind() == Struct {
			if x, err = hashStruct(f, depth+1); err != nil {
				return 0, err
			}
		} else if x, err = hashOf(f); err != nil {
			return 0, &Unhashable{"a struct holding " + err.(*Unhashable).What}
		}

		h = mix(h ^ x)
	}

	return h, nil
}

// sameKey reports whether two hashable keys are equal.
func sameKey(a, b Value) bool {
	if a.IsInt() && b.IsInt() {
		return a.n == b.n
	}

	k := a.Kind()
	if k != b.Kind() {
		return false
	}

	switch k {
	case Nil:
		return true
	case Number:
		x, xInt := ToInt(a)
		y, yInt := ToInt(b)

		if xInt || yInt {
			return xInt && yInt && x == y
		}

		return a.n == b.n
	case String:
		return a.Str() == b.Str()
	case Bytes:
		return string(a.Bytes()) == string(b.Bytes())
	case Struct:
		if a.Shape() != b.Shape() {
			return false
		}

		bf := b.Fields()
		for i, f := range a.Fields() {
			if !sameKey(f, bf[i]) {
				return false
			}
		}

		return true
	}

	return a.p == b.p && a.n == b.n
}

func (m *HashMap) Len() int { return len(m.keys) - m.holes }

// find returns the entry of key k, whose hash is h, or -1.
func (m *HashMap) find(k Value, h uint64) int {
	if len(m.slots) == 0 {
		return -1
	}

	mask := uint64(len(m.slots) - 1)

	for i := h >> m.shift; ; i = (i + 1) & mask {
		s := m.slots[i]
		if s == 0 {
			return -1
		}

		if e := int(s - 1); m.hashes[e] == h && sameKey(m.keys[e], k) {
			return e
		}
	}
}

// Get returns the value of key k, and whether there is one.
func (m *HashMap) Get(k Value) (Value, bool, error) {
	h, err := hashOf(k)
	if err != nil {
		return NilValue, false, err
	}

	e := m.find(k, h)
	if e < 0 {
		return NilValue, false, nil
	}

	return m.vals[e], true, nil
}

// Set sets key k to v.
func (m *HashMap) Set(k, v Value) error {
	h, err := hashOf(k)
	if err != nil {
		return err
	}

	if e := m.find(k, h); e >= 0 {
		m.vals[e] = v
		return nil
	}

	m.add(k, v, h)

	return nil
}

// add appends an entry for k, which the table does not have.
func (m *HashMap) add(k, v Value, h uint64) {
	if n := m.Len(); (len(m.keys)+1)*4 > len(m.slots)*3 || m.holes > 16 && m.holes > n {
		m.rebuild(slotsFor(n + 1))
	}

	m.version++
	m.keys = append(m.keys, k)
	m.vals = append(m.vals, v)
	m.hashes = append(m.hashes, h)
	m.place(len(m.keys)-1, h)
}

// place points the first free slot from h at entry e.
func (m *HashMap) place(e int, h uint64) {
	mask := uint64(len(m.slots) - 1)

	i := h >> m.shift
	for m.slots[i] != 0 {
		i = (i + 1) & mask
	}

	m.slots[i] = uint32(e + 1)
}

// slotsFor is the number of slots that n entries half fill at most.
func slotsFor(n int) int {
	size := minSlots
	for size < 2*n {
		size <<= 1
	}

	return size
}

// rebuild drops the holes and indexes the entries in size slots.
func (m *HashMap) rebuild(size int) {
	if m.holes > 0 {
		n := 0

		for i, k := range m.keys {
			if k.IsUndefined() {
				continue
			}

			m.keys[n], m.vals[n], m.hashes[n] = k, m.vals[i], m.hashes[i]
			n++
		}

		clear(m.keys[n:])
		clear(m.vals[n:])

		m.keys, m.vals, m.hashes, m.holes = m.keys[:n], m.vals[:n], m.hashes[:n], 0
	}

	if size == len(m.slots) {
		clear(m.slots)
	} else {
		m.slots = make([]uint32, size)
		m.shift = uint8(64 - bits.TrailingZeros(uint(size)))
	}

	for e, h := range m.hashes {
		m.place(e, h)
	}
}

// Delete removes key k and reports whether it was there.
func (m *HashMap) Delete(k Value) (bool, error) {
	h, err := hashOf(k)
	if err != nil {
		return false, err
	}

	e := m.find(k, h)
	if e < 0 {
		return false, nil
	}

	m.keys[e], m.vals[e] = UndefinedValue, NilValue
	m.holes++

	return true, nil
}

func (m *HashMap) Clear() {
	m.version++
	clear(m.slots)
	clear(m.keys)
	clear(m.vals)
	m.keys, m.vals, m.hashes, m.holes = m.keys[:0], m.vals[:0], m.hashes[:0], 0
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

	for i, k := range m.keys {
		if !k.IsUndefined() {
			c.add(k, m.vals[i], m.hashes[i])
		}
	}

	return c
}
