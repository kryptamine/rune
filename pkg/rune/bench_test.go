package rune

import (
	"io"
	"testing"
)

// benchSources are CPU-bound programs; print goes nowhere.
var benchSources = map[string]string{
	"fib":  `fun fib(n) { if n < 2 { return n; } return fib(n - 1) + fib(n - 2); } fib(20);`,
	"loop": `let mut i = 0; let mut s = 0; while i < 100000 { s = s + i; i = i + 1; }`,
	"closure": `fun counter() { let mut n = 0; fun inc() { n = n + 1; return n; } return inc; }
		let c = counter(); for i in 0..50000 { c(); }`,
	"array": `let mut a = []; for i in 0..2000 { a = a.append(i); }
		let mut s = 0; for i in 0..len(a) { s = s + a[i]; }`,
	"push": `let mut a = []; for i in 0..2000 { a.push(i); }
		let mut s = 0; for x in a { s += x; }`,
	"map":       `let mut a = []; for i in 0..2000 { a.push(i); } for k in 0..20 { a.map(x => x * 2).filter(x => x % 3 == 0); }`,
	"update":    `let mut s = 0; let mut i = 0; while i < 100000 { i++; s++; } let mut a = [0]; let mut j = 0; while j < 20000 { ++j; a[0]++; }`,
	"locals":    `fun run() { let mut s = 0; for i in 0..100000 { s = s + i % 7; } return s; } run();`,
	"strings":   `let mut s = ""; for i in 0..1000 { s = s + "ab"; } let mut n = 0; for c in s { if c == "a" { n += 1; } }`,
	"objects":   `let mut o = {x: 0, y: 0}; for i in 0..20000 { o.x = o.x + 1; o["y"] += o.x; }`,
	"structs":   `struct P { x, y, z } let mut s = 0; let mut p = P { x: 1, y: 2, z: 3 }; for i in 0..20000 { p = P { x: p.x + 1, y: p.y, z: p.z }; s += p.z + p.y; }`,
	"fields":    `let p = {x: 1, y: 2, z: 3}; let mut s = 0; for i in 0..50000 { s += p.x + p.y + p.z; }`,
	"setfield":  `struct P { x, y, z } impl P { fun run(mut self) { let mut s = 0; for i in 0..20000 { self.x += 1; s += self.z + self.y; } } } let mut p = P { x: 1, y: 2, z: 3 }; p.run();`,
	"sfields":   `struct P { x, y, z } let p = P { x: 1, y: 2, z: 3 }; let mut s = 0; for i in 0..50000 { s += p.x + p.y + p.z; }`,
	"methods":   `struct P { x } impl P { fun get(self, k) { return self.x + k; } } let p = P { x: 1 }; let mut s = 0; for i in 0..50000 { s += p.get(i); }`,
	"funcs":     `struct P { x } fun get(p, k) { return p.x + k; } let p = P { x: 1 }; let mut s = 0; for i in 0..50000 { s += get(p, i); }`,
	"mutself":   `struct C { n } impl C { fun incr(mut self) { self.n++; } } let mut c = C { n: 0 }; for i in 0..20000 { c.incr(); }`,
	"structset": `struct P { x, y } let mut p = P { x: 0, y: 0 }; for i in 0..20000 { p.x += 1; }`,
	"objcall":   `let o = {a: 1, b: 2}; let mut s = 0; for i in 0..20000 { s += o.len(); }`,
	"print":     `for i in 0..20000 { print(i); }`,
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
