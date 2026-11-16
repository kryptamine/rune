package rune

import (
	"path/filepath"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/stdlib"
	"rune/pkg/value"
)

type localVar struct {
	defined bool
	mutable bool
	// fixed is set on a function, type, module or catch variable, which
	// no mut makes mutable.
	fixed bool
	local *ast.Local
	name  ast.Token
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
	// inDefer is set in a defer body, which return, break and continue
	// cannot leave.
	inDefer bool
	fnDepth int
	errors  []error
	// mod is the top level of the file being resolved; modules holds
	// every file's, by file index.
	mod     *moduleScope
	modules []*moduleScope
	// root is where the program's files are on disk, for module.dir.
	root string
	// structs holds every struct type up front, by its name's declaration;
	// cases holds the constant each case of an enum is read as.
	structs map[*ast.Local]*value.Shape
	cases   map[*value.Shape]*ast.Local
	// fieldNames holds each field name's one string, shared by every struct
	// type that has the field.
	fieldNames map[string]string
	// main is the program's entry point, if it declares one.
	main *ast.FunctionStmt
	// kinds holds the methods of the built-in types; nmethods counts all.
	kinds    [value.NumKinds][]value.Method
	nmethods int
	// impls holds the impl blocks at the top level of a file.
	impls       map[*ast.ImplStmt]bool
	methodKinds map[string]methodKind
	// methodInout has, for each method name, a bit for each argument a
	// mut parameter of a method of that name takes.
	methodInout map[string]uint32
	// anyInout has a bit for each argument a mut parameter of some
	// function of the program takes.
	anyInout uint32
	// fns holds the function each fun, or let of an arrow function, binds.
	fns map[*ast.Local]*ast.FunctionStmt
	// uses counts the references to each variable so far.
	uses map[*ast.Local]int
	// consts holds the constants of the file being resolved, and how far
	// each one's value is; constDecls finds them by declaration.
	consts     map[*ast.VarStmt]constState
	constDecls map[*ast.Local]*ast.VarStmt
	// hints holds a fix for each name that names no variable, if one is
	// close enough to be what was meant.
	hints map[ast.Token]string
}

func NewResolver() *Resolver {
	mod := newModuleScope(newFileNamespace("", ""))

	p := &Resolver{
		mod:        mod,
		modules:    []*moduleScope{mod},
		structs:    map[*ast.Local]*value.Shape{},
		cases:      map[*value.Shape]*ast.Local{},
		fieldNames: stdlib.FieldNames(),
		impls:      map[*ast.ImplStmt]bool{},
		fns:        map[*ast.Local]*ast.FunctionStmt{},
		uses:       map[*ast.Local]int{},
		nmethods:   len(stdlib.Methods),
		hints:      map[ast.Token]string{},
	}

	for i, m := range stdlib.Methods {
		p.kinds[m.Kind] = append(p.kinds[m.Kind], value.Method{Name: p.intern(m.Name), ID: i, Self: true, InPlace: m.InPlace})
	}

	return p
}

func (p *Resolver) methodTable() methodTable { return methodTable{kinds: p.kinds, n: p.nmethods} }

// resolveTopLevel resolves the statements of one file.
func (p *Resolver) resolveTopLevel(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		fn, _ := stmt.(*ast.FunctionStmt)
		for _, b := range ast.Declared(stmt) {
			p.mod.globalMutable[b.Name.Lexeme] = b.Mutable
			p.mod.topLevel[b.Name.Lexeme] = &ast.Local{TopLevel: true}
			p.declareMain(b.Name, fn)

			if fn != nil {
				p.fns[p.mod.topLevel[b.Name.Lexeme]] = fn
			}
		}

		if s, ok := stmt.(*ast.StructStmt); ok {
			p.declareMain(s.Name, nil)
			p.mod.globalMutable[s.Name.Lexeme] = false
			p.mod.topLevel[s.Name.Lexeme] = &ast.Local{TopLevel: true}
		}
	}

	// Enums first, as constants and field defaults may name their
	// variants; structs after the constants their defaults may name.
	p.shapeStructs(stmts, true)
	p.resolveConsts(stmts)
	p.shapeStructs(stmts, false)

	for _, stmt := range stmts {
		if s, ok := stmt.(*ast.ImplStmt); ok {
			p.impls[s] = true
			p.declareImpl(s)
		}
	}

	p.resolveStmts(stmts)
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
	case fn.Rest:
		p.error(name, "main cannot take a rest parameter.")
	default:
		p.main = fn
	}
}

func (p *Resolver) VisitExprStmt(exprStmt *ast.ExprStmt) error {
	p.resolveExpr(exprStmt.Expr)
	return nil
}

func (p *Resolver) VisitVarStmt(stmt *ast.VarStmt) error {
	if stmt.Const {
		if _, topLevel := p.consts[stmt]; !topLevel {
			p.error(stmt.Name, "Constants must be declared at the top level.")
		}

		stmt.Local = p.declare(stmt.Binding)
		p.define(stmt.Name)

		return nil
	}

	stmt.Local = p.declare(stmt.Binding)

	if f, ok := stmt.Initializer.(*ast.FunctionExpr); ok && !stmt.Mutable {
		p.fns[stmt.Local] = f.Decl
	}

	p.resolveExpr(stmt.Initializer)

	p.define(stmt.Name)

	return nil
}

func (p *Resolver) VisitDestructureStmt(stmt *ast.DestructureStmt) error {
	p.resolveDestructure(stmt, stmt.Initializer)
	return nil
}

func (p *Resolver) VisitArrayDestructureStmt(stmt *ast.ArrayDestructureStmt) error {
	p.resolveDestructure(stmt, stmt.Initializer)
	return nil
}

// resolveDestructure declares the variables stmt binds, each defined
// once initializer and its default are resolved.
func (p *Resolver) resolveDestructure(stmt ast.Stmt, initializer ast.Expr) {
	vars := ast.PatternVars(stmt)
	for _, v := range vars {
		v.Local = p.declare(v.Binding)
	}

	p.resolveExpr(initializer)

	for _, v := range vars {
		p.resolveExpr(v.Default)

		p.define(v.Name)
	}
}

func (p *Resolver) requireInitialized(name ast.Token, verb string) {
	if l, exists := p.innermost()[name.Lexeme]; exists && !l.defined {
		kind := helpers.If(p.isScopesEmpty(), "global", "local")
		p.error(name, "Cannot "+verb+" "+kind+" variable in its own initializer.")
	}
}

// use resolves name, which is read or assigned to; ok is false for _,
// which names no variable.
func (p *Resolver) use(name ast.Token, verb string) (l *ast.Local, ok bool) {
	if (ast.Binding{Name: name}).Discards() {
		p.error(name, "Cannot "+verb+" '_'.")
		return nil, false
	}

	p.requireInitialized(name, verb)

	l = p.resolveLocal(name)
	if l == nil {
		if hint := p.unknownName(name.Lexeme); hint != "" {
			p.hints[name] = hint
		}
	}

	return l, true
}

func (p *Resolver) VisitVarExpr(expr *ast.VarExpr) (value.Value, error) {
	if expr.Module != nil {
		expr.Local = p.qualified(expr)
		return value.NilValue, nil
	}

	if q := p.qualifier(expr); q != "" {
		p.error(expr.Name, q+" '"+expr.Name.Lexeme+"' is not a value; use its members, as in "+expr.Name.Lexeme+".name.")
		return value.NilValue, nil
	}

	expr.Local, _ = p.use(expr.Name, "read")

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

	var root *ast.Local
	if stmt.Value.Mutable {
		p.requirePlace(stmt)

		if v := rootVar(stmt.Iterable); v != nil {
			root = v.Local
		}
	}

	uses := p.uses[root]

	p.beginScope()

	if stmt.Key != nil {
		if stmt.Key.Mutable {
			err := errors.NewSyntaxError(stmt.Key.Name, "A for-in key cannot be mut.")
			p.errors = append(p.errors, errors.WithHint(err, "copy it into a 'let mut' variable in the body"))
		}

		stmt.Key.Local = p.declare(*stmt.Key)
		p.define(stmt.Key.Name)
	}

	stmt.Value.Local = p.declare(stmt.Value)
	p.define(stmt.Value.Name)

	p.resolveLoopBody(stmt.Body)
	p.endScope()

	stmt.Alone = root != nil && p.uses[root] == uses

	return nil
}

func (p *Resolver) resolveLoopBody(body ast.Stmt) {
	p.loopDepth++
	p.resolveStmt(body)
	p.loopDepth--
}

func (p *Resolver) VisitBreakStmt(stmt *ast.BreakStmt) error {
	p.checkJump(stmt.Keyword)
	return nil
}

func (p *Resolver) VisitContinueStmt(stmt *ast.ContinueStmt) error {
	p.checkJump(stmt.Keyword)
	return nil
}

func (p *Resolver) checkJump(keyword ast.Token) {
	switch {
	case p.loopDepth == 0 && p.inDefer:
		p.error(keyword, "Cannot use '"+keyword.Lexeme+"' to leave a 'defer' body.")
	case p.loopDepth == 0:
		p.error(keyword, "Cannot use '"+keyword.Lexeme+"' outside of a loop.")
	}
}

func (p *Resolver) VisitDeferStmt(stmt *ast.DeferStmt) error {
	inDefer, loopDepth := p.inDefer, p.loopDepth
	p.inDefer, p.loopDepth = true, 0
	p.resolveStmt(stmt.Body)
	p.inDefer, p.loopDepth = inDefer, loopDepth

	return nil
}

func (p *Resolver) VisitFunctionExpr(expr *ast.FunctionExpr) (value.Value, error) {
	p.resolveFn(expr.Decl)

	return value.NilValue, nil
}

func (p *Resolver) VisitFunctionStmt(fnStmt *ast.FunctionStmt) error {
	fnStmt.Local = p.declare(fnStmt.Binding)
	p.fns[fnStmt.Local] = fnStmt
	p.fix(fnStmt.Name)
	p.define(fnStmt.Name)
	p.resolveFn(fnStmt)

	return nil
}

func (p *Resolver) VisitIsExpr(expr *ast.IsExpr) (value.Value, error) {
	p.resolveExpr(expr.Value)

	if t, ok := expr.Type.(*ast.VarExpr); ok {
		expr.Shape = p.typeNamed(t)
	} else {
		p.resolveExpr(expr.Type)
	}

	return value.NilValue, nil
}

func (p *Resolver) VisitReturnStmt(returnStmt *ast.ReturnStmt) error {
	switch {
	case p.inDefer:
		p.error(returnStmt.Keyword, "Cannot return from a 'defer' body.")
	case !p.isFunction:
		p.error(returnStmt.Keyword, "Cannot return from top-level code.")
	}

	p.resolveExpr(returnStmt.Value)

	return nil
}

func (p *Resolver) VisitTryStmt(tryStmt *ast.TryStmt) error {
	p.resolveStmt(tryStmt.Body)
	p.resolveCatches(tryStmt.Catches)

	return nil
}

func (p *Resolver) VisitTryExpr(expr *ast.TryExpr) (value.Value, error) {
	p.resolveExpr(expr.Value)
	p.resolveCatches(expr.Catches)

	return value.NilValue, nil
}

func (p *Resolver) resolveCatches(catches []ast.CatchClause) {
	caught := map[*value.Shape]bool{}

	for i := range catches {
		c := &catches[i]

		if len(c.Types) == 0 && i < len(catches)-1 {
			p.error(c.Keyword, "Catch-all must be the last catch clause.")
		}

		c.Shapes = p.resolveTypes(c.Types)

		for j, shape := range c.Shapes {
			if shape != nil && caught[shape] {
				t := c.Types[j]
				p.error(t.Name, "'"+qualified(t.Module, t.Name)+"' is already caught.")
			}

			caught[shape] = true
		}

		p.resolveCatch(c)
	}
}

func (p *Resolver) resolveCatch(c *ast.CatchClause) {
	scoped := c.Param != nil || declaresLocals(c.Body)
	if scoped {
		p.beginScope()
		defer p.endScope()
	}

	if c.Param != nil {
		c.ParamLocal = p.declare(ast.Binding{Name: *c.Param})
		p.fix(*c.Param)
		p.define(*c.Param)
	}

	p.resolveStmts(c.Body)

	p.resolveExpr(c.Value)
}

func (p *Resolver) VisitAssignExpr(expr *ast.AssignExpr) (value.Value, error) {
	p.resolveExpr(expr.Value)
	var ok bool
	if expr.Local, ok = p.use(expr.Name, "assign to"); ok {
		p.requireMutable(expr.Name, "assign to")
	}

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

	if prop, ok := callExpr.Callee.(*ast.IndexExpr); ok && isMethodCall(prop) {
		p.checkMutation(callExpr, prop)
	}

	p.inout(callExpr)

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

func (p *Resolver) VisitConditionalExpr(expr *ast.ConditionalExpr) (value.Value, error) {
	p.resolveExpr(expr.Cond)
	p.resolveExpr(expr.Then)
	p.resolveExpr(expr.Else)

	return value.NilValue, nil
}

func (p *Resolver) VisitThrowExpr(expr *ast.ThrowExpr) (value.Value, error) {
	p.resolveExpr(expr.Value)

	return value.NilValue, nil
}

func (p *Resolver) VisitJumpExpr(expr *ast.JumpExpr) (value.Value, error) {
	p.resolveStmt(expr.Stmt)

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

func (p *Resolver) VisitInterpolationExpr(expr *ast.InterpolationExpr) (value.Value, error) {
	for _, part := range expr.Parts {
		p.resolveExpr(part)
	}

	return value.NilValue, nil
}

func (p *Resolver) VisitOptionalExpr(expr *ast.OptionalExpr) (value.Value, error) {
	p.resolveExpr(expr.Expr)

	return value.NilValue, nil
}

func (p *Resolver) VisitSpreadExpr(expr *ast.SpreadExpr) (value.Value, error) {
	p.resolveExpr(expr.Expr)

	return value.NilValue, nil
}

func (p *Resolver) VisitComputedExpr(expr *ast.ComputedExpr) (value.Value, error) {
	p.resolveExpr(expr.Key)
	p.resolveExpr(expr.Value)

	return value.NilValue, nil
}

func (p *Resolver) VisitIndexExpr(expr *ast.IndexExpr) (value.Value, error) {
	switch p.qualifier(expr.Array) {
	case moduleQualifier:
		expr.Export = p.moduleMember(expr)
		return value.NilValue, nil
	case namespaceQualifier:
		p.namespaceMember(expr)
		return value.NilValue, nil
	}

	p.resolveMember(expr)
	p.resolveExpr(expr.Array)
	p.resolveExpr(expr.Index)
	p.resolveVariant(expr)

	return value.NilValue, nil
}

// resolveMember binds a literal member of an unshadowed namespace, as in
// string.fromCharCode, to the built-in it names, so reading it costs
// nothing at run time.
func (p *Resolver) resolveMember(expr *ast.IndexExpr) {
	ns, ok := expr.Array.(*ast.VarExpr)
	key, isLiteral := expr.Index.(*ast.LiteralExpr)

	if !ok || !isLiteral || !key.Value.IsString() || !p.isBuiltin(ns.Name.Lexeme) {
		return
	}

	members := p.namespace(ns.Name.Lexeme)
	if members == nil {
		return
	}

	if v, found := members[key.Value.Str()]; found {
		expr.Member = &v
		return
	}

	msg := ns.Name.Lexeme + " has no member '" + key.Value.Str() + "'."
	if owner := stdlib.Owner(key.Value.Str()); owner != "" {
		msg += " Did you mean " + owner + "." + key.Value.Str() + "?"
	}
	p.error(expr.Token, msg)
}

// namespaceMember resolves ns.name for a namespace that is no value, whose
// members must be named.
func (p *Resolver) namespaceMember(expr *ast.IndexExpr) {
	if key, ok := expr.Index.(*ast.LiteralExpr); !ok || !key.Value.IsString() {
		ns := expr.Array.(*ast.VarExpr).Name.Lexeme
		p.error(expr.Token, "Namespace members must be named, as in "+ns+".name.")
		return
	}

	p.resolveMember(expr)
}

// fileNamespace names the namespace of each file's own: module.dir and
// module.file, the directory and path of the file.
const fileNamespace = "module"

func newFileNamespace(root, path string) stdlib.Namespace {
	file := filepath.Join(root, filepath.FromSlash(path))

	return stdlib.Namespace{
		"dir":  value.NewString(filepath.Dir(file)),
		"file": value.NewString(file),
	}
}

// namespace is the built-in namespace called name, or nil.
func (p *Resolver) namespace(name string) stdlib.Namespace {
	if name == fileNamespace {
		return p.mod.file
	}

	return stdlib.Namespaces[name]
}

func (p *Resolver) VisitSetIndexExpr(expr *ast.SetIndexExpr) (value.Value, error) {
	if q := p.qualifier(expr.Array); q != "" {
		p.memberAssign(q, expr.Array, expr.Token)
		p.resolveExpr(expr.Value)

		return value.NilValue, nil
	}

	p.resolveExpr(expr.Array)
	p.resolveExpr(expr.Index)
	p.resolveExpr(expr.Value)
	p.requireMutableRoot(expr.Array)
	return value.NilValue, nil
}

func (p *Resolver) VisitRangeExpr(expr *ast.RangeExpr) (value.Value, error) {
	if expr.End == nil {
		p.error(expr.Operator, "A range needs an end; only a slice, as in xs[1..], may leave it open.")
		return value.NilValue, nil
	}

	p.resolveExpr(expr.Start)
	p.resolveExpr(expr.End)

	p.resolveExpr(expr.Step)

	return value.NilValue, nil
}

func (p *Resolver) VisitSliceExpr(expr *ast.SliceExpr) (value.Value, error) {
	p.resolveExpr(expr.Target)
	p.resolveExpr(expr.Range.Start)
	p.resolveExpr(expr.Range.End)

	return value.NilValue, nil
}

func (p *Resolver) VisitUpdateExpr(expr *ast.UpdateExpr) (value.Value, error) {
	p.resolveExpr(expr.Target)

	switch t := expr.Target.(type) {
	case *ast.VarExpr:
		p.requireMutable(t.Name, "assign to")
	case *ast.IndexExpr:
		if q := p.qualifier(t.Array); q != "" {
			p.memberAssign(q, t.Array, t.Token)
		} else {
			p.requireMutableRoot(t.Array)
		}
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
	switch {
	case p.constNamed(name.Lexeme) != nil:
		p.error(name, "Cannot "+verb+" constant '"+name.Lexeme+"'.")
	case !p.isMutable(name.Lexeme):
		p.immutable(name, "Cannot "+verb+" immutable variable '"+name.Lexeme+"'.")
	}
}

// requireMutableRoot checks the variable an element or property chain starts from.
func (p *Resolver) requireMutableRoot(expr ast.Expr) {
	if name := p.immutableRoot(expr); name != "" {
		p.immutable(rootVar(expr).Name, "Cannot mutate immutable variable '"+name+"'.")
	}
}

// immutableRoot names the variable an element or property chain starts
// from if it is bound immutably or is a built-in.
func (p *Resolver) immutableRoot(expr ast.Expr) string {
	root := rootVar(expr)

	switch {
	case root == nil:
	case root.Module != nil:
		if root.Local != nil && !p.mod.imports[root.Module.Lexeme].globalMutable[root.Name.Lexeme] {
			return qualified(root.Module, root.Name)
		}
	case !p.isMutable(root.Name.Lexeme) || p.isBuiltin(root.Name.Lexeme):
		return root.Name.Lexeme
	}

	return ""
}

// rootVar returns the variable an element or property chain starts from,
// a module member included, if any.
func rootVar(expr ast.Expr) *ast.VarExpr {
	for {
		switch e := expr.(type) {
		case *ast.IndexExpr:
			if e.Export != nil {
				return e.Export
			}

			expr = e.Array
		case *ast.OptionalExpr:
			expr = e.Expr
		case *ast.GroupingExpr:
			expr = e.Expr
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

	mutable, declared := p.mod.globalMutable[name]

	return !declared || mutable
}

func (p *Resolver) isBuiltin(name string) bool {
	_, declared := p.mod.globalMutable[name]

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

			p.uses[l.local]++

			return l.local
		}
	}

	l := p.mod.topLevel[name.Lexeme]
	if l == nil {
		l = p.builtin(name.Lexeme)
	}

	if l != nil && p.fnDepth > 0 {
		l.Captured = true
	}

	p.uses[l]++

	return l
}

func (p *Resolver) builtin(name string) *ast.Local {
	v, ok := builtinValues()[name]
	if !ok {
		return nil
	}

	l, ok := p.mod.builtins[name]
	if !ok {
		l = &ast.Local{TopLevel: true}
		p.mod.builtins[name] = l

		if v.Kind() == value.Type {
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

// resolveExpr resolves expr, which may be absent.
func (p *Resolver) resolveExpr(expr ast.Expr) {
	if expr != nil {
		_, _ = expr.Accept(p)
	}
}

func (p *Resolver) beginScope() {
	p.scopes = append(p.scopes, &Scope{vars: map[string]*localVar{}, fnDepth: p.fnDepth})
}

func (p *Resolver) endScope() {
	p.scopes = p.scopes[:len(p.scopes)-1]
}

func (p *Resolver) resolveFn(fn *ast.FunctionStmt) {
	enclosingIsFunction, enclosingLoopDepth, enclosingInDefer := p.isFunction, p.loopDepth, p.inDefer
	p.isFunction, p.loopDepth, p.inDefer = true, 0, false
	p.fnDepth++
	defer func() {
		p.isFunction, p.loopDepth, p.inDefer = enclosingIsFunction, enclosingLoopDepth, enclosingInDefer
		p.fnDepth--
	}()

	p.beginScope()

	for i, param := range fn.Parameters {
		p.resolveExpr(fn.Defaults[i])

		fn.Parameters[i].Local = p.declare(param)
		p.define(param.Name)
	}

	p.resolveStmts(fn.Body)
	p.endScope()
}

// declare adds name to the innermost scope and returns its declaration.
func (p *Resolver) declare(b ast.Binding) *ast.Local {
	name := b.Name

	if b.Discards() {
		return &ast.Local{}
	}

	if prev, exists := p.innermost()[name.Lexeme]; exists {
		p.errors = append(p.errors, errors.NewRedeclaredError(name, prev.name))
	}

	if p.isScopesEmpty() {
		p.mod.globals[name.Lexeme] = &localVar{name: name}
		return p.mod.topLevel[name.Lexeme]
	}

	l := &ast.Local{}
	p.peekScope().vars[name.Lexeme] = &localVar{mutable: b.Mutable, local: l, name: name}

	return l
}

// fix marks the variable name declares as one no mut makes mutable.
func (p *Resolver) fix(name ast.Token) {
	if l := p.innermost()[name.Lexeme]; l != nil {
		l.fixed = true
	}
}

// define marks the variable name declares defined; a discard declares none.
func (p *Resolver) define(name ast.Token) {
	if l := p.innermost()[name.Lexeme]; l != nil {
		l.defined = true
	}
}

// innermost returns the declarations of the current scope, the top level included.
func (p *Resolver) innermost() map[string]*localVar {
	if p.isScopesEmpty() {
		return p.mod.globals
	}

	return p.peekScope().vars
}

func (p *Resolver) isScopesEmpty() bool {
	return len(p.scopes) == 0
}

func (p *Resolver) peekScope() *Scope {
	return p.scopes[len(p.scopes)-1]
}
