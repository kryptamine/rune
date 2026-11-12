package rune

import (
	"rune/pkg/ast"
	"rune/pkg/helpers"
	"rune/pkg/stdlib"
	"rune/pkg/value"
)

func (p *Resolver) VisitImplStmt(stmt *ast.ImplStmt) error {
	if !p.impls[stmt] {
		p.error(stmt.Type, "Impl blocks must be at the top level.")
		return nil
	}

	for _, m := range stmt.Methods {
		p.resolveFn(m.Fn)
	}

	return nil
}

var builtinKinds = map[string]value.Kind{
	"string": value.String, "number": value.Number, "boolean": value.Bool,
	"array": value.Array, "object": value.Object, "function": value.Function,
	"Map": value.Map, "Set": value.Set, "bytes": value.Bytes,
}

// declareImpl adds the methods of stmt to its type, a struct of this file
// or a built-in type, before any code is resolved.
func (p *Resolver) declareImpl(stmt *ast.ImplStmt) {
	name := stmt.Type.Lexeme

	var (
		shape   *value.Shape
		methods *[]value.Method
	)

	kind, isKind := builtinKinds[name]

	switch l := p.mod.topLevel[name]; {
	case l != nil && p.structs[l] != nil:
		shape = p.structs[l]
		methods = &shape.Methods
	case l == nil && isKind:
		methods = &p.kinds[kind]
	case l == nil && builtinValues()[name].Kind() == value.Type:
		p.error(stmt.Type, "Cannot add methods to built-in struct '"+name+"'.")
		return
	default:
		p.error(stmt.Type, "'"+name+"' is not a struct of this file or a built-in type.")
		return
	}

	for i := range stmt.Methods {
		fn := stmt.Methods[i].Fn
		m := value.Method{Name: p.intern(fn.Name.Lexeme), ID: p.nmethods}
		m.Self, m.Mut = selfParam(fn)

		for i, param := range fn.Parameters {
			if param.Name.Lexeme == "self" && i > 0 {
				p.error(param.Name, "'self' must be the first parameter.")
			}
		}

		switch {
		case m.Self && fn.Rest && len(fn.Parameters) == 1:
			p.error(fn.Parameters[0].Name, "'self' cannot be a rest parameter.")
		case m.Self && fn.Defaults[0] != nil:
			p.error(fn.Parameters[0].Name, "'self' cannot have a default.")
		case shape == nil && !m.Self:
			p.error(fn.Name, "Methods of "+name+" must take self.")
		case shape == nil && m.Mut:
			p.error(fn.Name, "Only methods of structs can take 'mut self'.")
		case shape != nil && shape.Index(m.Name) >= 0:
			p.error(fn.Name, name+" has a field '"+m.Name+"'.")
		}

		// A method of the program replaces a built-in one.
		switch prev := value.FindMethod(*methods, m.Name); {
		case prev == nil:
			*methods = append(*methods, m)
		case prev.ID < len(stdlib.Methods):
			*prev = m
		default:
			p.error(fn.Name, name+" already has a method '"+m.Name+"'.")
			*methods = append(*methods, m)
		}

		p.nmethods++
		stmt.Methods[i].Method = m
	}
}

// isMethodCall reports whether a call of callee calls a method of its
// receiver rather than a namespace or module member.
func isMethodCall(callee *ast.IndexExpr) bool {
	return callee.Property && callee.Member == nil && callee.Export == nil
}

// writable reports whether a mutating method may write a new self back to
// its receiver: a mutable variable, or an element or property of a value
// that a mutable variable holds, or none does.
func (p *Resolver) writable(recv ast.Expr) bool {
	switch r := recv.(type) {
	case *ast.VarExpr:
		return r.Module == nil && r.Local != nil && p.isMutable(r.Name.Lexeme)
	case *ast.IndexExpr:
		if r.Optional || r.Member != nil || r.Export != nil {
			return false
		}

		root := rootVar(r.Array)

		switch {
		case root == nil:
			return true
		case root.Module != nil:
			return root.Local != nil && p.mod.imports[root.Module.Lexeme].globalMutable[root.Name.Lexeme]
		default:
			return p.isMutable(root.Name.Lexeme)
		}
	}

	return false
}

// selfParam reports whether fn takes self first, and whether mut self.
func selfParam(fn *ast.FunctionStmt) (self, mut bool) {
	self = len(fn.Parameters) > 0 && fn.Parameters[0].Name.Lexeme == "self"

	return self, self && fn.Parameters[0].Mutable
}

// mutators returns the names of the in-place built-in methods and the
// mut self methods of the program, but those another method shares, so
// that a call of one through an immutable variable is a compile error.
func mutators(files []*fileAST) map[string]bool {
	mut, other := map[string]bool{}, map[string]bool{}

	for _, m := range stdlib.Methods {
		helpers.If(m.InPlace, mut, other)[m.Name] = true
	}

	for _, f := range files {
		for _, stmt := range f.stmts {
			if impl, ok := stmt.(*ast.ImplStmt); ok {
				for _, m := range impl.Methods {
					_, isMut := selfParam(m.Fn)
					helpers.If(isMut, mut, other)[m.Fn.Name.Lexeme] = true
				}
			}
		}
	}

	for name := range other {
		delete(mut, name)
	}

	return mut
}
