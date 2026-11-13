package rune

import (
	"rune/pkg/ast"
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
		v, ok := constValue(s.Initializer)
		l := p.mod.topLevel[s.Name.Lexeme]

		switch {
		case !ok:
			p.error(s.Name, "Constant '"+s.Name.Lexeme+"' must be a constant expression: literals, other constants, operators on them and arrays of these.")
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
	}

	return true
}

// constValue is literalOf, or an array of such values: one array, which
// every read of the constant shares.
func constValue(e ast.Expr) (value.Value, bool) {
	arr, ok := unwrap(e).(*ast.ArrayExpr)
	if !ok {
		return literalOf(e)
	}

	items := make([]value.Value, len(arr.Items))
	for i, item := range arr.Items {
		if items[i], ok = constValue(item); !ok {
			return value.NilValue, false
		}
	}

	return value.NewArray(items), true
}

// constPattern is the test of the value of the constant l, named by tok.
func (p *Resolver) constPattern(tok ast.Token, l *ast.Local) ast.Pattern {
	if l.Const.Kind() == value.Array {
		p.error(tok, "Constant '"+tok.Lexeme+"' is an array, which a pattern cannot test.")
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
