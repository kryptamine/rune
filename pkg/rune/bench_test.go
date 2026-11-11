package rune

import "testing"

// benchSources are CPU-bound programs that print nothing.
var benchSources = map[string]string{
	"fib":  `fun fib(n) { if (n < 2) return n; return fib(n - 1) + fib(n - 2); } fib(20);`,
	"loop": `let mut i = 0; let mut s = 0; while (i < 100000) { s = s + i; i = i + 1; }`,
	"closure": `fun counter() { let mut n = 0; fun inc() { n = n + 1; return n; } return inc; }
		let c = counter(); for (let mut i = 0; i < 50000; i = i + 1) { c(); }`,
	"array": `let mut a = []; for (let mut i = 0; i < 2000; i = i + 1) { a = append(a, i); }
		let mut s = 0; for (let mut i = 0; i < len(a); i = i + 1) { s = s + a[i]; }`,
	"push": `let a = []; for (let mut i = 0; i < 2000; i = i + 1) { push(a, i); }
		let mut s = 0; for (x in a) { s += x; }`,
	"update":  `let mut s = 0; for (let mut i = 0; i < 100000; i++) { s++; } let a = [0]; for (let mut j = 0; j < 20000; ++j) { a[0]++; }`,
	"locals":  `fun run() { let mut s = 0; for (let mut i = 0; i < 100000; i += 1) { s = s + i % 7; } return s; } run();`,
	"strings": `let mut s = ""; for (let mut i = 0; i < 1000; i += 1) { s = s + "ab"; } let mut n = 0; for (c in s) { if (c == "a") n += 1; }`,
	"objects": `let o = {x: 0, y: 0}; for (let mut i = 0; i < 20000; i += 1) { o.x = o.x + 1; o["y"] += o.x; }`,
}

func BenchmarkPrograms(b *testing.B) {
	for name, src := range benchSources {
		b.Run(name, func(b *testing.B) {
			prog, errs := Load([]byte(src))
			if len(errs) > 0 {
				b.Fatal(errs)
			}

			b.ReportAllocs()

			for b.Loop() {
				if err := NewInterpreter().Run(prog); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
