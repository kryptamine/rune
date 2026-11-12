package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/stdlib"
	"rune/pkg/value"
)

// slow runs in, the instruction before the top frame's pc, for the cases
// the dispatch loop leaves out. It may grow the stack.
func (p *Interpreter) slow(in Instr) error {
	fr := &p.frames[len(p.frames)-1]
	f := fr.cl.proto
	regs := p.stack[fr.base : fr.base+f.nregs]
	consts := f.consts
	tok := f.toks[fr.pc-1]

	switch in.op() {
	case opAdd, opSub, opMul, opDiv, opMod, opCmp, opBitwise:
		return binaryInto(&regs[in.a()], in, tok, regs[in.b()], regs[in.c()])

	case opAddK, opSubK, opMulK, opDivK, opModK, opCmpK:
		return binaryInto(&regs[in.a()], in, tok, regs[in.b()], consts[in.c()])

	case opNeg:
		v, err := negate(tok, regs[in.b()])
		if err == nil {
			regs[in.a()] = v
		}

		return err

	case opIncr:
		v, err := increment(tok, regs[in.b()], int64(1-2*int(in.x())))
		if err == nil {
			regs[in.a()] = v
		}

		return err

	case opBitNot:
		v, err := bitNot(tok, regs[in.b()])
		if err == nil {
			regs[in.a()] = v
		}

		return err

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

		regs[in.a()+3] = value.NewInt(0)

	case opForIter:
		c, err := cursorOf(regs[in.a()], in.x() != 0, tok)
		if err == nil {
			regs[in.a()] = value.NewInternal(c)
		}

		return err

	case opForIterNext:
		k, v, ok, err := (*cursor)(regs[in.a()].Ptr()).advance()
		if ok {
			regs[in.a()+1] = k
			regs[in.a()+2] = v
			fr.pc = in.c()
		}

		return err

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
	case *BoundNative:
		p.grow(at + 2 + n)
		copy(p.stack[at+2:], p.stack[at+1:at+1+n])
		p.stack[at], p.stack[at+1] = value.NewFunction(f.fn), f.recv

		return p.callAt(at, n+1, tok)
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
		arr := dst.ArrRef()

		switch src.Kind() {
		case value.Array:
			*arr = append(*arr, src.Arr()...)
		case value.Set:
			src.HashMap().Each(func(k, _ value.Value) bool {
				*arr = append(*arr, k)
				return true
			})
		default:
			return spreadError(tok, src, "an array")
		}

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
	var (
		v   value.Value
		err error
	)

	switch in.op() {
	case opCmp, opCmpK:
		var b bool
		b, err = compareValues(in.x(), l, r, tok)
		v = value.NewBool(b)
	case opBitwise:
		v, err = bitwise(tok, value.Bit(in.x()), l, r)
	default:
		v, err = arith(tok, arithOf(in.op()), l, r)
	}

	if err == nil {
		*dst = v
	}

	return err
}

func compareValues(op uint8, l, r value.Value, tok *ast.Token) (bool, error) {
	if l.IsNumber() && r.IsNumber() {
		c, ok := value.CompareNum(l, r)
		if !ok {
			return op == cmpNE, nil
		}

		return compareNumbers(op, int64(c), 0), nil
	}

	if l.IsString() && r.IsString() {
		return compareStrings(op, l.Str(), r.Str()), nil
	}

	v, err := binaryOp(tok, l, r)

	return v.Bool(), err
}
