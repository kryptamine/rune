package rune

import (
	"rune/pkg/ast"
	"rune/pkg/errors"
)

type Scope = map[string]bool

// Resolver binds local variables to their scope depth before execution.
// Semantic errors do not confuse it, so it records them and keeps going;
// the visitor methods therefore always return nil errors.
type Resolver struct {
	interpreter *Interpreter
	scopes      []Scope
	isFunction  bool
	errors      []error
}

func NewResolver(interpreter *Interpreter) *Resolver {
	return &Resolver{
		interpreter: interpreter,
	}
}

// ResolveStmts resolves a whole program and returns all errors found.
func (p *Resolver) ResolveStmts(stmts []ast.Stmt) []error {
	p.resolveStmts(stmts)

	return p.errors
}

func (p *Resolver) VisitPrintStmt(printStmt *ast.PrintStmt) error {
	p.resolveExpr(printStmt.Expr)
	return nil
}

func (p *Resolver) VisitExprStmt(exprStmt *ast.ExprStmt) error {
	p.resolveExpr(exprStmt.Expr)
	return nil
}

func (p *Resolver) VisitVarStmt(stmt *ast.VarStmt) error {
	p.declare(stmt.Name)

	if stmt.Initializer != nil {
		p.resolveExpr(stmt.Initializer)
	}

	p.define(stmt.Name)

	return nil
}

func (p *Resolver) VisitVarExpr(expr *ast.VarExpr) (any, error) {
	if !p.isScopesEmpty() {
		if defined, exists := p.peekScope()[expr.Name.Lexeme]; exists && !defined {
			p.error(expr.Name, "Cannot read local variable in its own initializer.")
		}
	}

	p.resolveLocal(expr, expr.Name)

	return nil, nil
}

func (p *Resolver) VisitBlockStmt(blockStmt *ast.BlockStmt) error {
	p.beginScope()
	p.resolveStmts(blockStmt.Stmts)
	p.endScope()

	return nil
}

func (p *Resolver) VisitIfStmt(ifStmt *ast.IfStmt) error {
	p.resolveExpr(ifStmt.Condition)
	p.resolveStmt(ifStmt.Then)

	if ifStmt.El != nil {
		p.resolveStmt(ifStmt.El)
	}

	return nil
}

func (p *Resolver) VisitWhileStmt(whileStmt *ast.WhileStmt) error {
	p.resolveExpr(whileStmt.Condition)
	p.resolveStmt(whileStmt.Body)

	return nil
}

func (p *Resolver) VisitFunctionStmt(fnStmt *ast.FunctionStmt) error {
	p.declare(fnStmt.Name)
	p.define(fnStmt.Name)
	p.resolveFn(fnStmt)

	return nil
}

func (p *Resolver) VisitReturnStmt(returnStmt *ast.ReturnStmt) error {
	if !p.isFunction {
		p.error(returnStmt.Keyword, "Cannot return from top-level code.")
	}

	if returnStmt.Value != nil {
		p.resolveExpr(returnStmt.Value)
	}

	return nil
}

func (p *Resolver) VisitAssignExpr(expr *ast.AssignExpr) (any, error) {
	p.resolveExpr(expr.Value)
	p.resolveLocal(expr, expr.Name)

	return nil, nil
}

func (p *Resolver) VisitBinaryExpr(expr *ast.BinaryExpr) (any, error) {
	p.resolveExpr(expr.Left)
	p.resolveExpr(expr.Right)

	return nil, nil
}

func (p *Resolver) VisitCallExpr(callExpr *ast.CallExpr) (any, error) {
	p.resolveExpr(callExpr.Callee)

	for _, arg := range callExpr.Args {
		p.resolveExpr(arg)
	}

	return nil, nil
}

func (p *Resolver) VisitGroupingExpr(grExpr *ast.GroupingExpr) (any, error) {
	p.resolveExpr(grExpr.Expr)

	return nil, nil
}

func (p *Resolver) VisitLiteralExpr(_ *ast.LiteralExpr) (any, error) {
	return nil, nil
}

func (p *Resolver) VisitLogicalExpr(expr *ast.LogicalExpr) (any, error) {
	p.resolveExpr(expr.Left)
	p.resolveExpr(expr.Right)

	return nil, nil
}

func (p *Resolver) VisitUnaryExpr(expr *ast.UnaryExpr) (any, error) {
	p.resolveExpr(expr.Right)

	return nil, nil
}

func (p *Resolver) VisitArrayExpr(expr *ast.ArrayExpr) (any, error) {
	for _, item := range expr.Items {
		p.resolveExpr(item)
	}

	return nil, nil
}

func (p *Resolver) VisitIndexExpr(expr *ast.IndexExpr) (any, error) {
	p.resolveExpr(expr.Array)
	p.resolveExpr(expr.Index)

	return nil, nil
}

func (p *Resolver) VisitSetIndexExpr(expr *ast.SetIndexExpr) (any, error) {
	p.resolveExpr(expr.Array)
	p.resolveExpr(expr.Index)
	p.resolveExpr(expr.Value)

	return nil, nil
}

func (p *Resolver) VisitObjectExpr(expr *ast.ObjectExpr) (any, error) {
	for _, pair := range expr.Pairs {
		p.resolveExpr(pair)
	}

	return nil, nil
}

func (p *Resolver) error(token ast.Token, msg string) {
	p.errors = append(p.errors, errors.NewSyntaxError(token, msg))
}

func (p *Resolver) resolveLocal(expr ast.Expr, name ast.Token) {
	for i := len(p.scopes) - 1; i >= 0; i-- {
		if _, exists := p.scopes[i][name.Lexeme]; exists {
			p.interpreter.Resolve(expr, len(p.scopes)-1-i)
			return
		}
	}
}

func (p *Resolver) resolveStmts(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		p.resolveStmt(stmt)
	}
}

func (p *Resolver) resolveStmt(stmt ast.Stmt) {
	_ = stmt.Accept(p)
}

func (p *Resolver) resolveExpr(expr ast.Expr) {
	_, _ = expr.Accept(p)
}

func (p *Resolver) beginScope() {
	p.scopes = append(p.scopes, make(Scope))
}

func (p *Resolver) endScope() {
	p.scopes = p.scopes[:len(p.scopes)-1]
}

func (p *Resolver) resolveFn(fn *ast.FunctionStmt) {
	enclosingIsFunction := p.isFunction
	p.isFunction = true
	defer func() { p.isFunction = enclosingIsFunction }()

	p.beginScope()

	for _, fnParam := range fn.Parameters {
		p.declare(fnParam)
		p.define(fnParam)
	}

	p.resolveStmts(fn.Body)
	p.endScope()
}

func (p *Resolver) declare(name ast.Token) {
	if p.isScopesEmpty() {
		return
	}

	scope := p.peekScope()

	if _, ok := scope[name.Lexeme]; ok {
		p.error(name, "Variable with this name already declared in this scope.")
	}

	scope[name.Lexeme] = false
}

func (p *Resolver) define(name ast.Token) {
	if p.isScopesEmpty() {
		return
	}

	p.peekScope()[name.Lexeme] = true
}

func (p *Resolver) isScopesEmpty() bool {
	return len(p.scopes) == 0
}

func (p *Resolver) peekScope() Scope {
	return p.scopes[len(p.scopes)-1]
}
