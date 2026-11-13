package rune

import (
	"fmt"
	"math"
	"rune/pkg/errors"
	"rune/pkg/stdlib"
	"rune/pkg/value"
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
func (p *Interpreter) run(stop int) (value.Value, error) {
	var (
		fr     *frame
		code   []Instr
		consts []value.Value
		regs   []value.Value
		pc     int
		err    error
		ticks  = haltCheckInterval
	)

reload:
	fr = &p.frames[len(p.frames)-1]
	code = fr.cl.proto.code
	consts = fr.cl.proto.consts
	regs = p.stack[fr.base : fr.base+fr.cl.proto.nregs]
	pc = fr.pc

	// Counting ticks here as well measured ~10% slower on loops.
	if p.halted.Load() {
		return value.NilValue, p.unwind(ErrInterrupted, stop)
	}

	for {
		in := code[pc]
		pc++

		switch in.op() {
		case opMove:
			regs[in.a()] = regs[in.b()]

		case opLoadK:
			regs[in.a()] = consts[in.c()]

		case opLoadFresh:
			regs[in.a()] = value.Fresh(consts[in.c()])

		case opGetCell:
			v := (*Cell)(regs[in.b()].Ptr()).v
			if v.IsUndefined() {
				goto undefined
			}

			regs[in.a()] = v

		case opSetCell:
			(*Cell)(regs[in.a()].Ptr()).v = regs[in.b()]

		case opGetUpval:
			v := fr.cl.cells[in.b()].v
			if v.IsUndefined() {
				goto undefined
			}

			regs[in.a()] = v

		case opSetUpval:
			fr.cl.cells[in.b()].v = regs[in.a()]

		case opCheckDefined:
			if regs[in.a()].IsUndefined() {
				goto undefined
			}

		case opSetCellChecked:
			c := (*Cell)(regs[in.a()].Ptr())
			if c.v.IsUndefined() {
				goto undefined
			}

			c.v = regs[in.b()]

		case opSetUpvalChecked:
			c := fr.cl.cells[in.b()]
			if c.v.IsUndefined() {
				goto undefined
			}

			c.v = regs[in.a()]

		case opUndefined:
			goto undefined

		case opAdd:
			if l, r := regs[in.b()], regs[in.c()]; l.IsInt() && r.IsInt() {
				if n, ok := value.AddInt(l.Int(), r.Int()); ok {
					regs[in.a()] = value.NewInt(n)
					continue
				}
			} else if l.IsFloat() && r.IsFloat() {
				regs[in.a()] = value.NewNumber(l.Float() + r.Float())
				continue
			}

			goto slow

		case opSub:
			if l, r := regs[in.b()], regs[in.c()]; l.IsInt() && r.IsInt() {
				if n, ok := value.SubInt(l.Int(), r.Int()); ok {
					regs[in.a()] = value.NewInt(n)
					continue
				}
			} else if l.IsFloat() && r.IsFloat() {
				regs[in.a()] = value.NewNumber(l.Float() - r.Float())
				continue
			}

			goto slow

		case opMul:
			if l, r := regs[in.b()], regs[in.c()]; l.IsInt() && r.IsInt() {
				if n, ok := value.MulInt(l.Int(), r.Int()); ok {
					regs[in.a()] = value.NewInt(n)
					continue
				}
			} else if l.IsFloat() && r.IsFloat() {
				regs[in.a()] = value.NewNumber(l.Float() * r.Float())
				continue
			}

			goto slow

		case opDiv:
			if l, r := regs[in.b()], regs[in.c()]; l.IsNumber() && r.IsNumber() {
				regs[in.a()] = value.NewNumber(l.Num() / r.Num())
				continue
			}

			goto slow

		case opMod:
			if l, r := regs[in.b()], regs[in.c()]; l.IsInt() && r.IsInt() {
				if d := r.Int(); d != 0 {
					regs[in.a()] = value.NewInt(l.Int() % d)
					continue
				}
			} else if l.IsFloat() && r.IsFloat() {
				if m, ok := value.ExactMod(l.Float(), r.Float()); ok {
					regs[in.a()] = value.NewNumber(m)
					continue
				}
			}

			goto slow

		case opAddK:
			if l, k := regs[in.b()], consts[in.c()]; l.IsInt() && k.IsInt() {
				if n, ok := value.AddInt(l.Int(), k.Int()); ok {
					regs[in.a()] = value.NewInt(n)
					continue
				}
			} else if l.IsFloat() && k.IsFloat() {
				regs[in.a()] = value.NewNumber(l.Float() + k.Float())
				continue
			} else if l.IsString() && k.IsString() {
				fr.pc = pc
				dst := &regs[in.a()]
				*dst = value.Concat(l, k)

				goto reload
			}

			goto slow

		case opSubK:
			if l, k := regs[in.b()], consts[in.c()]; l.IsInt() && k.IsInt() {
				if n, ok := value.SubInt(l.Int(), k.Int()); ok {
					regs[in.a()] = value.NewInt(n)
					continue
				}
			} else if l.IsFloat() && k.IsFloat() {
				regs[in.a()] = value.NewNumber(l.Float() - k.Float())
				continue
			}

			goto slow

		case opMulK:
			if l, k := regs[in.b()], consts[in.c()]; l.IsInt() && k.IsInt() {
				if n, ok := value.MulInt(l.Int(), k.Int()); ok {
					regs[in.a()] = value.NewInt(n)
					continue
				}
			} else if l.IsFloat() && k.IsFloat() {
				regs[in.a()] = value.NewNumber(l.Float() * k.Float())
				continue
			}

			goto slow

		case opDivK:
			if l, k := regs[in.b()], consts[in.c()]; l.IsNumber() && k.IsNumber() {
				regs[in.a()] = value.NewNumber(l.Num() / k.Num())
				continue
			}

			goto slow

		case opModK:
			if l, k := regs[in.b()], consts[in.c()]; l.IsInt() && k.IsInt() {
				if d := k.Int(); d != 0 {
					regs[in.a()] = value.NewInt(l.Int() % d)
					continue
				}
			} else if l.IsFloat() && k.IsFloat() {
				if m, ok := value.ExactMod(l.Float(), k.Float()); ok {
					regs[in.a()] = value.NewNumber(m)
					continue
				}
			}

			goto slow

		case opCmp:
			if l, r := regs[in.b()], regs[in.c()]; l.IsInt() && r.IsInt() {
				regs[in.a()] = value.NewBool(compareNumbers(in.x(), l.Int(), r.Int()))
				continue
			} else if l.IsFloat() && r.IsFloat() {
				regs[in.a()] = value.NewBool(compareNumbers(in.x(), l.Float(), r.Float()))
				continue
			}

			goto slow

		case opCmpK:
			if l, k := regs[in.b()], consts[in.c()]; l.IsInt() && k.IsInt() {
				regs[in.a()] = value.NewBool(compareNumbers(in.x(), l.Int(), k.Int()))
				continue
			} else if l.IsFloat() && k.IsFloat() {
				regs[in.a()] = value.NewBool(compareNumbers(in.x(), l.Float(), k.Float()))
				continue
			}

			goto slow

		case opBitwise:
			if l, r := regs[in.b()], regs[in.c()]; l.IsInt() && r.IsInt() {
				if v, ok := value.Bitwise(value.Bit(in.x()), l.Int(), r.Int()); ok {
					regs[in.a()] = v
					continue
				}
			}

			goto slow

		case opBitNot:
			if v := regs[in.b()]; v.IsInt() {
				regs[in.a()] = value.NewInt(^v.Int())
				continue
			}

			goto slow

		case opIs:
			t := regs[in.c()]
			if t.Kind() != value.Type {
				goto slow
			}

			regs[in.a()] = value.NewBool(t.TypeShape().Has(regs[in.b()]))

		case opNeg:
			if v := regs[in.b()]; v.IsFloat() {
				regs[in.a()] = value.NewNumber(-v.Float())
				continue
			} else if v.IsInt() && v.Int() != math.MinInt64 {
				regs[in.a()] = value.NewInt(-v.Int())
				continue
			}

			goto slow

		case opNot:
			regs[in.a()] = value.NewBool(!regs[in.b()].Truthy())

		case opIncr:
			if v := regs[in.b()]; v.IsInt() {
				if n, ok := value.AddInt(v.Int(), 1-2*int64(in.x())); ok {
					regs[in.a()] = value.NewInt(n)
					continue
				}
			} else if v.IsFloat() {
				regs[in.a()] = value.NewNumber(v.Float() + float64(1-2*int(in.x())))
				continue
			}

			goto slow

		case opJmp:
			pc = in.c()
			if ticks--; ticks == 0 {
				goto tick
			}

		case opJmpIf:
			if regs[in.a()].Truthy() == in.y() {
				pc = in.c()
				if ticks--; ticks == 0 {
					goto tick
				}
			}

		case opJmpCmp:
			l, r := regs[in.a()], regs[in.b()]

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
			l, k := regs[in.a()], consts[in.b()]

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
			if !consts[in.b()].TypeShape().Has(regs[in.a()]) {
				pc = in.c()
			}

		case opJmpNil:
			if regs[in.a()].IsNil() == in.y() {
				pc = in.c()
			}

		case opJmpDefined:
			if !regs[in.a()].IsUndefined() {
				pc = in.c()
			}

		case opTestProp:
			t := regs[in.b()]
			if k := t.Kind(); k == value.Struct {
				if v, ok := t.InternedField(consts[in.c()]); ok {
					regs[in.a()] = v
					pc++
				}
			} else if k == value.Object {
				fr.pc = pc
				dst := &regs[in.a()]

				if v, ok := t.Obj().Get(consts[in.c()].Str()); ok {
					*dst = v
					fr.pc++
				}

				goto reload
			}

		case opGetIndex:
			if t, i := regs[in.b()], regs[in.c()]; t.Kind() == value.Array && i.IsInt() {
				if arr, idx := t.Arr(), i.Int(); uint64(idx) < uint64(len(arr)) {
					regs[in.a()] = arr[idx]
					continue
				}
			}

			goto slow

		case opGetProp:
			t := regs[in.b()]
			if k := t.Kind(); k == value.Struct {
				if v, ok := t.InternedField(consts[in.c()]); ok {
					regs[in.a()] = v
					continue
				}
			} else if k == value.Object {
				fr.pc = pc
				dst := &regs[in.a()]

				if v, ok := t.Obj().Get(consts[in.c()].Str()); ok {
					*dst = v
					goto reload
				}
			}

			goto slow

		case opSetProp:
			t := regs[in.a()]
			if k := t.Kind(); k == value.Object {
				fr.pc = pc
				t.Obj().Set(consts[in.b()].Str(), regs[in.c()])

				goto reload
			} else if k != value.Struct || !in.y() {
				goto slow
			}

			if slot := t.InternedSlot(consts[in.b()]); slot >= 0 {
				fr.pc = pc + int(in.x())
				dst := &regs[in.a()]
				*dst = t.With(slot, regs[in.c()])

				goto reload
			}

			goto slow

		case opSetOwned:
			t := regs[in.a()]
			if k := t.Kind(); k == value.Object {
				fr.pc = pc
				t.Obj().Set(consts[in.b()].Str(), regs[in.c()])

				goto reload
			} else if k != value.Struct {
				goto slow
			}

			slot := t.InternedSlot(consts[in.b()])
			if slot < 0 {
				goto slow
			}

			if own := &regs[in.a()+1]; *own != value.True {
				t = t.Clone()
				regs[in.a()] = t
				*own = value.True
			}

			t.Fields()[slot] = regs[in.c()]

		case opCheckProp:
			t := regs[in.a()]
			if k := t.Kind(); k == value.Object {
				continue
			} else if k == value.Struct && in.y() {
				if t.InternedSlot(consts[in.b()]) >= 0 {
					continue
				}
			}

			goto slow

		case opNewCell:
			fr.pc = pc
			dst := &regs[in.a()]
			*dst = value.NewInternal(&Cell{v: regs[in.b()]})

			goto reload

		case opCall:
			fn := regs[in.a()]
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

			switch recv := regs[in.a()+1]; recv.Kind() {
			case value.Struct:
				ms = recv.Shape().Methods
			case value.Enum:
				ms = recv.Variant().Shape.Methods
			case value.Type:
				goto slow
			default:
				ms = p.kinds[recv.Kind()]
			}

			m := value.InternedMethod(ms, consts[in.c()])
			if m == nil || !m.Self || m.Mut && !in.y() {
				goto slow
			}

			fn := p.methods[m.ID]
			if fn.TypeID() != closureID {
				if m.InPlace && in.x() != 0 {
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
			// the jump over its write-back.
			fr.pc = pc
			if m.Mut {
				fr.pc++
			}

			regs[in.a()] = fn

			base := fr.base + in.a() + 1
			if need := base + f.proto.nregs; need > len(p.stack) {
				p.grow(need)
			}

			p.frames = append(p.frames, frame{cl: f, base: base})

			goto reload

		case opReturn, opReturnNil:
			var v value.Value
			if in.op() == opReturn {
				v = regs[in.a()]
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
			s := regs[in.a() : in.a()+6]
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
			c := (*cursor)(regs[in.a()].Ptr())
			if c.next != nil {
				goto slow
			}

			if i := c.i; i < len(c.arr) {
				c.i++
				regs[in.a()+1] = value.NewInt(int64(i))
				regs[in.a()+2] = c.arr[i]
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
