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
	// mod is the top level of the file being resolved; modules holds
	// every file's, by file index.
	mod     *moduleScope
	modules []*moduleScope
	// root is where the program's files are on disk, for module.dir.
	root string
	// structs holds every struct type up front, by its name's declaration.
	structs map[*ast.Local]*value.Shape
	// fieldNames holds each field name's one string, shared by every struct
	// type that has the field.
	fieldNames map[string]string
	// main is the program's entry point, if it declares one.
	main *ast.FunctionStmt
	// kinds holds the methods of the built-in types; nmethods counts all.
	kinds    [value.NumKinds][]value.Method
	nmethods int
	// impls holds the impl blocks at the top level of a file.
	impls map[*ast.ImplStmt]bool
	// mutators holds the method names that always change their receiver.
	mutators map[string]bool
}

func NewResolver() *Resolver {
	mod := newModuleScope(newFileNamespace("", ""))

	p := &Resolver{
		mod:        mod,
		modules:    []*moduleScope{mod},
		structs:    map[*ast.Local]*value.Shape{},
		fieldNames: stdlib.FieldNames(),
		impls:      map[*ast.ImplStmt]bool{},
		nmethods:   len(stdlib.Methods),
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
		}

		if s, ok := stmt.(*ast.StructStmt); ok {
			p.declareMain(s.Name, nil)
			p.mod.globalMutable[s.Name.Lexeme] = false
			p.mod.topLevel[s.Name.Lexeme] = &ast.Local{TopLevel: true}
		}
	}

	for _, stmt := range stmts {
		if s, ok := stmt.(*ast.StructStmt); ok {
			s.Shape = p.shape(s)
			p.structs[p.mod.topLevel[s.Name.Lexeme]] = s.Shape
		}
	}

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

// ResolveExpr resolves a lone expression.
func (p *Resolver) ResolveExpr(expr ast.Expr) []error {
	p.resolveExpr(expr)

	return p.errors
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
		if v.Default != nil {
			p.resolveExpr(v.Default)
		}

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

	return p.resolveLocal(name), true
}

func (p *Resolver) VisitVarExpr(expr *ast.VarExpr) (value.Value, error) {
	if expr.Module != nil {
		expr.Local = p.member(*expr.Module, expr.Name)
		return value.NilValue, nil
	}

	if p.isModule(expr) {
		p.error(expr.Name, "Module '"+expr.Name.Lexeme+"' is not a value; use its members, as in "+expr.Name.Lexeme+".name.")
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

// VisitStructStmt checks stmt is at the top level, whose structs
// resolveTopLevel has given their shapes.
func (p *Resolver) VisitStructStmt(stmt *ast.StructStmt) error {
	if stmt.Shape == nil {
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
		name := p.intern(f.Name.Lexeme)

		var def value.Value
		if f.Default != nil {
			var ok bool
			if def, ok = p.constantOf(f.Default); !ok {
				p.error(f.DefaultAt, "Field default must be a constant: a literal, or an array, object, map() or set() of them.")
			}
		}
		fields[i] = value.Field{Name: name, Default: def, Required: f.Default == nil}
	}

	return value.NewShape(s.Name.Lexeme, fields)
}

func (p *Resolver) constantOf(e ast.Expr) (value.Value, bool) {
	if v, ok := literalOf(e); ok {
		return v, true
	}

	switch n := unwrap(e).(type) {
	case *ast.ArrayExpr:
		items := make([]value.Value, len(n.Items))
		for i, item := range n.Items {
			v, ok := p.constantOf(item)
			if !ok {
				return value.NilValue, false
			}
			items[i] = v
		}
		return value.NewArray(items), true
	case *ast.ObjectExpr:
		fields := value.NewObj(len(n.Pairs))
		for _, pair := range n.Pairs {
			v, ok := p.constantOf(pair.Value)
			if !ok {
				return value.NilValue, false
			}
			fields.Set(pair.Key, v)
		}
		return value.NewObject(fields), true
	case *ast.CallExpr:
		callee, ok := n.Callee.(*ast.VarExpr)
		if !ok || len(n.Args) > 0 || callee.Module != nil || !p.isBuiltin(callee.Name.Lexeme) {
			return value.NilValue, false
		}
		switch callee.Name.Lexeme {
		case "map":
			return value.NewMap(value.NewHashMap(0)), true
		case "set":
			return value.NewSet(value.NewHashMap(0)), true
		}
	}

	return value.NilValue, false
}

// intern returns the one string of a field or method name.
func (p *Resolver) intern(name string) string {
	if s, ok := p.fieldNames[name]; ok {
		return s
	}

	p.fieldNames[name] = name

	return name
}

func (p *Resolver) VisitStructExpr(expr *ast.StructExpr) (value.Value, error) {
	for _, pair := range expr.Pairs {
		p.resolveExpr(pair.Value)
	}

	p.resolveExpr(expr.Type)

	shape := p.structType(expr.Type)
	if shape == nil {
		return value.NilValue, nil
	}

	expr.Shape = shape
	given := make([]bool, len(shape.Fields))
	spread := false

	for i, pair := range expr.Pairs {
		if ast.IsSpread(pair.Value) {
			spread = true
			continue
		}

		slot := shape.Index(pair.Key.Lexeme)
		if slot < 0 {
			p.error(pair.Key, errors.NoField(shape.Name, pair.Key.Lexeme))
			continue
		}

		expr.Pairs[i].Slot = slot
		given[slot] = true
	}

	for i, f := range shape.Fields {
		if _, optional := shape.Default(i); !given[i] && !optional && !spread {
			p.error(expr.Type.Name, errors.MissingField(shape.Name, f))
		}
	}

	return value.NilValue, nil
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

	caught := map[*value.Shape]bool{}

	for i := range tryStmt.Catches {
		c := &tryStmt.Catches[i]

		if len(c.Types) == 0 && i < len(tryStmt.Catches)-1 {
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
		callExpr.Writable = p.writable(prop.Array)

		if name := p.immutableRoot(prop.Array); name != "" {
			callExpr.Frozen = true

			if p.mutators[prop.Token.Lexeme] {
				p.error(rootVar(prop.Array).Name, "Cannot mutate immutable variable '"+name+"'.")
			}
		}
	}

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

func (p *Resolver) VisitIndexExpr(expr *ast.IndexExpr) (value.Value, error) {
	if p.isModule(expr.Array) {
		expr.Export = p.moduleMember(expr)
		return value.NilValue, nil
	}

	p.resolveMember(expr)
	p.resolveExpr(expr.Array)
	p.resolveExpr(expr.Index)

	return value.NilValue, nil
}

// resolveMember binds a literal member of an unshadowed namespace to the
// built-in it names, so reading it costs nothing at run time.
func (p *Resolver) resolveMember(expr *ast.IndexExpr) {
	ns, ok := expr.Array.(*ast.VarExpr)
	key, isLiteral := expr.Index.(*ast.LiteralExpr)

	if !ok || !isLiteral || !key.Value.IsString() || !p.isBuiltin(ns.Name.Lexeme) {
		return
	}

	v, found, isNamespace := p.namespaceMember(ns.Name.Lexeme, key.Value.Str())

	switch {
	case found:
		expr.Member = &v
	case isNamespace:
		msg := ns.Name.Lexeme + " has no member '" + key.Value.Str() + "'."
		if owner := stdlib.Owner(key.Value.Str()); owner != "" {
			msg += " Did you mean " + owner + "." + key.Value.Str() + "?"
		}
		p.error(expr.Token, msg)
	}
}

// fileNamespace names the namespace of each file's own: module.dir and
// module.file, the directory and path of the file.
const fileNamespace = "module"

func newFileNamespace(root, path string) value.Value {
	file := filepath.Join(root, filepath.FromSlash(path))

	return stdlib.NewNamespace([]stdlib.Builtin{
		{Name: "dir", Value: value.NewString(filepath.Dir(file))},
		{Name: "file", Value: value.NewString(file)},
	})
}

func (p *Resolver) namespaceMember(ns, key string) (v value.Value, found, isNamespace bool) {
	if ns == fileNamespace {
		v, found = stdlib.NamespaceMember(p.mod.file, key)
		return v, found, true
	}

	return stdlib.Member(ns, key)
}

func (p *Resolver) VisitSetIndexExpr(expr *ast.SetIndexExpr) (value.Value, error) {
	if p.isModule(expr.Array) {
		p.moduleAssign(expr.Array, expr.Token)
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
	p.resolveExpr(expr.Start)
	p.resolveExpr(expr.End)

	if expr.Step != nil {
		p.resolveExpr(expr.Step)
	}

	return value.NilValue, nil
}

func (p *Resolver) VisitUpdateExpr(expr *ast.UpdateExpr) (value.Value, error) {
	p.resolveExpr(expr.Target)

	switch t := expr.Target.(type) {
	case *ast.VarExpr:
		p.requireMutable(t.Name, "assign to")
	case *ast.IndexExpr:
		if t.Export != nil {
			p.moduleAssign(t.Array, t.Token)
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
	if !p.isMutable(name.Lexeme) {
		p.error(name, "Cannot "+verb+" immutable variable '"+name.Lexeme+"'.")
	}
}

// requireMutableRoot checks the variable an element or property chain starts from.
func (p *Resolver) requireMutableRoot(expr ast.Expr) {
	if name := p.immutableRoot(expr); name != "" {
		p.error(rootVar(expr).Name, "Cannot mutate immutable variable '"+name+"'.")
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

	return l
}

func (p *Resolver) builtin(name string) *ast.Local {
	v, ok := p.mod.builtinValue(name)
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
		if fn.Defaults[i] != nil {
			p.resolveExpr(fn.Defaults[i])
		}

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
