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
		let mut s = 0; for i in 0..a.len() { s = s + a[i]; }`,
	"push": `let mut a = []; for i in 0..2000 { a.push(i); }
		let mut s = 0; for x in a { s += x; }`,
	"map":      `let mut a = []; for i in 0..2000 { a.push(i); } for k in 0..20 { a.map(x => x * 2).filter(x => x % 3 == 0); }`,
	"update":   `let mut s = 0; let mut i = 0; while i < 100000 { i++; s++; } let mut a = [0]; let mut j = 0; while j < 20000 { ++j; a[0]++; }`,
	"locals":   `fun run() { let mut s = 0; for i in 0..100000 { s = s + i % 7; } return s; } run();`,
	"strings":  `let mut s = ""; for i in 0..1000 { s = s + "ab"; } let mut n = 0; for c in s { if c == "a" { n += 1; } }`,
	"objects":  `let mut o = {x: 0, y: 0}; for i in 0..20000 { o.x = o.x + 1; o["y"] += o.x; }`,
	"structs":  `struct P { x, y, z } let mut s = 0; let mut p = P { x: 1, y: 2, z: 3 }; for i in 0..20000 { p = P { x: p.x + 1, y: p.y, z: p.z }; s += p.z + p.y; }`,
	"fields":   `let p = {x: 1, y: 2, z: 3}; let mut s = 0; for i in 0..50000 { s += p.x + p.y + p.z; }`,
	"setfield": `struct P { x, y, z } impl P { fun run(mut self) { let mut s = 0; for i in 0..20000 { self.x += 1; s += self.z + self.y; } } } let mut p = P { x: 1, y: 2, z: 3 }; p.run();`,
	"sfields":  `struct P { x, y, z } let p = P { x: 1, y: 2, z: 3 }; let mut s = 0; for i in 0..50000 { s += p.x + p.y + p.z; }`,
	"methods":  `struct P { x } impl P { fun get(self, k) { return self.x + k; } } let p = P { x: 1 }; let mut s = 0; for i in 0..50000 { s += p.get(i); }`,
	"funcs":    `struct P { x } fun get(p, k) { return p.x + k; } let p = P { x: 1 }; let mut s = 0; for i in 0..50000 { s += get(p, i); }`,
	"mutself":  `struct C { n } impl C { fun incr(mut self) { self.n++; } } let mut c = C { n: 0 }; for i in 0..20000 { c.incr(); }`,
	"objcall":  `let o = {a: 1, b: 2}; let mut s = 0; for i in 0..20000 { s += o.len(); }`,
	"print":    `for i in 0..20000 { print(i); }`,
	"format":   `let mut n = 0; for i in 0..20000 { n += "%-8s|%5.2f|%03d".format("ab", 3.14159, i).len(); }`,
}

var containerSources = map[string]string{
	"build":      `let mut a = []; for i in 0..5000 { a.push(i); }`,
	"buildcopy":  `let mut a = []; for i in 0..2000 { let mut b = a; b.push(i); a = b; }`,
	"copymut":    `let mut a = []; for i in 0..2000 { a.push(i); } let mut s = 0; for i in 0..200 { let mut b = a; b[0] = i; s += a[0] + b[0]; }`,
	"passread":   `fun sum(xs) { let mut s = 0; for x in xs { s += x; } return s; } let mut a = []; for i in 0..2000 { a.push(i); } let mut t = 0; for i in 0..50 { t += sum(a); }`,
	"passreturn": `fun add(xs, x) { let mut ys = xs; ys.push(x); return ys; } let mut a = []; for i in 0..2000 { a = add(a, i); }`,
	"mutparam":   `fun add(mut xs, x) { xs.push(x); } let mut a = []; for i in 0..2000 { add(a, i); }`,
	"grid":       `let mut g = []; for i in 0..200 { let mut r = []; for j in 0..200 { r.push(0); } g.push(r); } for i in 0..200 { for j in 0..200 { g[i][j] = i + j; } }`,
	"forrow":     `let mut g = []; for i in 0..200 { let mut r = []; for j in 0..200 { r.push(0); } g.push(r); } for i, mut r in g { if i % 7 == 0 { continue; } for j in 0..200 { r[j] = i + j; } if i == 190 { break; } }`,
	"objfield":   `let mut o = {xs: []}; for i in 0..2000 { o.xs.push(0); } for i in 0..2000 { o.xs[i] = i; }`,
	"deep":       `let mut o = {rows: []}; for i in 0..100 { o.rows.push({xs: []}); for j in 0..100 { o.rows[i].xs.push(0); } } for i in 0..100 { for j in 0..100 { o.rows[i].xs[j] = i + j; } }`,
	"mapfill":    `let mut m = map(); for i in 0..5000 { m[i] = i; }`,
	"mapparam":   `fun put(mut m, k) { m[k] = k; } let mut m = map(); for i in 0..2000 { put(m, i); }`,
	"setfill":    `let mut s = set(); for i in 0..5000 { s.add(i); }`,
	"structarr":  `struct B { xs } impl B { fun add(mut self, x) { self.xs.push(x); } fun set(mut self, i, x) { self.xs[i] = x; } } let mut b = B { xs: [] }; for i in 0..2000 { b.add(i); } for i in 0..2000 { b.set(i, i * 2); }`,
}

func TestBenchSources(t *testing.T) {
	for _, sources := range []map[string]string{benchSources, containerSources} {
		for name, src := range sources {
			t.Run(name, func(t *testing.T) {
				prog, errs := loadSource([]byte(src))
				if len(errs) > 0 {
					t.Fatal(errs)
				}
				if err := NewInterpreter(testStdio(io.Discard)).Run(prog, nil); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func BenchmarkPrograms(b *testing.B) {
	runBenchSources(b, benchSources)
}

func BenchmarkContainers(b *testing.B) {
	runBenchSources(b, containerSources)
}

func runBenchSources(b *testing.B, sources map[string]string) {
	for name, src := range sources {
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
