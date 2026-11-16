package stdlib

import (
	"fmt"
	"math"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"strings"
)

// structArg is the fields of argument i, which must be a struct of shape.
func (c *call) structArg(i int, shape *value.Shape) ([]value.Value, bool) {
	if v := c.args[i]; v.Kind() == value.Struct && v.Shape() == shape {
		return v.Fields(), true
	}

	c.typeError(i, shape.Name)

	return nil, false
}

// stringEntries is the entries of m, whose keys and values must be
// strings; what names them in an error.
func (c *call) stringEntries(m *value.HashMap, what string) [][2]string {
	out := make([][2]string, 0, m.Len())

	m.Each(func(k, v value.Value) bool {
		switch {
		case !k.IsString():
			c.fail(errors.TypeError, "expects %s names to be strings, got %s.", what, helpers.Quoted(k))
		case !v.IsString():
			c.fail(errors.TypeError, "expects %s '%s' to be a string, got %s.", what, k.Str(), helpers.TypeName(v))
		default:
			out = append(out, [2]string{k.Str(), v.Str()})
			return true
		}

		return false
	})

	return out
}

// option returns field i of an options struct of shape if it is set; it must be of kind.
func (c *call) option(shape *value.Shape, fields []value.Value, i int, kind value.Kind, want string) (value.Value, bool) {
	v := fields[i]
	if v.IsNil() || !c.ok() {
		return v, false
	}

	if v.Kind() != kind {
		c.fail(errors.TypeError, "expects %s.%s to be %s, got %s.", shape.Name, shape.Fields[i], want, helpers.TypeName(v))
		return v, false
	}

	return v, true
}

// Variant is Enum(v): v itself if a variant of it, or the variant at
// ordinal v, or named v.
func Variant(token *ast.Token, shape *value.Shape, v value.Value) (value.Value, error) {
	switch {
	case shape.Has(v):
		return v, nil
	case shape.HasCases():
		return value.NilValue, errors.NewRuntimeError(errors.TypeError, *token, NotOrdinal(shape))
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

// NotOrdinal says an enum with cases has no ordinals.
func NotOrdinal(enum *value.Shape) string {
	return enum.Name + " has variants with fields; values(), " + enum.Name + "(n) and int() are for enums without."
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
		switch v := c.args[0]; {
		case c.has(1):
			return intInBase(c)
		case v.Kind() == value.Enum:
			return ordinal(c, v.Variant())
		case v.IsNumber():
			return truncate(c, v)
		}

		return c.typeError(0, "a number")
	}},
}

// ordinal is int(v) of a variant: its number.
func ordinal(c *call, v *value.Variant) value.Value {
	if v.Shape.HasCases() {
		c.err = errors.NewRuntimeError(errors.TypeError, *c.token, NotOrdinal(v.Shape))
		return value.NilValue
	}

	return value.NewInt(int64(v.Ordinal))
}

// truncate is int(v) of a number: v toward zero.
func truncate(c *call, v value.Value) value.Value {
	if v.IsInt() {
		return v
	}

	n, ok := value.FloatToInt(math.Trunc(v.Float()))
	if !ok {
		return c.fail(errors.ValueError, "cannot convert %s to an integer.", helpers.Stringify(v))
	}

	return value.NewInt(n)
}

// intInBase is int(s, base): the digits of s in base.
func intInBase(c *call) value.Value {
	s, base := c.str(0), c.int(1)
	switch {
	case !c.ok():
		return value.NilValue
	case base < 2 || base > 36:
		return c.fail(errors.ValueError, "expects a base from 2 to 36, got %d.", base)
	}

	n, ok := helpers.ParseInt(strings.TrimSpace(s), base)
	if !ok {
		return c.fail(errors.ValueError, "cannot parse %s as an integer in base %d.", helpers.Quoted(c.args[0]), base)
	}

	return value.NewInt(n)
}
