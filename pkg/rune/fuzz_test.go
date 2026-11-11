package rune

import (
	"io"
	"os"
	"rune/pkg/errors"
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
		prog, errs := Load(src)
		for _, err := range errs {
			errors.Report(err, string(src))
		}

		if prog == nil {
			return
		}

		_ = prog.Disassemble()

		p := NewInterpreter()
		p.SetOutput(io.Discard)

		timer := time.AfterFunc(50*time.Millisecond, p.Interrupt)
		defer timer.Stop()

		if err := p.Run(prog); err != nil {
			errors.Report(err, string(src))
		}
	})
}
