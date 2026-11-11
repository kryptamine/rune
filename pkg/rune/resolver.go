package rune

import (
	"rune/pkg/ast"
	"rune/pkg/environment"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/stdlib"
	"rune/pkg/value"
)

type localVar struct {
	defined bool
	mutable bool
	slot    int
}

// Scope maps a local's name to its slot; size is the number of slots used.
type Scope struct {
	vars             map[string]*localVar
	size             int
	functionsAtStart int
}

// Resolver binds every local variable to a (depth, slot) pair stored in the
// AST before execution, and sizes each scope's environment. Blocks that
// declare nothing get no scope at all, matching the interpreter, which runs
// them in the enclosing environment.
// Semantic errors do not confuse it, so it records them and keeps going;
// the visitor methods therefore always return nil errors.
type Resolver struct {
	scopes     []*Scope
	isFunction bool
	// loopDepth counts the loops enclosing the current statement within
	// the current function; break and continue need it to be positive.
	loopDepth int
	functions int
	errors    []error
	// globals tracks top-level declarations like a scope's vars; their
	// slots are global ones and their mutability is in globalMutable, so
	// localVar.slot and localVar.mutable are unused.
	globals map[string]*localVar
	// globalMutable holds every top-level declaration up front, since a
	// function may assign to a global declared after it.
	globalMutable map[string]bool
}

func NewResolver() *Resolver {
	return &Resolver{globals: map[string]*localVar{}, globalMutable: map[string]bool{}}
}

// ResolveStmts resolves a whole program and returns all errors found.
func (p *Resolver) ResolveStmts(stmts []ast.Stmt) []error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.VarStmt:
			p.globalMutable[s.Name.Lexeme] = s.Mutable
		case *ast.FunctionStmt:
			p.globalMutable[s.Name.Lexeme] = false
		}
	}

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
	stmt.Local, stmt.Slot = p.declare(stmt.Binding)

	if stmt.Initializer != nil {
		p.resolveExpr(stmt.Initializer)
	}

	p.define(stmt.Name)

	return nil
}

func (p *Resolver) VisitVarExpr(expr *ast.VarExpr) (value.Value, error) {
	if l, exists := p.innermost()[expr.Name.Lexeme]; exists && !l.defined {
		kind := helpers.If(p.isScopesEmpty(), "global", "local")
		p.error(expr.Name, "Cannot read "+kind+" variable in its own initializer.")
	}

	expr.Depth, expr.Slot = p.resolveLocal(expr.Name)

	return value.NilValue, nil
}

func (p *Resolver) VisitBlockStmt(blockStmt *ast.BlockStmt) error {
	if !declaresLocals(blockStmt.Stmts) {
		p.resolveStmts(blockStmt.Stmts)
		return nil
	}

	p.beginScope()
	p.resolveStmts(blockStmt.Stmts)
	blockStmt.EnvSize, blockStmt.Pooled = p.endScope()

	return nil
}

// declaresLocals reports whether stmts directly declare a variable or function.
func declaresLocals(stmts []ast.Stmt) bool {
	for _, stmt := range stmts {
		switch stmt.(type) {
		case *ast.VarStmt, *ast.FunctionStmt:
			return true
		}
	}

	return false
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
	p.resolveLoopBody(whileStmt.Body)

	if whileStmt.Increment != nil {
		p.resolveExpr(whileStmt.Increment)
	}

	return nil
}

func (p *Resolver) VisitForInStmt(stmt *ast.ForInStmt) error {
	p.resolveExpr(stmt.Iterable)

	p.beginScope()

	if stmt.Key != nil {
		p.declare(*stmt.Key)
		p.define(stmt.Key.Name)
	}

	p.declare(stmt.Value)
	p.define(stmt.Value.Name)

	p.resolveLoopBody(stmt.Body)
	stmt.EnvSize, stmt.Pooled = p.endScope()

	return nil
}

func (p *Resolver) resolveLoopBody(body ast.Stmt) {
	p.loopDepth++
	p.resolveStmt(body)
	p.loopDepth--
}

func (p *Resolver) VisitBreakStmt(stmt *ast.BreakStmt) error {
	if p.loopDepth == 0 {
		p.error(stmt.Keyword, "Cannot use 'break' outside of a loop.")
	}

	return nil
}

func (p *Resolver) VisitContinueStmt(stmt *ast.ContinueStmt) error {
	if p.loopDepth == 0 {
		p.error(stmt.Keyword, "Cannot use 'continue' outside of a loop.")
	}

	return nil
}

func (p *Resolver) VisitFunctionExpr(expr *ast.FunctionExpr) (value.Value, error) {
	p.resolveFn(expr.Decl)

	return value.NilValue, nil
}

func (p *Resolver) VisitFunctionStmt(fnStmt *ast.FunctionStmt) error {
	fnStmt.Local, fnStmt.Slot = p.declare(ast.Binding{Name: fnStmt.Name})
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

func (p *Resolver) VisitTryStmt(tryStmt *ast.TryStmt) error {
	p.resolveStmt(tryStmt.Body)

	if tryStmt.Param == nil && !declaresLocals(tryStmt.Handler) {
		p.resolveStmts(tryStmt.Handler)
		return nil
	}

	p.beginScope()

	if tryStmt.Param != nil {
		p.declare(ast.Binding{Name: *tryStmt.Param})
		p.define(*tryStmt.Param)
	}

	p.resolveStmts(tryStmt.Handler)
	tryStmt.HandlerEnvSize, tryStmt.HandlerPooled = p.endScope()

	return nil
}

func (p *Resolver) VisitThrowStmt(throwStmt *ast.ThrowStmt) error {
	p.resolveExpr(throwStmt.Value)
	return nil
}

func (p *Resolver) VisitAssignExpr(expr *ast.AssignExpr) (value.Value, error) {
	p.resolveExpr(expr.Value)
	expr.Depth, expr.Slot = p.resolveLocal(expr.Name)
	p.requireMutable(expr.Name, "assign to")

	return value.NilValue, nil
}

func (p *Resolver) VisitBinaryExpr(expr *ast.BinaryExpr) (value.Value, error) {
	p.resolveExpr(expr.Left)
	p.resolveExpr(expr.Right)

	return value.NilValue, nil
}

func (p *Resolver) VisitCallExpr(callExpr *ast.CallExpr) (value.Value, error) {
	p.resolveExpr(callExpr.Callee)

	for _, arg := range callExpr.Args {
		p.resolveExpr(arg)
	}

	p.requireMutableArg(callExpr)

	return value.NilValue, nil
}

func (p *Resolver) VisitGroupingExpr(grExpr *ast.GroupingExpr) (value.Value, error) {
	p.resolveExpr(grExpr.Expr)

	return value.NilValue, nil
}

func (p *Resolver) VisitLiteralExpr(_ *ast.LiteralExpr) (value.Value, error) {
	return value.NilValue, nil
}

func (p *Resolver) VisitLogicalExpr(expr *ast.LogicalExpr) (value.Value, error) {
	p.resolveExpr(expr.Left)
	p.resolveExpr(expr.Right)

	return value.NilValue, nil
}

func (p *Resolver) VisitUnaryExpr(expr *ast.UnaryExpr) (value.Value, error) {
	p.resolveExpr(expr.Right)

	return value.NilValue, nil
}

func (p *Resolver) VisitArrayExpr(expr *ast.ArrayExpr) (value.Value, error) {
	for _, item := range expr.Items {
		p.resolveExpr(item)
	}

	return value.NilValue, nil
}

func (p *Resolver) VisitIndexExpr(expr *ast.IndexExpr) (value.Value, error) {
	p.resolveExpr(expr.Array)
	p.resolveExpr(expr.Index)

	return value.NilValue, nil
}

func (p *Resolver) VisitSetIndexExpr(expr *ast.SetIndexExpr) (value.Value, error) {
	p.resolveExpr(expr.Array)
	p.resolveExpr(expr.Index)
	p.resolveExpr(expr.Value)
	p.requireMutableRoot(expr.Array)

	return value.NilValue, nil
}

func (p *Resolver) VisitUpdateExpr(expr *ast.UpdateExpr) (value.Value, error) {
	p.resolveExpr(expr.Target)

	switch t := expr.Target.(type) {
	case *ast.VarExpr:
		p.requireMutable(t.Name, "assign to")
	case *ast.IndexExpr:
		p.requireMutableRoot(t.Array)
	}

	return value.NilValue, nil
}

func (p *Resolver) VisitObjectExpr(expr *ast.ObjectExpr) (value.Value, error) {
	for _, pair := range expr.Pairs {
		p.resolveExpr(pair.Value)
	}

	return value.NilValue, nil
}

// requireMutable reports name if it is bound immutably; verb says what was tried.
func (p *Resolver) requireMutable(name ast.Token, verb string) {
	if !p.isMutable(name.Lexeme) {
		p.error(name, "Cannot "+verb+" immutable variable '"+name.Lexeme+"'.")
	}
}

// requireMutableRoot checks the variable an element or property chain starts from.
func (p *Resolver) requireMutableRoot(expr ast.Expr) {
	if root := rootVar(expr); root != nil {
		p.requireMutable(root.Name, "mutate")
	}
}

// requireMutableArg checks the first argument of a call to an in-place built-in.
func (p *Resolver) requireMutableArg(call *ast.CallExpr) {
	callee, ok := call.Callee.(*ast.VarExpr)
	if ok && len(call.Args) > 0 && stdlib.InPlace[callee.Name.Lexeme] && p.isBuiltin(callee.Name.Lexeme) {
		p.requireMutableRoot(call.Args[0])
	}
}

// rootVar returns the variable an element or property chain starts from, if any.
func rootVar(expr ast.Expr) *ast.VarExpr {
	for {
		switch e := expr.(type) {
		case *ast.IndexExpr:
			expr = e.Array
		case *ast.VarExpr:
			return e
		default:
			return nil
		}
	}
}

// isMutable reports whether name, as bound here, may be assigned to;
// names the program never declares are built-ins, which may be redefined.
func (p *Resolver) isMutable(name string) bool {
	if l := p.lookupScopes(name); l != nil {
		return l.mutable
	}

	mutable, declared := p.globalMutable[name]

	return !declared || mutable
}

func (p *Resolver) isBuiltin(name string) bool {
	_, declared := p.globalMutable[name]

	return !declared && p.lookupScopes(name) == nil
}

func (p *Resolver) lookupScopes(name string) *localVar {
	for i := len(p.scopes) - 1; i >= 0; i-- {
		if l, exists := p.scopes[i].vars[name]; exists {
			return l
		}
	}

	return nil
}

func (p *Resolver) error(token ast.Token, msg string) {
	p.errors = append(p.errors, errors.NewSyntaxError(token, msg))
}

// resolveLocal returns the depth and slot of name; for a global the depth
// is Unresolved and the slot is the global one.
func (p *Resolver) resolveLocal(name ast.Token) (int, int) {
	for i := len(p.scopes) - 1; i >= 0; i-- {
		if l, exists := p.scopes[i].vars[name.Lexeme]; exists {
			return len(p.scopes) - 1 - i, l.slot
		}
	}

	return ast.Unresolved, environment.GlobalSlot(name.Lexeme)
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
	p.scopes = append(p.scopes, &Scope{vars: map[string]*localVar{}, functionsAtStart: p.functions})
}

func (p *Resolver) endScope() (size int, pooled bool) {
	scope := p.peekScope()
	p.scopes = p.scopes[:len(p.scopes)-1]

	return scope.size, scope.functionsAtStart == p.functions
}

func (p *Resolver) resolveFn(fn *ast.FunctionStmt) {
	enclosingIsFunction, enclosingLoopDepth := p.isFunction, p.loopDepth
	p.isFunction, p.loopDepth = true, 0
	p.functions++
	defer func() { p.isFunction, p.loopDepth = enclosingIsFunction, enclosingLoopDepth }()

	p.beginScope()

	for _, param := range fn.Parameters {
		p.declare(param)
		p.define(param.Name)
	}

	p.resolveStmts(fn.Body)
	fn.EnvSize, fn.Pooled = p.endScope()
}

// declare adds name to the innermost scope and returns its slot; at the
// top level it returns the name's global slot and local is false.
func (p *Resolver) declare(b ast.Binding) (local bool, slot int) {
	name := b.Name

	if _, exists := p.innermost()[name.Lexeme]; exists {
		p.error(name, "Variable with this name already declared in this scope.")
	}

	if p.isScopesEmpty() {
		p.globals[name.Lexeme] = &localVar{}
		return false, environment.GlobalSlot(name.Lexeme)
	}

	scope := p.peekScope()
	slot = scope.size
	scope.size++
	scope.vars[name.Lexeme] = &localVar{slot: slot, mutable: b.Mutable}

	return true, slot
}

func (p *Resolver) define(name ast.Token) {
	p.innermost()[name.Lexeme].defined = true
}

// innermost returns the declarations of the current scope, the top level included.
func (p *Resolver) innermost() map[string]*localVar {
	if p.isScopesEmpty() {
		return p.globals
	}

	return p.peekScope().vars
}

func (p *Resolver) isScopesEmpty() bool {
	return len(p.scopes) == 0
}

func (p *Resolver) peekScope() *Scope {
	return p.scopes[len(p.scopes)-1]
}
