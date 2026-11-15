package rune

import (
	"fmt"
	"iter"
	"maps"
	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
)

// The fixes errors suggest: a name close to a mistyped one, or what Rune
// calls what another language does.

// foreignKeywords maps keywords of other languages to the fix in Rune.
var foreignKeywords = map[string]string{
	"var":      "declare a variable with 'let', or 'let mut' to change it later",
	"val":      "declare a variable with 'let', or 'let mut' to change it later",
	"function": "declare a function with 'fun'",
	"func":     "declare a function with 'fun'",
	"fn":       "declare a function with 'fun'",
	"def":      "declare a function with 'fun'",
	"elif":     "write 'else if'",
	"elsif":    "write 'else if'",
	"elseif":   "write 'else if'",
	"class":    "declare a type with 'struct', and its methods in an 'impl' block",
}

// foreignNames maps names other languages give values to Rune's.
var foreignNames = map[string]string{
	"null":      "nil",
	"None":      "nil",
	"undefined": "nil",
	"nullptr":   "nil",
	"True":      "true",
	"False":     "false",
	"this":      "self",
}

// foreignMethods maps what other languages name a method to the one of
// Rune's that does its job.
var foreignMethods = map[string]string{
	"length":      "len",
	"size":        "len",
	"toUpperCase": "upper",
	"toLowerCase": "lower",
	"strip":       "trim",
	"lstrip":      "trimStart",
	"rstrip":      "trimEnd",
	"trimLeft":    "trimStart",
	"trimRight":   "trimEnd",
	"add":         "push",
}

// foreignTests are what other languages name a membership test, which in
// Rune is the in operator, and containers the kinds it tests.
var (
	foreignTests = []string{"includes", "contains", "has", "hasKey", "containsKey"}
	containers   = []value.Kind{value.Array, value.String, value.Object, value.Map, value.Set, value.Bytes, value.Buffer}
)

// foreignKeyword is the error of a statement that starts with a keyword
// of another language, which parsed as a name; others keep err.
func foreignKeyword(expr ast.Expr, err error) error {
	if call, ok := expr.(*ast.CallExpr); ok {
		expr = call.Callee
	}

	v, ok := expr.(*ast.VarExpr)
	if !ok || v.Module != nil {
		return err
	}

	fix, ok := foreignKeywords[v.Name.Lexeme]
	if !ok {
		return err
	}

	return errors.WithHint(errors.NewSyntaxError(v.Name, "'"+v.Name.Lexeme+"' is not a keyword of Rune."), fix)
}

// unknownName is the fix for a name that names no variable here.
func (p *Resolver) unknownName(name string) string {
	if fix, ok := foreignNames[name]; ok {
		return "Rune writes it '" + fix + "'"
	}

	seqs := []iter.Seq[string]{maps.Keys(p.mod.topLevel), maps.Keys(builtinValues())}
	for _, s := range p.scopes {
		seqs = append(seqs, maps.Keys(s.vars))
	}

	return errors.DidYouMean(name, seqs...)
}

// immutable reports msg about the immutable variable name, with the fix
// when its declaration can make it mutable.
func (p *Resolver) immutable(name ast.Token, msg string) {
	err := errors.NewSyntaxError(name, msg)

	v := p.lookupScopes(name.Lexeme)
	if v == nil {
		v = p.mod.globals[name.Lexeme]
	}

	switch {
	case v == nil || v.fixed:
	case p.constNamed(name.Lexeme) != nil:
		err = errors.WithHint(err, fmt.Sprintf("declare it with 'let mut' instead of 'const' on line %d", v.name.Line))
	default:
		err = errors.WithHint(err, fmt.Sprintf("declare it as 'mut %s' on line %d", name.Lexeme, v.name.Line))
	}

	p.errors = append(p.errors, err)
}

// noField reports the field key names, which shape lacks.
func (p *Resolver) noField(key ast.Token, shape *value.Shape) {
	err := errors.NewSyntaxError(key, errors.NoField(shape.Name, key.Lexeme))
	p.errors = append(p.errors, errors.WithHint(err, errors.DidYouMean(key.Lexeme, slices.Values(shape.Fields))))
}

// memberHint is the fix for a member name of t that none of its methods
// ms is: the method of Rune's that does the job it names elsewhere, or
// one close to it.
func memberHint(t value.Value, name string, ms []value.Method) string {
	if m, ok := foreignMethods[name]; ok && value.FindMethod(ms, m) != nil {
		return "use '" + m + "' instead"
	}

	if slices.Contains(foreignTests, name) && slices.Contains(containers, t.Kind()) {
		return "test membership with the in operator: 'x in xs'"
	}

	return errors.DidYouMean(name, methodNames(ms))
}

func methodNames(ms []value.Method) iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, m := range ms {
			if !yield(m.Name) {
				return
			}
		}
	}
}

// operandHint is the fix for l op r on operands it does not take.
func operandHint(op value.Arith, l, r value.Value) string {
	switch {
	case op == value.OpAdd && l.IsString() != r.IsString():
		return "to put a value in a string, interpolate it: \"text {x}\""
	case op == value.OpMul && repeatable(l) && r.IsInt():
		return "to repeat a " + helpers.TypeName(l) + ", call its repeat method: x.repeat(n)"
	case op == value.OpMul && l.IsInt() && repeatable(r):
		return "to repeat a " + helpers.TypeName(r) + ", call its repeat method: x.repeat(n)"
	}

	return ""
}

// repeatable is a value with a repeat method.
func repeatable(v value.Value) bool { return v.IsString() || v.Kind() == value.Array }
