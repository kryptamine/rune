package stdlib

import (
	"math"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"sync"
	"time"
	_ "time/tzdata"
)

const isoLayout = "%Y-%m-%dT%H:%M:%S%:z"

// The fields of Date, in its order.
const (
	dateYear = iota
	dateMonth
	dateDay
	dateHour
	dateMinute
	dateSecond
	dateWeekday
	dateYearDay
	dateZone
)

var Date = value.NewShape("Date", []value.Field{
	{Name: "year", Required: true},
	{Name: "month", Default: value.NewInt(1)},
	{Name: "day", Default: value.NewInt(1)},
	{Name: "hour", Default: value.NewInt(0)},
	{Name: "minute", Default: value.NewInt(0)},
	{Name: "second", Default: value.NewInt(0)},
	{Name: "weekday"},
	{Name: "yearDay"},
	{Name: "zone", Default: value.NewString("local")},
})

var timeNatives = []*Native{
	{"time.now", 0, 0, func(*call) value.Value {
		return value.NewNumber(float64(time.Now().UnixNano()) / 1e9)
	}},
	{"time.date", 0, 2, func(c *call) value.Value {
		t, zone := c.instant(0), c.optStr(1, "local")
		loc := c.location(zone)
		if !c.ok() {
			return value.NilValue
		}

		return dateOf(t.In(loc), zone)
	}},
	{"time.make", 1, 2, func(c *call) value.Value {
		d, err := Convert(c.token, Date, c.args[0])
		if err != nil {
			c.err = err
			return value.NilValue
		}

		f := d.Fields()

		var ymdhm [dateSecond]int
		for i := range ymdhm {
			ymdhm[i] = c.dateInt(f, i)
		}

		sec := c.dateField(f, dateSecond, value.Value.IsNumber, "a number").Num()
		zone := c.dateField(f, dateZone, value.Value.IsString, "a string").Str()
		loc := c.location(c.optStr(1, zone))

		if !c.ok() {
			return value.NilValue
		}

		if math.IsNaN(sec) || math.IsInf(sec, 0) {
			return c.fail(errors.ValueError, "expects a finite Date.second.")
		}

		whole := math.Floor(sec)
		nsec := int(math.Round((sec - whole) * 1e9))

		return seconds(time.Date(ymdhm[dateYear], time.Month(ymdhm[dateMonth]), ymdhm[dateDay], ymdhm[dateHour], ymdhm[dateMinute], int(whole), nsec, loc))
	}},
	{"time.format", 1, 3, func(c *call) value.Value {
		t, layout, loc := c.instant(0), c.optStr(1, isoLayout), c.location(c.optStr(2, "local"))
		if !c.ok() {
			return value.NilValue
		}

		s, err := strftime(t.In(loc), layout)
		if err != nil {
			return c.fail(errors.ValueError, "%s.", err)
		}

		return value.NewString(s)
	}},
	{"time.parse", 1, 3, func(c *call) value.Value {
		s, layout, loc := c.str(0), c.optStr(1, isoLayout), c.location(c.optStr(2, "local"))
		if !c.ok() {
			return value.NilValue
		}

		t, err := strptime(s, layout, loc)
		if err != nil {
			return c.fail(errors.ValueError, "cannot parse %q as %q: %s.", s, layout, err)
		}

		return seconds(t)
	}},
}

// instant is argument i, seconds since the epoch, or now if not given.
func (c *call) instant(i int) time.Time {
	if !c.has(i) || c.args[i].IsNil() {
		return time.Now()
	}

	n := c.num(i)
	if math.IsNaN(n) || math.IsInf(n, 0) {
		c.fail(errors.ValueError, "expects a finite time, got %v.", n)
		return time.Time{}
	}

	whole := math.Floor(n)

	return time.Unix(int64(whole), int64(math.Round((n-whole)*1e9)))
}

// optStr is argument i, a string, or def if not given or nil.
func (c *call) optStr(i int, def string) string {
	if !c.has(i) || c.args[i].IsNil() {
		return def
	}

	return c.str(i)
}

// dateField is field i of a Date, which is must accept.
func (c *call) dateField(f []value.Value, i int, is func(value.Value) bool, want string) value.Value {
	if c.ok() && !is(f[i]) {
		c.fail(errors.TypeError, "expects Date.%s to be %s, got %s.", Date.Fields[i], want, helpers.TypeName(f[i]))
	}

	if !c.ok() {
		return value.NewInt(0)
	}

	return f[i]
}

func (c *call) dateInt(f []value.Value, i int) int {
	n, _ := value.ToInt(c.dateField(f, i, isInt, "an integer"))
	return int(n)
}

func isInt(v value.Value) bool {
	_, ok := value.ToInt(v)
	return ok
}

var locations sync.Map

// location is the time zone called name: "local", "UTC" or an IANA name.
// After a failure it is UTC, so callers check once at the end.
func (c *call) location(name string) *time.Location {
	if !c.ok() {
		return time.UTC
	}

	if name == "local" {
		return time.Local
	}

	if loc, ok := locations.Load(name); ok {
		return loc.(*time.Location)
	}

	// LoadLocation takes these two for UTC and the local zone.
	loc, err := time.LoadLocation(name)
	if err != nil || name == "" || name == "Local" {
		c.fail(errors.ValueError, "unknown time zone %q.", name)
		return time.UTC
	}

	locations.Store(name, loc)

	return loc
}

// seconds is t since the epoch, an integer when t is a whole second.
func seconds(t time.Time) value.Value {
	if t.Nanosecond() == 0 {
		return value.NewInt(t.Unix())
	}

	return value.NewNumber(float64(t.UnixNano()) / 1e9)
}

func dateOf(t time.Time, zone string) value.Value {
	v, f := value.NewStruct(Date)
	for i, n := range [...]int{t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute(), t.Second(), isoWeekday(t), t.YearDay()} {
		f[i] = value.NewInt(int64(n))
	}

	f[dateZone] = value.NewString(zone)

	return v
}

// isoWeekday counts from Monday, 1, to Sunday, 7.
func isoWeekday(t time.Time) int {
	if t.Weekday() == time.Sunday {
		return 7
	}

	return int(t.Weekday())
}
