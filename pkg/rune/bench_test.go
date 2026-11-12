package rune

import (
	"io"
	"testing"
)

// benchSources are CPU-bound programs that print nothing.
var benchSources = map[string]string{
	"fib":  `fun fib(n) { if n < 2 { return n; } return fib(n - 1) + fib(n - 2); } fib(20);`,
	"loop": `let mut i = 0; let mut s = 0; while i < 100000 { s = s + i; i = i + 1; }`,
	"closure": `fun counter() { let mut n = 0; fun inc() { n = n + 1; return n; } return inc; }
		let c = counter(); for i in 0..50000 { c(); }`,
	"array": `let mut a = []; for i in 0..2000 { a = array.append(a, i); }
		let mut s = 0; for i in 0..len(a) { s = s + a[i]; }`,
	"push": `let mut a = []; for i in 0..2000 { array.push(a, i); }
		let mut s = 0; for x in a { s += x; }`,
	"update":  `let mut s = 0; let mut i = 0; while i < 100000 { i++; s++; } let mut a = [0]; let mut j = 0; while j < 20000 { ++j; a[0]++; }`,
	"locals":  `fun run() { let mut s = 0; for i in 0..100000 { s = s + i % 7; } return s; } run();`,
	"strings": `let mut s = ""; for i in 0..1000 { s = s + "ab"; } let mut n = 0; for c in s { if c == "a" { n += 1; } }`,
	"objects": `let mut o = {x: 0, y: 0}; for i in 0..20000 { o.x = o.x + 1; o["y"] += o.x; }`,
	"structs": `struct P { x, y, z } let mut s = 0; let mut p = P { x: 1, y: 2, z: 3 }; for i in 0..20000 { p = P { x: p.x + 1, y: p.y, z: p.z }; s += p.z + p.y; }`,
	"fields":  `let p = {x: 1, y: 2, z: 3}; let mut s = 0; for i in 0..50000 { s += p.x + p.y + p.z; }`,
	"with":    `struct P { x, y, z } let mut s = 0; let mut p = P { x: 1, y: 2, z: 3 }; for i in 0..20000 { p = p with { x: p.x + 1 }; s += p.z + p.y; }`,
	"sfields": `struct P { x, y, z } let p = P { x: 1, y: 2, z: 3 }; let mut s = 0; for i in 0..50000 { s += p.x + p.y + p.z; }`,
}

func BenchmarkPrograms(b *testing.B) {
	for name, src := range benchSources {
		b.Run(name, func(b *testing.B) {
			prog, errs := loadSource([]byte(src))
			if len(errs) > 0 {
				b.Fatal(errs)
			}

			b.ReportAllocs()

			for b.Loop() {
				if err := NewInterpreter(testStdio(io.Discard)).Run(prog, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
