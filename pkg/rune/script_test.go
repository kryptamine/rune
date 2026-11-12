package rune

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"rune/pkg/errors"
	"rune/pkg/golden"
	"rune/pkg/stdlib"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

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

	// Directories starting with _ hold modules the scripts import.
	err := filepath.WalkDir(scriptsDir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() && strings.HasPrefix(d.Name(), "_") {
			return filepath.SkipDir
		}

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

	got := execute(os.DirFS(filepath.Dir(path)), filepath.Base(path), &out, scriptTimeout)
	if got.runtimeError == ErrInterrupted {
		t.Fatalf("timed out after %v", scriptTimeout)
	}

	compareLines(t, "compile errors", errorStrings(got.compileErrors), want.errors)

	if msg := errorString(got.runtimeError); msg != want.runtimeError {
		t.Errorf("runtime error:\n got: %q\nwant: %q", msg, want.runtimeError)
	}

	compareLines(t, "output", splitLines(out.String()), want.out)
	golden.Check(t, strings.TrimSuffix(path, ".rn")+".stderr", got.stderr)
}

// outcome is what running a script produced besides its output.
type outcome struct {
	prog          *Program
	compileErrors []error
	runtimeError  error
	stderr        string
}

// loadSource loads src as a program of one file.
func loadSource(src []byte) (*Program, []error) {
	prog, _, errs := Load(fstest.MapFS{"main.rn": {Data: src}}, "", "main.rn")
	return prog, errs
}

// execute loads and runs the program at name in fsys as the CLI does,
// interrupting it after timeout.
func execute(fsys fs.FS, name string, out io.Writer, timeout time.Duration) outcome {
	var o outcome
	var stderr strings.Builder
	var files []errors.File

	o.prog, files, o.compileErrors = Load(fsys, "", name)
	for _, err := range o.compileErrors {
		stderr.WriteString(errors.Report(err, files))
	}

	if o.prog != nil {
		p := NewInterpreter(stdlib.NewStdio(strings.NewReader(""), out, &stderr))

		timer := time.AfterFunc(timeout, p.Interrupt)
		o.runtimeError = p.Run(o.prog, nil)
		if _, ok := o.runtimeError.(stdlib.Exit); ok {
			o.runtimeError = nil // exit codes are checked by the CLI tests
		}
		timer.Stop()

		if o.runtimeError != nil {
			stderr.WriteString(errors.Report(o.runtimeError, files))
		}
	}

	o.stderr = stderr.String()

	return o
}

func errorString(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}

func errorStrings(errs []error) []string {
	out := make([]string, len(errs))
	for i, err := range errs {
		out[i] = err.Error()
	}

	return out
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

// testStdio has no input, writes stdout to out and drops stderr.
func testStdio(out io.Writer) *stdlib.Stdio {
	return stdlib.NewStdio(strings.NewReader(""), out, io.Discard)
}
