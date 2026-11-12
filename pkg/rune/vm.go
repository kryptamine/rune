package rune

import (
	"fmt"
	"math"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
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
			if l, r := regs[in.b()], regs[in.c()]; l.IsNumber() && r.IsNumber() {
				regs[in.a()] = value.NewNumber(l.Num() + r.Num())
				continue
			}

			goto slow

		case opSub:
			if l, r := regs[in.b()], regs[in.c()]; l.IsNumber() && r.IsNumber() {
				regs[in.a()] = value.NewNumber(l.Num() - r.Num())
				continue
			}

			goto slow

		case opMul:
			if l, r := regs[in.b()], regs[in.c()]; l.IsNumber() && r.IsNumber() {
				regs[in.a()] = value.NewNumber(l.Num() * r.Num())
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
			if l, r := regs[in.b()], regs[in.c()]; l.IsNumber() && r.IsNumber() {
				if m, ok := intMod(l.Num(), r.Num()); ok {
					regs[in.a()] = value.NewNumber(m)
					continue
				}
			}

			goto slow

		case opAddK:
			l, k := regs[in.b()], consts[in.c()]
			if l.IsNumber() && k.IsNumber() {
				regs[in.a()] = value.NewNumber(l.Num() + k.Num())
				continue
			}

			if l.IsString() && k.IsString() {
				fr.pc = pc
				dst := &regs[in.a()]
				*dst = value.Concat(l, k)

				goto reload
			}

			goto slow

		case opSubK:
			if l, k := regs[in.b()], consts[in.c()]; l.IsNumber() && k.IsNumber() {
				regs[in.a()] = value.NewNumber(l.Num() - k.Num())
				continue
			}

			goto slow

		case opMulK:
			if l, k := regs[in.b()], consts[in.c()]; l.IsNumber() && k.IsNumber() {
				regs[in.a()] = value.NewNumber(l.Num() * k.Num())
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
			if l, k := regs[in.b()], consts[in.c()]; l.IsNumber() && k.IsNumber() {
				if m, ok := intMod(l.Num(), k.Num()); ok {
					regs[in.a()] = value.NewNumber(m)
					continue
				}
			}

			goto slow

		case opCmp:
			if l, r := regs[in.b()], regs[in.c()]; l.IsNumber() && r.IsNumber() {
				regs[in.a()] = value.NewBool(compareNumbers(in.x(), l.Num(), r.Num()))
				continue
			}

			goto slow

		case opCmpK:
			if l, k := regs[in.b()], consts[in.c()]; l.IsNumber() && k.IsNumber() {
				regs[in.a()] = value.NewBool(compareNumbers(in.x(), l.Num(), k.Num()))
				continue
			}

			goto slow

		case opIs:
			t := regs[in.c()]
			if t.Kind() != value.Type {
				goto slow
			}

			v := regs[in.b()]
			regs[in.a()] = value.NewBool(v.Kind() == value.Struct && v.Shape() == t.TypeShape())

		case opNeg:
			if v := regs[in.b()]; v.IsNumber() {
				regs[in.a()] = value.NewNumber(-v.Num())
				continue
			}

			goto slow

		case opNot:
			regs[in.a()] = value.NewBool(!regs[in.b()].Truthy())

		case opIncr:
			v := regs[in.b()]
			if !v.IsNumber() {
				goto slow
			}

			if in.x() == 0 {
				regs[in.a()] = value.NewNumber(v.Num() + 1)
			} else {
				regs[in.a()] = value.NewNumber(v.Num() - 1)
			}

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
			if !l.IsNumber() || !r.IsNumber() {
				goto slow
			}

			if compareNumbers(in.x(), l.Num(), r.Num()) == in.y() {
				pc = in.c()
				if ticks--; ticks == 0 {
					goto tick
				}
			}

		case opJmpCmpK:
			l, k := regs[in.a()], consts[in.b()]
			if !l.IsNumber() || !k.IsNumber() {
				if !l.IsString() || !k.IsString() {
					goto slow
				}

				fr.pc = pc
				if compareStrings(in.x(), l.Str(), k.Str()) == in.y() {
					fr.pc = in.c()
				}

				goto reload
			}

			if compareNumbers(in.x(), l.Num(), k.Num()) == in.y() {
				pc = in.c()
				if ticks--; ticks == 0 {
					goto tick
				}
			}

		case opJmpNotType:
			if v := regs[in.a()]; v.Kind() != value.Struct || v.Shape() != consts[in.b()].TypeShape() {
				pc = in.c()
			}

		case opJmpNil:
			if regs[in.a()].IsNil() == in.y() {
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

				if v, ok := t.Obj()[consts[in.c()].Str()]; ok {
					*dst = v
					fr.pc++
				}

				goto reload
			}

		case opGetIndex:
			if t, i := regs[in.b()], regs[in.c()]; t.Kind() == value.Array && i.IsNumber() {
				arr := t.Arr()
				if idx := int(i.Num()); float64(idx) == i.Num() && idx >= 0 && idx < len(arr) {
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

				if v, ok := t.Obj()[consts[in.c()].Str()]; ok {
					*dst = v
					goto reload
				}
			}

			goto slow

		case opSetProp:
			t := regs[in.a()]
			if k := t.Kind(); k == value.Object {
				fr.pc = pc
				t.Obj()[consts[in.b()].Str()] = regs[in.c()]

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
			recv := regs[in.a()+1]
			if recv.Kind() != value.Struct {
				goto slow
			}

			m := value.InternedMethod(recv.Shape().Methods, consts[in.c()])
			if m == nil || !m.Self || m.Mut && !in.y() {
				goto slow
			}

			fn := p.methods[m.ID]
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
			r := numRange{start: s[0].Num(), end: s[1].Num(), step: s[2].Num(), inclusive: in.x() != 0}

			i := s[3].Num()
			if x, ok := r.at(i); ok {
				s[3] = value.NewNumber(i + 1)
				s[4] = value.NewNumber(i)
				s[5] = value.NewNumber(x)
				pc = in.c()
				if ticks--; ticks == 0 {
					goto tick
				}
			}

		case opForIterNext:
			c := (*cursor)(regs[in.a()].Ptr())
			if c.mode != arrayCursor {
				goto slow
			}

			if i := c.i; i < len(c.arr) {
				c.i++
				regs[in.a()+1] = value.NewNumber(float64(i))
				regs[in.a()+2] = c.arr[i]
				pc = in.c()
				if ticks--; ticks == 0 {
					goto tick
				}
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

// slow runs in, the instruction before the top frame's pc, for the cases
// the dispatch loop leaves out. It may grow the stack.
func (p *Interpreter) slow(in Instr) error {
	fr := &p.frames[len(p.frames)-1]
	f := fr.cl.proto
	regs := p.stack[fr.base : fr.base+f.nregs]
	consts := f.consts
	tok := f.toks[fr.pc-1]

	switch in.op() {
	case opAdd, opSub, opMul, opDiv, opMod, opCmp:
		return binaryInto(&regs[in.a()], in, tok, regs[in.b()], regs[in.c()])

	case opAddK, opSubK, opMulK, opDivK, opModK, opCmpK:
		return binaryInto(&regs[in.a()], in, tok, regs[in.b()], consts[in.c()])

	case opNeg, opIncr:
		return errors.NewRuntimeError(errors.TypeError, *tok, "Operand must be a number.")

	case opIs:
		return errors.NewRuntimeError(errors.TypeError, *tok, fmt.Sprintf(
			"Right operand of 'is' must be a struct, got %s.", helpers.TypeName(regs[in.c()])))

	case opJmpCmp, opJmpCmpK:
		var r value.Value
		if in.op() == opJmpCmp {
			r = regs[in.b()]
		} else {
			r = consts[in.b()]
		}

		b, err := compareValues(in.x(), regs[in.a()], r, tok)
		if err == nil && b == in.y() {
			fr.pc = in.c()
		}

		return err

	case opArray:
		items := regs[in.b() : in.b()+in.c()]
		if in.x() == 0 {
			regs[in.a()] = value.NewArray(append([]value.Value(nil), items...))
		} else {
			arr := regs[in.a()].ArrRef()
			*arr = append(*arr, items...)
		}

	case opConcat:
		buf := p.buf[:0]
		if in.x() != 0 {
			buf = append(buf, regs[in.a()].Str()...)
		}

		for _, v := range regs[in.b() : in.b()+in.c()] {
			buf = helpers.AppendValue(buf, v)
		}

		p.buf = buf
		regs[in.a()] = value.NewString(string(buf))

	case opObject:
		keys := f.keys[in.c()]

		var obj map[string]value.Value
		if in.x() == 0 {
			obj = make(map[string]value.Value, len(keys))
			regs[in.a()] = value.NewObject(obj)
		} else {
			obj = regs[in.a()].Obj()
		}

		for i, k := range keys {
			obj[k] = regs[in.b()+i]
		}

	case opGetIndex:
		v, err := getIndex(tok, regs[in.b()], regs[in.c()])
		if err == nil {
			regs[in.a()] = v
		}

		return err

	case opGetIndexOpt:
		v, _, err := lookup(tok, regs[in.b()], regs[in.c()])
		if err == nil {
			regs[in.a()] = v
		}

		return err

	case opGetProp, opGetPropOpt:
		v, err := p.property(tok, regs[in.b()], consts[in.c()], in.op() == opGetPropOpt)
		if err == nil {
			regs[in.a()] = v
		}

		return err

	case opStruct:
		v, fields := value.NewStruct(consts[in.c()].TypeShape())
		copy(fields, regs[in.b():])
		regs[in.a()] = v

	case opSetIndex, opSetProp:
		t, key := regs[in.a()], setKey(in, regs, consts)

		switch {
		case t.Kind() == value.Struct && in.y():
			v, err := setField(tok, t, key, regs[in.c()])
			if err == nil {
				regs[in.a()] = v
				fr.pc += int(in.x())
			}

			return err
		case in.op() == opSetIndex:
			return setIndex(tok, t, key, regs[in.c()])
		default:
			// SetProp gets here only for a target that is not an object.
			return propertyTarget(tok, t)
		}

	case opCheckProp:
		if t := regs[in.a()]; t.Kind() == value.Struct && in.y() {
			if t.Shape().Index(tok.Lexeme) < 0 {
				return undefinedField(tok, t.Shape(), tok.Lexeme)
			}

			return nil
		}

		return propertyTarget(tok, regs[in.a()])

	case opRange:
		r, err := rangeOf(regs[in.b():], in.x() != 0, tok)
		if err != nil {
			return err
		}

		v, err := r.array(tok)
		if err == nil {
			regs[in.a()] = v
		}

		return err

	case opClosure:
		proto := f.protos[in.c()]
		cl := &Closure{proto: proto}

		if len(proto.upvals) > 0 {
			cl.cells = make([]*Cell, len(proto.upvals))

			for i, u := range proto.upvals {
				if u.fromLocal {
					cl.cells[i] = (*Cell)(regs[u.index].Ptr())
				} else {
					cl.cells[i] = fr.cl.cells[u.index]
				}
			}
		}

		regs[in.a()] = value.NewFunction(cl)

	case opCall:
		return p.callAt(fr.base+in.a(), in.b(), tok)

	case opInvoke:
		return p.invoke(fr, in, tok)

	case opMethod:
		p.methods[in.c()] = regs[in.a()]

	case opCallArray:
		args := regs[in.a()+1]
		if args.Kind() != value.Array {
			return spreadError(tok, args, "an array")
		}

		at := fr.base + in.a()
		items := args.Arr()
		p.grow(at + 1 + len(items))
		copy(p.stack[at+1:], items)

		return p.callAt(at, len(items), tok)

	case opSpread:
		return spread(tok, regs[in.a()], regs[in.b()])

	case opUnpack:
		src := regs[in.b()]
		if src.Kind() != value.Array {
			return errors.NewRuntimeError(errors.TypeError, *tok, "Cannot destructure "+helpers.TypeName(src)+" as an array.")
		}

		arr, n := src.Arr(), in.c()
		if len(arr) < n {
			return errors.NewRuntimeError(errors.IndexError, *tok, fmt.Sprintf("Expected at least %s but got %d.", helpers.Plural(n, "element"), len(arr)))
		}

		copy(regs[in.a():], arr[:n])

		if in.x() != 0 {
			regs[in.a()+n] = value.NewArray(append([]value.Value(nil), arr[n:]...))
		}

	case opPrint:
		p.buf = append(helpers.AppendValue(p.buf[:0], regs[in.a()]), '\n')
		p.host.Stdio.Out(p.buf)

	case opThrow:
		v := regs[in.a()]
		if v.Kind() != value.Struct {
			return errors.NewRuntimeError(errors.TypeError, *tok, "Can only throw struct instances.")
		}

		return errors.NewThrownError(*tok, v)

	case opNoMatch:
		return errors.NewRuntimeError(errors.ValueError, *tok, "No match arm for "+helpers.Quoted(regs[in.a()])+".")

	case opRethrow:
		return *regs[in.a()].Internal().(*errors.RuntimeError)

	case opTry:
		p.handlers = append(p.handlers, handler{frame: len(p.frames) - 1, pc: in.c(), reg: in.a(), bind: in.y(), keep: in.x() != 0})

	case opForRange:
		if _, err := rangeOf(regs[in.a():], false, tok); err != nil {
			return err
		}

		regs[in.a()+3] = value.NewNumber(0)

	case opForIter:
		c, err := cursorOf(regs[in.a()], in.x() != 0, tok)
		if err == nil {
			regs[in.a()] = value.NewInternal(c)
		}

		return err

	case opForIterNext:
		if k, v, ok := (*cursor)(regs[in.a()].Ptr()).next(); ok {
			regs[in.a()+1] = k
			regs[in.a()+2] = v
			fr.pc = in.c()
		}

	default:
		panic(fmt.Sprintf("vm: unknown op %d", in.op()))
	}

	return nil
}

// callAt calls the function at stack[at] with the n arguments after it,
// leaving the result there unless it pushes a frame.
func (p *Interpreter) callAt(at, n int, tok *ast.Token) error {
	fn := p.stack[at]
	if fn.Kind() == value.Type {
		v, err := convert(tok, fn.TypeShape(), p.stack[at+1:at+1+n])
		if err == nil {
			p.stack[at] = v
		}

		return err
	}

	if fn.Kind() != value.Function {
		return errors.NewRuntimeError(errors.TypeError, *tok, "Can only call functions.")
	}

	switch f := fn.Fn().(type) {
	case *Closure:
		return p.enter(f, at+1, n, tok)
	case *BoundMethod:
		if !f.fn.proto.accepts(n + 1) {
			return argumentError(f.fn.proto, n+1, 1, tok)
		}

		p.grow(at + 2 + n)
		copy(p.stack[at+2:], p.stack[at+1:at+1+n])
		p.stack[at], p.stack[at+1] = value.NewFunction(f.fn), f.recv

		return p.enter(f.fn, at+1, n+1, tok)
	}

	v, err := p.host.Call(fn.Fn().(*stdlib.Native), p, p.stack[at+1:at+1+n], tok)
	if err == nil {
		// The call may have grown the stack.
		p.stack[at] = v
	}

	return err
}

// setKey is the key a set writes: K[B] for a property, R[B] for an element.
func setKey(in Instr, regs, consts []value.Value) value.Value {
	if in.op() == opSetProp {
		return consts[in.b()]
	}

	return regs[in.b()]
}

// spread adds the elements of src to the array dst, or the properties or
// fields of src to the object dst.
func spread(tok *ast.Token, dst, src value.Value) error {
	if dst.Kind() == value.Array {
		if src.Kind() != value.Array {
			return spreadError(tok, src, "an array")
		}

		arr := dst.ArrRef()
		*arr = append(*arr, src.Arr()...)

		return nil
	}

	obj := dst.Obj()

	switch src.Kind() {
	case value.Object:
		for k, v := range src.Obj() {
			obj[k] = v
		}
	case value.Struct:
		for i, f := range src.Shape().Fields {
			obj[f] = src.Fields()[i]
		}
	default:
		return spreadError(tok, src, "an object")
	}

	return nil
}

func spreadError(tok *ast.Token, v value.Value, into string) error {
	return errors.NewRuntimeError(errors.TypeError, *tok, "Cannot spread "+helpers.TypeName(v)+" into "+into+".")
}

// binaryInto stores the result of in's operator in dst unless it fails.
func binaryInto(dst *value.Value, in Instr, tok *ast.Token, l, r value.Value) error {
	if in.op() == opCmp || in.op() == opCmpK {
		b, err := compareValues(in.x(), l, r, tok)
		if err == nil {
			*dst = value.NewBool(b)
		}

		return err
	}

	v, err := binaryOp(tok, l, r)
	if err == nil {
		*dst = v
	}

	return err
}

// intMod is mod for integers that floats hold exactly.
func intMod(l, r float64) (float64, bool) {
	if l <= -maxExactInt || l >= maxExactInt || r <= -maxExactInt || r >= maxExactInt || r == 0 {
		return 0, false
	}

	li, ri := int64(l), int64(r)
	if float64(li) != l || float64(ri) != r {
		return 0, false
	}

	return math.Copysign(float64(li%ri), l), true
}

func undefinedVariable(f *Proto, pc int) error {
	tok := f.toks[pc-1]
	return errors.NewRuntimeError(errors.NameError, *tok, fmt.Sprintf("Undefined variable '%s'.", tok.Lexeme))
}

func compareNumbers(op uint8, l, r float64) bool {
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

func compareValues(op uint8, l, r value.Value, tok *ast.Token) (bool, error) {
	if l.IsNumber() && r.IsNumber() {
		return compareNumbers(op, l.Num(), r.Num()), nil
	}

	if l.IsString() && r.IsString() {
		return compareStrings(op, l.Str(), r.Str()), nil
	}

	v, err := binaryOp(tok, l, r)

	return v.Bool(), err
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
