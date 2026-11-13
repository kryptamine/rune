package stdlib

import (
	"regexp"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"strings"
	"unicode/utf8"
)

// Match is a regex match; index counts characters, groups holds nil for a
// group that did not take part.
var Match = value.NewShape("Match", []value.Field{
	{Name: "text", Required: true},
	{Name: "index", Required: true},
	{Name: "groups", Required: true},
	{Name: "named", Required: true},
})

const regexCacheSize = 256

// regex compiles argument i, reusing what this host compiled before.
func (c *call) regex(i int) *regexp.Regexp {
	src := c.str(i)
	if !c.ok() {
		return nil
	}

	if re, ok := c.host.regexes[src]; ok {
		return re
	}

	re, err := regexp.Compile(src)
	if err != nil {
		c.fail(errors.ValueError, "bad pattern: %s.", strings.TrimPrefix(err.Error(), "error parsing regexp: "))
		return nil
	}

	if c.host.regexes == nil || len(c.host.regexes) >= regexCacheSize {
		c.host.regexes = map[string]*regexp.Regexp{}
	}

	c.host.regexes[src] = re

	return re
}

func (c *call) limit(i int) int {
	if !c.has(i) {
		return -1
	}

	return c.int(i)
}

func newMatch(re *regexp.Regexp, s string, loc []int) value.Value {
	groups := make([]value.Value, re.NumSubexp())
	named := value.NewObj(0)

	for g := range groups {
		if start := loc[2*g+2]; start >= 0 {
			groups[g] = value.NewString(s[start:loc[2*g+3]])
		}

		if name := re.SubexpNames()[g+1]; name != "" {
			named.Set(name, groups[g])
		}
	}

	v, fields := value.NewStruct(Match)
	fields[0] = value.NewString(s[loc[0]:loc[1]])
	fields[1] = count(utf8.RuneCountInString(s[:loc[0]]))
	fields[2] = value.NewArray(groups)
	fields[3] = value.NewObject(named)

	return v
}

var regexNatives = []*Native{
	{"regex.test", 2, 2, func(c *call) value.Value {
		re, s := c.regex(0), c.str(1)
		if !c.ok() {
			return value.NilValue
		}

		return value.NewBool(re.MatchString(s))
	}},
	{"regex.find", 2, 2, func(c *call) value.Value {
		re, s := c.regex(0), c.str(1)
		if !c.ok() {
			return value.NilValue
		}

		loc := re.FindStringSubmatchIndex(s)
		if loc == nil {
			return value.NilValue
		}

		return newMatch(re, s, loc)
	}},
	{"regex.findAll", 2, 3, func(c *call) value.Value {
		re, s, n := c.regex(0), c.str(1), c.limit(2)
		if !c.ok() {
			return value.NilValue
		}

		locs := re.FindAllStringSubmatchIndex(s, n)
		out := make([]value.Value, len(locs))

		for i, loc := range locs {
			out[i] = newMatch(re, s, loc)
		}

		return value.NewArray(out)
	}},
	{"regex.replace", 3, 3, func(c *call) value.Value {
		re, s := c.regex(0), c.str(1)
		if !c.ok() {
			return value.NilValue
		}

		if c.args[2].IsString() {
			return value.NewString(re.ReplaceAllString(s, c.args[2].Str()))
		}

		if c.args[2].Kind() != value.Function {
			return c.typeError(2, "a string or a function")
		}

		return replaceFunc(c, re, s, c.args[2])
	}},
	{"regex.split", 2, 3, func(c *call) value.Value {
		re, s, n := c.regex(0), c.str(1), c.limit(2)
		if !c.ok() {
			return value.NilValue
		}

		parts := re.Split(s, n)
		out := make([]value.Value, len(parts))

		for i, p := range parts {
			out[i] = value.NewString(p)
		}

		return value.NewArray(out)
	}},
	{"regex.escape", 1, 1, func(c *call) value.Value {
		return value.NewString(regexp.QuoteMeta(c.str(0)))
	}},
}

// replaceFunc replaces each match with what fn returns for its Match.
func replaceFunc(c *call, re *regexp.Regexp, s string, fn value.Value) value.Value {
	var b strings.Builder

	last := 0
	arg := make([]value.Value, 1)

	for _, loc := range re.FindAllStringSubmatchIndex(s, -1) {
		arg[0] = newMatch(re, s, loc)

		r := c.invoke(fn, arg)
		if !c.ok() {
			return value.NilValue
		}

		if !r.IsString() {
			return c.fail(errors.TypeError, "expects the function to return a string, got %s.", helpers.TypeName(r))
		}

		b.WriteString(s[last:loc[0]])
		b.WriteString(r.Str())
		last = loc[1]
	}

	b.WriteString(s[last:])

	return value.NewString(b.String())
}
