package rune

import (
	"fmt"
	"io"
	"runtime"
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

// A copy of the array takes 16KB; reads of it take none.
func TestReadsDoNotCopy(t *testing.T) {
	src := `fun sum(xs) { let mut s = 0; for x in xs { s += x; } return s; }
		let mut a = []; for i in 0..1000 { a.push(i); }
		let o = {a};
		for k in 0..%d { let b = a; sum(b); let c = o.a; let [h, ...t] = c[..1]; a.len(); a.filter(x => false); }
		a.push(0);`

	if one, many := allocated(t, src, 1), allocated(t, src, 101); many-one > 100<<10 {
		t.Fatalf("reads copy: %d bytes for 1 pass, %d for 101", one, many)
	}
}

// Each pass makes one 16KB array; a copy of it would take as much again.
func TestMovesDoNotCopy(t *testing.T) {
	for name, src := range map[string]string{
		"return":   `fun build() { let mut r = [0].repeat(1000); return r; } for k in 0..%d { let mut xs = build(); xs[0] = 1; }`,
		"let":      `for k in 0..%d { let mut a = [0].repeat(1000); let mut b = a; b[0] = 1; }`,
		"assign":   `for k in 0..%d { let mut a = [0].repeat(1000); let mut b = []; b = a; b[0] = 1; }`,
		"rebind":   `let mut a = [0].repeat(1000); for k in 0..%d { let mut b = a; b[0] = k; a = b; let c = [0].repeat(1000); }`,
		"pass":     `fun first(xs) { return xs[0]; } for k in 0..%d { let mut a = [0].repeat(1000); first(a); a[0] = 1; }`,
		"passlast": `fun put(mut xs) { xs[0] = 1; } for k in 0..%d { let mut a = [0].repeat(1000); let mut b = a; put(b); }`,
		"cond":     `fun pick(c) { let mut a = [0].repeat(1000); let mut b = [1]; return c ? a : b; } for k in 0..%d { let mut xs = pick(k %% 2 == 0); xs[0] = 1; }`,
	} {
		t.Run(name, func(t *testing.T) {
			if one, many := allocated(t, src, 1), allocated(t, src, 101); many-one > 100*20<<10 {
				t.Fatalf("copies: %d bytes for 1 pass, %d for 101", one, many)
			}
		})
	}
}

// A container a pass hands from variable to variable is never copied.
func TestContainerMovesDoNotCopy(t *testing.T) {
	for name, src := range map[string]string{
		"object": `let mut a = {}; for i in 0..1000 { a[string(i)] = i; } for k in 0..%d { let mut b = a; b["0"] = k; a = b; }`,
		"map":    `let mut a = map(); for i in 0..1000 { a[i] = i; } for k in 0..%d { let mut b = a; b[0] = k; a = b; }`,
		"set":    `let mut a = set(); for i in 0..1000 { a.add(i); } for k in 0..%d { let mut b = a; b.delete(k); a = b; }`,
		"buffer": `let mut a = bytes.buffer(); for i in 0..1000 { a.writeUint(0, 8); } for k in 0..%d { let mut b = a; b[0] = k %% 256; a = b; }`,
		"path":   `let mut o = { m: map({ a: {} }) }; for i in 0..1000 { o.m["a"][string(i)] = i; } for k in 0..%d { o.m["a"]["0"] = k; o.m["a"].delete("1"); }`,
	} {
		t.Run(name, func(t *testing.T) {
			if one, many := allocated(t, src, 1), allocated(t, src, 101); many > one+20<<10 {
				t.Fatalf("copies: %d bytes for 1 pass, %d for 101", one, many)
			}
		})
	}
}

// A grid of 100 rows of 100 takes some 170KB; writes along paths no
// other holder shares take none.
func TestPathWritesInPlace(t *testing.T) {
	src := `let mut g = []; for i in 0..100 { g.push([0].repeat(100)); }
		let mut o = { g };
		for k in 0..%d { for i in 0..100 { g[i][k %% 100] = k; g[i][0] += 1; g[i][1]++; o.g[i][0] = k; } g[0].push(k); g[0].pop(); }`

	if one, many := allocated(t, src, 1), allocated(t, src, 101); many > one+50<<10 {
		t.Fatalf("path writes copy: %d bytes for 1 pass, %d for 101", one, many)
	}
}

// So do the write-backs of mut parameters and of for mut.
func TestInoutWritesInPlace(t *testing.T) {
	src := `fun bump(mut row, k) { row[0] = k; row[1] += 1; }
		let mut g = []; for i in 0..100 { g.push([0].repeat(100)); }
		for k in 0..%d { for i, mut row in g { row[k %% 100] = k; bump(row, k); } bump(g[k %% 100], k); }`

	if one, many := allocated(t, src, 1), allocated(t, src, 101); many > one+50<<10 {
		t.Fatalf("inout writes copy: %d bytes for 1 pass, %d for 101", one, many)
	}
}

// allocated is how many bytes src, with its %d for the passes, takes.
func allocated(t *testing.T, src string, passes int) uint64 {
	prog, errs := loadSource([]byte(fmt.Sprintf(src, passes)))
	if len(errs) > 0 {
		t.Fatal(errs)
	}

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	if err := NewInterpreter(testStdio(io.Discard)).Run(prog, nil); err != nil {
		t.Fatal(err)
	}
	runtime.ReadMemStats(&after)

	return after.TotalAlloc - before.TotalAlloc
}
