package rune

import (
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/stdlib"
	"rune/pkg/value"
)

type localVar struct {
	defined bool
	mutable bool
	local   *ast.Local
	name    ast.Token
}

// Scope maps a local's name to its declaration; fnDepth is the nesting
// depth of the function the scope belongs to.
type Scope struct {
	vars    map[string]*localVar
	fnDepth int
}

// Resolver binds every reference to a local variable to its declaration
// and marks the locals that nested functions capture.
// Semantic errors do not confuse it, so it records them and keeps going;
// the visitor methods therefore always return nil errors.
type Resolver struct {
	scopes     []*Scope
	isFunction bool
	// loopDepth counts the loops enclosing the current statement within
	// the current function; break and continue need it to be positive.
	loopDepth int
	fnDepth   int
	errors    []error
	// globals tracks top-level declarations like a scope's vars; their
	// mutability is in globalMutable, so localVar.mutable is unused.
	globals map[string]*localVar
	// globalMutable holds every top-level declaration up front, since a
	// function may assign to a global declared after it.
	globalMutable map[string]bool
	// topLevel holds the declarations of every top-level name up front,
	// for the same reason.
	topLevel map[string]*ast.Local
	// builtins holds a declaration for each built-in the program names
	// without declaring it.
	builtins map[string]*ast.Local
	// structs holds every struct type up front, by its name's declaration.
	structs map[*ast.Local]*value.Shape
	// fieldNames holds each field name's one string, shared by every struct
	// type that has the field.
	fieldNames map[string]string
	// main is the program's entry point, if it declares one.
	main *ast.FunctionStmt
}

func NewResolver() *Resolver {
	return &Resolver{
		globals:       map[string]*localVar{},
		globalMutable: map[string]bool{},
		topLevel:      map[string]*ast.Local{},
		builtins:      map[string]*ast.Local{},
		structs:       map[*ast.Local]*value.Shape{},
		fieldNames:    errorFieldNames(),
	}
}

func errorFieldNames() map[string]string {
	names := map[string]string{}
	for _, k := range errors.Kinds {
		for _, f := range k.Fields {
			names[f] = f
		}
	}

	return names
}

// ResolveStmts resolves a whole program and returns all errors found.
func (p *Resolver) ResolveStmts(stmts []ast.Stmt) []error {
	for _, stmt := range stmts {
		fn, _ := stmt.(*ast.FunctionStmt)
		for _, b := range ast.Declared(stmt) {
			p.globalMutable[b.Name.Lexeme] = b.Mutable
			p.topLevel[b.Name.Lexeme] = &ast.Local{TopLevel: true}
			p.declareMain(b.Name, fn)
		}

		if s, ok := stmt.(*ast.StructStmt); ok {
			p.declareMain(s.Name, nil)
			p.globalMutable[s.Name.Lexeme] = false
			l := &ast.Local{TopLevel: true}
			p.topLevel[s.Name.Lexeme] = l
			s.Shape = p.shape(s)
			p.structs[l] = s.Shape
		}
	}

	p.resolveStmts(stmts)

	return p.errors
}

// declareMain takes a top-level declaration of name as the entry point
// if it is main, which must be a function of the entry file taking at
// most the program's arguments.
func (p *Resolver) declareMain(name ast.Token, fn *ast.FunctionStmt) {
	switch {
	case name.Lexeme != "main":
	case fn == nil:
		p.error(name, "main must be a function.")
	case name.File != 0:
		p.error(name, "Only the entry file can declare main.")
	case len(fn.Parameters) > 1:
		p.error(name, "main takes at most one parameter.")
	default:
		p.main = fn
	}
}

// ResolveExpr resolves a lone expression.
func (p *Resolver) ResolveExpr(expr ast.Expr) []error {
	p.resolveExpr(expr)

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
	stmt.Local = p.declare(stmt.Binding)

	if stmt.Initializer != nil {
		p.resolveExpr(stmt.Initializer)
	}

	p.define(stmt.Name)

	return nil
}

func (p *Resolver) VisitDestructureStmt(stmt *ast.DestructureStmt) error {
	bindings := ast.Declared(stmt)
	for _, b := range bindings {
		b.Local = p.declare(*b)
	}

	p.resolveExpr(stmt.Initializer)

	for _, b := range bindings {
		p.define(b.Name)
	}

	return nil
}

func (p *Resolver) requireInitialized(name ast.Token, verb string) {
	if l, exists := p.innermost()[name.Lexeme]; exists && !l.defined {
		kind := helpers.If(p.isScopesEmpty(), "global", "local")
		p.error(name, "Cannot "+verb+" "+kind+" variable in its own initializer.")
	}
}

func (p *Resolver) VisitVarExpr(expr *ast.VarExpr) (value.Value, error) {
	p.requireInitialized(expr.Name, "read")
	expr.Local = p.resolveLocal(expr.Name)

	return value.NilValue, nil
}

func (p *Resolver) VisitBlockStmt(blockStmt *ast.BlockStmt) error {
	if !declaresLocals(blockStmt.Stmts) {
		p.resolveStmts(blockStmt.Stmts)
		return nil
	}

	p.beginScope()
	p.resolveStmts(blockStmt.Stmts)
	p.endScope()

	return nil
}

// declaresLocals reports whether stmts directly declare a variable or function.
func declaresLocals(stmts []ast.Stmt) bool {
	for _, stmt := range stmts {
		if len(ast.Declared(stmt)) > 0 {
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

	return nil
}

func (p *Resolver) VisitForInStmt(stmt *ast.ForInStmt) error {
	p.resolveExpr(stmt.Iterable)

	p.beginScope()

	if stmt.Key != nil {
		stmt.Key.Local = p.declare(*stmt.Key)
		p.define(stmt.Key.Name)
	}

	stmt.Value.Local = p.declare(stmt.Value)
	p.define(stmt.Value.Name)

	p.resolveLoopBody(stmt.Body)
	p.endScope()

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
	fnStmt.Local = p.declare(fnStmt.Binding)
	p.define(fnStmt.Name)
	p.resolveFn(fnStmt)

	return nil
}

func (p *Resolver) VisitStructStmt(stmt *ast.StructStmt) error {
	if !p.isScopesEmpty() {
		p.error(stmt.Name, "Structs must be declared at the top level.")
		return nil
	}

	stmt.Local = p.declare(ast.Binding{Name: stmt.Name})
	p.define(stmt.Name)

	return nil
}

func (p *Resolver) shape(s *ast.StructStmt) *value.Shape {
	fields := make([]value.Field, len(s.Fields))

	for i, f := range s.Fields {
		name, ok := p.fieldNames[f.Name.Lexeme]
		if !ok {
			name = f.Name.Lexeme
			p.fieldNames[name] = name
		}

		def, _ := literalOf(f.Default)
		fields[i] = value.Field{Name: name, Default: def, Required: f.Default == nil}
	}

	return value.NewShape(s.Name.Lexeme, fields)
}

func (p *Resolver) VisitStructExpr(expr *ast.StructExpr) (value.Value, error) {
	for _, pair := range expr.Pairs {
		p.resolveExpr(pair.Value)
	}

	expr.Local = p.resolveLocal(expr.Name)

	shape := p.structs[expr.Local]
	if shape == nil {
		p.error(expr.Name, "'"+expr.Name.Lexeme+"' is not a struct.")
		return value.NilValue, nil
	}

	expr.Shape = shape
	given := make([]bool, len(shape.Fields))

	for i, pair := range expr.Pairs {
		slot := shape.Index(pair.Key.Lexeme)
		if slot < 0 {
			p.error(pair.Key, noField(shape.Name, pair.Key.Lexeme))
			continue
		}

		expr.Pairs[i].Slot = slot
		given[slot] = true
	}

	for i, f := range shape.Fields {
		if _, optional := shape.Default(i); !given[i] && !optional {
			p.error(expr.Name, missingField(shape.Name, f))
		}
	}

	return value.NilValue, nil
}

func (p *Resolver) VisitWithExpr(expr *ast.WithExpr) (value.Value, error) {
	p.resolveExpr(expr.Target)

	for _, pair := range expr.Pairs {
		p.resolveExpr(pair.Value)
	}

	return value.NilValue, nil
}

func (p *Resolver) VisitIsExpr(expr *ast.IsExpr) (value.Value, error) {
	p.resolveExpr(expr.Value)
	p.resolveExpr(expr.Type)

	return value.NilValue, nil
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

	caught := map[string]bool{}

	for i := range tryStmt.Catches {
		c := &tryStmt.Catches[i]

		if len(c.Types) == 0 && i < len(tryStmt.Catches)-1 {
			p.error(c.Keyword, "Catch-all must be the last catch clause.")
		}

		for _, t := range c.Types {
			p.resolveExpr(t)

			if p.structs[t.Local] == nil {
				p.error(t.Name, "'"+t.Name.Lexeme+"' is not a struct.")
			} else if caught[t.Name.Lexeme] {
				p.error(t.Name, "'"+t.Name.Lexeme+"' is already caught.")
			}

			caught[t.Name.Lexeme] = true
		}

		p.resolveCatch(c)
	}

	return nil
}

func (p *Resolver) resolveCatch(c *ast.CatchClause) {
	if c.Param == nil && !declaresLocals(c.Body) {
		p.resolveStmts(c.Body)
		return
	}

	p.beginScope()

	if c.Param != nil {
		c.ParamLocal = p.declare(ast.Binding{Name: *c.Param})
		p.define(*c.Param)
	}

	p.resolveStmts(c.Body)
	p.endScope()
}

func (p *Resolver) VisitThrowStmt(throwStmt *ast.ThrowStmt) error {
	p.resolveExpr(throwStmt.Value)
	return nil
}

func (p *Resolver) VisitAssignExpr(expr *ast.AssignExpr) (value.Value, error) {
	p.resolveExpr(expr.Value)
	p.requireInitialized(expr.Name, "assign to")
	expr.Local = p.resolveLocal(expr.Name)
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
	if p.resolveMember(expr) {
		return value.NilValue, nil
	}

	p.resolveExpr(expr.Array)
	p.resolveExpr(expr.Index)

	return value.NilValue, nil
}

// resolveMember binds a literal member of an unshadowed namespace to the
// built-in it names, so it costs nothing at run time.
func (p *Resolver) resolveMember(expr *ast.IndexExpr) bool {
	ns, ok := expr.Array.(*ast.VarExpr)
	if !ok || !p.isBuiltin(ns.Name.Lexeme) {
		return false
	}

	v := builtinValues()[ns.Name.Lexeme]
	key, ok := expr.Index.(*ast.LiteralExpr)

	if v.Kind() != value.Struct || !ok || !key.Value.IsString() {
		return false
	}

	i := v.Shape().Index(key.Value.Str())
	if i < 0 {
		p.error(expr.Token, ns.Name.Lexeme+" has no member '"+key.Value.Str()+"'.")
		return true
	}

	expr.Member = &v.Fields()[i]

	return true
}

func (p *Resolver) VisitSetIndexExpr(expr *ast.SetIndexExpr) (value.Value, error) {
	p.resolveExpr(expr.Array)
	p.resolveExpr(expr.Index)
	p.resolveExpr(expr.Value)
	p.requireMutableRoot(expr.Array)

	return value.NilValue, nil
}

func (p *Resolver) VisitRangeExpr(expr *ast.RangeExpr) (value.Value, error) {
	p.resolveExpr(expr.Start)
	p.resolveExpr(expr.End)

	if expr.Step != nil {
		p.resolveExpr(expr.Step)
	}

	return value.NilValue, nil
}

func (p *Resolver) VisitUpdateExpr(expr *ast.UpdateExpr) (value.Value, error) {
	switch t := expr.Target.(type) {
	case *ast.VarExpr:
		p.resolveExpr(t)
		p.requireMutable(t.Name, "assign to")
	case *ast.IndexExpr:
		// Not resolveExpr: a written member must stay a real lookup.
		p.resolveExpr(t.Array)
		p.resolveExpr(t.Index)
		p.requireMutableRoot(t.Array)
	default:
		p.resolveExpr(t)
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
	callee, ok := call.Callee.(*ast.IndexExpr)
	if ok && callee.Member != nil && len(call.Args) > 0 && stdlib.InPlace(*callee.Member) {
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

// resolveLocal returns the declaration name refers to, or nil for a name
// neither declared nor built in.
func (p *Resolver) resolveLocal(name ast.Token) *ast.Local {
	for i := len(p.scopes) - 1; i >= 0; i-- {
		if l, exists := p.scopes[i].vars[name.Lexeme]; exists {
			if p.scopes[i].fnDepth < p.fnDepth {
				l.local.Captured = true
			}

			return l.local
		}
	}

	l := p.topLevel[name.Lexeme]
	if l == nil {
		l = p.builtin(name.Lexeme)
	}

	if l != nil && p.fnDepth > 0 {
		l.Captured = true
	}

	return l
}

func (p *Resolver) builtin(name string) *ast.Local {
	if _, ok := builtinValues()[name]; !ok {
		return nil
	}

	l, ok := p.builtins[name]
	if !ok {
		l = &ast.Local{TopLevel: true}
		p.builtins[name] = l

		if v := builtinValues()[name]; v.Kind() == value.Type {
			p.structs[l] = v.TypeShape()
		}
	}

	return l
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
	p.scopes = append(p.scopes, &Scope{vars: map[string]*localVar{}, fnDepth: p.fnDepth})
}

func (p *Resolver) endScope() {
	p.scopes = p.scopes[:len(p.scopes)-1]
}

func (p *Resolver) resolveFn(fn *ast.FunctionStmt) {
	enclosingIsFunction, enclosingLoopDepth := p.isFunction, p.loopDepth
	p.isFunction, p.loopDepth = true, 0
	p.fnDepth++
	defer func() { p.isFunction, p.loopDepth = enclosingIsFunction, enclosingLoopDepth; p.fnDepth-- }()

	p.beginScope()

	for i, param := range fn.Parameters {
		fn.Parameters[i].Local = p.declare(param)
		p.define(param.Name)
	}

	p.resolveStmts(fn.Body)
	p.endScope()
}

// declare adds name to the innermost scope and returns its declaration.
func (p *Resolver) declare(b ast.Binding) *ast.Local {
	name := b.Name

	if prev, exists := p.innermost()[name.Lexeme]; exists {
		p.errors = append(p.errors, errors.NewRedeclaredError(name, prev.name))
	}

	if p.isScopesEmpty() {
		p.globals[name.Lexeme] = &localVar{name: name}
		return p.topLevel[name.Lexeme]
	}

	l := &ast.Local{}
	p.peekScope().vars[name.Lexeme] = &localVar{mutable: b.Mutable, local: l, name: name}

	return l
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
