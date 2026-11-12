# Built-in Functions

`print`, `type`, `object`, `str`, `number`, `int`, `Map` and `Set`
are global; the operations on a value are its methods, as `xs.map(f)` or
`s.split(",")`; everything else lives in a namespace, used as
`math.floor(x)`.

## print

`print(v)` writes `v` to stdout followed by a newline: a string as it is,
anything else as it would appear in an array. `io.write`, `io.eprint` and
`io.ewrite` do the same without a newline or to stderr.

```rune
print("hi");             // hi
print([1, "two", nil]);  // [1, "two", nil]
["a", "b"].map(print);
// a
// b
```

## type

`type(value)` is `"nil"`, `"boolean"`, `"number"`, `"string"`, `"array"`,
`"object"`, `"function"`, `"Map"`, `"Set"`, `"bytes"`, `"Stream"`, a
struct's name, `"struct"` for a struct type, or `"namespace"`.

```rune
struct User { name }

print(type(nil));                  // nil
print(type(1.5));                  // number
print(type([1]));                  // array
print(type(x => x));               // function
print(type(User { name: "ann" })); // User
print(type(User));                 // struct
print(type(math));                 // namespace
```

## object and str

`object(s)` is a new object with the fields of the struct `s`; `str(v)` is
`v` as `print` shows it.

```rune
struct User { name, role = "guest" }

let u = User { name: "ann" };
print(object(u));          // {name: "ann", role: "guest"}
print(str(u));             // User {name: "ann", role: "guest"}
print(str([1, "two"]));    // [1, "two"]
print("items: " + str(3)); // items: 3
```

## number and int

`number(v)` is a number as is, or a string holding a number literal,
optionally negated and surrounded by spaces (`"42"`, `" -2.5e1 "`). It is
an integer unless it has a fraction or an exponent or int64 cannot hold it;
anything else, `".5"`, `"+1"` or `"inf"` included, is a `ValueError`.

`int(v[, base])` is a number truncated toward zero (one past the integers
is a `ValueError`), or a string parsed as an integer in `base`, 2 to 36,
default 10.

```rune
print(number("42"));         // 42
print(number(" -2.5e1 "));   // -25
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
int("12.5"); // ValueError: int() cannot parse "12.5" as an integer in base 10.
```

## Namespaces

A namespace is an immutable struct of its members. Unless a program declares
a variable of its name, `ns.member` is bound when the program is compiled:
it costs nothing at run time, and a misspelled member is a compile error.

```rune
math.flor(1); // Error at 'flor': math has no member 'flor'.
```

Built-ins are immutable, as `let` variables are:

```rune
math.PI = 3; // Error at 'math': Cannot mutate immutable variable 'math'.
```
