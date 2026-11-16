# String Methods

Strings never change: every method returns a new string, and the original
stays as it was. Indices and lengths count characters, not bytes, so
`"héllo".len()` is 5.

| Method | Does |
|---|---|
| `len()` | number of characters |
| `split(sep)` | array of the parts between `sep` |
| `upper()`, `lower()` | change case |
| `startsWith(p)`, `endsWith(s)` | test the start or the end |
| `indexOf(sub)` | index of the first `sub`, or `nil` |
| `replace(old, new)` | replace every `old` |
| `repeat(n)` | the string `n` times |
| `trim([chars])`, `trimStart([chars])`, `trimEnd([chars])` | strip whitespace or `chars` |
| `removePrefix(p)`, `removeSuffix(s)` | strip one exact prefix or suffix |
| `padStart(w[, pad])`, `padEnd(w[, pad])` | fill to width `w` |
| `charCode([i])` | code point of character `i` |
| `format(...)` | printf-style formatting |

For patterns, see [regex](regex.md). For indexing, slices and `in`, see
[Strings](../language/strings.md).

## Splitting and case

`split(sep)` cuts the string at each `sep`. An empty `sep` splits it into
characters.

```rune
let row = "2026-11-12,ann,checkout,42.50";
print(row.split(","));        // ["2026-11-12", "ann", "checkout", "42.50"]
print("a,,b".split(","));     // ["a", "", "b"]
print("héy".split(""));       // ["h", "é", "y"]

let [date, user, action, amount] = row.split(",");
print("{user} spent {amount} on {date}");   // ann spent 42.50 on 2026-11-12

print("Ann".upper());         // ANN
print("ÀNN".lower());         // ànn
```

`startsWith` and `endsWith` test the ends of a string:

```rune
let files = ["report.csv", "notes.txt", "data.csv"];
print(files.filter(f => f.endsWith(".csv")));       // ["report.csv", "data.csv"]
print("https://x.io".startsWith("https://"));       // true
```

## Searching and replacing

- `indexOf(sub)` is the index of the first `sub`, or `nil` if there is
  none. To ask only whether it is there, use `sub in s`.
- `replace(old, new)` replaces every `old`.
- `repeat(n)` is the string `n` times over.

```rune
let line = "ERROR disk full on /dev/sda1";

print("disk" in line);                   // true
print(line.indexOf("full"));             // 11
print(line.indexOf("cpu"));              // nil
print(line[line.indexOf("/")..]);        // /dev/sda1

print("2026-11-12".replace("-", "/"));   // 2026/11/12
print("=".repeat(12));                   // ============
```

## Trimming

- `trim()`, `trimStart()`, `trimEnd()` strip whitespace from both ends,
  the start, or the end. Given `chars`, they strip any of those characters
  instead.
- `removePrefix(p)` and `removeSuffix(s)` strip `p` or `s` once, if the
  string has it.

```rune
print("[" + "  ann  ".trim() + "]");        // [ann]
print("[" + "  ann  ".trimStart() + "]");   // [ann  ]
print("[" + "  ann  ".trimEnd() + "]");     // [  ann]
print("--title--".trim("-"));               // title
print("/usr/local/".trim("/"));             // usr/local
print("0042".trimStart("0"));               // 42

print("v1.2.3".removePrefix("v"));          // 1.2.3
print("report.csv".removeSuffix(".csv"));   // report
print("report.csv".removePrefix("x"));      // report.csv
```

`trim` strips every matching character, while `removePrefix` strips the
exact prefix once:

```rune
print("vvv1".trimStart("v"));      // 1
print("vvv1".removePrefix("v"));   // vv1
```

## Padding and characters

- `padStart(width[, pad])` and `padEnd(width[, pad])` fill the string to
  `width` characters with `pad`, a space by default. A longer string stays
  as it is.
- `charCode([i])` is the code point of character `i`, 0 by default.
  `string.fromCharCode(c, ...)` makes a string from code points.

```rune
print("7".padStart(3, "0"));           // 007
print("ann".padEnd(6, ".") + "|");     // ann...|
print("ab".padStart(7, "xy"));         // xyxyxab
print("long name".padEnd(4) + "|");    // long name|

print("A".charCode());                 // 65
print("héllo".charCode(1));            // 233
print(string.fromCharCode(72, 105));   // Hi
```

A Caesar shift, built from character codes:

```rune
fun shift(s, n) {
    let a = "a".charCode();
    s.split("").map(c => string.fromCharCode((c.charCode() - a + n) % 26 + a)).join("")
}
print(shift("hello", 3));   // khoor
```

## format

`s.format(...)` is printf for numbers: `s` is the format, and each `%`
verb takes the next argument. Text goes in by
[interpolation](../language/strings.md#joining-and-interpolation), and is
aligned with `padStart` and `padEnd`.

| Verb | Shows |
|---|---|
| `%q` | a value as it appears inside an array (strings quoted) |
| `%d` | an integer |
| `%x`, `%X`, `%o`, `%b` | an integer in hex, upper hex, octal, binary |
| `%f`, `%e`, `%E`, `%g`, `%G` | a number |
| `%%` | `%` |

Between `%` and the verb come optional flags, a width and a precision:
`%[flags][width][.precision]verb`.

- Flags: `-` pads on the right, `+` always shows the sign, `0` pads with
  zeros, ` ` leaves a space for the sign, `#` adds `0x` and the like.
- Width is the minimum number of characters.
- Precision is the number of digits after the point for `%f`.
- `*` for a width or a precision takes it from the next argument.

```rune
print("%d items".format(3));                       // 3 items
print("%.2f".format(3.14159));                      // 3.14
print("%5.1f%%".format(12.345));                    //  12.3%
print("%03d|%-5d|%5d|".format(7, 12, 34));          // 007|12   |   34|
print("%q".format("hi"));                          // "hi"
print("%x %X %o %b".format(255, 255, 8, 5));       // ff FF 10 101
print("%#x".format(255));                          // 0xff
print("%+d %e".format(5, 12345.678));              // +5 1.234568e+04
```

A receipt, its columns aligned:

```rune
let items = [["coffee", 2, 3.5], ["bagel", 1, 2.25], ["juice", 3, 4]];
for [name, qty, price] in items {
    print("{name.padEnd(8)} {"%2d x %5.2f".format(qty, price)}");
}
// coffee    2 x  3.50
// bagel     1 x  2.25
// juice     3 x  4.00
```

A column can take the width of its widest value:

```rune
let rows = [["GET /users", 12], ["POST /orders/batch", 340]];
let width = rows.reduce((w, [path, _]) => math.max(w, path.len()), 0);
for [path, ms] in rows {
    print("{path.padEnd(width)} {"%4d".format(ms)}ms");
}
// GET /users           12ms
// POST /orders/batch  340ms
```

An argument of the wrong type is a `TypeError`, and a missing or extra one
an `ArgumentError`:

```rune
"%d".format("7"); // TypeError: string.format() expects an integer as argument 1, got string.
```

```rune
"%d and %d".format(1); // ArgumentError: string.format() '%d' at 7 has no argument.
```
