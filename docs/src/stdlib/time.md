# time

A time is a number of seconds since 1970-01-01 UTC. A time zone is
`"local"` (the default), `"UTC"` or an IANA name, `"Europe/Kyiv"`; an
unknown one is a `ValueError`.

The examples use a fixed moment, Thursday 2026-11-12 14:30 UTC:

```rune
let t = 1794493800;
print time.format(t, nil, "UTC");   // 2026-11-12T14:30:00Z
```

## now

`now()` is the current time, with sub-second precision. Subtract two to
measure a duration:

```rune
let start = time.now();
let mut sum = 0;
for i in 0..100000 { sum += i; }
let ms = (time.now() - start) * 1000;
print ms < 1000;   // true
```

## date and make

`date([t[, zone]])` is `t` (default now) on the calendar of `zone`: a
`Date { year, month = 1, day = 1, hour = 0, minute = 0, second = 0, weekday, yearDay, zone = "local" }`,
`weekday` from 1, Monday, to 7.

```rune
let t = 1794493800;
let d = time.date(t, "UTC");
print d;
// Date {year: 2026, month: 11, day: 12, hour: 14, minute: 30, second: 0, weekday: 4, yearDay: 316, zone: "UTC"}
print "{d.day}.{d.month}.{d.year}";   // 12.11.2026
print time.date(t, "Europe/Kyiv").hour; // 16
```

`make(d[, zone])` is the time of the `Date` `d`, or an object converted to
one, in `zone` or else `d.zone`. Fields past their range carry over,
`day: 32` into the next month; `weekday` and `yearDay` are ignored. A whole
second gives an integer.

```rune
print time.make({ year: 2026, month: 11, day: 12, hour: 14, minute: 30, zone: "UTC" }); // 1794493800

let newYear = time.make({ year: 2026, month: 12, day: 32, zone: "UTC" });
print time.format(newYear, "%Y-%m-%d", "UTC");   // 2027-01-01
```

Date arithmetic is a `make` of a changed `Date`, so it keeps the wall
clock, across DST too:

```rune
let d = time.date(1794493800, "UTC");
let tomorrow = time.make(Date { ...d, day: d.day + 1 });
let nextMonth = time.make(Date { ...d, month: d.month + 1 });
print time.format(tomorrow, "%a %d %b %H:%M", "UTC");  // Fri 13 Nov 14:30
print time.format(nextMonth, "%Y-%m-%d", "UTC");      // 2026-12-12
```

## format

`format(t[, layout[, zone]])` is `t` as a string, `layout` defaulting to
ISO 8601, `"%Y-%m-%dT%H:%M:%S%:z"`.

```rune
let t = 1794493800;
print time.format(t, "%Y-%m-%d %H:%M", "UTC");            // 2026-11-12 14:30
print time.format(t, "%a %d %b %Y, %I:%M %p", "Europe/Kyiv"); // Thu 12 Nov 2026, 04:30 PM
print time.format(t, nil, "Europe/Kyiv");                 // 2026-11-12T16:30:00+02:00
print time.format(t, "%A, %B %e", "UTC");                 // Thursday, November 12
print time.format(t + 0.25, "%H:%M:%S.%f", "UTC");        // 14:30:00.250000
```

## parse

`parse(s[, layout[, zone]])` is the time `s` holds in `layout`, read in
`zone` unless it has an offset; anything else, or a date that does not
exist, is a `ValueError`.

```rune
print time.parse("2026-11-12", "%Y-%m-%d", "UTC");       // 1794441600
print time.parse("2026-11-12T16:30:00+02:00");           // 1794493800
print time.parse("12/Nov/2026:14:30:00 +0000", "%d/%b/%Y:%H:%M:%S %z"); // 1794493800
```

```rune
time.parse("2026-02-30", "%Y-%m-%d", "UTC"); // ValueError: time.parse() cannot parse "2026-02-30" as "%Y-%m-%d": 2026-02-30 is not a valid date.
```

## Layouts

| | |
|---|---|
| `%Y`, `%y` | year, two-digit year |
| `%m`, `%d`, `%e` | month, day, day padded with a space |
| `%j` | day of the year |
| `%H`, `%I`, `%p` | hour, 12-hour hour, `AM`/`PM` |
| `%M`, `%S` | minute, second |
| `%f` | microseconds; parsing takes any fraction |
| `%b`, `%B` | month name, short and long |
| `%a`, `%A`, `%u` | weekday name, short and long; weekday from 1 |
| `%z`, `%:z` | offset `+0200`; `+02:00`, `Z` for UTC |
| `%Z` | zone abbreviation; parsing takes only `UTC` and `GMT` |
| `%s` | seconds since the epoch |
| `%%` | `%` |

Parsing ignores case in names; a weekday it reads is not checked against
the date.
