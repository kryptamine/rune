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
}

func BenchmarkPrograms(b *testing.B) {
	for name, src := range benchSources {
		b.Run(name, func(b *testing.B) {
			tokens, scanErrs := Scan([]byte(src))
			stmts, parseErrs := ParseStmts(tokens)
			if len(scanErrs) > 0 || len(parseErrs) > 0 {
				b.Fatal(scanErrs, parseErrs)
			}

			if errs := NewResolver().ResolveStmts(stmts); len(errs) > 0 {
				b.Fatal(errs)
			}

			b.ReportAllocs()

			for b.Loop() {
				if err := NewInterpreter().EvaluateStmts(stmts); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
