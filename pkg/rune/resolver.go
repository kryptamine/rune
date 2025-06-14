package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
)

type Scope = map[string]bool

type Resolver struct {
	interpreter *Interpreter
	scopes      []Scope
	isFunction  bool
}

func NewResolver(interpreter *Interpreter) *Resolver {
	return &Resolver{
		interpreter: interpreter,
	}
}

func (p *Resolver) VisitPrintStmt(printStmt *ast.PrintStmt) error {
	_, err := p.resolveExpr(printStmt.Expr)

	return err
}

func (p *Resolver) VisitExprStmt(exprStmt *ast.ExprStmt) error {
	_, err := p.resolveExpr(exprStmt.Expr)
	return err
}

func (p *Resolver) VisitVarStmt(stmt *ast.VarStmt) error {
	if err := p.declare(stmt.Name); err != nil {
		return err
	}

	if stmt.Initializer != nil {
		if _, err := p.resolveExpr(stmt.Initializer); err != nil {
			return err
		}
	}

	p.define(stmt.Name)

	return nil
}

func (p *Resolver) has(name string) (bool, bool) {
	if p.isScopesEmpty() {
		return false, false
	}

	v, ok := p.peekScope()[name]
	if !ok {
		return false, false
	}

	return true, v
}

func (p *Resolver) VisitVarExpr(expr *ast.VarExpr) (any, error) {
	exists, defined := p.has(expr.Name.Lexeme)

	if !p.isScopesEmpty() && exists && !defined {
		return nil, errors.NewRuntimeError(
			expr.Name,
			fmt.Sprintf("Error at '%s': Cannot read local variable in its own initializer.", expr.Name.Lexeme),
		)
	}

	p.resolveLocal(expr, expr.Name)

	return nil, nil
}

func (p *Resolver) VisitBlockStmt(blockStmt *ast.BlockStmt) error {
	p.beginScope()
	if err := p.ResolveStmts(blockStmt.Stmts); err != nil {
		return err
	}
	p.endScope()

	return nil
}

func (p *Resolver) VisitIfStmt(blockStmt *ast.IfStmt) error {
	if _, err := p.resolveExpr(blockStmt.Condition); err != nil {
		return err
	}

	if err := p.resolveStmt(blockStmt.Then); err != nil {
		return err
	}

	if blockStmt.El != nil {
		if err := p.resolveStmt(blockStmt.El); err != nil {
			return err
		}
	}

	return nil
}

func (p *Resolver) VisitWhileStmt(blockStmt *ast.WhileStmt) error {
	if _, err := p.resolveExpr(blockStmt.Condition); err != nil {
		return err
	}

	return p.resolveStmt(blockStmt.Body)
}

func (p *Resolver) VisitFunctionStmt(fnStmt *ast.FunctionStmt) error {
	if err := p.declare(fnStmt.Name); err != nil {
		return err
	}

	p.define(fnStmt.Name)
	return p.resolveFn(fnStmt)
}

func (p *Resolver) VisitReturnStmt(returnStmt *ast.ReturnStmt) error {
	if !p.isFunction {
		return errors.NewRuntimeError(
			returnStmt.Keyword,
			fmt.Sprintf("Error at '%s': Cannot return from top-level code.", returnStmt.Keyword.Lexeme),
		)
	}

	if returnStmt.Value != nil {
		_, err := p.resolveExpr(returnStmt.Value)
		return err
	}

	return nil
}

func (p *Resolver) VisitAssignExpr(expr *ast.AssignExpr) (any, error) {
	_, err := p.resolveExpr(expr.Value)

	if err != nil {
		return nil, err
	}

	p.resolveLocal(expr, expr.Name)
	return nil, nil
}

func (p *Resolver) VisitBinaryExpr(expr *ast.BinaryExpr) (any, error) {
	if _, err := p.resolveExpr(expr.Left); err != nil {
		return nil, err
	}

	if _, err := p.resolveExpr(expr.Right); err != nil {
		return nil, err
	}

	return nil, nil
}

func (p *Resolver) VisitCallExpr(callExpr *ast.CallExpr) (any, error) {
	if _, err := p.resolveExpr(callExpr.Callee); err != nil {
		return nil, err
	}

	for _, arg := range callExpr.Args {
		if _, err := p.resolveExpr(arg); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (p *Resolver) VisitGroupingExpr(grExpr *ast.GroupingExpr) (any, error) {
	return p.resolveExpr(grExpr.Expr)
}

func (p *Resolver) VisitLiteralExpr(_ *ast.LiteralExpr) (any, error) {
	return nil, nil
}

func (p *Resolver) VisitLogicalExpr(expr *ast.LogicalExpr) (any, error) {
	if _, err := p.resolveExpr(expr.Left); err != nil {
		return nil, err
	}

	if _, err := p.resolveExpr(expr.Right); err != nil {
		return nil, err
	}

	return nil, nil
}

func (p *Resolver) VisitUnaryExpr(expr *ast.UnaryExpr) (any, error) {
	return p.resolveExpr(expr.Right)
}

func (p *Resolver) VisitArrayExpr(expr *ast.ArrayExpr) (any, error) {
	for _, item := range expr.Items {
		if _, err := p.resolveExpr(item); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (p *Resolver) VisitIndexExpr(expr *ast.IndexExpr) (any, error) {
	if _, err := p.resolveExpr(expr.Array); err != nil {
		return nil, err
	}

	if _, err := p.resolveExpr(expr.Index); err != nil {
		return nil, err
	}

	return nil, nil
}

func (p *Resolver) VisitSetIndexExpr(expr *ast.SetIndexExpr) (any, error) {
	if _, err := p.resolveExpr(expr.Array); err != nil {
		return nil, err
	}

	if _, err := p.resolveExpr(expr.Index); err != nil {
		return nil, err
	}

	if _, err := p.resolveExpr(expr.Value); err != nil {
		return nil, err
	}
	return nil, nil
}

func (p *Resolver) VisitObjectExpr(expr *ast.ObjectExpr) (any, error) {
	for _, pair := range expr.Pairs {
		if _, err := p.resolveExpr(pair); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (p *Resolver) ResolveStmts(stmts []ast.Stmt) error {
	for _, stmt := range stmts {
		if err := p.resolveStmt(stmt); err != nil {
			return err
		}
	}

	return nil
}

func (p *Resolver) resolveLocal(expr ast.Expr, name ast.Token) {
	for i := len(p.scopes) - 1; i >= 0; i-- {
		s := p.scopes[i]
		if _, exists := s[name.Lexeme]; exists {
			depth := len(p.scopes) - 1 - i

			p.interpreter.Resolve(expr, depth)
			return
		}
	}
}

func (p *Resolver) resolveStmt(stmt ast.Stmt) error {
	return stmt.Accept(p)
}

func (p *Resolver) resolveExpr(expr ast.Expr) (any, error) {
	return expr.Accept(p)
}

func (p *Resolver) beginScope() {
	p.scopes = append(p.scopes, make(Scope))
}

func (p *Resolver) endScope() {
	if len(p.scopes) > 0 {
		p.scopes = p.scopes[:len(p.scopes)-1]
	}
}

func (p *Resolver) resolveFn(fn *ast.FunctionStmt) error {
	p.isFunction = true
	p.beginScope()

	for _, fnParam := range fn.Parameters {
		if err := p.declare(fnParam); err != nil {
			return err
		}

		p.define(fnParam)
	}

	if err := p.ResolveStmts(fn.Body); err != nil {
		return err
	}

	p.endScope()

	return nil
}

func (p *Resolver) declare(name ast.Token) error {
	if p.isScopesEmpty() {
		return nil
	}

	if _, ok := p.peekScope()[name.Lexeme]; ok {
		return errors.NewRuntimeError(
			name,
			fmt.Sprintf("Error at '%s': Variable with this name already declared in this scope.", name.Lexeme),
		)
	}

	scope := p.peekScope()

	scope[name.Lexeme] = false

	return nil
}

func (p *Resolver) define(name ast.Token) {
	if p.isScopesEmpty() {
		return
	}

	scope := p.peekScope()
	scope[name.Lexeme] = true
}

func (p *Resolver) isScopesEmpty() bool {
	return len(p.scopes) == 0
}

func (p *Resolver) peekScope() Scope {
	return p.scopes[len(p.scopes)-1]
}
