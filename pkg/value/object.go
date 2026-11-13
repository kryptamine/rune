package value

import (
	"iter"
)

// Obj is the table of an Object: string keys in the order they were
// first added. Small tables are scanned; larger ones keep an index.
type Obj struct {
	keys  []string
	vals  []Value
	index map[string]int
}

const indexedObj = 8

func NewObj(size int) *Obj {
	return &Obj{keys: make([]string, 0, size), vals: make([]Value, 0, size)}
}

func (o *Obj) Len() int { return len(o.keys) }

// All walks the entries in order; the table must not change meanwhile.
func (o *Obj) All() iter.Seq2[string, Value] {
	return func(yield func(string, Value) bool) {
		for i, k := range o.keys {
			if !yield(k, o.vals[i]) {
				return
			}
		}
	}
}

func (o *Obj) find(k string) int {
	if o.index != nil {
		if i, ok := o.index[k]; ok {
			return i
		}

		return -1
	}

	for i, x := range o.keys {
		if x == k {
			return i
		}
	}

	return -1
}

func (o *Obj) Get(k string) (Value, bool) {
	if i := o.find(k); i >= 0 {
		return o.vals[i], true
	}

	return NilValue, false
}

func (o *Obj) Has(k string) bool { return o.find(k) >= 0 }

func (o *Obj) Set(k string, v Value) {
	if i := o.find(k); i >= 0 {
		o.vals[i] = v
		return
	}

	o.keys, o.vals = append(o.keys, k), append(o.vals, v)

	switch n := len(o.keys); {
	case o.index != nil:
		o.index[k] = n - 1
	case n > indexedObj:
		o.reindex(0)
	}
}

// Delete removes k, keeping the order of the rest; found is whether it
// was there.
func (o *Obj) Delete(k string) (found bool) {
	i := o.find(k)
	if i < 0 {
		return false
	}

	last := len(o.keys) - 1
	o.keys = append(o.keys[:i], o.keys[i+1:]...)
	copy(o.vals[i:], o.vals[i+1:])
	o.vals[last] = NilValue
	o.vals = o.vals[:last]

	if o.index != nil {
		delete(o.index, k)
		o.reindex(i)
	}

	return true
}

func (o *Obj) reindex(from int) {
	if o.index == nil {
		o.index = make(map[string]int, len(o.keys))
	}

	for i := from; i < len(o.keys); i++ {
		o.index[o.keys[i]] = i
	}
}
