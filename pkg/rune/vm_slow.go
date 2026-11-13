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
// missingProp is what a property read gives for one an object or struct
// does not have, or nil for an error.
func missingProp(in Instr) *value.Value {
	switch {
	case in.x() != 0:
		return &value.UndefinedValue
	case in.op() == opGetPropOpt:
		return &value.NilValue
	}

	return nil
}

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
			"Right operand of 'is' must be a type, got %s.", helpers.TypeName(regs[in.c()])))

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

		var obj *value.Obj
		if in.x() == 0 {
			obj = value.NewObj(len(keys))
			regs[in.a()] = value.NewObject(obj)
		} else {
			obj = regs[in.a()].Obj()
		}

		for i, k := range keys {
			obj.Set(k, regs[in.b()+i])
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
		v, err := p.property(tok, regs[in.b()], consts[in.c()], missingProp(in))
		if err == nil {
			regs[in.a()] = v
		}

		return err

	case opStruct:
		v, fields := value.NewStruct(consts[in.c()].TypeShape())
		copy(fields, regs[in.b():])
		regs[in.a()] = v

	case opUndef:
		for i := range in.b() {
			regs[in.a()+i] = value.UndefinedValue
		}

	case opStructSpread:
		return spreadFields(tok, consts[in.c()].TypeShape(), regs[in.a():], regs[in.b()])

	case opStructFill:
		shape := consts[in.c()].TypeShape()
		v, fields := value.NewStruct(shape)
		copy(fields, regs[in.b():])

		if missing, ok := shape.Fill(fields); !ok {
			return errors.NewRuntimeError(errors.KeyError, *tok, errors.MissingField(shape.Name, missing))
		}

		regs[in.a()] = v

	case opSetIndex, opSetProp, opSetOwned:
		t, key := regs[in.a()], setKey(in, regs, consts)

		switch {
		case t.Kind() == value.Struct && (in.y() || in.op() == opSetOwned):
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
		items, ok := spreadArgs(args)
		if !ok {
			return spreadError(tok, args, "an array")
		}

		at := fr.base + in.a()
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

		arr, n, req := src.Arr(), in.c()&maxUnpack, in.c()>>unpackShift

		if len(arr) < req {
			return errors.NewRuntimeError(errors.IndexError, *tok, fmt.Sprintf("Expected at least %s but got %d.", helpers.Plural(req, "element"), len(arr)))
		}

		got := copy(regs[in.a():in.a()+n], arr)
		for i := got; i < n; i++ {
			regs[in.a()+i] = value.UndefinedValue
		}

		if in.x() != 0 {
			regs[in.a()+n] = value.NewArray(append([]value.Value(nil), arr[got:]...))
		}

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
		c, err := p.cursorOf(regs[in.a()], in.x() != 0, tok)
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
		p.stack[at+1] = f.recv

		return p.callNativeMethod(at, n, f.fn, tok)
	}

	v, err := p.host.Call(fn.Fn().(*stdlib.Native), p, p.stack[at+1:at+1+n], tok)
	if err == nil {
		// The call may have grown the stack.
		p.stack[at] = v
	}

	return err
}

// callNativeMethod calls nat on the receiver at stack[at+1] with the n
// arguments after it, leaving the result at stack[at].
func (p *Interpreter) callNativeMethod(at, n int, nat *stdlib.Native, tok *ast.Token) error {
	v, err := p.host.CallMethod(nat, p, p.stack[at+1:at+2+n], tok)
	if err == nil {
		p.stack[at] = v
	}

	return err
}

// setKey is the key a set writes: K[B] for a property, R[B] for an element.
func setKey(in Instr, regs, consts []value.Value) value.Value {
	if in.op() != opSetIndex {
		return consts[in.b()]
	}

	return regs[in.b()]
}

// spread adds the elements of src to the array dst, or the properties or
// fields of src to the object dst.
func spread(tok *ast.Token, dst, src value.Value) error {
	if dst.Kind() == value.Array {
		arr := dst.ArrRef()

		items, ok := spreadItems(*arr, src)
		if !ok {
			return spreadError(tok, src, "an array")
		}

		*arr = items

		return nil
	}

	obj := dst.Obj()

	switch src.Kind() {
	case value.Object:
		for k, v := range src.Obj().All() {
			obj.Set(k, v)
		}
	case value.Struct:
		for k, v := range src.FieldPairs() {
			obj.Set(k, v)
		}
	default:
		return spreadError(tok, src, "an object")
	}

	return nil
}

// spreadItems appends the elements of src: an array's, a set's, a map's
// [key, value] pairs or a bytes' integers.
func spreadItems(items []value.Value, src value.Value) ([]value.Value, bool) {
	switch src.Kind() {
	case value.Array:
		return append(items, src.Arr()...), true
	case value.Set:
		src.HashMap().Each(func(k, _ value.Value) bool {
			items = append(items, k)
			return true
		})
	case value.Map:
		src.HashMap().Each(func(k, v value.Value) bool {
			items = append(items, value.NewArray([]value.Value{k, v}))
			return true
		})
	case value.Bytes:
		for _, b := range src.Bytes() {
			items = append(items, value.NewInt(int64(b)))
		}
	default:
		return items, false
	}

	return items, true
}

// spreadArgs is spreadItems for a call, which takes an array's own items.
func spreadArgs(src value.Value) ([]value.Value, bool) {
	if src.Kind() == value.Array {
		return src.Arr(), true
	}

	return spreadItems(nil, src)
}

// spreadFields copies src, a struct of type shape or an object of some of
// its fields, into the field registers fields.
func spreadFields(tok *ast.Token, shape *value.Shape, fields []value.Value, src value.Value) error {
	switch {
	case src.Kind() == value.Struct && src.Shape() == shape:
		copy(fields, src.Fields())
	case src.Kind() == value.Object:
		if extra, ok := shape.Assign(fields, src.Obj()); !ok {
			return errors.NewRuntimeError(errors.KeyError, *tok, errors.NoField(shape.Name, extra))
		}
	default:
		return spreadError(tok, src, shape.Name)
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
