package rune

import (
	"io"
	"os"
	"testing"
	"time"
)

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
		if o := execute(src, io.Discard, 50*time.Millisecond); o.prog != nil {
			_ = o.prog.Disassemble()
		}
	})
}
