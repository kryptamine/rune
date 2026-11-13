package rune

import (
	"rune/pkg/ast"
	"rune/pkg/helpers"
	"rune/pkg/value"
)

type constState int

const (
	constPending constState = iota
	constActive
	constFailed
	constDone
)

// resolveConsts gives every constant of the file its value, so it can be
// used before its declaration.
func (p *Resolver) resolveConsts(stmts []ast.Stmt) {
	p.consts = map[*ast.VarStmt]constState{}
	p.constDecls = map[*ast.Local]*ast.VarStmt{}

	for _, stmt := range stmts {
		if s, ok := stmt.(*ast.VarStmt); ok && s.Const {
			p.consts[s] = constPending

			if l := p.mod.topLevel[s.Name.Lexeme]; l != nil && p.constDecls[l] == nil {
				p.constDecls[l] = s
			}
		}
	}

	for _, stmt := range stmts {
		if s, ok := stmt.(*ast.VarStmt); ok && s.Const {
			p.evalConst(s)
		}
	}
}

// evalConst computes the value of s, after those of the constants it
// names; it reports whether s has one.
func (p *Resolver) evalConst(s *ast.VarStmt) bool {
	if state := p.consts[s]; state != constPending {
		return state == constDone
	}

	p.consts[s] = constActive
	p.resolveExpr(s.Initializer)

	state := constFailed

	if p.evalOperands(s.Initializer) {
		v, ok := p.constantOf(s.Initializer)
		l := p.mod.topLevel[s.Name.Lexeme]

		switch {
		case !ok:
			p.error(s.Name, "Constant '"+s.Name.Lexeme+"' must be a constant: a literal, a const or operators on them, an enum variant, or an array, object, map() or set() of them.")
		case l != nil && p.constDecls[l] == s:
			l.Const, l.IsConst = v, true
			state = constDone
		}
	}

	p.consts[s] = state

	return state == constDone
}

// evalOperands computes the constants e is made of; it is false if one
// has no value, which is reported already.
func (p *Resolver) evalOperands(e ast.Expr) bool {
	switch n := unwrap(e).(type) {
	case *ast.VarExpr:
		s := p.constDecls[n.Local]

		switch {
		case s == nil || n.Module != nil:
		case p.consts[s] == constActive:
			p.error(n.Name, "Constant '"+n.Name.Lexeme+"' depends on itself.")
			return false
		default:
			return p.evalConst(s)
		}
	case *ast.UnaryExpr:
		return p.evalOperands(n.Right)
	case *ast.BinaryExpr:
		l, r := p.evalOperands(n.Left), p.evalOperands(n.Right)
		return l && r
	case *ast.ArrayExpr:
		ok := true
		for _, item := range n.Items {
			ok = p.evalOperands(item) && ok
		}

		return ok
	case *ast.ObjectExpr:
		ok := true
		for _, pair := range n.Pairs {
			ok = p.evalOperands(pair.Value) && ok
		}

		return ok
	}

	return true
}

// constPattern is the test of the value of the constant l, named by tok.
func (p *Resolver) constPattern(tok ast.Token, l *ast.Local) ast.Pattern {
	if value.Mutable(l.Const) {
		p.error(tok, "Constant '"+tok.Lexeme+"' is a mutable "+helpers.TypeName(l.Const)+", which a pattern cannot test.")
		return &ast.LiteralPattern{Token: tok}
	}

	return &ast.LiteralPattern{Token: tok, Value: l.Const}
}

// constNamed returns the constant name denotes here, if it is one.
func (p *Resolver) constNamed(name string) *ast.Local {
	if p.lookupScopes(name) != nil {
		return nil
	}

	if l := p.mod.topLevel[name]; l != nil && l.IsConst {
		return l
	}

	return nil
}
