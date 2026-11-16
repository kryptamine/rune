package rune

import (
	"io"
	"rune/pkg/errors"
	"strings"
	"testing"
	"testing/fstest"
)

func report(t *testing.T, src string) string {
	t.Helper()

	prog, files, errs := Load(fstest.MapFS{"main.rn": {Data: []byte(src)}}, "", "main.rn")
	if len(errs) > 0 {
		return errors.Report(errs[0], files, false)
	}

	p := NewInterpreter(testStdio(io.Discard))

	err := p.Run(prog, nil)
	if err == nil {
		t.Fatal("expected an error")
	}

	return errors.Report(err, files, false)
}

func TestReport(t *testing.T) {
	tests := []struct {
		name, src, want string
	}{
		{
			"syntax error points at the token",
			"let x = 1;\nprint(x +);\n",
			"error: Expect expression.\n" +
				" --> main.rn:2:10\n" +
				"  |\n" +
				"1 | let x = 1;\n" +
				"2 | print(x +);\n" +
				"  |          ^\n",
		},
		{
			"caret column counts characters and keeps tabs",
			"\tprint(\"é\" + é);\n",
			"error: Unexpected character: é\n" +
				" --> main.rn:1:14\n" +
				"  |\n" +
				"1 | \tprint(\"é\" + é);\n" +
				"  | \t            ^\n",
		},
		{
			"missing token points past the line before",
			"let x = 1\nprint(x);\n",
			"error: Expect ';' after variable declaration.\n" +
				" --> main.rn:1:10\n" +
				"  |\n" +
				"1 | let x = 1\n" +
				"  |          ^\n",
		},
		{
			"runtime error unwinds through closures and natives",
			"fun g(x) {\n  return x.y;\n}\nlet a = f => [1].map(f);\na(g);\n",
			"TypeError: Cannot read property 'y' of number.\n" +
				" --> main.rn:2:12\n" +
				"  |\n" +
				"1 | fun g(x) {\n" +
				"2 |   return x.y;\n" +
				"  |            ^\n" +
				"\n" +
				"  at g (main.rn:2)\n" +
				"  at array.map (native)\n" +
				"  at <anonymous> (main.rn:4)\n" +
				"  at <script> (main.rn:5)\n",
		},
		{
			"recursion collapses repeated frames",
			"fun f() { f(); }\nf();\n",
			"error: Stack overflow.\n" +
				" --> main.rn:1:13\n" +
				"  |\n" +
				"1 | fun f() { f(); }\n" +
				"  |             ^\n" +
				"\n" +
				"  at f (main.rn:1)\n" +
				"  ... repeated 9999 more times\n" +
				"  at <script> (main.rn:2)\n",
		},
		{
			"top-level error has no trace",
			"print(nope);\n",
			"NameError: Undefined variable 'nope'.\n" +
				" --> main.rn:1:7\n" +
				"  |\n" +
				"1 | print(nope);\n" +
				"  |       ^^^^\n",
		},
		{
			"uncaught struct without a message",
			"struct Oops { code }\nthrow Oops { code: 1 };\n",
			"error: Oops {code: 1}\n" +
				" --> main.rn:2:1\n" +
				"  |\n" +
				"1 | struct Oops { code }\n" +
				"2 | throw Oops { code: 1 };\n" +
				"  | ^^^^^\n",
		},
		{
			"redeclaration hints at the first",
			"let a = 1;\n{\n  let b = 1;\n  let b = 2;\n}\n",
			"error: Variable with this name already declared in this scope.\n" +
				" --> main.rn:4:7\n" +
				"  |\n" +
				"3 |   let b = 1;\n" +
				"4 |   let b = 2;\n" +
				"  |       ^\n" +
				"  |\n" +
				"  = hint: 'b' is first declared on line 3; rename one of them\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := report(t, tt.src); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestHints(t *testing.T) {
	tests := []struct {
		name, src, hint string
	}{
		{"typo of a variable", "let count = 1;\nprint(cout);\n", "did you mean 'count'?"},
		{"typo of a local", "fun f() {\n  let total = 1;\n  return totl;\n}\nf();\n", "did you mean 'total'?"},
		{"typo of a builtin", "prnt(1);\n", "did you mean 'print'?"},
		{"name of another language", "let x = null;\n", "Rune writes it 'nil'"},
		{"typo of a field", "struct User { name }\nprint(User { name: 1 }.nmae);\n", "did you mean 'name'?"},
		{"typo of a field in a literal", "struct User { name }\nUser { nmae: 1 };\n", "did you mean 'name'?"},
		{"typo of a property", "let o = {width: 1};\nprint(o.widht);\n", "did you mean 'width'?"},
		{"typo of a method", "struct U {}\nimpl U { fun greet(self) { 1 } }\nU {}.gret();\n", "did you mean 'greet'?"},
		{"method of another language", "print([1].length);\n", "use 'len' instead"},
		{"membership method", "print([1].includes(1));\n", "test membership with the in operator: 'x in xs'"},
		{"assign to immutable", "let a = 1;\na = 2;\n", "declare it as 'mut a' on line 1"},
		{"mutate a constant", "const A = [1];\nA.push(2);\n", "declare it with 'let mut' instead of 'const' on line 1"},
		{"keyword of another language", "var x = 1;\n", "declare a variable with 'let', or 'let mut' to change it later"},
		{"string plus number", "print(\"n: \" + 1);\n", "to put a value in a string, interpolate it: \"text {x}\""},
		{"string times number", "print(\"ab\" * 2);\n", "to repeat a string, call its repeat method: x.repeat(n)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := report(t, tt.src); !strings.Contains(got, "= hint: "+tt.hint+"\n") {
				t.Errorf("want hint %q in:\n%s", tt.hint, got)
			}
		})
	}
}

func TestNoHint(t *testing.T) {
	for _, src := range []string{
		"print(zzz);\n",
		"let x = 1;\nprint(y);\n",
		"fun f() {}\nf = 1;\n",
		"try { throw ValueError { message: \"m\" }; } catch e { e = 1; }\n",
	} {
		if got := report(t, src); strings.Contains(got, "hint:") {
			t.Errorf("want no hint for %q:\n%s", src, got)
		}
	}
}
