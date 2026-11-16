package rune

import (
	"fmt"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/stdlib"
	"rune/pkg/value"
	"slices"
)

// slow runs in, the instruction before the top frame's pc, for the cases
// the dispatch loop leaves out. It may grow the stack.
// missingProp is what a property read gives for a field a struct does
// not have, or nil for an error.
func missingProp(in Instr) *value.Value {
	switch {
	case in.x()&getMissing != 0:
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
		v, err := arith(tok, value.Arith(in.x()), regs[in.b()], value.NewInt(1))
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

	case opMap:
		var m *value.HashMap
		if in.x() == 0 {
			m = value.NewHashMap(int(in.c()))
			regs[in.a()] = value.NewMap(m)
		} else {
			m = regs[in.a()].HashMap()
		}

		for i := in.b(); i < in.b()+2*in.c(); i += 2 {
			if err := m.Set(regs[i], regs[i+1]); err != nil {
				return keyError(tok, err)
			}
		}

	case opGetIndex:
		v, err := getIndex(tok, regs[in.b()], regs[in.c()])
		if err == nil {
			regs[in.a()] = v
			shareGot(in, regs[in.b()], v)
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
			shareGot(in, regs[in.b()], v)
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

	case opSetIndex, opSetProp:
		t, key := regs[in.a()], setKey(in, regs, consts)
		field := in.op() == opSetProp && t.Kind() == value.Struct && in.y()

		switch {
		case field || in.op() == opSetIndex:
			if t.Shared() {
				t = value.Unshare(t)
				regs[in.a()] = t
				fr.pc += int(in.x())
			}

			if field {
				return setField(tok, t, key, regs[in.c()])
			}

			return setIndex(tok, t, key, regs[in.c()])
		default:
			// SetProp gets here only for a target that takes no property.
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

	case opIn:
		b, err := contains(tok, regs[in.b()], regs[in.c()])
		if err == nil {
			regs[in.a()] = value.NewBool(b != in.y())
		}

		return err

	case opInRange:
		r, err := rangeOf(regs[in.c():], in.x() != 0, tok)
		if err == nil {
			regs[in.a()] = value.NewBool(r.contains(regs[in.b()]) != in.y())
		}

		return err

	case opUnshare:
		regs[in.a()] = value.Unshare(regs[in.a()])
		fr.pc += int(in.x())

	case opSlice:
		v, err := slice(tok, regs[in.b()], regs[in.c()], regs[in.c()+1], in.x())
		if err == nil {
			if v.Kind() == value.Array {
				value.ShareAll(v.Arr())
			}

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
		return p.callAt(fr.base+in.a(), in.b(), tok, fr.site(in))

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

		return p.callAt(at, len(items), tok, callSite{})

	case opSpread:
		return spread(tok, regs[in.a()], regs[in.b()])

	case opUnpack:
		src := regs[in.b()]
		if src.Kind() != value.Array {
			return errors.NewRuntimeError(errors.TypeError, *tok, "Cannot destructure "+helpers.TypeName(src)+" as an array.")
		}

		arr, n, req := src.Arr(), in.c()&maxUnpack, in.c()>>unpackShift
		value.ShareAll(arr)

		rest := in.x() != 0
		if len(arr) < req || !rest && !in.y() && len(arr) > n {
			return unpackError(tok, len(arr), req, n, rest)
		}

		got := copy(regs[in.a():in.a()+n], arr)
		for i := got; i < n; i++ {
			regs[in.a()+i] = value.UndefinedValue
		}

		if rest {
			regs[in.a()+n] = value.NewArray(append([]value.Value(nil), arr[got:]...))
		}

	case opOmit:
		v, err := omit(tok, regs[in.b()], f.keys[in.c()])
		if err == nil {
			regs[in.a()] = v
		}

		return err

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
		v := regs[in.a()]
		if in.x()&forMut != 0 {
			if k := v.Kind(); k != value.Array && k != value.Map {
				return errors.NewRuntimeError(errors.TypeError, *tok, "A 'for mut' loop writes back into an array or map, not "+helpers.TypeName(v)+".")
			}
		}

		// The loop walks the array as it was: a write in the body copies
		// it. A for mut has it to itself, if nothing else can read it.
		if !in.y() {
			value.Share(v)
		}

		c, err := p.cursorOf(v, in.x()&1 != 0, tok)
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
// leaving the result there unless it pushes a frame, to which a closure
// called directly gets what site passes on.
func (p *Interpreter) callAt(at, n int, tok *ast.Token, site callSite) error {
	fn := p.stack[at]
	if isEnum(fn) {
		v, err := variant(tok, fn.TypeShape(), p.stack[at+1:at+1+n])
		if err == nil {
			p.stack[at] = v
		}

		return err
	}

	if !fn.IsFunction() {
		return errors.NewRuntimeError(errors.TypeError, *tok, "Can only call functions.")
	}

	switch f := fn.Fn().(type) {
	case *Closure:
		return p.enter(f, at+1, n, tok, site)
	}

	v, err := p.host.Call(fn.Fn().(*stdlib.Native), p.stack[at+1:at+1+n], tok)
	if err == nil {
		// The call may have grown the stack.
		p.stack[at] = v
	}

	return err
}

// callNativeMethod calls nat on the receiver at stack[at+1] with the n
// arguments after it, leaving the result at stack[at].
func (p *Interpreter) callNativeMethod(at, n int, nat *stdlib.Native, tok *ast.Token) error {
	v, err := p.host.CallMethod(nat, p.stack[at+1:at+2+n], tok)
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

// spread adds the elements of src to the array dst.
func spread(tok *ast.Token, dst, src value.Value) error {
	arr := dst.ArrRef()

	items, ok := spreadItems(*arr, src)
	if !ok {
		return spreadError(tok, src, "an array")
	}

	*arr = items

	return nil
}

// spreadItems appends the elements of src: an array's, a set's or a
// bytes' integers.
func spreadItems(items []value.Value, src value.Value) ([]value.Value, bool) {
	switch src.Kind() {
	case value.Array:
		value.ShareAll(src.Arr())
		return append(items, src.Arr()...), true
	case value.Set:
		src.HashMap().Each(func(k, _ value.Value) bool {
			items = append(items, k)
			return true
		})
	case value.Bytes, value.Buffer:
		data, _ := src.Binary()
		for _, b := range data {
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

// spreadFields copies src, a struct of type shape, into the field
// registers fields.
func spreadFields(tok *ast.Token, shape *value.Shape, fields []value.Value, src value.Value) error {
	switch {
	case src.Kind() == value.Map:
		return stdlib.AssignFields(tok, shape, fields, src.HashMap())
	case src.Kind() == value.Struct && src.Shape() == shape:
		value.ShareAll(src.Fields())
		copy(fields, src.Fields())
	default:
		return spreadError(tok, src, shape.Name)
	}

	return nil
}

// omit is the rest of a record pattern: the fields of the struct src
// but keys, as a map.
func omit(tok *ast.Token, src value.Value, keys []string) (value.Value, error) {
	if src.Kind() != value.Struct {
		return value.Value{}, errors.NewRuntimeError(errors.TypeError, *tok, "Cannot destructure "+helpers.TypeName(src)+" by fields.")
	}

	m := value.NewHashMap(0)
	for k, v := range src.FieldPairs() {
		if !slices.Contains(keys, k) {
			_ = m.Set(value.NewString(k), value.Share(v))
		}
	}

	return value.NewMap(m), nil
}

// unpackError says an array of got elements does not fit a pattern
// taking from req to n of them, or more with rest.
func unpackError(tok *ast.Token, got, req, n int, rest bool) error {
	want := "Expected " + helpers.Plural(req, "element")
	switch {
	case got < req && (rest || req < n):
		want = "Expected at least " + helpers.Plural(req, "element")
	case got > n && req < n:
		want = "Expected at most " + helpers.Plural(n, "element")
	}

	return errors.NewRuntimeError(errors.IndexError, *tok, fmt.Sprintf("%s but got %d.", want, got))
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

// shareGot shares v, which a get read from t: always with Y, and on the
// path of a write with X&getShareIf if t is shared or, with X&getFrozen,
// a struct, as opShareIf would at the write.
func shareGot(in Instr, t, v value.Value) {
	if in.y() || in.x()&getShareIf != 0 && (t.Shared() || in.x()&getFrozen != 0 && t.Kind() == value.Struct) {
		value.Share(v)
	}
}
