package stdlib

import (
	"fmt"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"strings"
)

// format is printf: %[flags][width][.precision]verb, flags from "-+ 0#".
// %v and %s show a value as print does, %q as it shows inside an array;
// %d %x %X %o %b take integers, %f %e %E %g %G numbers; %% is a percent sign.
func format(c *call, f string, args []value.Value) value.Value {
	var b strings.Builder

	next := 0

	for i := 0; i < len(f); i++ {
		if f[i] != '%' {
			b.WriteByte(f[i])
			continue
		}

		start := i

		i++
		for i < len(f) && strings.IndexByte("-+ 0#", f[i]) >= 0 {
			i++
		}

		for i < len(f) && (f[i] >= '0' && f[i] <= '9' || f[i] == '.') {
			i++
		}

		if i == len(f) {
			return c.fail(errors.ValueError, "incomplete directive '%s' at %d.", f[start:], start)
		}

		directive := f[start : i+1]
		verb := f[i]

		if verb == '%' {
			if len(directive) > 2 {
				return c.fail(errors.ValueError, "bad directive '%s' at %d.", directive, start)
			}

			b.WriteByte('%')

			continue
		}

		if !validDirective(directive) {
			return c.fail(errors.ValueError, "bad directive '%s' at %d.", directive, start)
		}

		if next >= len(args) {
			return c.fail(errors.ArgumentError, "'%s' at %d has no argument.", directive, start)
		}

		arg, ok := formatArg(c, verb, next+1)
		if !ok {
			return value.NilValue
		}

		next++

		fmt.Fprintf(&b, directive[:len(directive)-1]+goVerb(verb), arg)
	}

	if next < len(args) {
		return c.fail(errors.ArgumentError, "got %d arguments for %d directives.", len(args), next)
	}

	return value.NewString(b.String())
}

// validDirective checks width and precision are plain numbers of sane size.
func validDirective(d string) bool {
	body := strings.TrimLeft(d[1:len(d)-1], "-+ 0#")
	width, precision, _ := strings.Cut(body, ".")

	return len(width) <= 4 && len(precision) <= 3 && !strings.Contains(precision, ".")
}

// goVerb is the fmt verb for verb, given the string formatArg makes of
// a value for %v and %q.
func goVerb(verb byte) string {
	if verb == 'v' || verb == 'q' {
		return "s"
	}

	return string(verb)
}

// formatArg converts c.args[i] for verb.
func formatArg(c *call, verb byte, i int) (any, bool) {
	v := c.args[i]

	switch verb {
	case 'v', 's':
		return string(helpers.AppendValue(nil, v)), true
	case 'q':
		return helpers.Quoted(v), true
	case 'd', 'x', 'X', 'o', 'b':
		if n, ok := value.ToInt(v); ok {
			return n, true
		}

		c.typeError(i, "an integer")
	case 'f', 'e', 'E', 'g', 'G':
		if v.IsNumber() {
			return v.Num(), true
		}

		c.typeError(i, "a number")
	default:
		c.fail(errors.ValueError, "unknown verb '%%%c'.", verb)
	}

	return nil, false
}
