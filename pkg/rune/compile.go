package rune

import (
	"fmt"
	"maps"
	"rune/pkg/ast"
	"rune/pkg/environment"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
)

type (
	evalFn func(p *Interpreter) (value.Value, error)
	execFn func(p *Interpreter) error
	condFn func(p *Interpreter) (bool, error)
)

// Program holds no run state and can be run any number of times.
type Program struct {
	run execFn
}

func Compile(stmts []ast.Stmt) *Program {
	return &Program{run: compileStmts(stmts)}
}

type function struct {
	name    string
	arity   int
	envSize int
	pooled  bool
	body    execFn
}

type Closure struct {
	fn  *function
	env *environment.Environment
}

func (c *Closure) String() string {
	if c.fn.name == "" {
		return "<fn>"
	}

	return fmt.Sprintf("<fn %s>", c.fn.name)
}

func (f *function) displayName() string {
	if f.name == "" {
		return "<anonymous>"
	}

	return f.name
}

func compileFunction(decl *ast.FunctionStmt) *function {
	return &function{
		name:    decl.Name.Lexeme,
		arity:   len(decl.Parameters),
		envSize: decl.EnvSize,
		pooled:  decl.Pooled,
		body:    compileStmts(decl.Body),
	}
}

func nop(*Interpreter) error { return nil }

func compileStmts(stmts []ast.Stmt) execFn {
	fns := make([]execFn, len(stmts))
	for i, stmt := range stmts {
		fns[i] = compileStmt(stmt)
	}

	switch len(fns) {
	case 0:
		return nop
	case 1:
		return fns[0]
	case 2:
		a, b := fns[0], fns[1]

		return func(p *Interpreter) error {
			if err := a(p); err != nil {
				return err
			}

			return b(p)
		}
	}

	return func(p *Interpreter) error {
		for _, f := range fns {
			if err := f(p); err != nil {
				return err
			}
		}

		return nil
	}
}

func compileStmt(stmt ast.Stmt) execFn {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		e := compileExpr(s.Expr)

		return func(p *Interpreter) error {
			_, err := e(p)
			return err
		}
	case *ast.PrintStmt:
		return compilePrint(s)
	case *ast.VarStmt:
		return compileVar(s)
	case *ast.BlockStmt:
		return compileScope(compileStmts(s.Stmts), s.EnvSize, s.Pooled)
	case *ast.IfStmt:
		return compileIf(s)
	case *ast.WhileStmt:
		return compileWhile(s)
	case *ast.ForInStmt:
		return compileForIn(s)
	case *ast.FunctionStmt:
		return compileFunctionStmt(s)
	case *ast.ReturnStmt:
		return compileReturn(s)
	case *ast.TryStmt:
		return compileTry(s)
	case *ast.ThrowStmt:
		return compileThrow(s)
	case *ast.BreakStmt:
		return func(*Interpreter) error { return errBreak }
	case *ast.ContinueStmt:
		return func(*Interpreter) error { return errContinue }
	}

	panic(fmt.Sprintf("compile: unknown statement %T", stmt))
}

func compilePrint(s *ast.PrintStmt) execFn {
	e := compileExpr(s.Expr)

	return func(p *Interpreter) error {
		v, err := e(p)
		if err != nil {
			return err
		}

		p.line = append(helpers.AppendValue(p.line[:0], v), '\n')
		p.write(p.line)

		return nil
	}
}

func compileVar(s *ast.VarStmt) execFn {
	init := func(*Interpreter) (value.Value, error) { return value.NilValue, nil }
	if s.Initializer != nil {
		init = compileExpr(s.Initializer)
	}

	if s.Local {
		slot := s.Slot

		return func(p *Interpreter) error {
			v, err := init(p)
			if err != nil {
				return err
			}

			p.environment.Set(slot, v)

			return nil
		}
	}

	slot := globalSlot(s.Slot, s.Name.Lexeme)

	return func(p *Interpreter) error {
		v, err := init(p)
		if err != nil {
			return err
		}

		p.globals.Define(slot, v)

		return nil
	}
}

func compileScope(body execFn, size int, pooled bool) execFn {
	if size == 0 {
		return body
	}

	return func(p *Interpreter) error {
		return p.runIn(body, p.newEnv(pooled, p.environment, size), pooled)
	}
}

func compileIf(s *ast.IfStmt) execFn {
	cond := compileCond(s.Condition)
	then := compileStmt(s.Then)

	if s.El == nil {
		return func(p *Interpreter) error {
			c, err := cond(p)
			if err != nil {
				return err
			}

			if c {
				return then(p)
			}

			return nil
		}
	}

	el := compileStmt(s.El)

	return func(p *Interpreter) error {
		c, err := cond(p)
		if err != nil {
			return err
		}

		if c {
			return then(p)
		}

		return el(p)
	}
}

func compileWhile(s *ast.WhileStmt) execFn {
	cond := compileCond(s.Condition)
	body := compileStmt(s.Body)

	var incr evalFn
	if s.Increment != nil {
		incr = compileExpr(s.Increment)
	}

	return func(p *Interpreter) error {
		for {
			c, err := cond(p)
			if err != nil {
				return err
			}

			if !c {
				return nil
			}

			if err := body(p); err != nil && err != errContinue {
				if err == errBreak {
					return nil
				}

				return err
			}

			if incr != nil {
				if _, err := incr(p); err != nil {
					return err
				}
			}
		}
	}
}

// Elements are snapshotted when the loop starts.
func compileForIn(s *ast.ForInStmt) execFn {
	iterable := compileExpr(s.Iterable)
	body := compileStmt(s.Body)
	size, pooled, hasKey, keyword := s.EnvSize, s.Pooled, s.Key != nil, s.Keyword

	visit := func(p *Interpreter, env *environment.Environment, key, val value.Value) (bool, error) {
		if hasKey {
			env.Set(0, key)
			env.Set(1, val)
		} else {
			env.Set(0, val)
		}

		prev := p.environment
		p.environment = env
		err := body(p)
		p.environment = prev

		switch err {
		case nil, errContinue:
			return false, nil
		case errBreak:
			return true, nil
		}

		return true, err
	}

	return func(p *Interpreter) error {
		it, err := iterable(p)
		if err != nil {
			return err
		}

		var shared *environment.Environment
		if pooled {
			shared = p.pool.Get(p.environment, size)
			defer p.pool.Put(shared)
		}

		scope := func() *environment.Environment {
			if shared != nil {
				return shared
			}

			return environment.NewEnvironment(p.environment, size)
		}

		switch it.Kind() {
		case value.Array:
			for i, v := range slices.Clone(it.Arr()) {
				if stop, err := visit(p, scope(), value.NewNumber(float64(i)), v); stop {
					return err
				}
			}
		case value.String:
			for i, ch := range helpers.Chars(it.Str()) {
				if stop, err := visit(p, scope(), value.NewNumber(float64(i)), value.NewString(ch)); stop {
					return err
				}
			}
		case value.Object:
			obj := it.Obj()

			for _, k := range slices.Sorted(maps.Keys(obj)) {
				key := value.NewString(k)

				val := key
				if hasKey {
					val = obj[k]
				}

				if stop, err := visit(p, scope(), key, val); stop {
					return err
				}
			}
		default:
			return errors.NewRuntimeError(keyword, fmt.Sprintf(
				"Can only iterate over arrays, strings and objects, got %s.", helpers.TypeName(it)))
		}

		return nil
	}
}

func compileFunctionStmt(s *ast.FunctionStmt) execFn {
	fn := compileFunction(s)

	if s.Local {
		slot := s.Slot

		return func(p *Interpreter) error {
			p.environment.Set(slot, value.NewFunction(&Closure{fn: fn, env: p.environment}))
			return nil
		}
	}

	slot := globalSlot(s.Slot, s.Name.Lexeme)

	return func(p *Interpreter) error {
		p.globals.Define(slot, value.NewFunction(&Closure{fn: fn, env: p.environment}))
		return nil
	}
}

func compileReturn(s *ast.ReturnStmt) execFn {
	if s.Value == nil {
		return func(p *Interpreter) error {
			p.returnValue = value.NilValue
			return errReturn
		}
	}

	e := compileExpr(s.Value)

	return func(p *Interpreter) error {
		v, err := e(p)
		if err != nil {
			return err
		}

		p.returnValue = v

		return errReturn
	}
}

func compileTry(s *ast.TryStmt) execFn {
	body := compileStmt(s.Body)
	handler := compileStmts(s.Handler)
	size, pooled, hasParam := s.HandlerEnvSize, s.HandlerPooled, s.Param != nil

	return func(p *Interpreter) error {
		err := body(p)

		rtErr, ok := err.(errors.RuntimeError)
		if !ok {
			return err
		}

		if size == 0 {
			return handler(p)
		}

		env := p.newEnv(pooled, p.environment, size)

		if hasParam {
			env.Set(0, rtErr.Value())
		}

		return p.runIn(handler, env, pooled)
	}
}

func compileThrow(s *ast.ThrowStmt) execFn {
	e := compileExpr(s.Value)
	keyword := s.Keyword

	return func(p *Interpreter) error {
		v, err := e(p)
		if err != nil {
			return err
		}

		return errors.NewThrownError(keyword, v, helpers.Stringify(v))
	}
}

func compileExpr(expr ast.Expr) evalFn {
	switch n := expr.(type) {
	case *ast.LiteralExpr:
		return constant(n.Value)
	case *ast.GroupingExpr:
		return compileExpr(n.Expr)
	case *ast.VarExpr:
		return compileVarExpr(n)
	case *ast.AssignExpr:
		return compileAssign(n)
	case *ast.BinaryExpr:
		return compileBinary(n)
	case *ast.LogicalExpr:
		return compileLogical(n)
	case *ast.UnaryExpr:
		return compileUnary(n)
	case *ast.CallExpr:
		return compileCall(n)
	case *ast.ArrayExpr:
		return compileArray(n)
	case *ast.ObjectExpr:
		return compileObject(n)
	case *ast.IndexExpr:
		return compileIndex(n)
	case *ast.SetIndexExpr:
		return compileSetIndex(n)
	case *ast.FunctionExpr:
		fn := compileFunction(n.Decl)

		return func(p *Interpreter) (value.Value, error) {
			return value.NewFunction(&Closure{fn: fn, env: p.environment}), nil
		}
	}

	panic(fmt.Sprintf("compile: unknown expression %T", expr))
}

func constant(v value.Value) evalFn {
	return func(*Interpreter) (value.Value, error) { return v, nil }
}

func literalOf(expr ast.Expr) (value.Value, bool) {
	switch n := expr.(type) {
	case *ast.LiteralExpr:
		return n.Value, true
	case *ast.GroupingExpr:
		return literalOf(n.Expr)
	}

	return value.NilValue, false
}

func compileVarExpr(n *ast.VarExpr) evalFn {
	slot := n.Slot

	switch n.Depth {
	case ast.Unresolved:
		slot := globalSlot(n.Slot, n.Name.Lexeme)
		name := &n.Name

		return func(p *Interpreter) (value.Value, error) {
			return p.globals.Get(slot, name)
		}
	case 0:
		return func(p *Interpreter) (value.Value, error) {
			return p.environment.Get(slot), nil
		}
	case 1:
		return func(p *Interpreter) (value.Value, error) {
			return p.environment.Enclosing().Get(slot), nil
		}
	}

	depth := n.Depth

	return func(p *Interpreter) (value.Value, error) {
		return p.environment.GetAt(depth, slot), nil
	}
}

func compileAssign(n *ast.AssignExpr) evalFn {
	if f := compileIncrement(n); f != nil {
		return f
	}

	val := compileExpr(n.Value)
	depth, slot := n.Depth, n.Slot

	switch depth {
	case ast.Unresolved:
		slot := globalSlot(n.Slot, n.Name.Lexeme)
		name := &n.Name

		return func(p *Interpreter) (value.Value, error) {
			v, err := val(p)
			if err != nil {
				return value.NilValue, err
			}

			return v, p.globals.Assign(slot, name, v)
		}
	case 0:
		return func(p *Interpreter) (value.Value, error) {
			v, err := val(p)
			if err != nil {
				return value.NilValue, err
			}

			p.environment.Set(slot, v)

			return v, nil
		}
	}

	return func(p *Interpreter) (value.Value, error) {
		v, err := val(p)
		if err != nil {
			return value.NilValue, err
		}

		p.environment.AssignAt(depth, slot, v)

		return v, nil
	}
}

// compileIncrement handles x = x ± c (and x += c); nil otherwise.
func compileIncrement(n *ast.AssignExpr) evalFn {
	bin, ok := n.Value.(*ast.BinaryExpr)
	if !ok || (bin.Operator.TokenType != ast.PLUS && bin.Operator.TokenType != ast.MINUS) {
		return nil
	}

	target, ok := bin.Left.(*ast.VarExpr)
	if !ok || target.Name.Lexeme != n.Name.Lexeme || target.Depth != n.Depth || target.Slot != n.Slot {
		return nil
	}

	c, ok := literalOf(bin.Right)
	if !ok || !c.IsNumber() {
		return nil
	}

	delta := c.Num()
	if bin.Operator.TokenType == ast.MINUS {
		delta = -delta
	}

	op := &bin.Operator

	switch n.Depth {
	case ast.Unresolved:
		slot := globalSlot(n.Slot, n.Name.Lexeme)
		name := &n.Name

		return func(p *Interpreter) (value.Value, error) {
			old, err := p.globals.Get(slot, name)
			if err != nil {
				return value.NilValue, err
			}

			if !old.IsNumber() {
				_, err := binaryOp(op, old, c)
				return value.NilValue, err
			}

			v := value.NewNumber(old.Num() + delta)

			return v, p.globals.Assign(slot, name, v)
		}
	case 0:
		slot := n.Slot

		return func(p *Interpreter) (value.Value, error) {
			env := p.environment

			old := env.Get(slot)
			if !old.IsNumber() {
				_, err := binaryOp(op, old, c)
				return value.NilValue, err
			}

			v := value.NewNumber(old.Num() + delta)
			env.Set(slot, v)

			return v, nil
		}
	}

	return nil
}

func compileBinary(n *ast.BinaryExpr) evalFn {
	op := &n.Operator
	opType := op.TokenType

	lc, lConst := literalOf(n.Left)
	rc, rConst := literalOf(n.Right)

	// A fold that fails is left to fail at run time.
	if lConst && rConst {
		if v, err := binaryOp(op, lc, rc); err == nil {
			return constant(v)
		}
	}

	left := compileExpr(n.Left)

	if rConst && rc.IsNumber() {
		r := rc.Num()

		return func(p *Interpreter) (value.Value, error) {
			l, err := left(p)
			if err != nil {
				return value.NilValue, err
			}

			if l.IsNumber() {
				return numberOp(opType, l.Num(), r), nil
			}

			return binaryOp(op, l, rc)
		}
	}

	right := compileExpr(n.Right)

	return func(p *Interpreter) (value.Value, error) {
		l, err := left(p)
		if err != nil {
			return value.NilValue, err
		}

		r, err := right(p)
		if err != nil {
			return value.NilValue, err
		}

		if l.IsNumber() && r.IsNumber() {
			return numberOp(opType, l.Num(), r.Num()), nil
		}

		return binaryOp(op, l, r)
	}
}

func compileLogical(n *ast.LogicalExpr) evalFn {
	left := compileExpr(n.Left)
	right := compileExpr(n.Right)
	isOr := n.Op.TokenType == ast.OR

	return func(p *Interpreter) (value.Value, error) {
		l, err := left(p)
		if err != nil {
			return value.NilValue, err
		}

		if helpers.IsTruthy(l) == isOr {
			return l, nil
		}

		return right(p)
	}
}

func compileUnary(n *ast.UnaryExpr) evalFn {
	op := n.Operator

	if c, ok := literalOf(n.Right); ok && op.TokenType == ast.MINUS && c.IsNumber() {
		return constant(value.NewNumber(-c.Num()))
	}

	right := compileExpr(n.Right)

	if op.TokenType == ast.BANG {
		return func(p *Interpreter) (value.Value, error) {
			v, err := right(p)
			if err != nil {
				return value.NilValue, err
			}

			return value.NewBool(!helpers.IsTruthy(v)), nil
		}
	}

	return func(p *Interpreter) (value.Value, error) {
		v, err := right(p)
		if err != nil {
			return value.NilValue, err
		}

		if !v.IsNumber() {
			return value.NilValue, errors.NewRuntimeError(op, "Operand must be a number.")
		}

		return value.NewNumber(-v.Num()), nil
	}
}

func compileCond(expr ast.Expr) condFn {
	switch n := expr.(type) {
	case *ast.GroupingExpr:
		return compileCond(n.Expr)

	case *ast.BinaryExpr:
		if !isComparison(n.Operator.TokenType) {
			break
		}

		op := &n.Operator
		opType := op.TokenType
		left := compileExpr(n.Left)

		if rc, ok := literalOf(n.Right); ok && rc.IsNumber() {
			r := rc.Num()

			return func(p *Interpreter) (bool, error) {
				l, err := left(p)
				if err != nil {
					return false, err
				}

				if l.IsNumber() {
					return compare(opType, l.Num(), r), nil
				}

				v, err := binaryOp(op, l, rc)

				return v.Bool(), err
			}
		}

		right := compileExpr(n.Right)

		return func(p *Interpreter) (bool, error) {
			l, err := left(p)
			if err != nil {
				return false, err
			}

			r, err := right(p)
			if err != nil {
				return false, err
			}

			if l.IsNumber() && r.IsNumber() {
				return compare(opType, l.Num(), r.Num()), nil
			}

			v, err := binaryOp(op, l, r)

			return v.Bool(), err
		}

	case *ast.LogicalExpr:
		left, right := compileCond(n.Left), compileCond(n.Right)

		if n.Op.TokenType == ast.OR {
			return func(p *Interpreter) (bool, error) {
				if l, err := left(p); err != nil || l {
					return l, err
				}

				return right(p)
			}
		}

		return func(p *Interpreter) (bool, error) {
			if l, err := left(p); err != nil || !l {
				return l, err
			}

			return right(p)
		}

	case *ast.UnaryExpr:
		if n.Operator.TokenType != ast.BANG {
			break
		}

		inner := compileCond(n.Right)

		return func(p *Interpreter) (bool, error) {
			c, err := inner(p)
			return !c, err
		}
	}

	e := compileExpr(expr)

	return func(p *Interpreter) (bool, error) {
		v, err := e(p)
		if err != nil {
			return false, err
		}

		return helpers.IsTruthy(v), nil
	}
}

func compileArray(n *ast.ArrayExpr) evalFn {
	items := make([]evalFn, len(n.Items))
	for i, item := range n.Items {
		items[i] = compileExpr(item)
	}

	return func(p *Interpreter) (value.Value, error) {
		vals := make([]value.Value, len(items))

		for i, item := range items {
			v, err := item(p)
			if err != nil {
				return value.NilValue, err
			}

			vals[i] = v
		}

		return value.NewArray(vals), nil
	}
}

func compileObject(n *ast.ObjectExpr) evalFn {
	keys := make([]string, len(n.Pairs))
	vals := make([]evalFn, len(n.Pairs))

	for i, pair := range n.Pairs {
		keys[i] = pair.Key
		vals[i] = compileExpr(pair.Value)
	}

	return func(p *Interpreter) (value.Value, error) {
		obj := make(map[string]value.Value, len(keys))

		for i, val := range vals {
			v, err := val(p)
			if err != nil {
				return value.NilValue, err
			}

			obj[keys[i]] = v
		}

		return value.NewObject(obj), nil
	}
}

// compileProperty compiles obj.name, whose key is known up front.
func compileProperty(n *ast.IndexExpr) evalFn {
	target := compileExpr(n.Array)
	token, key := &n.Token, n.Token.Lexeme

	return func(p *Interpreter) (value.Value, error) {
		t, err := target(p)
		if err != nil {
			return value.NilValue, err
		}

		if t.Kind() != value.Object {
			return value.NilValue, propertyError("read", token, t)
		}

		if v, ok := t.Obj()[key]; ok {
			return v, nil
		}

		return value.NilValue, undefinedProperty(token, key)
	}
}

func compileIndex(n *ast.IndexExpr) evalFn {
	if n.Property {
		return compileProperty(n)
	}

	target := compileExpr(n.Array)
	index := compileExpr(n.Index)
	token := &n.Token

	return func(p *Interpreter) (value.Value, error) {
		t, err := target(p)
		if err != nil {
			return value.NilValue, err
		}

		i, err := index(p)
		if err != nil {
			return value.NilValue, err
		}

		if t.Kind() == value.Array && i.IsNumber() {
			arr := t.Arr()
			if idx := int(i.Num()); float64(idx) == i.Num() && idx >= 0 && idx < len(arr) {
				return arr[idx], nil
			}
		}

		return getIndex(token, t, i)
	}
}

func compileSetIndex(n *ast.SetIndexExpr) evalFn {
	target := compileExpr(n.Array)
	index := compileExpr(n.Index)
	val := compileExpr(n.Value)
	token, op, property := &n.Token, n.Op, n.Property

	return func(p *Interpreter) (value.Value, error) {
		t, err := target(p)
		if err != nil {
			return value.NilValue, err
		}

		if property && t.Kind() != value.Object {
			return value.NilValue, propertyError("set", token, t)
		}

		i, err := index(p)
		if err != nil {
			return value.NilValue, err
		}

		// For a compound assignment, read the old value before evaluating
		// the right-hand side, as "a[i] = a[i] op v" would.
		var old value.Value

		if op != nil {
			if old, err = getIndex(token, t, i); err != nil {
				return value.NilValue, err
			}
		}

		v, err := val(p)
		if err != nil {
			return value.NilValue, err
		}

		if op != nil {
			if v, err = binaryOp(op, old, v); err != nil {
				return value.NilValue, err
			}
		}

		if err := setIndex(token, t, i, v); err != nil {
			return value.NilValue, err
		}

		return v, nil
	}
}
