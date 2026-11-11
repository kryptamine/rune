package rune

import (
	"io"
	"rune/pkg/errors"
	"testing"
)

func report(t *testing.T, src string) string {
	t.Helper()

	prog, errs := Load([]byte(src))
	if len(errs) > 0 {
		return errors.Report(errs[0], src)
	}

	p := NewInterpreter()
	p.SetOutput(io.Discard)

	err := p.Run(prog)
	if err == nil {
		t.Fatal("expected an error")
	}

	return errors.Report(err, src)
}

func TestReport(t *testing.T) {
	tests := []struct {
		name, src, want string
	}{
		{
			"syntax error points at the token",
			"let x = 1;\nprint x +;\n",
			"[line: 2] Error at ';': Expect expression.\n" +
				"    2 | print x +;\n" +
				"      |          ^\n",
		},
		{
			"caret column counts characters and keeps tabs",
			"\tprint \"é\" + é;\n",
			"[line: 1] Error: Unexpected character: é\n" +
				"    1 | \tprint \"é\" + é;\n" +
				"      | \t            ^\n",
		},
		{
			"runtime error unwinds through closures and natives",
			"fun g(x) {\n  return x.y;\n}\nlet a = fun (f) { return map([1], f); };\na(g);\n",
			"[line: 2] Cannot read property 'y' of number.\n" +
				"    2 |   return x.y;\n" +
				"      |            ^\n" +
				"  at g (line 2)\n" +
				"  at map (native)\n" +
				"  at <anonymous> (line 4)\n" +
				"  at <script> (line 5)\n",
		},
		{
			"recursion collapses repeated frames",
			"fun f() { f(); }\nf();\n",
			"[line: 1] Stack overflow.\n" +
				"    1 | fun f() { f(); }\n" +
				"      |             ^\n" +
				"  at f (line 1)\n" +
				"  ... repeated 9999 more times\n" +
				"  at <script> (line 2)\n",
		},
		{
			"top-level error has no trace",
			"print nope;\n",
			"[line: 1] Undefined variable 'nope'.\n" +
				"    1 | print nope;\n" +
				"      |       ^\n",
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
