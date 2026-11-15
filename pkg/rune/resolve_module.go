package rune

import (
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/stdlib"
	"rune/pkg/value"
	"slices"
	"strings"
)

// moduleScope is the top level of one file.
type moduleScope struct {
	// globals tracks top-level declarations like a scope's vars; their
	// mutability is in globalMutable, so localVar.mutable is unused. A
	// module the file imports is one of them, immutable, so it shadows
	// a built-in namespace of its name.
	globals map[string]*localVar
	// globalMutable holds every top-level declaration up front, since a
	// function may assign to a global declared after it.
	globalMutable map[string]bool
	// topLevel holds the declarations of every top-level name up front,
	// for the same reason.
	topLevel map[string]*ast.Local
	// builtins holds a declaration for each built-in the file names
	// without declaring it.
	builtins map[string]*ast.Local
	// imports holds the modules the file imports, by the name it binds.
	imports map[string]*moduleScope
	exports map[string]*ast.Local
	// file is the file's namespace, module.
	file stdlib.Namespace
}

func newModuleScope(file stdlib.Namespace) *moduleScope {
	return &moduleScope{
		file:          file,
		globals:       map[string]*localVar{},
		globalMutable: map[string]bool{},
		topLevel:      map[string]*ast.Local{},
		builtins:      map[string]*ast.Local{},
		imports:       map[string]*moduleScope{},
		exports:       map[string]*ast.Local{},
	}
}

// ResolveModules resolves a program's files, each after the ones it
// imports, and returns all errors found.
func (p *Resolver) ResolveModules(files []*fileAST) []error {
	p.modules = make([]*moduleScope, len(files))
	p.methodKinds = methodKinds(files)
	p.methodInout = methodInout(files)
	for _, f := range files {
		p.anyInout |= f.inout
	}

	for _, f := range files {
		p.mod = newModuleScope(newFileNamespace(p.root, f.path))
		p.modules[f.file] = p.mod

		for _, imp := range f.imports {
			p.bindModule(imp.as, p.modules[imp.file])
		}

		p.resolveTopLevel(f.stmts)

		for _, stmt := range f.exports {
			for _, name := range declaredNames(stmt) {
				if l := p.mod.topLevel[name]; l != nil {
					l.Exported = true
					p.mod.exports[name] = l
				}
			}
		}
	}

	p.ownVars()

	return p.errors
}

func declaredNames(stmt ast.Stmt) []string {
	if s, ok := stmt.(*ast.StructStmt); ok {
		return []string{s.Name.Lexeme}
	}

	var names []string
	for _, b := range ast.Declared(stmt) {
		names = append(names, b.Name.Lexeme)
	}

	return names
}

// bindModule declares name as the module m in the current file.
func (p *Resolver) bindModule(name ast.Token, m *moduleScope) {
	if prev, exists := p.mod.globals[name.Lexeme]; exists {
		p.errors = append(p.errors, errors.NewRedeclaredError(name, prev.name))
		return
	}

	p.mod.globals[name.Lexeme] = &localVar{defined: true, fixed: true, name: name}
	p.mod.globalMutable[name.Lexeme] = false
	p.mod.imports[name.Lexeme] = m
}

// namedLocal is a built-in's declaration in one file.
type namedLocal struct {
	name  string
	local *ast.Local
	value value.Value
}

// builtinLocals lists the declarations made for the built-ins each file
// names, file by file, sorted by name.
func (p *Resolver) builtinLocals() []namedLocal {
	var out []namedLocal

	for _, m := range p.modules {
		names := make([]string, 0, len(m.builtins))
		for name := range m.builtins {
			names = append(names, name)
		}

		slices.Sort(names)

		for _, name := range names {
			v := builtinValues()[name]
			out = append(out, namedLocal{name, m.builtins[name], v})
		}
	}

	return out
}

// isModule reports whether e is a bare name of a module imported here.
func (p *Resolver) isModule(e ast.Expr) bool {
	v, ok := e.(*ast.VarExpr)

	return ok && v.Module == nil && p.lookupScopes(v.Name.Lexeme) == nil && p.mod.imports[v.Name.Lexeme] != nil
}

// The qualifiers, names that are no values: only their members are.
const (
	moduleQualifier    = "Module"
	namespaceQualifier = "Namespace"
)

// qualifier is what e names when it is a bare name of a module imported
// here or of a built-in namespace that is no function, or "".
func (p *Resolver) qualifier(e ast.Expr) string {
	v, ok := e.(*ast.VarExpr)

	switch {
	case !ok || v.Module != nil:
		return ""
	case p.isModule(e):
		return moduleQualifier
	case !p.isBuiltin(v.Name.Lexeme) || p.namespace(v.Name.Lexeme) == nil:
		return ""
	}

	if _, callable := builtinValues()[v.Name.Lexeme]; callable {
		return ""
	}

	return namespaceQualifier
}

// member returns the variable module.name names, reporting it if module
// is not a module or does not export name.
func (p *Resolver) member(module, name ast.Token) *ast.Local {
	m := p.mod.imports[module.Lexeme]
	if m == nil || p.lookupScopes(module.Lexeme) != nil {
		p.error(module, "'"+module.Lexeme+"' is not a module.")
		return nil
	}

	l := m.exports[name.Lexeme]
	if l == nil {
		p.error(name, "Module '"+module.Lexeme+"' has no export '"+name.Lexeme+"'.")
		return nil
	}

	if p.fnDepth > 0 {
		l.Captured = true
	}

	return l
}

// qualified returns the variable m.name names, or the constant of the
// case Enum.Case or m.Enum.Case.
func (p *Resolver) qualified(e *ast.VarExpr) *ast.Local {
	if e.Case != nil {
		if l := p.member(*e.Module, e.Name); l != nil {
			return p.caseNamed(p.structs[l], e.Name, *e.Case)
		}

		return nil
	}

	if enum := p.localEnum(*e.Module); enum != nil {
		return p.caseNamed(enum, *e.Module, e.Name)
	}

	return p.member(*e.Module, e.Name)
}

// moduleMember returns the member m.name or m["name"] of the module m
// names; any other key is an error.
func (p *Resolver) moduleMember(expr *ast.IndexExpr) *ast.VarExpr {
	module := expr.Array.(*ast.VarExpr).Name
	name := expr.Token

	key, isLiteral := expr.Index.(*ast.LiteralExpr)
	if !isLiteral || !key.Value.IsString() {
		p.error(name, "Module members must be named, as in "+module.Lexeme+".name.")
		return nil
	}

	name.Lexeme = key.Value.Str()

	return &ast.VarExpr{Module: &module, Name: name, Local: p.member(module, name)}
}

// memberAssign reports an assignment to a member of the qualifier q that
// e names.
func (p *Resolver) memberAssign(q string, e ast.Expr, tok ast.Token) {
	p.error(tok, "Cannot assign to a member of "+strings.ToLower(q)+" '"+e.(*ast.VarExpr).Name.Lexeme+"'.")
}

// structType returns the struct type t names, reporting it if t names
// something else; a missing module member is reported already.
func (p *Resolver) structType(t *ast.VarExpr) *value.Shape {
	shape := p.structs[t.Local]
	if shape == nil && (t.Module == nil || t.Local != nil) {
		p.error(t.Name, "'"+qualified(t.Module, t.Name)+"' is not a struct.")
	}

	return shape
}

// qualified spells a name as written, with its module if any.
func qualified(module *ast.Token, name ast.Token) string {
	if module == nil {
		return name.Lexeme
	}

	return module.Lexeme + "." + name.Lexeme
}
