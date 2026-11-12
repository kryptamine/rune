package stdlib

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// A layout is text with %-directives, as strftime takes, each of which
// knows how to write its part of a time and to read it back.
type directive struct {
	write func(b []byte, t time.Time) []byte
	read  func(p *parsed, s string) (string, error)
}

// The parts of a time a layout reads.
type part int

const (
	partYear part = iota
	partYear2
	partMonth
	partDay
	partYearDay
	partHour
	partHour12
	partMinute
	partSecond
	nparts
)

// parsed is what a layout has read so far.
type parsed struct {
	v    [nparts]int
	has  [nparts]bool
	nsec int
	pm   bool
	loc  *time.Location
}

func (p *parsed) set(k part, n int) {
	p.v[k], p.has[k] = n, true
}

func (p *parsed) get(k part, def int) int {
	if p.has[k] {
		return p.v[k]
	}

	return def
}

var (
	monthNames = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	dayNames   = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
)

var directives = map[string]directive{
	"Y": number("%04d", 4, partYear, 0, 9999, "year", time.Time.Year),
	"y": number("%02d", 2, partYear2, 0, 99, "year", func(t time.Time) int { return t.Year() % 100 }),
	"m": number("%02d", 2, partMonth, 1, 12, "month", func(t time.Time) int { return int(t.Month()) }),
	"d": number("%02d", 2, partDay, 1, 31, "day", time.Time.Day),
	"e": spaced(number("%2d", 2, partDay, 1, 31, "day", time.Time.Day)),
	"j": number("%03d", 3, partYearDay, 1, 366, "day of the year", time.Time.YearDay),
	"H": number("%02d", 2, partHour, 0, 23, "hour", time.Time.Hour),
	"I": number("%02d", 2, partHour12, 1, 12, "hour", func(t time.Time) int { return (t.Hour()+11)%12 + 1 }),
	"M": number("%02d", 2, partMinute, 0, 59, "minute", time.Time.Minute),
	"S": number("%02d", 2, partSecond, 0, 59, "second", time.Time.Second),
	"u": {
		write: func(b []byte, t time.Time) []byte { return strconv.AppendInt(b, int64(isoWeekday(t)), 10) },
		read: func(_ *parsed, s string) (string, error) {
			_, s, err := digits(s, 1, "a weekday")
			return s, err
		},
	},
	"f": {
		write: func(b []byte, t time.Time) []byte { return fmt.Appendf(b, "%06d", t.Nanosecond()/1000) },
		read: func(p *parsed, s string) (string, error) {
			frac, rest, err := digits(s, 9, "a fraction")
			p.nsec = frac * pow10(9-(len(s)-len(rest)))

			return rest, err
		},
	},
	"p": {
		write: func(b []byte, t time.Time) []byte { return append(b, t.Format("PM")...) },
		read: func(p *parsed, s string) (string, error) {
			i, rest, err := oneOf(s, []string{"AM", "PM"}, "AM or PM")
			p.pm = i == 1

			return rest, err
		},
	},
	"b":  names(monthNames, 3, "a month name", func(t time.Time) int { return int(t.Month()) - 1 }, func(p *parsed, i int) { p.set(partMonth, i+1) }),
	"B":  names(monthNames, 0, "a month name", func(t time.Time) int { return int(t.Month()) - 1 }, func(p *parsed, i int) { p.set(partMonth, i+1) }),
	"a":  names(dayNames, 3, "a weekday name", func(t time.Time) int { return isoWeekday(t) - 1 }, func(*parsed, int) {}),
	"A":  names(dayNames, 0, "a weekday name", func(t time.Time) int { return isoWeekday(t) - 1 }, func(*parsed, int) {}),
	"z":  offsetDirective("-0700"),
	":z": offsetDirective("Z07:00"),
	"Z": {
		write: func(b []byte, t time.Time) []byte { return append(b, t.Format("MST")...) },
		read: func(p *parsed, s string) (string, error) {
			_, rest, err := oneOf(s, []string{"UTC", "GMT"}, "UTC or GMT")
			if err == nil {
				p.loc = time.UTC
			}

			return rest, err
		},
	},
	"s": {
		write: func(b []byte, t time.Time) []byte { return strconv.AppendInt(b, t.Unix(), 10) },
		read: func(p *parsed, s string) (string, error) {
			neg := strings.HasPrefix(s, "-")

			n, rest, err := digits(strings.TrimPrefix(s, "-"), 18, "seconds")
			if neg {
				n = -n
			}

			t := time.Unix(int64(n), 0).UTC()
			p.set(partYear, t.Year())
			p.set(partMonth, int(t.Month()))
			p.set(partDay, t.Day())
			p.set(partHour, t.Hour())
			p.set(partMinute, t.Minute())
			p.set(partSecond, t.Second())
			p.loc = time.UTC

			return rest, err
		},
	},
	"%": {
		write: func(b []byte, _ time.Time) []byte { return append(b, '%') },
		read: func(_ *parsed, s string) (string, error) {
			_, rest, err := oneOf(s, []string{"%"}, `"%"`)
			return rest, err
		},
	},
}

// number is a zero- or space-padded integer from lo to hi.
func number(format string, width int, k part, lo, hi int, what string, get func(time.Time) int) directive {
	return directive{
		write: func(b []byte, t time.Time) []byte { return fmt.Appendf(b, format, get(t)) },
		read: func(p *parsed, s string) (string, error) {
			n, rest, err := digits(s, width, what)
			if err == nil && (n < lo || n > hi) {
				err = fmt.Errorf("%s %d out of range", what, n)
			}

			p.set(k, n)

			return rest, err
		},
	}
}

// spaced lets d read after spaces, as it writes them.
func spaced(d directive) directive {
	read := d.read
	d.read = func(p *parsed, s string) (string, error) { return read(p, strings.TrimLeft(s, " ")) }

	return d
}

// names writes names[get(t)], cut to short characters unless 0, and reads
// a full or short one, ignoring case.
func names(list []string, short int, what string, get func(time.Time) int, set func(*parsed, int)) directive {
	cut := make([]string, len(list))
	for i, n := range list {
		cut[i] = n[:3]
	}

	return directive{
		write: func(b []byte, t time.Time) []byte {
			n := list[get(t)]
			if short > 0 {
				n = n[:short]
			}

			return append(b, n...)
		},
		read: func(p *parsed, s string) (string, error) {
			i, rest, err := oneOf(s, list, what)
			if err != nil {
				i, rest, err = oneOf(s, cut, what)
			}

			set(p, i)

			return rest, err
		},
	}
}

// offsetDirective writes the UTC offset in the Go layout of it and reads
// Z, ±hh, ±hhmm or ±hh:mm.
func offsetDirective(goLayout string) directive {
	return directive{
		write: func(b []byte, t time.Time) []byte { return append(b, t.Format(goLayout)...) },
		read: func(p *parsed, s string) (string, error) {
			if _, rest, err := oneOf(s, []string{"Z"}, ""); err == nil {
				p.loc = time.UTC
				return rest, nil
			}

			sign, rest, err := oneOf(s, []string{"+", "-"}, "a UTC offset")
			if err != nil {
				return s, err
			}

			h, rest, err := digits(rest, 2, "a UTC offset")
			if err != nil {
				return s, err
			}

			m := 0
			if rest = strings.TrimPrefix(rest, ":"); rest != "" && isDigit(rest[0]) {
				if m, rest, err = digits(rest, 2, "a UTC offset"); err != nil {
					return s, err
				}
			}

			off := (h*60 + m) * 60
			if sign == 1 {
				off = -off
			}

			p.loc = time.FixedZone("", off)

			return rest, nil
		},
	}
}

func strftime(t time.Time, layout string) (string, error) {
	var b []byte

	err := walk(layout, func(lit byte) error {
		b = append(b, lit)
		return nil
	}, func(d directive) error {
		b = d.write(b, t)
		return nil
	})

	return string(b), err
}

func strptime(s, layout string, loc *time.Location) (time.Time, error) {
	p := parsed{loc: loc}

	err := walk(layout, func(lit byte) error {
		if s == "" || s[0] != lit {
			return expected(strconv.QuoteRune(rune(lit)), s)
		}

		s = s[1:]

		return nil
	}, func(d directive) (err error) {
		s, err = d.read(&p, s)
		return err
	})

	switch {
	case err != nil:
		return time.Time{}, err
	case s != "":
		return time.Time{}, fmt.Errorf("unexpected %q at the end", s)
	}

	return p.time()
}

// walk calls literal with each byte of layout outside a directive, and
// each with each directive.
func walk(layout string, literal func(byte) error, each func(directive) error) error {
	for i := 0; i < len(layout); i++ {
		if layout[i] != '%' {
			if err := literal(layout[i]); err != nil {
				return err
			}

			continue
		}

		verb := layout[i+1 : min(i+2, len(layout))]
		if strings.HasPrefix(layout[i+1:], ":z") {
			verb = ":z"
		}

		d, ok := directives[verb]
		if !ok {
			return fmt.Errorf("unknown directive %%%s in %q", verb, layout)
		}

		if err := each(d); err != nil {
			return err
		}

		i += len(verb)
	}

	return nil
}

func (p *parsed) time() (time.Time, error) {
	year := p.get(partYear, 1970)
	if p.has[partYear2] {
		if year = 2000 + p.v[partYear2]; year >= 2069 {
			year -= 100
		}
	}

	month, day := p.get(partMonth, 1), p.get(partDay, 1)

	if p.has[partYearDay] {
		d := time.Date(year, 1, p.v[partYearDay], 0, 0, 0, 0, time.UTC)
		if d.Year() != year {
			return time.Time{}, fmt.Errorf("day of the year %d out of range", p.v[partYearDay])
		}

		month, day = int(d.Month()), d.Day()
	}

	hour := p.v[partHour]
	if p.has[partHour12] {
		hour = p.v[partHour12] % 12
	}

	if p.pm && hour < 12 {
		hour += 12
	}

	if d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC); int(d.Month()) != month {
		return time.Time{}, fmt.Errorf("%04d-%02d-%02d is not a valid date", year, month, day)
	}

	return time.Date(year, time.Month(month), day, hour, p.v[partMinute], p.v[partSecond], p.nsec, p.loc), nil
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func digits(s string, width int, what string) (int, string, error) {
	n := 0
	for n < len(s) && n < width && isDigit(s[n]) {
		n++
	}

	if n == 0 {
		return 0, s, expected(what, s)
	}

	v, _ := strconv.Atoi(s[:n])

	return v, s[n:], nil
}

// oneOf matches the first of options that s starts with, ignoring case.
func oneOf(s string, options []string, what string) (int, string, error) {
	for i, o := range options {
		if len(s) >= len(o) && strings.EqualFold(s[:len(o)], o) {
			return i, s[len(o):], nil
		}
	}

	return 0, s, expected(what, s)
}

func pow10(n int) int {
	p := 1
	for range n {
		p *= 10
	}

	return p
}

func expected(what, at string) error {
	if at == "" {
		return fmt.Errorf("expected %s at the end", what)
	}

	return fmt.Errorf("expected %s at %q", what, at)
}
