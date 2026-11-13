package stdlib

import (
	"fmt"
	"math"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"strconv"
	"strings"
)

// Convert is Shape(v): v itself if already of that shape, or a struct
// built from an object holding every required field and no other.
func Convert(token *ast.Token, shape *value.Shape, v value.Value) (value.Value, error) {
	if shape.Enum {
		return variantOf(token, shape, v)
	}

	switch v.Kind() {
	case value.Struct:
		if v.Shape() == shape {
			return v, nil
		}
	case value.Object:
		return structOf(token, shape, v.Obj())
	}

	return value.NilValue, errors.NewRuntimeError(errors.TypeError, *token, fmt.Sprintf(
		"Cannot convert %s to %s.", helpers.TypeName(v), shape.Name))
}

// variantOf is Enum(v): v itself if a variant of it, or the variant at
// ordinal v, or named v.
func variantOf(token *ast.Token, shape *value.Shape, v value.Value) (value.Value, error) {
	switch {
	case v.Kind() == value.Enum && v.Variant().Shape == shape:
		return v, nil
	case v.IsInt():
		if i := v.Int(); i >= 0 && i < int64(len(shape.Variants)) {
			return shape.Variants[i], nil
		}
	case v.IsString():
		if x, ok := shape.Variant(v.Str()); ok {
			return x, nil
		}
	default:
		return value.NilValue, errors.NewRuntimeError(errors.TypeError, *token, fmt.Sprintf(
			"Cannot convert %s to %s.", helpers.TypeName(v), shape.Name))
	}

	return value.NilValue, errors.NewRuntimeError(errors.ValueError, *token, fmt.Sprintf(
		"%s has no variant %s.", shape.Name, helpers.Quoted(v)))
}

func structOf(token *ast.Token, shape *value.Shape, obj *value.Obj) (value.Value, error) {
	v, fields := value.NewStruct(shape)
	for i := range fields {
		fields[i] = value.UndefinedValue
	}

	extra, known := shape.Assign(fields, obj)

	if missing, ok := shape.Fill(fields); !ok {
		return value.NilValue, errors.NewRuntimeError(errors.KeyError, *token, errors.MissingField(shape.Name, missing))
	}

	if !known {
		return value.NilValue, errors.NewRuntimeError(errors.KeyError, *token, errors.NoField(shape.Name, extra))
	}

	return v, nil
}

// toString is string(v): v as print shows it.
var toString = &Native{"string", 1, 1, func(c *call) value.Value {
	if v := c.args[0]; v.IsString() {
		return v
	}

	return value.NewString(string(helpers.AppendValue(nil, c.args[0])))
}}

// conversionNatives are global, like the struct types, which convert too.
var conversionNatives = []*Native{
	{"number", 1, 1, func(c *call) value.Value {
		switch v := c.args[0]; {
		case v.IsNumber():
			return v
		case v.IsString():
			if n, ok := helpers.ParseNumber(strings.TrimSpace(v.Str())); ok {
				return n
			}

			return c.fail(errors.ValueError, "cannot parse %s as a number.", helpers.Quoted(v))
		}

		return c.typeError(0, "a number or a string")
	}},
	{"int", 1, 2, func(c *call) value.Value {
		if v := c.args[0]; v.Kind() == value.Enum && !c.has(1) {
			return value.NewInt(int64(v.Variant().Ordinal))
		}

		if v := c.args[0]; v.IsNumber() && !c.has(1) {
			if v.IsInt() {
				return v
			}

			n, ok := value.FloatToInt(math.Trunc(v.Float()))
			if !ok {
				return c.fail(errors.ValueError, "cannot convert %s to an integer.", helpers.Stringify(v))
			}

			return value.NewInt(n)
		}

		base := 10
		if c.has(1) {
			base = c.int(1)
		}

		s := c.str(0)
		if !c.ok() {
			return value.NilValue
		}

		if base < 2 || base > 36 {
			return c.fail(errors.ValueError, "expects a base from 2 to 36, got %d.", base)
		}

		n, err := strconv.ParseInt(strings.TrimSpace(s), base, 64)
		if err != nil {
			return c.fail(errors.ValueError, "cannot parse %s as an integer in base %d.", helpers.Quoted(c.args[0]), base)
		}

		return value.NewInt(n)
	}},
}
