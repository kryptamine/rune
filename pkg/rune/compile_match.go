package rune

import (
	"rune/pkg/ast"
)

// match compiles m into dst; for a statement (dst noReg) the value is
// dropped. No arm matching is an error either way.
func (fs *funcState) match(m *ast.MatchExpr, dst int32) {
	fs.scoped(func() {
		var guards []ast.Expr
		for _, arm := range m.Arms {
			if arm.Guard != nil {
				guards = append(guards, arm.Guard)
			}
		}

		subj := fs.operand(m.Subject, guards...)
		fs.nactive = fs.free

		var notEnum []int
		if m.EnumShape != nil {
			notEnum = []int{fs.typeTest(subj, m.EnumShape)}
		}

		last := &m.Arms[len(m.Arms)-1]
		fails := !reachedAll(last) || notEnum != nil

		var ends []int

		for i := range m.Arms {
			arm := &m.Arms[i]

			fs.scoped(func() {
				miss := fs.arm(arm, subj, dst)

				if arm != last || fails {
					ends = append(ends, fs.emit(ins{op: opJmp}, nil))
				}

				fs.patch(miss)
			})
		}

		if fails {
			fs.patch(notEnum)
			fs.emit(ins{op: opNoMatch, a: subj}, &m.Keyword)
		}

		fs.patch(ends)
	})
}

// arm compiles arm against R[subj] and returns the jumps taken when it
// does not match.
func (fs *funcState) arm(arm *ast.MatchArm, subj, dst int32) []int {
	bindings := ast.Bindings(arm.Pattern)
	for _, b := range bindings {
		fs.declare(b.Local)
	}

	miss := fs.pattern(arm.Pattern, subj)
	fs.free = fs.nactive
	fs.cellBindings(arm.Pattern)

	if arm.Guard != nil {
		miss = append(miss, fs.jumpIf(arm.Guard, false)...)
	}

	fs.block(arm.Body, dst)

	return miss
}
