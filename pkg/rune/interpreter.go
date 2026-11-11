package rune

import (
	"bufio"
	"io"
	"os"
	"rune/pkg/ast"
	"rune/pkg/callable"
	"rune/pkg/environment"
	"rune/pkg/value"
)

// maxRecursionDepth caps nested calls; 10k frames take ~40MB of memory.
const maxRecursionDepth = 10_000

// returnSignal unwinds statements up to the enclosing call. The value being
// returned travels in Interpreter.returnValue, so returning never allocates.
type returnSignal struct{}

func (returnSignal) Error() string { return "<fn return>" }

var errReturn error = returnSignal{}

// loopSignal unwinds statements up to the enclosing loop for break and
// continue. The resolver guarantees they only appear inside loops.
type loopSignal struct{ name string }

func (s loopSignal) Error() string { return "<" + s.name + ">" }

var (
	errBreak    error = loopSignal{"break"}
	errContinue error = loopSignal{"continue"}
)

type Interpreter struct {
	// environment is the innermost local scope, or nil at the top level.
	environment    *environment.Environment
	globals        *environment.Globals
	pool           environment.Pool
	returnValue    value.Value
	recursionDepth int
	maxRecursion   int
	args           []value.Value
	out            *bufio.Writer
	dest           io.Writer
	line           []byte
}

func NewInterpreter() *Interpreter {
	p := &Interpreter{
		globals:      environment.NewGlobals(),
		maxRecursion: maxRecursionDepth,
		dest:         os.Stdout,
	}

	// Global functions.
	p.registerGlobalCallable("clock", callable.NewClockCallable())
	p.registerGlobalCallable("len", callable.NewLenCallable())
	p.registerGlobalCallable("append", callable.NewAppendCallable())
	p.registerGlobalCallable("push", callable.NewPushCallable())
	p.registerGlobalCallable("pop", callable.NewPopCallable())
	p.registerGlobalCallable("json", callable.NewJsonCallable())

	return p
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

func (p *Interpreter) registerGlobalCallable(name string, fn callable.Callable) {
	p.globals.Define(environment.GlobalSlot(name), value.NewFunction(fn))
}

func EvaluateExpr(expr ast.Expr) (value.Value, error) {
	p := NewInterpreter()
	defer p.flush()

	return compileExpr(expr)(p)
}

func (p *Interpreter) EvaluateStmts(stmts []ast.Stmt) error {
	return p.Run(Compile(stmts))
}

func (p *Interpreter) Run(prog *Program) error {
	defer p.flush()

	return prog.run(p)
}

func (p *Interpreter) newEnv(pooled bool, enclosing *environment.Environment, size int) *environment.Environment {
	if pooled {
		return p.pool.Get(enclosing, size)
	}

	return environment.NewEnvironment(enclosing, size)
}

func (p *Interpreter) runIn(body execFn, env *environment.Environment, pooled bool) error {
	prev := p.environment
	p.environment = env

	err := body(p)

	p.environment = prev

	if pooled {
		p.pool.Put(env)
	}

	return err
}
