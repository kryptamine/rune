package rune

import (
	"io"
	"rune/pkg/errors"
	"testing"
	"testing/fstest"
)

func report(t *testing.T, src string) string {
	t.Helper()

	prog, files, errs := Load(fstest.MapFS{"main.rn": {Data: []byte(src)}}, "", "main.rn")
	if len(errs) > 0 {
		return errors.Report(errs[0], files)
	}

	p := NewInterpreter(testStdio(io.Discard))

	err := p.Run(prog, nil)
	if err == nil {
		t.Fatal("expected an error")
	}

	return errors.Report(err, files)
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
			"fun g(x) {\n  return x.y;\n}\nlet a = fun (f) { return [1].map(f); };\na(g);\n",
			"[line: 2] TypeError: Cannot read property 'y' of number.\n" +
				"    2 |   return x.y;\n" +
				"      |            ^\n" +
				"  at g (line 2)\n" +
				"  at array.map (native)\n" +
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
			"[line: 1] NameError: Undefined variable 'nope'.\n" +
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
