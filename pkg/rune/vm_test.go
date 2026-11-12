package rune

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

func TestProgramRunsConcurrently(t *testing.T) {
	prog, errs := loadSource([]byte(`
		let mut s = "";
		fun f(n) { if n < 2 { return n; } return f(n - 1) + f(n - 2); }
		for i in 0..50 { s = s + "ab"; }
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

				if _, err := p.Run(prog, nil); err != nil {
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

func TestInterrupt(t *testing.T) {
	srcs := map[string]string{
		"while":     `while true {}`,
		"for":       `for i in 0..1e18 {}`,
		"object":    `let o = {a: 1}; while true { for k in o {} }`,
		"recursion": `fun f(n) { if n < 2 { return n; } return f(n - 1) + f(n - 2); } f(100);`,
		"try":       `while true { try { while true {} } catch (e) {} }`,
		"native":    `map([1], fun (x) { while true {} });`,
	}

	for name, src := range srcs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			prog, errs := loadSource([]byte(src))
			if len(errs) > 0 {
				t.Fatal(errs)
			}

			p := NewInterpreter()
			time.AfterFunc(10*time.Millisecond, p.Interrupt)

			if _, err := p.Run(prog, nil); err != ErrInterrupted {
				t.Fatalf("got %v, want %v", err, ErrInterrupted)
			}
		})
	}
}
