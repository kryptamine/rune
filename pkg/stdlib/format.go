package stdlib

import (
	"fmt"
	"math"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"strconv"
	"strings"
)

// format is printf: %[flags][width][.precision]verb, flags from "-+ 0#".
// A width or precision of * takes the next argument, an integer; a
// negative width pads on the right. %v and %s show a value as print does,
// %q as it shows inside an array; %d %x %X %o %b take integers, %f %e %E
// %g %G numbers; %% is a percent sign.
func format(c *call, f string, args []value.Value) value.Value {
	var b strings.Builder

	p := printf{c: c, args: args}

	for i := 0; i < len(f); i++ {
		if f[i] != '%' {
			b.WriteByte(f[i])
			continue
		}

		d, end := scanDirective(f, i)
		if end < 0 {
			return c.fail(errors.ValueError, "incomplete directive '%s' at %d.", f[i:], i)
		}

		p.text, p.start = f[i:end], i
		i = end - 1

		if d.verb == '%' {
			if len(p.text) > 2 {
				return p.bad()
			}

			b.WriteByte('%')

			continue
		}

		width, wok := p.size(d.width, 4)
		precision, pok := p.size(d.precision, 3)

		switch {
		case !c.ok():
			return value.NilValue
		case !wok || !pok || d.verb == '.' || d.star && strings.HasPrefix(precision, "-"):
			return p.bad()
		}

		if d.star && strings.HasPrefix(width, "-") {
			d.flags, width = "-"+d.flags, width[1:]
		}

		j, ok := p.take()
		if !ok {
			return value.NilValue
		}

		arg, ok := formatArg(c, d.verb, j)
		if !ok {
			return value.NilValue
		}

		goDirective := p.text[:len(p.text)-1]
		if d.star {
			goDirective = "%" + d.flags + width
			if d.dot {
				goDirective += "." + precision
			}
		}

		fmt.Fprintf(&b, goDirective+goVerb(d.verb), arg)
	}

	if p.next < len(args) {
		return c.fail(errors.ArgumentError, "got %d arguments for %d directives.", len(args), p.next)
	}

	return value.NewString(b.String())
}

// printf walks the arguments of a format call; text and start are the
// directive at hand, for errors.
type printf struct {
	c     *call
	args  []value.Value
	next  int
	text  string
	start int
}

func (p *printf) bad() value.Value {
	return p.c.fail(errors.ValueError, "bad directive '%s' at %d.", p.text, p.start)
}

// take returns the index in c.args of the next argument.
func (p *printf) take() (int, bool) {
	if p.next == len(p.args) {
		p.c.fail(errors.ArgumentError, "'%s' at %d has no argument.", p.text, p.start)
		return 0, false
	}

	p.next++

	return p.next, true
}

// size is a width or precision as written: up to digits digits, or *
// for the next argument, an integer of as many digits at most.
func (p *printf) size(field string, digits int) (string, bool) {
	if field != "*" {
		return field, len(field) <= digits && !strings.Contains(field, "*")
	}

	limit := int64(math.Pow10(digits)) - 1

	j, ok := p.take()
	if !ok {
		return "", false
	}

	n, ok := value.ToInt(p.c.args[j])
	if !ok {
		p.c.typeError(j, "an integer")
		return "", false
	}

	if n < -limit || n > limit {
		p.c.fail(errors.ValueError, "'%s' at %d got %d for '*', out of range %d to %d.", p.text, p.start, n, -limit, limit)
		return "", false
	}

	return strconv.FormatInt(n, 10), true
}

// printfSpec is %[flags][width][.precision]verb as written; width and
// precision are digits or *.
type printfSpec struct {
	flags, width, precision string
	dot, star               bool
	verb                    byte
}

// scanDirective reads the directive whose '%' is f[i] and returns it with
// the index past its verb, or -1 if f ends first.
func scanDirective(f string, i int) (printfSpec, int) {
	var d printfSpec

	j := i + 1
	d.flags, j = span(f, j, flagByte)
	d.width, j = span(f, j, sizeByte)

	if j < len(f) && f[j] == '.' {
		d.dot = true
		d.precision, j = span(f, j+1, sizeByte)
	}

	if j == len(f) {
		return d, -1
	}

	d.verb = f[j]
	d.star = strings.IndexByte(d.width, '*') >= 0 || strings.IndexByte(d.precision, '*') >= 0

	return d, j + 1
}

const (
	flagByte = 1 << iota
	sizeByte
)

// directiveBytes classes the bytes of a directive's flags and sizes.
var directiveBytes = func() (t [256]uint8) {
	for _, c := range "-+ 0#" {
		t[c] |= flagByte
	}

	for _, c := range "0123456789*" {
		t[c] |= sizeByte
	}

	return t
}()

// span is the run of bytes of class at f[j:], and the index past it.
func span(f string, j int, class uint8) (string, int) {
	from := j
	for j < len(f) && directiveBytes[f[j]]&class != 0 {
		j++
	}

	return f[from:j], j
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
