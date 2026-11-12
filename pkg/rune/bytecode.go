package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/value"
	"strings"
)

type Op uint8

// R is a register of the current frame, K a constant, U an upvalue of
// the running closure. Jump targets are absolute, in C.
// A and B take up to 16 bits, C up to 20.
const (
	opMove     Op = iota // R[A] = R[B]
	opLoadK              // R[A] = K[C]
	opNewCell            // R[A] = cell(R[B])
	opGetCell            // R[A] = cell R[B], which must be defined
	opSetCell            // cell R[A] = R[B]
	opGetUpval           // R[A] = U[B], which must be defined
	opSetUpval           // U[B] = R[A]
	// The checked forms fail on a top-level variable not yet defined.
	opCheckDefined // R[A] must be defined
	opSetCellChecked
	opSetUpvalChecked
	opUndefined // fail on a name neither declared nor built in

	// The arithmetic ops come in the order of value.Arith, each op with
	// a register then with a constant: a fast path apiece, as one op
	// switching on its operator measured slower.
	opAdd // R[A] = R[B] + R[C]
	opSub
	opMul
	opDiv
	opMod
	opAddK // R[A] = R[B] + K[C]
	opSubK
	opMulK
	opDivK
	opModK
	opCmp     // R[A] = R[B] <X> R[C]
	opCmpK    // R[A] = R[B] <X> K[C]
	opIs      // R[A] = R[B] is R[C]
	opNeg     // R[A] = -R[B]
	opNot     // R[A] = !R[B]
	opIncr    // R[A] = R[B] + 1, or - 1 if X
	opBitwise // R[A] = R[B] <X> R[C], X a value.Bit
	opBitNot  // R[A] = ^R[B]

	opJmp        // pc = C
	opJmpIf      // if truthy(R[A]) == Y: pc = C
	opJmpCmp     // if (R[A] <X> R[B]) == Y: pc = C
	opJmpCmpK    // if (R[A] <X> K[B]) == Y: pc = C
	opJmpNotType // if R[A] is not a struct of type K[B]: pc = C
	opJmpNil     // if (R[A] is nil) == Y: pc = C
	opJmpDefined // if R[A] is defined: pc = C

	opArray    // R[A] = [R[B], ..., R[B+C-1]]
	opConcat   // R[A] = R[B] .. R[B+C-1] as printed, appended to R[A] if X
	opObject   // R[A] = {keys[C][i]: R[B+i]}
	opStruct   // R[A] = struct of type K[C] with fields R[B...]
	opGetIndex // R[A] = R[B][R[C]]
	opGetProp  // R[A] = R[B].K[C]
	// The optional forms give nil for a missing element, property or field.
	opGetIndexOpt
	opGetPropOpt
	// The sets take a struct only with Y, for a mut self, copying it into
	// R[A]; with X they resume past the jump that follows, over code
	// writing R[A] back.
	opSetIndex  // R[A][R[B]] = R[C]
	opSetProp   // R[A].K[B] = R[C]
	opCheckProp // R[A] must take property K[B]: an object, or with Y a struct with that field
	// TestProp needs four operands, so it skips over the instruction after
	// it, a jump to the mismatch, instead of taking a target.
	opTestProp // if R[B] is an object or struct with K[C]: R[A] = R[B].K[C], pc++
	opRange    // R[A] = array of R[B]..R[B+1] step R[B+2], inclusive if X
	opSpread   // append the elements of array R[B] to array R[A], or merge R[B] into object R[A]
	opUnpack   // R[A+i] = R[B][i] for i < C, and if X R[A+C] = R[B][C:]

	opClosure   // R[A] = closure of protos[C]
	opCall      // R[A] = R[A](R[A+1], ..., R[A+B])
	opCallArray // R[A] = R[A](the elements of array R[A+1])
	// Invoke calls method K[C] of R[A+1] with it and R[A+2], ..., R[A+1+B].
	// With Y the next instruction jumps over code that writes R[A+1] back
	// to the receiver; a mutating method resumes after that jump.
	opInvoke
	opMethod    // methods[C] = R[A]
	opReturn    // return R[A]
	opReturnNil // return nil
	opPrint     // print R[A]
	opThrow     // throw R[A]
	opRethrow   // throw the caught error in R[A] again
	opNoMatch   // fail: no match arm matches R[A]
	opTry       // catch errors at C, into R[A] if Y, and the error into R[A+1] if X
	opEndTry    // drop B handlers

	// A for-in over a range keeps start, end, step and the step count in
	// R[A..A+3]; it binds the index to R[A+4] and the number to R[A+5].
	opForRange     // check R[A..A+2] and zero R[A+3]
	opForRangeNext // bind the next number and pc = C, unless done (X: inclusive)
	// Any other for-in keeps a cursor in R[A] and binds R[A+1] and R[A+2].
	opForIter     // R[A] = cursor over R[A] (X: binds a key)
	opForIterNext // bind the next pair and pc = C, unless done
)

// Comparisons, in X of opCmp and the conditional jumps.
const (
	cmpLT uint8 = iota
	cmpLE
	cmpGT
	cmpGE
	cmpEQ
	cmpNE
)

// Instr packs op in bits 0-7, x in 8-10, y in 11, a in 12-27, b in
// 28-43 and c in 44-63.
type Instr uint64

const (
	maxAB = 1<<16 - 1
	maxC  = 1<<20 - 1
)

func (i Instr) op() Op   { return Op(i) }
func (i Instr) x() uint8 { return uint8(i>>8) & 7 }
func (i Instr) y() bool  { return i&(1<<11) != 0 }
func (i Instr) a() int   { return int(i>>12) & maxAB }
func (i Instr) b() int   { return int(i>>28) & maxAB }
func (i Instr) c() int   { return int(i >> 44) }

// ins is an instruction being compiled.
type ins struct {
	op      Op
	x, y    uint8
	a, b, c int32
}

// tooLarge is the error of a function past the limits of the encoding.
type tooLarge struct{ name string }

func (e tooLarge) Error() string {
	return fmt.Sprintf("Error: Function %s is too large to compile.", e.name)
}

func (in ins) encode(name string) Instr {
	if in.a < 0 || in.a > maxAB || in.b < 0 || in.b > maxAB || in.c < 0 || in.c > maxC {
		panic(tooLarge{name})
	}

	return Instr(in.op) | Instr(in.x&7)<<8 | Instr(in.y&1)<<11 | Instr(in.a)<<12 | Instr(in.b)<<28 | Instr(in.c)<<44
}

// upvalDesc says where a new closure finds an upvalue: the cell in its
// creator's register index, or its creator's own upvalue index.
type upvalDesc struct {
	fromLocal bool
	index     int32
}

// Proto is a compiled function. toks[i] locates errors of code[i].
type Proto struct {
	name string
	// arity counts the parameters, except a rest one, if rest; the
	// first required of them have no default.
	arity    int
	required int
	rest     bool
	// exact is the argument count the inlined call takes: arity, or -1
	// with a rest parameter, so it checks one field.
	exact  int
	nregs  int
	code   []Instr
	toks   []*ast.Token
	consts []value.Value
	protos []*Proto
	keys   [][]string
	upvals []upvalDesc
}

// accepts reports whether f takes n arguments.
func (f *Proto) accepts(n int) bool {
	return n >= f.required && (n <= f.arity || f.rest)
}

func (f *Proto) displayName() string {
	if f.name == "" {
		return "<anonymous>"
	}

	return f.name
}

type Cell struct {
	v value.Value
}

type Closure struct {
	proto *Proto
	cells []*Cell
}

func (c *Closure) String() string {
	if c.proto.name == "" {
		return "<fn>"
	}

	return fmt.Sprintf("<fn %s>", c.proto.name)
}

// Program holds no run state and can be run any number of times.
type Program struct {
	script *Proto
	// main is the name of the entry point, nil without one.
	main    *ast.Token
	methods methodTable
}

// methodTable holds the methods of the built-in types, by kind, and the
// number of methods in the program.
type methodTable struct {
	kinds [value.NumKinds][]value.Method
	n     int
}

var opNames = [...]string{
	"MOVE", "LOADK", "NEWCELL", "GETCELL", "SETCELL", "GETUPVAL", "SETUPVAL",
	"CHECKDEFINED", "SETCELL!", "SETUPVAL!", "UNDEFINED",
	"ADD", "SUB", "MUL", "DIV", "MOD", "ADDK", "SUBK", "MULK", "DIVK", "MODK", "CMP", "CMPK", "IS", "NEG", "NOT", "INCR", "BITWISE", "BITNOT",
	"JMP", "JMPIF", "JMPCMP", "JMPCMPK", "JMPNOTTYPE", "JMPNIL", "JMPDEFINED",
	"ARRAY", "CONCAT", "OBJECT", "STRUCT", "GETINDEX", "GETPROP", "GETINDEX?", "GETPROP?", "SETINDEX", "SETPROP", "CHECKPROP", "TESTPROP", "RANGE", "SPREAD", "UNPACK",
	"CLOSURE", "CALL", "CALLARRAY", "INVOKE", "METHOD", "RETURN", "RETURNNIL", "PRINT", "THROW", "RETHROW", "NOMATCH", "TRY", "ENDTRY",
	"FORRANGE", "FORRANGENEXT", "FORITER", "FORITERNEXT",
}

var bitNames = [...]string{"AND", "OR", "XOR", "SHL", "SHR"}

// opName names in; ownX says the name already shows its X.
func opName(in Instr) (string, bool) {
	if in.op() == opBitwise {
		return bitNames[in.x()], true
	}

	return opNames[in.op()], false
}

// arithOp is the instruction of op on a register, or a constant if k;
// arithOf is the operator of one.
func arithOp(op value.Arith, k bool) Op {
	if k {
		return opAddK + Op(op)
	}

	return opAdd + Op(op)
}

func arithOf(op Op) value.Arith {
	if op >= opAddK {
		return value.Arith(op - opAddK)
	}

	return value.Arith(op - opAdd)
}

// Disassemble lists the program's code, function by function.
func (prog *Program) Disassemble() string {
	var b strings.Builder

	var walk func(f *Proto)
	walk = func(f *Proto) {
		arity := fmt.Sprint(f.arity)
		if f.required < f.arity {
			arity = fmt.Sprintf("%d-%d", f.required, f.arity)
		}

		if f.rest {
			arity += "+"
		}

		fmt.Fprintf(&b, "%s: arity %s, %d registers, %d upvalues\n", f.displayName(), arity, f.nregs, len(f.upvals))

		for i, in := range f.code {
			name, ownX := opName(in)
			fmt.Fprintf(&b, "%5d  %-13s %d %d %d", i, name, in.a(), in.b(), in.c())

			if !ownX && in.x() != 0 || in.y() {
				fmt.Fprintf(&b, "  x=%d y=%v", in.x(), in.y())
			}

			b.WriteByte('\n')
		}

		for _, p := range f.protos {
			b.WriteByte('\n')
			walk(p)
		}
	}

	walk(prog.script)

	return b.String()
}
