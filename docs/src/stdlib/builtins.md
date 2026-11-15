# Built-in Functions

`print`, `type`, `object`, `string`, `bytes`, `number`, `int`, `map` and
`set` are global; the name of a type converts to it, as `string(v)`,
`number(v)` or `map(v)` do, and `string` and `bytes` also have members,
such as `string.fromCharCode` and `bytes.fromHex`. The
operations on a value are its methods, as `xs.map(f)` or `s.split(",")`;
everything else lives in a namespace, used as `math.floor(x)`.

## print

`print(v, ...)` writes its arguments to stdout, separated by spaces and
followed by a newline: a string as it is, anything else as it would appear
in an array. `io.write`, `io.eprint` and `io.ewrite` do the same without a
newline or to stderr.

```rune
print("hi");             // hi
print([1, "two", nil]);  // [1, "two", nil]
print("total:", 3, [1]); // total: 3 [1]
print();                 // (an empty line)
["a", "b"].map(print);
// a
// b
```

## type

`type(value)` is `"nil"`, `"boolean"`, `"number"`, `"string"`, `"array"`,
`"object"`, `"function"`, `"map"`, `"set"`, `"bytes"`, `"buffer"`,
`"stream"`, the name of a value's struct, `"Enum.Variant"` for a variant,
`"struct"` for a struct type or `"enum"` for an enum type.

```rune
struct User { name }

print(type(nil));                  // nil
print(type(1.5));                  // number
print(type([1]));                  // array
print(type(x => x));               // function
print(type(User { name: "ann" })); // User
print(type(User));                 // struct
```

## object and string

`object()` makes an empty object and `object(src)` a new one from a map or
an array of `[key, value]` pairs, with string keys; an object or a struct
is copied with spread, `{ ...src }`. `string(v)` is `v` as `print` shows
it.

```rune
struct User { name, role = "guest" }

let u = User { name: "ann" };
print(object([["id", 7], ["ok", true]])); // {id: 7, ok: true}
print(string(u));                       // User {name: "ann", role: "guest"}
print(string([1, "two"]));              // [1, "two"]
print("items: " + string(3));           // items: 3
print([1, 2].map(string));              // ["1", "2"]
```

```rune
object(map([[1, "one"]])); // TypeError: object() expects string keys, got 1.
```

## number and int

`number(v)` is a number as is, or a string holding a
[number literal](../language/numbers.md), optionally negated and
surrounded by spaces (`"42"`, `" -2.5e1 "`, `"0xff"`, `"1_000"`). It is an
integer unless it has a fraction or an exponent or, in decimal, int64
cannot hold it; anything else, `".5"`, `"+1"` or `"inf"` included, is a
`ValueError`.

`int(v[, base])` is a number truncated toward zero (one past the integers
is a `ValueError`), or a string: without `base`, an integer literal as
`number` reads it (`"42"`, `"0xff"`, `"1_000"`); with it, digits in `base`,
2 to 36, optionally separated by `_`.

```rune
print(number("42"));         // 42
print(number(" -2.5e1 "));   // -25
print(number("0xff"));       // 255
print(number("1_000"));      // 1000
print(int("1_000"));         // 1000
print(int(3.9));             // 3
print(int(-3.9));            // -3
print(int("42"));            // 42
print(int("ff", 16));        // 255
print(int("0755", 8));       // 493
print(["10", "2"].map(int)); // [10, 2]
```

```rune
number("12 apples"); // ValueError: number() cannot parse "12 apples" as a number.
```

```rune
int("12.5"); // ValueError: int() cannot parse "12.5" as an integer.
```

## Namespaces

A namespace is no value, only its members are, as for a
[module](../programs/modules.md). Unless a program declares a variable of
its name, `ns.member` is bound when the program is compiled: it costs
nothing at run time, and a misspelled member is a compile error.

```rune
math.flor(1); // Error at 'flor': math has no member 'flor'.
```

```rune
let m = math; // Error at 'math': Namespace 'math' is not a value; use its members, as in math.name.
```

Its members cannot be assigned:

```rune
math.PI = 3; // Error at 'PI': Cannot assign to a member of namespace 'math'.
```
