# regex

Patterns are RE2 syntax (no backreferences; matching is linear in the
input). A bad pattern is a `ValueError`. Indices count characters.

Write patterns as raw strings, `r"..."`, so `\d` and `{3}` are taken as
written; in a plain string `\` is an escape and `{` starts interpolation.

## test

`test(re, s)` — whether `re` matches anywhere in `s`.

```rune
print(regex.test(r"^\d{3}-\d{4}$", "555-1234"));  // true
print(regex.test(r"^\d{3}-\d{4}$", "555-12"));    // false
```

## find and findAll

`find(re, s)` is the first `Match { text, index, groups, named }`, or `nil`:
`groups` holds each group's text, `nil` if it did not take part, and
`named` the named ones (`(?P<name>...)`). `findAll(re, s[, n])` is all
matches, at most `n`.

```rune
let line = "GET /api/users 200 12ms";
let m = regex.find(r"(\w+) (\S+) (\d+)", line);
print(m.text);     // GET /api/users 200
print(m.index);    // 0
print(m.groups);   // ["GET", "/api/users", "200"]

let [method, path, status] = m.groups;
print("{method} {path} -> {status}"); // GET /api/users -> 200

let d = regex.find(r"(?P<year>\d{4})-(?P<month>\d\d)", "released 2026-11");
print(d.named);          // {year: "2026", month: "11"}
print(d.index);          // 9

print(regex.find(r"\d+", "no digits"));            // nil
print(regex.find(r"(cat)|(dog)", "dog").groups);   // [nil, "dog"]
```

```rune
let ids = regex.findAll(r"\d+", "ids: 7, 42, 108");
print(ids.map(m => m.text));                                   // ["7", "42", "108"]
print(regex.findAll(r"\d+", "ids: 7, 42, 108", 2).map(m => m.text)); // ["7", "42"]
```

## replace

`replace(re, s, with)` replaces every match; `with` is a string, where `$1`
or `${name}` is a group, or a function from a `Match` to a string.

```rune
print(regex.replace(r"\s+", "a  b   c", " "));                // a b c
print(regex.replace(r"(\w+)@(\w+)", "ann@mail", r"$2: $1"));  // mail: ann
print(regex.replace(r"(?P<user>\w+)@(?P<host>\w+)", "ann@mail", r"${host}: ${user}")); // mail: ann
print(regex.replace(r"\d+", "3 apples, 12 pears", m => string(int(m.text) * 2))); // 6 apples, 24 pears
```

## split and escape

`split(re, s[, n])` is the parts between matches, at most `n`;
`escape(s)` is `s` with the special characters quoted, to match it
literally.

```rune
print(regex.split(r"\s*,\s*", "a , b,c ,  d"));  // ["a", "b", "c", "d"]
print(regex.split(r",", "a,b,c,d", 2));          // ["a", "b,c,d"]
print(regex.escape("1.5*2?"));                   // 1\.5\*2\?
print(regex.test(regex.escape("1.5"), "105"));   // false
```

```rune
regex.test("(", "x"); // ValueError: regex.test() bad pattern: missing closing ): `(`.
```
