# regex

Regular expressions, in [RE2 syntax](https://github.com/google/re2/wiki/Syntax).
Matching takes time linear in the input; in exchange, RE2 has no
backreferences. Indices count characters.

| Function | Gives |
|---|---|
| `regex.test(re, s)` | whether `re` matches anywhere in `s` |
| `regex.find(re, s)` | the first `Match`, or `nil` |
| `regex.findAll(re, s[, n])` | all matches, at most `n` |
| `regex.replace(re, s, with)` | `s` with every match replaced |
| `regex.split(re, s[, n])` | the parts between matches, at most `n` |
| `regex.escape(s)` | `s` with special characters escaped |

Write patterns as raw strings, `r"..."`. In a raw string `\d` and `{3}`
mean what they say; in a plain string `\` starts an escape and `{` starts
interpolation.

## test

`regex.test(re, s)` is whether `re` matches anywhere in `s`. Anchor with
`^` and `$` to match the whole string.

```rune
let phone = r"^\d{3}-\d{4}$";
print(regex.test(phone, "555-1234"));   // true
print(regex.test(phone, "555-12"));     // false
print(regex.test(r"\d", "room 101"));   // true
```

## find and findAll

`regex.find(re, s)` is the first match, or `nil`. A match is a
`Match { text, index, groups, named }`:

- `text` — the matched text;
- `index` — where it starts in `s`;
- `groups` — the text of each group `(...)`, `nil` for a group that did
  not take part;
- `named` — an object of the named groups, `(?P<name>...)`.

```rune
let line = "GET /api/users 200 12ms";
let m = regex.find(r"(\w+) (\S+) (\d+)", line);
print(m.text);     // GET /api/users 200
print(m.index);    // 0
print(m.groups);   // ["GET", "/api/users", "200"]

let [method, path, status] = m.groups;
print("{method} {path} -> {status}");   // GET /api/users -> 200
```

Named groups read better when there are several:

```rune
let d = regex.find(r"(?P<year>\d{4})-(?P<month>\d\d)", "released 2026-11");
print(d.named);         // {year: "2026", month: "11"}
print(d.named.year);    // 2026
print(d.index);         // 9

print(regex.find(r"\d+", "no digits"));             // nil
print(regex.find(r"(cat)|(dog)", "dog").groups);    // [nil, "dog"]
```

`regex.findAll(re, s[, n])` is an array of every match, or of the first
`n`:

```rune
let text = "ids: 7, 42, 108";
print(regex.findAll(r"\d+", text).map(m => int(m.text)));      // [7, 42, 108]
print(regex.findAll(r"\d+", text, 2).map(m => m.text));        // ["7", "42"]

let pairs = regex.findAll(r"(\w+)=(\w+)", "user=ann role=admin");
print(object(pairs.map(m => m.groups)));                       // {user: "ann", role: "admin"}
```

## replace

`regex.replace(re, s, with)` replaces every match. `with` is either:

- a string, where `$1` is the first group and `${name}` a named one;
- a function that takes a `Match` and returns a string.

```rune
print(regex.replace(r"\s+", "a  b   c", " "));                 // a b c
print(regex.replace(r"(\w+)@(\w+)", "ann@mail", r"$2: $1"));   // mail: ann
print(regex.replace(r"(?P<user>\w+)@(?P<host>\w+)", "ann@mail", r"${host}: ${user}")); // mail: ann
```

With a function, each replacement can be computed:

```rune
let recipe = "3 eggs, 200 g flour, 1 cup milk";
print(regex.replace(r"\d+", recipe, m => string(int(m.text) * 2)));
// 6 eggs, 400 g flour, 2 cup milk
```

## split and escape

`regex.split(re, s[, n])` is the parts of `s` between matches, at most
`n`. `regex.escape(s)` escapes the special characters of `s`, so it
matches literally.

```rune
print(regex.split(r"\s*,\s*", "a , b,c ,  d"));   // ["a", "b", "c", "d"]
print(regex.split(r",", "a,b,c,d", 2));           // ["a", "b,c,d"]

print(regex.escape("1.5*2?"));                    // 1\.5\*2\?
print(regex.test("1.5", "105"));                  // true
print(regex.test(regex.escape("1.5"), "105"));    // false
```

A bad pattern is a `ValueError`:

```rune
regex.test("(", "x"); // ValueError: regex.test() bad pattern: missing closing ): `(`.
```
