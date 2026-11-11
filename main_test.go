package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"rune/pkg/golden"
	"strings"
	"testing"
)

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

				golden.Check(t, strings.TrimSuffix(path, ".rn")+".golden", got)
			})
		}
	}
}
