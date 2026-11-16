package rune

import "rune/pkg/ast"

// resolveMethods resolves the methods of stmt, marking the self of those
// taking mut self.
func (p *Resolver) resolveMethods(stmt *ast.ImplStmt) {
	for _, m := range stmt.Methods {
		p.resolveFn(m.Fn)

		if m.Method.Self {
			m.Fn.Parameters[0].Local.MutSelf = m.Fn.Parameters[0].Mutable
		}
	}
}
