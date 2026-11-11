// Package golden compares test output with files kept beside the test data.
package golden

import (
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files")

// Check compares got with the file at path, where a missing file means
// empty. With -update it rewrites the file instead, removing it if got is
// empty.
func Check(t *testing.T, path, got string) {
	t.Helper()

	if *update {
		write(t, path, got)
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

func write(t *testing.T, path, got string) {
	t.Helper()

	var err error
	if got == "" {
		err = os.Remove(path)
	} else {
		err = os.WriteFile(path, []byte(got), 0o644)
	}

	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
}
