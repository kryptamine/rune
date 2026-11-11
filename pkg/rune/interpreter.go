package rune

import (
	"bufio"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/stdlib"
	"rune/pkg/value"
	"sync"
	"sync/atomic"
)

// ErrInterrupted is what Run returns after Interrupt; try does not catch it.
var ErrInterrupted = stderrors.New("Interrupted.")

// maxRecursionDepth caps nested calls.
const maxRecursionDepth = 10_000

// frame is a running call: its registers start at stack[base], and the
// callee itself sits just below them, where its result goes.
type frame struct {
	cl   *Closure
	pc   int
	base int
}

// handler is an active try: an error in frames[frame] or above resumes at
// pc there, with the caught value in register reg if bind, and if keep
// the error itself, traced up to there, in the register after it.
type handler struct {
	frame int
	pc    int
	reg   int
	bind  bool
	keep  bool
}

type Interpreter struct {
	stack        []value.Value
	frames       []frame
	handlers     []handler
	natives      stdlib.Frames
	maxRecursion int
	out          *bufio.Writer
	dest         io.Writer
	line         []byte
	halted       atomic.Bool
}

func NewInterpreter() *Interpreter {
	return &Interpreter{maxRecursion: maxRecursionDepth, dest: os.Stdout}
}

// Interrupt stops Run at the next call or loop iteration, from any
// goroutine. The interpreter stays interrupted.
func (p *Interpreter) Interrupt() {
	p.halted.Store(true)
}

func (p *Interpreter) SetOutput(w io.Writer) {
	p.flush()
	p.out, p.dest = nil, w
}

func (p *Interpreter) write(b []byte) {
	if p.out == nil {
		p.out = bufio.NewWriterSize(p.dest, 32<<10)
	}

	_, _ = p.out.Write(b)
}

func (p *Interpreter) flush() {
	if p.out != nil {
		_ = p.out.Flush()
	}
}

var builtinValues = sync.OnceValue(func() map[string]value.Value {
	m := make(map[string]value.Value, len(stdlib.Builtins))
	for _, b := range stdlib.Builtins {
		m[b.Name] = b.Value
	}

	return m
})

func EvaluateExpr(expr ast.Expr) (value.Value, error) {
	r := NewResolver()
	if errs := r.ResolveExpr(expr); len(errs) > 0 {
		return value.NilValue, errs[0]
	}

	prog, err := compileExpr(expr, r.builtins)
	if err != nil {
		return value.NilValue, err
	}

	p := NewInterpreter()
	defer p.flush()

	return p.exec(prog)
}

func (p *Interpreter) Run(prog *Program) error {
	defer p.flush()

	_, err := p.exec(prog)

	return err
}

func (p *Interpreter) exec(prog *Program) (value.Value, error) {
	cl := &Closure{proto: prog.main}

	p.grow(1 + cl.proto.nregs)
	p.stack[0] = value.NewFunction(cl)
	p.frames = append(p.frames[:0], frame{cl: cl, base: 1})
	p.handlers = p.handlers[:0]

	return p.run(0)
}

// grow makes the stack hold at least n registers.
func (p *Interpreter) grow(n int) {
	if n <= len(p.stack) {
		return
	}

	stack := make([]value.Value, max(n, 2*len(p.stack), 256))
	copy(stack, p.stack)
	p.stack = stack
}

// CallValue calls fn with args from a native; fn does not keep args.
func (p *Interpreter) CallValue(fn value.Value, args []value.Value, token *ast.Token) (value.Value, error) {
	if fn.Kind() == value.Function {
		switch f := fn.Fn().(type) {
		case *Closure:
			top := p.frames[len(p.frames)-1]
			at := top.base + top.cl.proto.nregs

			p.grow(at + 1 + len(args))
			p.stack[at] = fn
			copy(p.stack[at+1:], args)

			if err := p.enter(f, at+1, len(args), token); err != nil {
				return value.NilValue, err
			}

			return p.run(len(p.frames) - 1)
		case *stdlib.Native:
			return p.natives.Call(f, p, args, token)
		}
	}

	return value.NilValue, errors.NewRuntimeError(errors.TypeError, *token, "Can only call functions.")
}

// enter pushes a frame calling cl with the nargs arguments at stack[base:].
func (p *Interpreter) enter(cl *Closure, base, nargs int, tok *ast.Token) error {
	if nargs != cl.proto.arity || len(p.frames) > p.maxRecursion {
		return p.callError(cl, nargs, tok)
	}

	if need := base + cl.proto.nregs; need > len(p.stack) {
		p.grow(need)
	}

	p.frames = append(p.frames, frame{cl: cl, base: base})

	return nil
}

func (p *Interpreter) callError(cl *Closure, nargs int, tok *ast.Token) error {
	if nargs != cl.proto.arity {
		return errors.NewRuntimeError(errors.ArgumentError, *tok, fmt.Sprintf("Expected %d arguments but got %d.", cl.proto.arity, nargs))
	}

	return errors.NewFatalError(*tok, "Stack overflow.")
}

// unwind handles err raised in the top frame of a run that started at
// frames[stop]. It resumes at the innermost handler of that run and
// returns nil, if there is one; otherwise it drops the run's frames and
// returns err with them traced.
func (p *Interpreter) unwind(err error, stop int) error {
	// Only a RuntimeError is catchable, so ErrInterrupted always unwinds.
	rt, ok := err.(errors.RuntimeError)

	if n := len(p.handlers); ok && !rt.Fatal() && n > 0 && p.handlers[n-1].frame >= stop {
		h := p.handlers[n-1]
		p.handlers = p.handlers[:n-1]
		f := &p.frames[h.frame]

		if h.keep {
			rt = p.trace(rt, h.frame+1)
			p.stack[f.base+h.reg+1] = value.NewInternal(&rt)
		}

		if h.bind {
			p.stack[f.base+h.reg] = rt.Value()
		}

		p.frames = p.frames[:h.frame+1]
		f.pc = h.pc

		return nil
	}

	if ok {
		err = p.trace(rt, max(stop, 1))
	}

	p.frames = p.frames[:stop]
	p.dropHandlers(stop)

	return err
}

// trace records in rt the frames from the top down to frames[from].
func (p *Interpreter) trace(rt errors.RuntimeError, from int) errors.RuntimeError {
	for i := len(p.frames) - 1; i >= from; i-- {
		caller := p.frames[i-1]
		line := caller.cl.proto.toks[caller.pc-1].Line
		rt = rt.WithFrame(errors.Frame{Name: p.frames[i].cl.proto.displayName(), Line: line})
	}

	return rt
}

// dropHandlers drops the handlers of frames[from] and above.
func (p *Interpreter) dropHandlers(from int) {
	n := len(p.handlers)
	for n > 0 && p.handlers[n-1].frame >= from {
		n--
	}

	p.handlers = p.handlers[:n]
}
