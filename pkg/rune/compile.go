package rune

import (
	"rune/pkg/ast"
	"rune/pkg/stdlib"
	"rune/pkg/value"
	"unsafe"
)

// funcState compiles a function of a resolved AST into register bytecode.
// Every local lives in a register of its function; a local some nested
// function captures holds a cell instead, so closures share it and each
// declaration that runs (once per loop iteration, say) makes a fresh one.
type funcState struct {
	parent *funcState
	proto  *Proto
	locals map[*ast.Local]int32
	upvals map[*ast.Local]int32
	consts map[constKey]int32
	// Registers below nactive hold locals; temporaries start at free.
	nactive int32
	free    int32
	code    []ins
	loops   []*loop
	tries   int
	// chain holds the jumps of the ?. of the optional chain being compiled.
	chain []int
	// defined holds the top-level variables whose declaration the script
	// has passed, so reading them needs no check.
	defined map[*ast.Local]bool
	unit    *unit
	// A mut self lives outside register 0 when a closure captures it,
	// in a cell there if selfCell, or when it is owned, in register
	// selfReg; returns put it back into register 0 for the caller.
	selfCell bool
	selfReg  int32
}

// unit is what the functions of a program share: each field name's one
// string, which constants equal to it reuse so the VM can match a field by
// pointer.
type unit struct {
	names map[string]string
}

type loop struct {
	breaks, continues []int
	tries             int
}

type constKey struct {
	kind  value.Kind
	isInt bool
	num   float64
	int   int64
	str   string
	ptr   unsafe.Pointer
}

// compile compiles a program; builtins are the declarations the resolver
// made for the built-ins it names. The script takes the program's
// arguments and returns what main, if any, returns.
func compile(stmts []ast.Stmt, builtins []namedLocal, main *ast.FunctionStmt, methods methodTable) (*Program, error) {
	return compileScript(methods, func(fs *funcState, args *ast.Local) *ast.Token {
		fs.declareStructs(stmts)
		fs.declareTopLevel(stmts, builtins)
		fs.declareMethods(stmts)
		fs.stmts(stmts)

		if main == nil {
			fs.emit(ins{op: opReturnNil}, nil)
			return nil
		}

		call := &ast.CallExpr{Token: main.Name, Callee: &ast.VarExpr{Name: main.Name, Local: main.Local}}
		if len(main.Parameters) == 1 {
			call.Args = []ast.Expr{&ast.VarExpr{Local: args}}
		}

		r := fs.alloc()
		fs.call(call, r)
		fs.emit(ins{op: opReturn, a: r}, nil)

		return &main.Name
	})
}

// compileExpr compiles a lone expression into a program returning it.
func compileExpr(expr ast.Expr, builtins []namedLocal, methods methodTable) (*Program, error) {
	return compileScript(methods, func(fs *funcState, _ *ast.Local) *ast.Token {
		fs.declareTopLevel(nil, builtins)
		r := fs.alloc()
		fs.expr(expr, r)
		fs.emit(ins{op: opReturn, a: r}, nil)

		return nil
	})
}

// compileScript compiles the top level, whose one parameter is args; body
// returns the name of main, if it calls one. A call names a method of a
// built-in type by the string the method has.
func compileScript(methods methodTable, body func(fs *funcState, args *ast.Local) *ast.Token) (prog *Program, err error) {
	defer func() {
		switch e := recover().(type) {
		case nil:
		case tooLarge:
			prog, err = nil, e
		default:
			panic(e)
		}
	}()

	fs := newFunc(nil, "<script>")

	for _, ms := range methods.kinds {
		for _, m := range ms {
			fs.unit.names[m.Name] = m.Name
		}
	}

	args := &ast.Local{}
	fs.declare(args)
	fs.proto.arity, fs.proto.required = 1, 1

	main := body(fs, args)
	fs.finish()

	return &Program{script: fs.proto, main: main, methods: methods}, nil
}

func (fs *funcState) finish() {
	fs.proto.exact = fs.proto.arity
	if fs.proto.rest {
		fs.proto.exact = -1
	}

	fs.proto.code = make([]Instr, len(fs.code))
	for i, in := range fs.code {
		fs.proto.code[i] = in.encode(fs.proto.displayName())
	}
}

func newFunc(parent *funcState, name string) *funcState {
	fs := &funcState{
		parent:  parent,
		proto:   &Proto{name: name},
		locals:  map[*ast.Local]int32{},
		upvals:  map[*ast.Local]int32{},
		consts:  map[constKey]int32{},
		defined: map[*ast.Local]bool{},
	}

	if parent != nil {
		fs.unit = parent.unit
	} else {
		fs.unit = &unit{names: stdlib.FieldNames()}
	}

	return fs
}

// declareStructs binds the struct types' names and collects their field
// names, before any constant is made.
func (fs *funcState) declareStructs(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		s, ok := stmt.(*ast.StructStmt)
		if !ok {
			continue
		}

		for _, f := range s.Shape.Fields {
			fs.unit.names[f] = f
		}

		fs.declareAs(s.Local, value.NewType(s.Shape))
		fs.defined[s.Local] = true
	}
}

// declareMethods makes the closures of every method before any other
// code runs, once the top-level variables they may capture exist. It
// first collects the method names, before any string constant is made.
func (fs *funcState) declareMethods(stmts []ast.Stmt) {
	var impls []*ast.ImplStmt

	for _, stmt := range stmts {
		if impl, ok := stmt.(*ast.ImplStmt); ok {
			impls = append(impls, impl)

			for _, m := range impl.Methods {
				fs.unit.names[m.Name] = m.Name
			}
		}
	}

	for _, impl := range impls {
		for _, m := range impl.Methods {
			var self *ast.Local
			if m.Mut {
				self = m.Fn.Parameters[0].Local
			}

			r := fs.alloc()
			fs.emit(ins{op: opClosure, a: r, c: fs.functionNamed(m.Fn, impl.Type.Lexeme+"."+m.Name, self)}, nil)
			fs.emit(ins{op: opMethod, a: r, c: int32(m.ID)}, nil)
			fs.free = fs.nactive
		}
	}
}

// declareTopLevel gives every top-level variable a register of the
// script, holding the built-in of that name until it is declared, and
// every built-in the program names one holding it from the start.
func (fs *funcState) declareTopLevel(stmts []ast.Stmt, builtins []namedLocal) {
	for _, stmt := range stmts {
		for _, b := range ast.Declared(stmt) {
			if _, done := fs.locals[b.Local]; !done {
				fs.declareTopLevelVar(b.Local, b.Name.Lexeme)
			}
		}
	}

	for _, b := range builtins {
		fs.declareAs(b.local, b.value)
		fs.defined[b.local] = true
	}
}

func (fs *funcState) declareTopLevelVar(l *ast.Local, name string) {
	v, ok := builtinValues()[name]
	if !ok {
		v = value.UndefinedValue
	}

	fs.declareAs(l, v)
}

// declareAs binds l to a register holding v, in a cell if l is captured.
func (fs *funcState) declareAs(l *ast.Local, v value.Value) {
	r := fs.declare(l)
	fs.emit(ins{op: opLoadK, a: r, c: fs.constant(v)}, nil)

	if l.Captured {
		fs.emit(ins{op: opNewCell, a: r, b: r}, nil)
	}
}

// checked reports whether reading l must check it is defined.
func (fs *funcState) checked(l *ast.Local) bool {
	return l != nil && l.TopLevel && !fs.defined[l]
}

func (fs *funcState) function(decl *ast.FunctionStmt) int32 {
	return fs.functionNamed(decl, decl.Name.Lexeme, nil)
}

// functionNamed compiles decl as name; mutSelf is the parameter of a
// method taking mut self, or nil.
func (fs *funcState) functionNamed(decl *ast.FunctionStmt, name string, mutSelf *ast.Local) int32 {
	f := newFunc(fs, name)
	f.proto.arity = len(decl.Parameters)
	if decl.Rest {
		f.proto.arity, f.proto.rest = f.proto.arity-1, true
	}

	f.proto.required = decl.Required()

	regs := make([]int32, len(decl.Parameters))
	for i, param := range decl.Parameters {
		regs[i] = f.declare(param.Local)
	}

	for i, param := range decl.Parameters {
		if decl.Defaults[i] != nil {
			skip := f.emit(ins{op: opJmpDefined, a: regs[i]}, nil)
			f.expr(decl.Defaults[i], regs[i])
			f.patch([]int{skip})
		}

		if param.Local.Captured {
			f.emit(ins{op: opNewCell, a: regs[i], b: regs[i]}, nil)
		}
	}

	f.selfCell = mutSelf != nil && mutSelf.Captured
	if mutSelf != nil && mutSelf.Owned {
		f.selfReg = f.declare(mutSelf)
		f.emit(ins{op: opMove, a: f.selfReg, b: 0}, nil)
		f.ownership(f.declare(nil), nil)
	}

	f.stmts(decl.Body)
	f.returnSelf()
	f.emit(ins{op: opReturnNil}, nil)
	f.finish()

	fs.proto.protos = append(fs.proto.protos, f.proto)

	return int32(len(fs.proto.protos) - 1)
}

// returnSelf puts a mut self kept elsewhere back into register 0 before a return.
func (fs *funcState) returnSelf() {
	switch {
	case fs.selfCell:
		fs.emit(ins{op: opGetCell, a: 0, b: 0}, nil)
	case fs.selfReg != 0:
		fs.move(0, fs.selfReg)
	}
}

func (fs *funcState) emit(in ins, tok *ast.Token) int {
	fs.code = append(fs.code, in)
	fs.proto.toks = append(fs.proto.toks, tok)

	return len(fs.code) - 1
}

func (fs *funcState) here() int32 { return int32(len(fs.code)) }

func (fs *funcState) patch(jumps []int) {
	for _, j := range jumps {
		fs.code[j].c = fs.here()
	}
}

func (fs *funcState) alloc() int32 {
	r := fs.free
	fs.free++

	if int(fs.free) > fs.proto.nregs {
		fs.proto.nregs = int(fs.free)
	}

	return r
}

// declare binds l to the next register; temporaries must all be free.
func (fs *funcState) declare(l *ast.Local) int32 {
	r := fs.alloc()
	fs.nactive = fs.free

	if l != nil {
		fs.locals[l] = r
	}

	return r
}

func (fs *funcState) constant(v value.Value) int32 {
	key := constKey{kind: v.Kind()}

	switch v.Kind() {
	case value.Number:
		key.isInt, key.num = v.IsInt(), v.Num()
		if v.IsInt() {
			key.int = v.Int()
		}
	case value.Bool:
		key.num = boolNum(v.Bool())
	case value.Bytes:
		key.str = string(v.Bytes())
	case value.String:
		key.str = v.Str()
		if name, ok := fs.unit.names[key.str]; ok {
			v = value.NewString(name)
		}
	case value.Function, value.Type, value.Enum, value.Struct, value.Array, value.Object, value.Map, value.Set:
		key.ptr = v.Ptr()
	}

	if k, ok := fs.consts[key]; ok {
		return k
	}

	k := int32(len(fs.proto.consts))
	fs.proto.consts = append(fs.proto.consts, v.Detach())
	fs.consts[key] = k

	return k
}

func boolNum(b bool) float64 {
	if b {
		return 1
	}

	return 0
}

type varKind uint8

const (
	varUndefined varKind = iota // a name neither declared nor built in
	varLocal                    // a plain register
	varCell                     // a register holding a cell
	varUpval
)

// lookup says where the variable l lives.
func (fs *funcState) lookup(l *ast.Local) (varKind, int32) {
	if l == nil {
		return varUndefined, 0
	}

	if r, ok := fs.locals[l]; ok {
		if l.Captured {
			return varCell, r
		}

		return varLocal, r
	}

	return varUpval, fs.upval(l)
}

func (fs *funcState) upval(l *ast.Local) int32 {
	if i, ok := fs.upvals[l]; ok {
		return i
	}

	var desc upvalDesc
	if r, ok := fs.parent.locals[l]; ok {
		desc = upvalDesc{fromLocal: true, index: r}
	} else {
		desc = upvalDesc{index: fs.parent.upval(l)}
	}

	i := int32(len(fs.proto.upvals))
	fs.proto.upvals = append(fs.proto.upvals, desc)
	fs.upvals[l] = i

	return i
}
