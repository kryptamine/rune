package rune

import (
	"bytes"
	stderrors "errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"rune/pkg/errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

var updateGolden = flag.Bool("update", false, "rewrite golden files")

const (
	scriptsDir    = "../../test"
	scriptTimeout = 5 * time.Second
)

var (
	expectRe       = regexp.MustCompile(`// expect: ?(.*)`)
	errorRe        = regexp.MustCompile(`// (\[line: \d+\] )?(Error.*)`)
	runtimeErrorRe = regexp.MustCompile(`// expect runtime error: (.+)`)
)

type expectations struct {
	out          []string
	errors       []string
	runtimeError string
}

func parseExpectations(src []byte) expectations {
	var e expectations

	for i, line := range strings.Split(string(src), "\n") {
		if m := expectRe.FindStringSubmatch(line); m != nil {
			e.out = append(e.out, m[1])
		}

		if m := errorRe.FindStringSubmatch(line); m != nil {
			at := m[1]
			if at == "" {
				at = "[line: " + strconv.Itoa(i+1) + "] "
			}

			e.errors = append(e.errors, at+m[2])
		}

		if m := runtimeErrorRe.FindStringSubmatch(line); m != nil {
			e.runtimeError = m[1]
		}
	}

	return e
}

func scripts(t testing.TB) []string {
	var paths []string

	err := filepath.WalkDir(scriptsDir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(path) == ".rn" {
			paths = append(paths, path)
		}

		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	return paths
}

func TestScripts(t *testing.T) {
	for _, path := range scripts(t) {
		name, _ := filepath.Rel(scriptsDir, strings.TrimSuffix(path, ".rn"))

		t.Run(filepath.ToSlash(name), func(t *testing.T) {
			t.Parallel()
			runScript(t, path)
		})
	}
}

func runScript(t *testing.T, path string) {
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := parseExpectations(src)
	if len(want.errors) > 0 && want.runtimeError != "" {
		t.Fatal("cannot expect both compile and runtime errors")
	}

	var out bytes.Buffer
	var stderr strings.Builder
	var gotErrors []string
	var gotRuntime string

	prog, errs := Load(src)
	for _, err := range errs {
		gotErrors = append(gotErrors, err.Error())
		stderr.WriteString(errors.Report(err, string(src)))
	}

	if prog != nil {
		p := NewInterpreter()
		p.SetOutput(&out)

		timer := time.AfterFunc(scriptTimeout, p.Interrupt)
		err := p.Run(prog)
		timer.Stop()

		if stderrors.Is(err, ErrInterrupted) {
			t.Fatalf("timed out after %v", scriptTimeout)
		}

		if err != nil {
			gotRuntime = err.Error()
			stderr.WriteString(errors.Report(err, string(src)))
		}
	}

	compareLines(t, "compile errors", gotErrors, want.errors)

	if gotRuntime != want.runtimeError {
		t.Errorf("runtime error:\n got: %q\nwant: %q", gotRuntime, want.runtimeError)
	}

	compareLines(t, "output", splitLines(out.String()), want.out)
	golden(t, strings.TrimSuffix(path, ".rn")+".stderr", stderr.String())
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}

	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

func compareLines(t *testing.T, what string, got, want []string) {
	t.Helper()

	for i := range max(len(got), len(want)) {
		switch {
		case i >= len(want):
			t.Errorf("%s %d: unexpected %q", what, i+1, got[i])
		case i >= len(got):
			t.Errorf("%s %d: missing %q", what, i+1, want[i])
		case got[i] != want[i]:
			t.Errorf("%s %d:\n got: %q\nwant: %q", what, i+1, got[i], want[i])
		}
	}
}

// golden compares got with the file at path, where a missing file means
// empty; -update rewrites it instead.
func golden(t *testing.T, path, got string) {
	t.Helper()

	if *updateGolden {
		if got == "" {
			err := os.Remove(path)
			if err != nil && !stderrors.Is(err, fs.ErrNotExist) {
				t.Fatal(err)
			}

			return
		}

		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}

		return
	}

	want, err := os.ReadFile(path)
	if err != nil && !stderrors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}

	if got != string(want) {
		t.Errorf("%s mismatch (run with -update to accept):\n got:\n%s\nwant:\n%s", filepath.Base(path), got, want)
	}
}
