package rune

import "testing"

// benchSources are CPU-bound programs that print nothing.
var benchSources = map[string]string{
	"fib":  `fun fib(n) { if (n < 2) return n; return fib(n - 1) + fib(n - 2); } fib(20);`,
	"loop": `var i = 0; var s = 0; while (i < 100000) { s = s + i; i = i + 1; }`,
	"closure": `fun counter() { var n = 0; fun inc() { n = n + 1; return n; } return inc; }
		var c = counter(); for (var i = 0; i < 50000; i = i + 1) { c(); }`,
	"array": `var a = []; for (var i = 0; i < 2000; i = i + 1) { a = append(a, i); }
		var s = 0; for (var i = 0; i < len(a); i = i + 1) { s = s + a[i]; }`,
	"push": `var a = []; for (var i = 0; i < 2000; i = i + 1) { push(a, i); }
		var s = 0; for (x in a) { s += x; }`,
	"locals":  `fun run() { var s = 0; for (var i = 0; i < 100000; i += 1) { s = s + i % 7; } return s; } run();`,
	"strings": `var s = ""; for (var i = 0; i < 1000; i += 1) { s = s + "ab"; } var n = 0; for (c in s) { if (c == "a") n += 1; }`,
	"objects": `var o = {x: 0, y: 0}; for (var i = 0; i < 20000; i += 1) { o.x = o.x + 1; o["y"] += o.x; }`,
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
