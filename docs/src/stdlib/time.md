# time

A time is a number: seconds since 1970-01-01 00:00 UTC. Times subtract to
durations in seconds, and compare with `<` and `>`.

| Function | Gives |
|---|---|
| `time.now()` | the current time |
| `time.date([t[, zone]])` | `t` split into a `Date` on the calendar |
| `time.make(d[, zone])` | the time of a `Date` |
| `time.format(t[, layout[, zone]])` | `t` as text |
| `time.parse(s[, layout[, zone]])` | the time written in `s` |

A time zone is `"local"` (the default), `"UTC"` or an IANA name such as
`"Europe/Kyiv"`. An unknown zone is a `ValueError`.

The examples use a fixed moment, Thursday 2026-11-12 14:30 UTC:

```rune
let t = 1794493800;
print(time.format(t, nil, "UTC"));   // 2026-11-12T14:30:00Z
```

## now

`time.now()` is the current time, with fractions of a second. Subtract two
to measure how long something took:

```rune
let start = time.now();
let mut sum = 0;
for i in 0..100000 { sum += i; }
let ms = (time.now() - start) * 1000;
print(ms < 1000);   // true
```

## date and make

`time.date([t[, zone]])` splits `t` (default: now) into its calendar
fields in `zone`. The result is a `Date`:

```
Date { year, month = 1, day = 1, hour = 0, minute = 0, second = 0, weekday, yearDay, zone = "local" }
```

`weekday` runs from 1, Monday, to 7, Sunday; `yearDay` from 1.

```rune
let t = 1794493800;
let d = time.date(t, "UTC");
print(d);
// Date {year: 2026, month: 11, day: 12, hour: 14, minute: 30, second: 0, weekday: 4, yearDay: 316, zone: "UTC"}
print("{d.day}.{d.month}.{d.year}");       // 12.11.2026
print(time.date(t, "Europe/Kyiv").hour);   // 16
```

`time.make(d[, zone])` goes the other way: the time of the `Date` `d`, in
`zone`, or else `d.zone`. A field past
its range carries over, so `day: 32` of December is January 1.
`weekday` and `yearDay` are ignored.

```rune
print(time.make(Date { year: 2026, month: 11, day: 12, hour: 14, minute: 30, zone: "UTC" }));
// 1794493800

let newYear = time.make(Date { year: 2026, month: 12, day: 32, zone: "UTC" });
print(time.format(newYear, "%Y-%m-%d", "UTC"));   // 2027-01-01
```

### Date arithmetic

To move by days or months, change a field of a `Date` and `make` it again.
That keeps the wall-clock time, and handles month lengths and leap years:

```rune
let d = time.date(1794493800, "UTC");
let tomorrow = time.make(Date { ...d, day: d.day + 1 });
let nextMonth = time.make(Date { ...d, month: d.month + 1 });
print(time.format(tomorrow, "%a %d %b %H:%M", "UTC"));   // Fri 13 Nov 14:30
print(time.format(nextMonth, "%Y-%m-%d", "UTC"));        // 2026-12-12
```

It works across a daylight-saving change too. In Kyiv, clocks move forward
on 2026-03-29, so "tomorrow at noon" is only 23 hours away:

```rune
let noon = time.parse("2026-03-28 12:00", "%Y-%m-%d %H:%M", "Europe/Kyiv");
let d = time.date(noon, "Europe/Kyiv");
let next = time.make(Date { ...d, day: d.day + 1 });
print(time.format(next, nil, "Europe/Kyiv"));   // 2026-03-29T12:00:00+03:00
print((next - noon) / 3600);                    // 23
```

To move by a fixed duration instead, add seconds: `t + 3600` is an hour
later.

## format

`time.format(t[, layout[, zone]])` writes `t` as text. The
[layout](#layouts) defaults to ISO 8601, `"%Y-%m-%dT%H:%M:%S%:z"`.

```rune
let t = 1794493800;
print(time.format(t, "%Y-%m-%d %H:%M", "UTC"));                 // 2026-11-12 14:30
print(time.format(t, "%a %d %b %Y, %I:%M %p", "Europe/Kyiv"));  // Thu 12 Nov 2026, 04:30 PM
print(time.format(t, nil, "Europe/Kyiv"));                      // 2026-11-12T16:30:00+02:00
print(time.format(t, "%A, %B %e", "UTC"));                      // Thursday, November 12
print(time.format(t + 0.25, "%H:%M:%S.%f", "UTC"));             // 14:30:00.250000
```

## parse

`time.parse(s[, layout[, zone]])` reads the time written in `s`, by
`layout` (ISO 8601 by default). A time without an offset is read in
`zone`.

```rune
print(time.parse("2026-11-12", "%Y-%m-%d", "UTC"));                       // 1794441600
print(time.parse("2026-11-12T16:30:00+02:00"));                           // 1794493800
print(time.parse("12/Nov/2026:14:30:00 +0000", "%d/%b/%Y:%H:%M:%S %z"));  // 1794493800
```

Days between two dates:

```rune
let start = time.parse("2026-11-01", "%Y-%m-%d", "UTC");
let end = time.parse("2026-11-12", "%Y-%m-%d", "UTC");
print((end - start) / 86400);   // 11
```

Text that does not fit the layout, or a date that does not exist, is a
`ValueError`:

```rune
time.parse("2026-02-30", "%Y-%m-%d", "UTC"); // ValueError: time.parse() cannot parse "2026-02-30" as "%Y-%m-%d": 2026-02-30 is not a valid date.
```

## Layouts

| Code | Means | Example |
|---|---|---|
| `%Y`, `%y` | year, two-digit year | `2026`, `26` |
| `%m` | month | `11` |
| `%d`, `%e` | day; day padded with a space | `05`, ` 5` |
| `%j` | day of the year | `316` |
| `%H`, `%I`, `%p` | hour; 12-hour hour; `AM`/`PM` | `14`, `02`, `PM` |
| `%M`, `%S` | minute, second | `30`, `00` |
| `%f` | microseconds; parsing takes any fraction | `250000` |
| `%b`, `%B` | month name, short and long | `Nov`, `November` |
| `%a`, `%A` | weekday name, short and long | `Thu`, `Thursday` |
| `%u` | weekday, 1 (Monday) to 7 | `4` |
| `%z`, `%:z` | offset; offset with a colon, `Z` for UTC | `+0200`, `+02:00` |
| `%Z` | zone abbreviation; parsing takes only `UTC` and `GMT` | `UTC` |
| `%s` | seconds since the epoch | `1794493800` |
| `%%` | `%` | `%` |

Parsing ignores the case of names. A weekday it reads is not checked
against the date.
