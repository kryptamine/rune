package rune

import (
	"fmt"
	"maps"
	"rune/pkg/ast"
	"slices"
)

// liveSet holds the variables read again before they are set.
type liveSet map[*ast.Local]bool

func (s liveSet) add(o liveSet) { maps.Copy(s, o) }

func (s liveSet) reset(o liveSet) {
	clear(s)
	maps.Copy(s, o)
}

// liveness walks a function's body backward, from each point to the
// variables live there. A loop is walked until its head stops changing,
// so the last walk of a read decides whether it moves.
type liveness struct {
	// owners are the variables a let gives a value of their own: one that
	// is either shared or held by nothing else.
	owners map[*ast.Local]bool
	reads  map[*ast.VarExpr]bool
	// named counts the reads and assignments of each variable walked.
	named map[*ast.Local]int
	// pinned are live everywhere in the part walked: what a handler or a
	// defer, which an error or any exit runs, reads.
	pinned liveSet
	loops  []liveLoop
	// inDefer walks a defer, compiled at each of its exits: its reads never
	// move.
	inDefer int
}

type liveLoop struct{ head, out liveSet }

// markMoves sets Moves on the reads in body after which the variable is
// dead. A variable captured, exported or not given its value by a let or
// as a Param never moves: a mut parameter, a loop, catch or match
// variable holds what its giver may still hold. A Param owns its value
// only if the caller gave it up, which its moves test when they run.
func markMoves(params []ast.Binding, body []ast.Stmt) {
	lv := &liveness{owners: map[*ast.Local]bool{}, reads: map[*ast.VarExpr]bool{}, named: map[*ast.Local]int{}, pinned: liveSet{}}
	for _, p := range params {
		if p.Local.Param > 0 {
			lv.owners[p.Local] = true
		}
	}

	lv.stmts(body, liveSet{})

	for v := range lv.reads {
		l := v.Local
		v.Moves = v.Moves && lv.owners[l] && !l.Captured && !l.Exported
	}
}

func (lv *liveness) own(l *ast.Local, live liveSet) {
	if l != nil {
		lv.owners[l] = true
		delete(live, l)
	}
}

// stmts takes live from after stmts to before them. The statements after a
// defer run before its body, which reads what it likes on every exit.
func (lv *liveness) stmts(stmts []ast.Stmt, live liveSet) {
	for i, s := range stmts {
		if d, ok := s.(*ast.DeferStmt); ok {
			reads := lv.deferred(d.Body)
			pinned := lv.pin(reads)
			live.add(reads)
			lv.stmts(stmts[i+1:], live)
			lv.pinned = pinned
			stmts = stmts[:i]

			break
		}
	}

	for i := len(stmts) - 1; i >= 0; i-- {
		lv.stmt(stmts[i], live)
	}
}

// deferred walks a defer's body and returns what it reads.
func (lv *liveness) deferred(body ast.Stmt) liveSet {
	reads := liveSet{}
	lv.inDefer++
	lv.stmt(body, reads)
	lv.inDefer--

	return reads
}

// pin adds s to the pinned variables and returns those before.
func (lv *liveness) pin(s liveSet) liveSet {
	pinned := lv.pinned
	lv.pinned = maps.Clone(pinned)
	lv.pinned.add(s)

	return pinned
}

func (lv *liveness) stmt(stmt ast.Stmt, live liveSet) {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		lv.expr(s.Expr, live)
	case *ast.VarStmt:
		if s.Const {
			return
		}

		lv.own(s.Local, live)
		lv.expr(s.Initializer, live)
	case *ast.DestructureStmt:
		lv.destructure(s, s.Initializer, live)
	case *ast.ArrayDestructureStmt:
		lv.destructure(s, s.Initializer, live)
	case *ast.BlockStmt:
		lv.stmts(s.Stmts, live)
	case *ast.IfStmt:
		then := maps.Clone(live)
		lv.stmt(s.Then, then)
		if s.El != nil {
			lv.stmt(s.El, live)
		}

		live.add(then)
		lv.expr(s.Condition, live)
	case *ast.WhileStmt:
		lv.loop(live, func(live, out liveSet) {
			lv.stmt(s.Body, live)
			live.add(out)
			lv.expr(s.Condition, live)
		})
	case *ast.ForInStmt:
		var root *ast.Local
		if v := rootVar(s.Iterable); v != nil && s.Value.Mutable {
			root = v.Local
		}

		named := lv.named[root]
		lv.loop(live, func(live, out liveSet) {
			lv.stmt(s.Body, live)
			live.add(out)
		})
		s.Alone = root != nil && lv.named[root] == named
		lv.expr(s.Iterable, live)
	case *ast.ReturnStmt:
		clear(live)
		lv.expr(s.Value, live)
	case *ast.TryStmt:
		lv.try(live, s.Catches, func() { lv.stmt(s.Body, live) })
	case *ast.DeferStmt:
		live.add(lv.deferred(s.Body))
	case *ast.MatchStmt:
		lv.match(s.Match, live)
	case *ast.BreakStmt:
		live.reset(lv.loops[len(lv.loops)-1].out)
	case *ast.ContinueStmt:
		live.reset(lv.loops[len(lv.loops)-1].head)
	case *ast.FunctionStmt, *ast.StructStmt, *ast.ImplStmt:
	default:
		panic(fmt.Sprintf("moves: unknown statement %T", stmt))
	}
}

func (lv *liveness) destructure(s ast.Stmt, init ast.Expr, live liveSet) {
	vars := ast.PatternVars(s)
	for i := len(vars) - 1; i >= 0; i-- {
		lv.own(vars[i].Local, live)
		lv.expr(vars[i].Default, live)
	}

	lv.expr(init, live)
}

// loop takes live from after a loop to its head. body takes live from
// the head, where an iteration ends, back to it, given what is live past
// the loop.
func (lv *liveness) loop(live liveSet, body func(live, out liveSet)) {
	out := maps.Clone(live)
	head := maps.Clone(live)

	for {
		next := maps.Clone(head)
		lv.loops = append(lv.loops, liveLoop{head: head, out: out})
		body(next, out)
		lv.loops = lv.loops[:len(lv.loops)-1]

		if maps.Equal(next, head) {
			break
		}

		head = next
	}

	live.reset(head)
}

// try walks body, any point of which may go to the catches.
func (lv *liveness) try(live liveSet, catches []ast.CatchClause, body func()) {
	caught := liveSet{}
	for _, c := range catches {
		in := maps.Clone(live)
		if c.Value != nil {
			lv.expr(c.Value, in)
		} else {
			lv.stmts(c.Body, in)
		}

		caught.add(in)
	}

	pinned := lv.pin(caught)
	body()
	lv.pinned = pinned
	live.add(caught)
}

// match walks the arms in reverse: one that does not match, or whose
// guard fails, goes on to the next.
func (lv *liveness) match(m *ast.MatchExpr, live liveSet) {
	lv.holds(m.Subject, live)
	next := maps.Clone(live)

	for i := len(m.Arms) - 1; i >= 0; i-- {
		arm := m.Arms[i]
		in := maps.Clone(live)
		lv.stmt(arm.Body, in)
		in.add(next)
		lv.expr(arm.Guard, in)
		next.add(in)
	}

	live.reset(next)
	lv.expr(m.Subject, live)
}

// expr takes live from after e to before it.
func (lv *liveness) expr(e ast.Expr, live liveSet) {
	switch n := e.(type) {
	case nil, *ast.LiteralExpr, *ast.FunctionExpr:
	case *ast.VarExpr:
		if n.Local == nil {
			return
		}

		lv.reads[n] = true
		lv.named[n.Local]++
		n.Moves = lv.inDefer == 0 && !live[n.Local] && !lv.pinned[n.Local]
		live[n.Local] = true
	case *ast.GroupingExpr:
		lv.expr(n.Expr, live)
	case *ast.OptionalExpr:
		lv.expr(n.Expr, live)
	case *ast.SpreadExpr:
		lv.expr(n.Expr, live)
	case *ast.UnaryExpr:
		lv.expr(n.Right, live)
	case *ast.BinaryExpr:
		lv.holds(n.Left, live)
		lv.expr(n.Right, live)
		lv.expr(n.Left, live)
	case *ast.LogicalExpr:
		right := maps.Clone(live)
		lv.expr(n.Right, right)
		live.add(right)
		lv.expr(n.Left, live)
	case *ast.ConditionalExpr:
		then := maps.Clone(live)
		lv.expr(n.Then, then)
		lv.expr(n.Else, live)
		live.add(then)
		lv.expr(n.Cond, live)
	case *ast.AssignExpr:
		lv.named[n.Local]++
		delete(live, n.Local)
		lv.expr(n.Value, live)
	case *ast.CallExpr:
		args, prop := n.Args, (*ast.IndexExpr)(nil)
		if p, ok := n.Callee.(*ast.IndexExpr); ok {
			args, prop = append(slices.Clip(args), p.Array), p
		}

		given := lv.givable(args, live)

		if prop != nil {
			lv.holds(prop.Array, live)
		}

		// A method that keeps its arguments takes copies, which may move.
		if n.Methods&ast.InPlace == 0 {
			for _, arg := range n.Args {
				lv.holds(arg, live)
			}
		}

		lv.exprs(n.Args, live)
		lv.expr(n.Callee, live)
		lv.give(given)
	case *ast.IndexExpr:
		if n.Export != nil {
			lv.expr(n.Export, live)
			return
		}

		lv.holds(n.Array, live)
		lv.expr(n.Index, live)
		lv.expr(n.Array, live)
	case *ast.SetIndexExpr:
		lv.holds(n.Array, live)
		lv.holds(n.Index, live)
		lv.expr(n.Value, live)
		lv.expr(n.Index, live)
		lv.expr(n.Array, live)
	case *ast.UpdateExpr:
		lv.holds(n.Target, live)
		lv.expr(n.Target, live)
	case *ast.ArrayExpr:
		lv.exprs(n.Items, live)
	case *ast.InterpolationExpr:
		for _, part := range n.Parts {
			lv.holds(part, live)
		}

		lv.exprs(n.Parts, live)
	case *ast.ObjectExpr:
		for i := len(n.Pairs) - 1; i >= 0; i-- {
			lv.expr(n.Pairs[i].Value, live)
		}
	case *ast.ComputedExpr:
		lv.expr(n.Value, live)
		lv.expr(n.Key, live)
	case *ast.StructExpr:
		for i := len(n.Pairs) - 1; i >= 0; i-- {
			lv.expr(n.Pairs[i].Value, live)
		}
	case *ast.IsExpr:
		if n.Shape == nil {
			lv.holds(n.Value, live)
			lv.expr(n.Type, live)
		}

		lv.expr(n.Value, live)
	case *ast.RangeExpr:
		lv.expr(n.Step, live)
		lv.expr(n.End, live)
		lv.expr(n.Start, live)
	case *ast.SliceExpr:
		lv.holds(n.Target, live)
		lv.expr(n.Range, live)
		lv.expr(n.Target, live)
	case *ast.ThrowExpr:
		clear(live)
		lv.expr(n.Value, live)
	case *ast.TryExpr:
		lv.try(live, n.Catches, func() { lv.expr(n.Value, live) })
	case *ast.JumpExpr:
		lv.stmt(n.Stmt, live)
	case *ast.MatchExpr:
		lv.match(n, live)
	default:
		panic(fmt.Sprintf("moves: unknown expression %T", e))
	}
}

func (lv *liveness) exprs(es []ast.Expr, live liveSet) {
	for i := len(es) - 1; i >= 0; i-- {
		lv.expr(es[i], live)
	}
}

// holds keeps live the variable e reads in place, with no copy: what an
// operation reads that way it holds till done, a write through it too.
func (lv *liveness) holds(e ast.Expr, live liveSet) {
	if sp, ok := e.(*ast.SpreadExpr); ok {
		e = sp.Expr
	}

	if v := rootVar(e); v != nil && v.Local != nil {
		live[v.Local] = true
	}
}

// givable are the arguments of a call that are a variable alone, dead
// after it, with the count of its reads so far.
func (lv *liveness) givable(args []ast.Expr, live liveSet) map[*ast.VarExpr]int {
	var given map[*ast.VarExpr]int

	for _, arg := range args {
		if v, ok := unwrap(arg).(*ast.VarExpr); ok && v.Local != nil && !live[v.Local] && !lv.pinned[v.Local] && lv.inDefer == 0 {
			if given == nil {
				given = map[*ast.VarExpr]int{}
			}

			given[v] = lv.named[v.Local]
		}
	}

	return given
}

// give moves each of given whose variable the call read only there: the
// call holds what it reads in place, so a caller gives an argument up
// only if nothing else in the call has it.
func (lv *liveness) give(given map[*ast.VarExpr]int) {
	for v, n := range given {
		v.Moves = lv.named[v.Local] == n+1
	}
}
