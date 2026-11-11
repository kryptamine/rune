package rune

import (
	"bytes"
	"sync"
	"testing"
)

func TestProgramRunsConcurrently(t *testing.T) {
	prog, errs := Load([]byte(`
		let mut s = "";
		fun f(n) { if (n < 2) return n; return f(n - 1) + f(n - 2); }
		for (i in 0..50) { s = s + "ab"; }
		print len(s) + f(15);
		print map([1, 2], fun (x) { return x * 2; });
	`))
	if len(errs) > 0 {
		t.Fatal(errs)
	}

	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			for range 20 {
				var out bytes.Buffer

				p := NewInterpreter()
				p.SetOutput(&out)

				if err := p.Run(prog); err != nil {
					t.Error(err)
				}

				if got := out.String(); got != "710\n[2, 4]\n" {
					t.Errorf("got %q", got)
				}
			}
		})
	}

	wg.Wait()
}
