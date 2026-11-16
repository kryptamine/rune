# Built-in Functions

The standard library comes in three parts:

- **Global functions**, listed on this page: `print`, `string`, `number`,
  `int`, plus `map`, `set` and `bytes`, which make
  [maps, sets](../data/map-and-set.md) and [bytes](../data/bytes.md).
  The name of a type converts a value to it: `string(v)`, `number(v)`,
  `map(v)`.
- **Methods**, called on a value: `xs.map(f)`, `s.split(",")`. See
  [arrays](array.md), [strings](string.md),
  [maps and sets](../data/map-and-set.md), [bytes](../data/bytes.md) and
  [streams](streams.md).
- **Namespaces**, groups of functions under one name:
  [`math`](math.md), [`regex`](regex.md), [`json`, `http`](json-http.md),
  [`time`](time.md), [`fs`](fs.md), [`io`](io.md) and [`os`](os.md).
  `string` and `bytes` are namespaces too, with members such as
  `string.fromCharCode` and `bytes.fromHex`.

## print

`print(v, ...)` writes its arguments to stdout, separated by spaces, and
ends the line. A string is written as it is; any other value as it would
appear inside an array, so a nested string keeps its quotes.

```rune
let user = "ann";
let cart = ["tea", "milk"];

print("hello");                       // hello
print("user:", user, "items:", 2);    // user: ann items: 2
print(cart);                          // ["tea", "milk"]
print(["id": 7, "tags": cart]);       // ["id": 7, "tags": ["tea", "milk"]]
print(nil, true, 3.5);                // nil true 3.5
```

`print()` with no arguments writes an empty line. `print` is a function
like any other, so it can be passed along:

```rune
["build", "test", "deploy"].map(print);
// build
// test
// deploy
```

To write without a newline, or to stderr, use [`io`](io.md):
`io.write`, `io.eprint`, `io.ewrite`.

## len

Every collection has a `len()` method: the number of elements of an array,
characters of a string, entries of a map or a set, and
bytes of `bytes`.

```rune
print([10, 20, 30].len());           // 3
print("héllo".len());                // 5
print(["host": "x", "port": 80].len()); // 2
print(set(["a", "b", "a"]).len());   // 2
print(b"héllo".len());               // 6
```

## string

`string(v)` is `v` as `print` shows it. It is the way to turn a number
into text, or to build a key from any value.

```rune
struct User { name, role = "guest" }

print("items: " + string(3));             // items: 3
print(string(3.0));                       // 3
print(string(User { name: "ann" }));      // User {name: "ann", role: "guest"}
print([1, 2.5, nil].map(string));         // ["1", "2.5", "nil"]
```

Inside a string, interpolation does the same: `"items: {n}"`.

## number and int

`number(v)` reads a number from a string. The string holds a
[number literal](../language/numbers.md), with an optional minus sign and
spaces around it: `"42"`, `" -2.5e1 "`, `"0xff"`, `"1_000"`. The result is
an integer, unless the literal has a fraction or an exponent, or is too big
for a 64-bit integer.

`int(v)` makes an integer from a number, by cutting off the fraction
(toward zero). `int(s, base)` reads the digits of `s` in `base`, 2 to 36,
optionally separated by `_`.

```rune
print(number("42"));         // 42
print(number(" -2.5e1 "));   // -25
print(number("0xff"));       // 255
print(number("1_000"));      // 1000

print(int(3.9));             // 3
print(int(-3.9));            // -3
print(int("ff", 16));        // 255
print(int("0755", 8));       // 493
print(int("1010", 2));       // 10
```

Reading the numbers out of a CSV row:

```rune
let row = "2026-11-12,3,12.50";
let [date, qty, price] = row.split(",");
print(number(qty) * number(price));   // 37.5
```

Text that is not a number is a `ValueError`. That includes `".5"`, `"+1"`,
`"inf"` and anything with extra characters:

```rune
number("12 apples"); // ValueError: number() cannot parse "12 apples" as a number.
```

A string is read by `number`; `int` takes it only with a base:

```rune
int("12"); // TypeError: int() expects a number as argument 1, got string.
```

A number too big for a 64-bit integer is a `ValueError`:

```rune
int(1e300); // ValueError: int() cannot convert 1e+300 to an integer.
```

## map

`map(src)` makes a map from a struct, its fields as keys, or from an
array of `[key, value]` pairs. For a map written out, use a
[literal](../data/map-and-set.md#making-one): `["a": 1]`, `[:]`.

```rune
struct Server { host, port }

let pairs = [["host", "localhost"], ["port", 8080]];
print(map(pairs));                                // ["host": "localhost", "port": 8080]
print(map(Server { host: "x.io", port: 443 }));   // ["host": "x.io", "port": 443]

let index = ["a", "b"].map((k, i) => [k, i]);
print(map(index));                                // ["a": 0, "b": 1]
```

## Namespaces

A namespace such as `math` is not a value; only its members are, as with a
[module](../programs/modules.md). Unless the program declares a variable of
the same name, `math.floor` is looked up when the program is compiled. A
call costs nothing extra at run time, and a misspelled member is caught
before the program runs:

```rune
math.flor(1); // Error at 'flor': math has no member 'flor'.
```

A namespace cannot be stored in a variable:

```rune
let m = math; // Error at 'math': Namespace 'math' is not a value; use its members, as in math.name.
```

Its members cannot be assigned:

```rune
math.PI = 3; // Error at 'PI': Cannot assign to a member of namespace 'math'.
```
