package rune

import (
	"rune/pkg/ast"
	"rune/pkg/errors"
	"slices"
)

// inout marks the arguments of call that a mut parameter writes back to.
// A function the call names takes a mutable place for each of its mut
// parameters, and a built-in none. Any other function is known only when
// called, so every mutable place is marked where some function of the
// program has a mut parameter, and the call writes back those its
// function has.
func (p *Resolver) inout(call *ast.CallExpr) {
	if slices.ContainsFunc(call.Args, ast.IsSpread) {
		return
	}

	mask := p.anyInout

	switch callee := call.Callee.(type) {
	case *ast.VarExpr:
		if callee.Module == nil && p.isBuiltin(callee.Name.Lexeme) {
			return
		}

		if fn := p.fns[callee.Local]; fn != nil && callee.Module == nil {
			mask = fn.Inout()
			for i, arg := range call.Args {
				if mask&(1<<i) != 0 && !p.mutablePlace(arg) {
					p.notPlace(call, arg, fn.Parameters[i].Name.Lexeme)
				}
			}
		}
	case *ast.IndexExpr:
		switch {
		case callee.Member != nil:
			return
		case isMethodCall(callee):
			mask = p.methodInout[callee.Token.Lexeme]
		}
	}

	for i, arg := range call.Args {
		if mask&(1<<i) != 0 && p.mutablePlace(arg) {
			call.Inout |= 1 << i
		}
	}
}

// mutablePlace reports whether e is a mutable variable, or an element or
// property on a path from one, which a mut parameter can write back to.
func (p *Resolver) mutablePlace(e ast.Expr) bool {
	for n := e; ; {
		switch x := n.(type) {
		case *ast.GroupingExpr:
			n = x.Expr
			continue
		case *ast.IndexExpr:
			if x.Optional || x.Member != nil {
				return false
			}

			if x.Export == nil {
				n = x.Array
				continue
			}
		case *ast.VarExpr:
			if x.Local == nil || x.Local.IsConst {
				return false
			}
		default:
			return false
		}

		return p.immutableRoot(e) == ""
	}
}

// notPlace reports arg of call, which mut parameter param cannot write
// back to.
func (p *Resolver) notPlace(call *ast.CallExpr, arg ast.Expr, param string) {
	if root := rootVar(arg); root != nil && p.immutableRoot(arg) != "" {
		p.immutable(root.Name, "Cannot pass immutable variable '"+p.immutableRoot(arg)+"' to mut parameter '"+param+"'.")
		return
	}

	err := errors.NewSyntaxError(call.Token, "Mut parameter '"+param+"' takes a mutable variable, element or property.")
	p.errors = append(p.errors, errors.WithHint(err, "put the value in a 'let mut' variable and pass that"))
}

// methodInout gives, for each method name of the program, a bit for each
// argument a mut parameter of a method of that name takes.
func methodInout(files []*fileAST) map[string]uint32 {
	masks := map[string]uint32{}

	for _, f := range files {
		for _, stmt := range f.stmts {
			if impl, ok := stmt.(*ast.ImplStmt); ok {
				for _, m := range impl.Methods {
					masks[m.Fn.Name.Lexeme] |= argInout(m.Fn)
				}
			}
		}
	}

	return masks
}

// requirePlace reports the collection of a for mut loop unless it is a
// mutable place, which each iteration writes its variable back into.
func (p *Resolver) requirePlace(stmt *ast.ForInStmt) {
	if p.mutablePlace(stmt.Iterable) {
		return
	}

	if root := rootVar(stmt.Iterable); root != nil && p.immutableRoot(stmt.Iterable) != "" {
		p.immutable(root.Name, "Cannot write 'for mut' variable '"+stmt.Value.Name.Lexeme+"' back into immutable variable '"+p.immutableRoot(stmt.Iterable)+"'.")
		return
	}

	err := errors.NewSyntaxError(stmt.Value.Name, "A 'for mut' loop walks a mutable variable, element or property, which it writes back into.")
	p.errors = append(p.errors, errors.WithHint(err, "put the collection in a 'let mut' variable and loop over that"))
}

// argInout has a bit for each argument a mut parameter of fn takes, self
// aside.
func argInout(fn *ast.FunctionStmt) uint32 {
	if self, _ := selfParam(fn); self {
		return fn.Inout() >> 1
	}

	return fn.Inout()
}
