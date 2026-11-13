package rune

import (
	"fmt"
	"math"
	"rune/pkg/errors"
	"rune/pkg/stdlib"
	"rune/pkg/value"
	"unsafe"
)

var closureID = value.TypeIDOf((*Closure)(nil))

// haltCheckInterval is how many jumps run takes between checks for Interrupt.
const haltCheckInterval = 1024

// run executes from the top frame until frames[stop] returns.
//
// Go keeps a variable that lives across a call on the stack, which the
// dispatch loop would pay for on every instruction. So fast paths make no
// calls, and a path that does saves pc in the frame first and then only
// jumps to reload or fail, which rebuild the loop state from the frame.
//
// Registers, constants and code are read through pointers, unchecked: the
// compiler keeps every operand within its frame.
func (p *Interpreter) run(stop int) (value.Value, error) {
	var (
		fr     *frame
		code   unsafe.Pointer
		consts unsafe.Pointer
		regs   unsafe.Pointer
		pc     int
		err    error
		ticks  = haltCheckInterval
	)

reload:
	fr = &p.frames[len(p.frames)-1]
	code = unsafe.Pointer(unsafe.SliceData(fr.cl.proto.code))
	consts = unsafe.Pointer(unsafe.SliceData(fr.cl.proto.consts))
	regs = unsafe.Pointer(at(unsafe.Pointer(unsafe.SliceData(p.stack)), fr.base))
	pc = fr.pc

	// Counting ticks here as well measured ~10% slower on loops.
	if p.halted.Load() {
		return value.NilValue, p.unwind(ErrInterrupted, stop)
	}

	for {
		in := *(*Instr)(unsafe.Add(code, pc<<3))
		pc++

		switch in.op() {
		case opMove:
			*at(regs, in.a()) = *at(regs, in.b())

		case opLoadK:
			*at(regs, in.a()) = *at(consts, in.c())

		case opLoadFresh:
			*at(regs, in.a()) = value.Fresh(*at(consts, in.c()))

		case opGetCell:
			v := (*Cell)(at(regs, in.b()).Ptr()).v
			if v.IsUndefined() {
				goto undefined
			}

			*at(regs, in.a()) = v

		case opSetCell:
			(*Cell)(at(regs, in.a()).Ptr()).v = *at(regs, in.b())

		case opGetUpval:
			v := fr.cl.cells[in.b()].v
			if v.IsUndefined() {
				goto undefined
			}

			*at(regs, in.a()) = v

		case opSetUpval:
			fr.cl.cells[in.b()].v = *at(regs, in.a())

		case opCheckDefined:
			if at(regs, in.a()).IsUndefined() {
				goto undefined
			}

		case opSetCellChecked:
			c := (*Cell)(at(regs, in.a()).Ptr())
			if c.v.IsUndefined() {
				goto undefined
			}

			c.v = *at(regs, in.b())

		case opSetUpvalChecked:
			c := fr.cl.cells[in.b()]
			if c.v.IsUndefined() {
				goto undefined
			}

			c.v = *at(regs, in.a())

		case opUndefined:
			goto undefined

		case opAdd:
			if l, r := *at(regs, in.b()), *at(regs, in.c()); l.IsInt() && r.IsInt() {
				if n, ok := value.AddInt(l.Int(), r.Int()); ok {
					*at(regs, in.a()) = value.NewInt(n)
					continue
				}
			} else if l.IsFloat() && r.IsFloat() {
				*at(regs, in.a()) = value.NewNumber(l.Float() + r.Float())
				continue
			}

			goto slow

		case opSub:
			if l, r := *at(regs, in.b()), *at(regs, in.c()); l.IsInt() && r.IsInt() {
				if n, ok := value.SubInt(l.Int(), r.Int()); ok {
					*at(regs, in.a()) = value.NewInt(n)
					continue
				}
			} else if l.IsFloat() && r.IsFloat() {
				*at(regs, in.a()) = value.NewNumber(l.Float() - r.Float())
				continue
			}

			goto slow

		case opMul:
			if l, r := *at(regs, in.b()), *at(regs, in.c()); l.IsInt() && r.IsInt() {
				if n, ok := value.MulInt(l.Int(), r.Int()); ok {
					*at(regs, in.a()) = value.NewInt(n)
					continue
				}
			} else if l.IsFloat() && r.IsFloat() {
				*at(regs, in.a()) = value.NewNumber(l.Float() * r.Float())
				continue
			}

			goto slow

		case opDiv:
			if l, r := *at(regs, in.b()), *at(regs, in.c()); l.IsNumber() && r.IsNumber() {
				*at(regs, in.a()) = value.NewNumber(l.Num() / r.Num())
				continue
			}

			goto slow

		case opMod:
			if l, r := *at(regs, in.b()), *at(regs, in.c()); l.IsInt() && r.IsInt() {
				if d := r.Int(); d != 0 {
					*at(regs, in.a()) = value.NewInt(l.Int() % d)
					continue
				}
			} else if l.IsFloat() && r.IsFloat() {
				if m, ok := value.ExactMod(l.Float(), r.Float()); ok {
					*at(regs, in.a()) = value.NewNumber(m)
					continue
				}
			}

			goto slow

		case opAddK:
			if l, k := *at(regs, in.b()), *at(consts, in.c()); l.IsInt() && k.IsInt() {
				if n, ok := value.AddInt(l.Int(), k.Int()); ok {
					*at(regs, in.a()) = value.NewInt(n)
					continue
				}
			} else if l.IsFloat() && k.IsFloat() {
				*at(regs, in.a()) = value.NewNumber(l.Float() + k.Float())
				continue
			} else if l.IsString() && k.IsString() {
				fr.pc = pc
				dst := at(regs, in.a())
				*dst = value.Concat(l, k)

				goto reload
			}

			goto slow

		case opSubK:
			if l, k := *at(regs, in.b()), *at(consts, in.c()); l.IsInt() && k.IsInt() {
				if n, ok := value.SubInt(l.Int(), k.Int()); ok {
					*at(regs, in.a()) = value.NewInt(n)
					continue
				}
			} else if l.IsFloat() && k.IsFloat() {
				*at(regs, in.a()) = value.NewNumber(l.Float() - k.Float())
				continue
			}

			goto slow

		case opMulK:
			if l, k := *at(regs, in.b()), *at(consts, in.c()); l.IsInt() && k.IsInt() {
				if n, ok := value.MulInt(l.Int(), k.Int()); ok {
					*at(regs, in.a()) = value.NewInt(n)
					continue
				}
			} else if l.IsFloat() && k.IsFloat() {
				*at(regs, in.a()) = value.NewNumber(l.Float() * k.Float())
				continue
			}

			goto slow

		case opDivK:
			if l, k := *at(regs, in.b()), *at(consts, in.c()); l.IsNumber() && k.IsNumber() {
				*at(regs, in.a()) = value.NewNumber(l.Num() / k.Num())
				continue
			}

			goto slow

		case opModK:
			if l, k := *at(regs, in.b()), *at(consts, in.c()); l.IsInt() && k.IsInt() {
				if d := k.Int(); d != 0 {
					*at(regs, in.a()) = value.NewInt(l.Int() % d)
					continue
				}
			} else if l.IsFloat() && k.IsFloat() {
				if m, ok := value.ExactMod(l.Float(), k.Float()); ok {
					*at(regs, in.a()) = value.NewNumber(m)
					continue
				}
			}

			goto slow

		case opCmp:
			if l, r := *at(regs, in.b()), *at(regs, in.c()); l.IsInt() && r.IsInt() {
				*at(regs, in.a()) = value.NewBool(compareNumbers(in.x(), l.Int(), r.Int()))
				continue
			} else if l.IsFloat() && r.IsFloat() {
				*at(regs, in.a()) = value.NewBool(compareNumbers(in.x(), l.Float(), r.Float()))
				continue
			}

			goto slow

		case opCmpK:
			if l, k := *at(regs, in.b()), *at(consts, in.c()); l.IsInt() && k.IsInt() {
				*at(regs, in.a()) = value.NewBool(compareNumbers(in.x(), l.Int(), k.Int()))
				continue
			} else if l.IsFloat() && k.IsFloat() {
				*at(regs, in.a()) = value.NewBool(compareNumbers(in.x(), l.Float(), k.Float()))
				continue
			}

			goto slow

		case opBitwise:
			if l, r := *at(regs, in.b()), *at(regs, in.c()); l.IsInt() && r.IsInt() {
				if v, ok := value.Bitwise(value.Bit(in.x()), l.Int(), r.Int()); ok {
					*at(regs, in.a()) = v
					continue
				}
			}

			goto slow

		case opBitNot:
			if v := *at(regs, in.b()); v.IsInt() {
				*at(regs, in.a()) = value.NewInt(^v.Int())
				continue
			}

			goto slow

		case opIs:
			t := *at(regs, in.c())
			if t.Kind() != value.Type {
				goto slow
			}

			*at(regs, in.a()) = value.NewBool(t.TypeShape().Has(*at(regs, in.b())))

		case opNeg:
			if v := *at(regs, in.b()); v.IsFloat() {
				*at(regs, in.a()) = value.NewNumber(-v.Float())
				continue
			} else if v.IsInt() && v.Int() != math.MinInt64 {
				*at(regs, in.a()) = value.NewInt(-v.Int())
				continue
			}

			goto slow

		case opNot:
			*at(regs, in.a()) = value.NewBool(!at(regs, in.b()).Truthy())

		case opIncr:
			if v := *at(regs, in.b()); v.IsInt() {
				if n, ok := value.AddInt(v.Int(), 1-2*int64(in.x())); ok {
					*at(regs, in.a()) = value.NewInt(n)
					continue
				}
			} else if v.IsFloat() {
				*at(regs, in.a()) = value.NewNumber(v.Float() + float64(1-2*int(in.x())))
				continue
			}

			goto slow

		case opJmp:
			pc = in.c()
			if ticks--; ticks == 0 {
				goto tick
			}

		case opJmpIf:
			if at(regs, in.a()).Truthy() == in.y() {
				pc = in.c()
				if ticks--; ticks == 0 {
					goto tick
				}
			}

		case opJmpCmp:
			l, r := *at(regs, in.a()), *at(regs, in.b())

			var b bool
			if l.IsInt() && r.IsInt() {
				b = compareNumbers(in.x(), l.Int(), r.Int())
			} else if l.IsFloat() && r.IsFloat() {
				b = compareNumbers(in.x(), l.Float(), r.Float())
			} else {
				goto slow
			}

			if b == in.y() {
				pc = in.c()
				if ticks--; ticks == 0 {
					goto tick
				}
			}

		case opJmpCmpK:
			l, k := *at(regs, in.a()), *at(consts, in.b())

			var b bool
			if l.IsInt() && k.IsInt() {
				b = compareNumbers(in.x(), l.Int(), k.Int())
			} else if l.IsFloat() && k.IsFloat() {
				b = compareNumbers(in.x(), l.Float(), k.Float())
			} else if l.IsString() && k.IsString() {
				fr.pc = pc
				if compareStrings(in.x(), l.Str(), k.Str()) == in.y() {
					fr.pc = in.c()
				}

				goto reload
			} else {
				goto slow
			}

			if b == in.y() {
				pc = in.c()
				if ticks--; ticks == 0 {
					goto tick
				}
			}

		case opJmpNotType:
			if !at(consts, in.b()).TypeShape().Has(*at(regs, in.a())) {
				pc = in.c()
			}

		case opJmpNil:
			if at(regs, in.a()).IsNil() == in.y() {
				pc = in.c()
			}

		case opJmpDefined:
			if !at(regs, in.a()).IsUndefined() {
				pc = in.c()
			}

		case opJmpNotLen:
			if v := at(regs, in.a()); v.Kind() != value.Array || !(len(v.Arr()) == in.b() || in.x() != 0 && len(v.Arr()) > in.b()) {
				pc = in.c()
			}

		case opTestProp:
			t := *at(regs, in.b())
			if k := t.Kind(); k == value.Struct {
				if v, ok := t.InternedField(*at(consts, in.c())); ok {
					*at(regs, in.a()) = v
					pc++
				}
			} else if k == value.Object {
				fr.pc = pc
				dst := at(regs, in.a())

				if v, ok := t.Obj().Get(at(consts, in.c()).Str()); ok {
					*dst = v
					fr.pc++
				}

				goto reload
			}

		case opGetIndex:
			if t, i := *at(regs, in.b()), *at(regs, in.c()); t.Kind() == value.Array && i.IsInt() {
				if arr, idx := t.Arr(), i.Int(); uint64(idx) < uint64(len(arr)) {
					*at(regs, in.a()) = arr[idx]
					continue
				}
			}

			goto slow

		case opGetProp:
			t := *at(regs, in.b())
			if k := t.Kind(); k == value.Struct {
				if v, ok := t.InternedField(*at(consts, in.c())); ok {
					*at(regs, in.a()) = v
					continue
				}
			} else if k == value.Object {
				fr.pc = pc
				dst := at(regs, in.a())

				if v, ok := t.Obj().Get(at(consts, in.c()).Str()); ok {
					*dst = v
					goto reload
				}
			}

			goto slow

		case opSetProp:
			t := *at(regs, in.a())
			if k := t.Kind(); k == value.Object {
				fr.pc = pc
				t.Obj().Set(at(consts, in.b()).Str(), *at(regs, in.c()))

				goto reload
			} else if k != value.Struct || !in.y() {
				goto slow
			}

			if slot := t.InternedSlot(*at(consts, in.b())); slot >= 0 {
				fr.pc = pc + int(in.x())
				dst := at(regs, in.a())
				*dst = t.With(slot, *at(regs, in.c()))

				goto reload
			}

			goto slow

		case opSetOwned:
			t := *at(regs, in.a())
			slot := t.InternedSlot(*at(consts, in.b()))
			if slot < 0 {
				goto slow
			}

			if own := at(regs, in.a()+1); *own != value.True {
				t = t.Clone()
				*at(regs, in.a()) = t
				*own = value.True
			}

			t.Fields()[slot] = *at(regs, in.c())

		case opCheckProp:
			t := *at(regs, in.a())
			if k := t.Kind(); k == value.Object {
				continue
			} else if k == value.Struct && in.y() {
				if t.InternedSlot(*at(consts, in.b())) >= 0 {
					continue
				}
			}

			goto slow

		case opNewCell:
			fr.pc = pc
			dst := at(regs, in.a())
			*dst = value.NewInternal(&Cell{v: *at(regs, in.b())})

			goto reload

		case opCall:
			fn := *at(regs, in.a())
			if fn.Kind() != value.Function || fn.TypeID() != closureID {
				goto slow
			}

			// enter, inlined.
			f := (*Closure)(fn.Ptr())
			if in.b() != f.proto.exact || len(p.frames) > p.maxRecursion {
				goto slow
			}

			fr.pc = pc

			base := fr.base + in.a() + 1
			if need := base + f.proto.nregs; need > len(p.stack) {
				p.grow(need)
			}

			p.frames = append(p.frames, frame{cl: f, base: base})

			goto reload

		case opInvoke:
			var ms []value.Method

			switch recv := *at(regs, in.a()+1); recv.Kind() {
			case value.Struct:
				ms = recv.Shape().Methods
			case value.Enum:
				ms = recv.Variant().Shape.Methods
			case value.Type:
				goto slow
			default:
				ms = p.kinds[recv.Kind()]
			}

			m := value.InternedMethod(ms, *at(consts, in.c()))
			if m == nil || !m.Self || m.Mut && !in.y() {
				goto slow
			}

			fn := p.methods[m.ID]
			if fn.TypeID() != closureID {
				if m.InPlace && in.x()&1 != 0 {
					goto slow
				}

				fr.pc = pc
				if err = p.callNativeMethod(fr.base+in.a(), in.b(), fn.Fn().(*stdlib.Native), fr.cl.proto.toks[pc-1]); err != nil {
					goto fail
				}

				goto reload
			}

			f := (*Closure)(fn.Ptr())
			if in.b()+1 != f.proto.exact || len(p.frames) > p.maxRecursion {
				goto slow
			}

			// enter, inlined as for opCall; a mutating method resumes past
			// the jump over its write-back. An error must not leave the
			// caller a struct half changed, so the method takes the
			// caller's own only if no try is open in the caller.
			fr.pc = pc
			entry := 0

			if m.Mut {
				fr.pc++

				if in.x()&2 != 0 {
					if f.proto.selfReg == 0 {
						fr.pc++
					} else if n := len(p.handlers); *at(regs, in.a()) == value.True && (n == 0 || p.handlers[n-1].frame < len(p.frames)-1) {
						entry = 2
					}
				}
			}

			*at(regs, in.a()) = fn

			base := fr.base + in.a() + 1
			if need := base + f.proto.nregs; need > len(p.stack) {
				p.grow(need)
			}

			if entry != 0 {
				self := base + f.proto.selfReg
				p.stack[self], p.stack[self+1] = p.stack[base], value.True
			}

			p.frames = append(p.frames, frame{cl: f, pc: entry, base: base})

			goto reload

		case opReturn, opReturnNil:
			var v value.Value
			if in.op() == opReturn {
				v = *at(regs, in.a())
			}

			top := len(p.frames) - 1
			p.stack[fr.base-1] = v
			p.frames = p.frames[:top]

			if len(p.handlers) > 0 {
				p.dropHandlers(top)
			}

			if top == stop {
				return v, nil
			}

			goto reload

		case opEndTry:
			p.handlers = p.handlers[:len(p.handlers)-in.b()]

		case opForRangeNext:
			s := (*[6]value.Value)(unsafe.Pointer(at(regs, in.a())))
			i := s[3].Int()

			// Integers inline; floats, the rare case, through at.
			var x value.Value
			if s[0].IsInt() {
				n, ok := intRangeAt(s[0].Int(), s[1].Int(), s[2].Int(), i, in.x() != 0)
				if !ok {
					continue
				}

				x = value.NewInt(n)
			} else if v, ok := (numRange{s[0], s[1], s[2], in.x() != 0}).at(i); ok {
				x = v
			} else {
				continue
			}

			s[3] = value.NewInt(i + 1)
			s[4] = value.NewInt(i)
			s[5] = x
			pc = in.c()
			if ticks--; ticks == 0 {
				goto tick
			}

		case opForIterNext:
			c := (*cursor)(at(regs, in.a()).Ptr())
			if c.next != nil {
				goto slow
			}

			if i := c.i; i < len(c.arr) {
				c.i++
				*at(regs, in.a()+1) = value.NewInt(int64(i))
				*at(regs, in.a()+2) = c.arr[i]
				pc = in.c()
				if ticks--; ticks == 0 {
					goto tick
				}
			} else {
				p.release(c)
			}

		default:
			goto slow
		}

		continue

	undefined:
		fr.pc = pc
		err = undefinedVariable(fr.cl.proto, pc)

		goto fail

	tick:
		ticks = haltCheckInterval
		if !p.halted.Load() {
			continue
		}

		fr.pc = pc
		err = ErrInterrupted

		goto fail

	slow:
		fr.pc = pc
		if err = p.slow(in); err != nil {
			goto fail
		}

		goto reload

	fail:
		if err = p.unwind(err, stop); err != nil {
			return value.NilValue, err
		}

		goto reload
	}
}

// at is the i-th value from p.
func at(p unsafe.Pointer, i int) *value.Value {
	return (*value.Value)(unsafe.Add(p, i<<4))
}

func undefinedVariable(f *Proto, pc int) error {
	tok := f.toks[pc-1]
	return errors.NewRuntimeError(errors.NameError, *tok, fmt.Sprintf("Undefined variable '%s'.", tok.Lexeme))
}

func compareNumbers[T int64 | float64](op uint8, l, r T) bool {
	switch op {
	case cmpLT:
		return l < r
	case cmpLE:
		return l <= r
	case cmpGT:
		return l > r
	case cmpGE:
		return l >= r
	case cmpEQ:
		return l == r
	default:
		return l != r
	}
}

func compareStrings(op uint8, l, r string) bool {
	switch op {
	case cmpLT:
		return l < r
	case cmpLE:
		return l <= r
	case cmpGT:
		return l > r
	case cmpGE:
		return l >= r
	case cmpEQ:
		return l == r
	default:
		return l != r
	}
}
