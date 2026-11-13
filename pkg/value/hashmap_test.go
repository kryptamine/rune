package value

import "testing"

func TestHashMapChurn(t *testing.T) {
	m := NewHashMap(0)

	for round := range 3 {
		for i := range 5000 {
			if err := m.Set(NewInt(int64(i)), NewInt(int64(i+round))); err != nil {
				t.Fatal(err)
			}
		}

		for i := 0; i < 5000; i += 2 {
			if found, _ := m.Delete(NewNumber(float64(i))); !found {
				t.Fatalf("round %d: %d not deleted", round, i)
			}
		}

		if m.Len() != 2500 {
			t.Fatalf("round %d: len %d", round, m.Len())
		}

		for i := range 5000 {
			v, found, _ := m.Get(NewInt(int64(i)))
			if found != (i%2 == 1) || found && v.Int() != int64(i+round) {
				t.Fatalf("round %d: key %d: got %v, %v", round, i, v.Int(), found)
			}
		}
	}

	prev := int64(-1)
	m.Each(func(k, _ Value) bool {
		if k.Int() <= prev {
			t.Fatalf("order: %d after %d", k.Int(), prev)
		}

		prev = k.Int()

		return true
	})

	c := m.Clone()
	m.Clear()

	if m.Len() != 0 || c.Len() != 2500 {
		t.Fatalf("len %d, clone %d", m.Len(), c.Len())
	}
}

func TestHashMapKeys(t *testing.T) {
	shape := NewShape("P", []Field{{Name: "x", Required: true}})
	point := func(x Value) Value {
		v, fields := NewStruct(shape)
		fields[0] = x

		return v
	}

	m := NewHashMap(0)
	keys := []Value{NilValue, True, NewInt(1), NewNumber(0.5), NewString("1"), NewBytes([]byte("1")), point(NewInt(1)), point(NewString("a"))}

	for i, k := range keys {
		if err := m.Set(k, NewInt(int64(i))); err != nil {
			t.Fatal(err)
		}
	}

	same := []Value{NilValue, True, NewNumber(1), NewNumber(0.5), NewString("1"), NewBytes([]byte("1")), point(NewNumber(1)), point(NewString("a"))}
	for i, k := range same {
		if v, found, _ := m.Get(k); !found || v.Int() != int64(i) {
			t.Fatalf("key %d: got %v, %v", i, v.Int(), found)
		}
	}

	if m.Len() != len(keys) {
		t.Fatalf("len %d", m.Len())
	}

	for _, k := range []Value{NewNumber(nan()), NewArray(nil), point(NewArray(nil))} {
		if err := m.Set(k, NilValue); err == nil {
			t.Fatal("expected an unhashable key")
		}
	}
}

func nan() float64 {
	z := 0.0
	return z / z
}
