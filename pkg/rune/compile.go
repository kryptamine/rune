package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/environment"
	"rune/pkg/value"
)

type (
	evalFn func(p *Interpreter) (value.Value, error)
	execFn func(p *Interpreter) error
	condFn func(p *Interpreter) (bool, error)
)

// Program holds no run state and can be run any number of times.
type Program struct {
	run execFn
}

func Compile(stmts []ast.Stmt) *Program {
	return &Program{run: compileStmts(stmts)}
}

type function struct {
	name    string
	arity   int
	envSize int
	pooled  bool
	body    execFn
}

type Closure struct {
	fn  *function
	env *environment.Environment
}

func (c *Closure) String() string {
	if c.fn.name == "" {
		return "<fn>"
	}

	return fmt.Sprintf("<fn %s>", c.fn.name)
}

func (f *function) displayName() string {
	if f.name == "" {
		return "<anonymous>"
	}

	return f.name
}

func compileFunction(decl *ast.FunctionStmt) *function {
	return &function{
		name:    decl.Name.Lexeme,
		arity:   len(decl.Parameters),
		envSize: decl.EnvSize,
		pooled:  decl.Pooled,
		body:    compileStmts(decl.Body),
	}
}

func nop(*Interpreter) error { return nil }

func constant(v value.Value) evalFn {
	return func(*Interpreter) (value.Value, error) { return v, nil }
}

func literalOf(expr ast.Expr) (value.Value, bool) {
	switch n := expr.(type) {
	case *ast.LiteralExpr:
		return n.Value, true
	case *ast.GroupingExpr:
		return literalOf(n.Expr)
	}

	return value.NilValue, false
}
