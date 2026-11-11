package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update", false, "rewrite golden files")

// TestCommands runs each testdata/<command>/*.rn through the command and
// compares exit code, stdout and stderr with the .golden file beside it.
func TestCommands(t *testing.T) {
	for name, cmd := range commands {
		paths, err := filepath.Glob(filepath.Join("testdata", name, "*.rn"))
		if err != nil {
			t.Fatal(err)
		}

		for _, path := range paths {
			t.Run(name+"/"+strings.TrimSuffix(filepath.Base(path), ".rn"), func(t *testing.T) {
				t.Parallel()

				src, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}

				var stdout, stderr bytes.Buffer
				code := cmd(&stdout, &stderr, src)
				got := fmt.Sprintf("exit: %d\n-- stdout --\n%s-- stderr --\n%s", code, stdout.String(), stderr.String())

				golden(t, strings.TrimSuffix(path, ".rn")+".golden", got)
			})
		}
	}
}

func golden(t *testing.T, path, got string) {
	t.Helper()

	if *updateGolden {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}

		return
	}

	want, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}

	if got != string(want) {
		t.Errorf("%s mismatch (run with -update to accept):\n got:\n%s\nwant:\n%s", filepath.Base(path), got, want)
	}
}
