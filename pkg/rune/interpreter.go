package rune

import (
	"bufio"
	"io"
	"os"
	"rune/pkg/ast"
	"rune/pkg/environment"
	"rune/pkg/stdlib"
	"rune/pkg/value"
	"sync"
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
	globals        environment.Globals
	pool           environment.Pool
	frames         stdlib.Frames
	returnValue    value.Value
	recursionDepth int
	maxRecursion   int
	args           []value.Value
	out            *bufio.Writer
	dest           io.Writer
	line           []byte
}

func NewInterpreter() *Interpreter {
	return &Interpreter{
		globals:      builtinGlobals().Clone(),
		maxRecursion: maxRecursionDepth,
		dest:         os.Stdout,
	}
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

// builtinGlobals holds the builtins; each interpreter starts from a clone.
var builtinGlobals = sync.OnceValue(func() *environment.Globals {
	var g environment.Globals
	for _, b := range stdlib.Builtins {
		g.Define(environment.GlobalSlot(b.Name), b.Value)
	}

	return &g
})

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
