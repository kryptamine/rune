package rune

import (
	"io"
	"os"
	"regexp"
	"testing"
	"testing/fstest"
	"time"
)

var sideEffects = regexp.MustCompile(`\b(fs|os)\b`)

// FuzzRun checks that no program panics the pipeline, the disassembler,
// the VM or error reporting. Seeds are the scripts in test/.
func FuzzRun(f *testing.F) {
	for _, path := range scripts(f) {
		src, err := os.ReadFile(path)
		if err != nil {
			f.Fatal(err)
		}

		f.Add(src)
	}

	f.Fuzz(func(t *testing.T, src []byte) {
		// A mutated fs or os call could touch any file or run any command; both are reached only by name.
		if sideEffects.Match(src) {
			return
		}

		if o := execute(fstest.MapFS{"main.rn": {Data: src}}, "main.rn", io.Discard, 50*time.Millisecond); o.prog != nil {
			_ = o.prog.Disassemble()
		}
	})
}
