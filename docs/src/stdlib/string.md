# String and Number Methods

Strings are immutable: every method returns a new value. Indices and
lengths count characters, not bytes.

## Splitting and case

`split(sep)` (`""` splits into characters), `upper()`, `lower()`,
`startsWith(p)`, `endsWith(p)`.

```rune
let row = "2026-11-12,ann,checkout,42.50";
print(row.split(","));               // ["2026-11-12", "ann", "checkout", "42.50"]
print("abc".split(""));              // ["a", "b", "c"]
print("Ann".upper());                // ANN
print("Ann".lower());                // ann
print("report.csv".endsWith(".csv")); // true
print("https://x.io".startsWith("https://")); // true
```

## Searching and replacing

`replace(old, new)` replaces all occurrences; `repeat(n)`, and `slice`,
`reverse`, `contains`, `indexOf` work as for arrays.

```rune
print("2026-11-12".replace("-", "/"));   // 2026/11/12
print("=".repeat(10));                   // ==========
print("héllo".slice(1, 3));              // él
print("héllo".slice(-2));                // lo
print("héllo".reverse());                // olléh
print("hello world".contains("wor"));    // true
print("héllo".indexOf("l"));             // 2
print("hello".indexOf("z"));             // -1
```

## Trimming

- `trim([chars])`, `trimStart([chars])`, `trimEnd([chars])` — the string without whitespace, or any of the characters of `chars`, at both ends, the start or the end.
- `removePrefix(p)`, `removeSuffix(s)` — the string without `p` at its start or `s` at its end, if there.

```rune
print("  ann  ".trim() + "|");            // ann|
print("  ann  ".trimStart() + "|");       // ann  |
print("  ann  ".trimEnd() + "|");         //   ann|
print("--title--".trim("-"));             // title
print("/usr/local/".trim("/"));           // usr/local
print("v1.2.3".removePrefix("v"));        // 1.2.3
print("report.csv".removeSuffix(".csv")); // report
print("report.csv".removePrefix("x"));    // report.csv
```

## Characters and padding

- `charCode([i])` — the code point of character `i` (default 0); `string.fromCharCode(c, ...)` — the string of those code points.
- `padStart(width[, pad])`, `padEnd(width[, pad])` — the string filled to `width` characters with `pad` (default `" "`), repeated and cut.

```rune
print("A".charCode());                 // 65
print("héllo".charCode(1));            // 233
print(string.fromCharCode(72, 105));   // Hi
print("7".padStart(3, "0"));           // 007
print("ann".padEnd(6) + "|");          // ann   |
print("ab".padStart(7, "xy"));         // xyxyxab
```

## format

`format(...)` is printf, the string being the format:
`%[flags][width][.precision]verb`, flags `-+ 0#`. A width or precision
of `*` takes the next argument, so a column can fit its widest value; a
negative width pads on the right.

- `%v` and `%s` show a value as `print` does, `%q` as inside an array;
- `%d %x %X %o %b` take integers, `%f %e %E %g %G` numbers;
- `%%` is `%`.

Widths count characters; a missing or extra argument is an `ArgumentError`.

```rune
print("%s has %d items".format("cart", 3));        // cart has 3 items
print("%-6s|%5.2f|%03d".format("ab", 3.14159, 7)); // ab    | 3.14|007
print("%v and %q".format(["a"], "a"));             // ["a"] and "a"
print("%x %X %o %b".format(255, 255, 8, 5));       // ff FF 10 101
print("%+d %e".format(5, 12345.678));              // +5 1.234568e+04
print("100%%".format());                           // 100%
```

A table, columns aligned:

```rune
let items = [["coffee", 2, 3.5], ["bagel", 1, 2.25]];
for [name, qty, price] in items {
    print("%-8s %2d x %5.2f".format(name, qty, price));
}
// coffee    2 x  3.50
// bagel     1 x  2.25
```

```rune
let rows = [["GET /users", 12], ["POST /orders/batch", 340]];
let width = rows.reduce((w, [path]) => math.max(w, path.len()), 0);
for [path, ms] in rows {
    print("%-*s %4dms".format(width, path, ms));
}
// GET /users           12ms
// POST /orders/batch  340ms
```

```rune
"%d".format("7"); // TypeError: string.format() expects an integer as argument 1, got string.
```

## Numbers

A number has `fixed(digits)`, with 0 to 100 decimals.

```rune
print(3.14159.fixed(2));  // 3.14
print(2.fixed(2));        // 2.00
print((19.999).fixed(0)); // 20
```
