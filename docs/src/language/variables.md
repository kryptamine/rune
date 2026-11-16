# Variables

`let` declares a variable that never changes. `let mut` declares one that
may. Immutable is the default because most values are set once: when a
variable is plain `let`, you know it holds the same value everywhere you
read it.

```rune
let name = "ann";
let mut visits = 0;

visits += 1;
print("{name} visited {visits} times");    // ann visited 1 times
```

Changing a `let` variable, with `=`, `+=` or any other assignment, is
caught at compile time, before the program runs:

```rune
let name = "ann";
name = "bob";    // Error at 'name': Cannot assign to immutable variable 'name'.
```

A `let` variable needs a value right away. A `let mut` variable may be
declared without one and starts as `nil`:

```rune
let mut lastLogin;
print(lastLogin);    // nil
lastLogin = "2026-11-16";
print(lastLogin);    // 2026-11-16
```

```rune
let limit;    // Error at 'limit': Immutable variable must be initialized.
```

## Immutable Arrays, Maps and Structs

`let` freezes the whole value, not only the variable. Through a `let`
variable you cannot change an element, a key or a field, however deep,
nor call a method that changes the value in place, such as `push`, `pop`,
`sort` or a `mut self` method:

```rune
let config = ["db": ["hosts": ["db1"]]];
config["db"]["hosts"][0] = "db2";    // Error at 'config': Cannot mutate immutable variable 'config'.
```

```rune
let scores = [70, 95];
scores.push(80);    // Error at 'scores': Cannot mutate immutable variable 'scores'.
```

Methods that return a new value, such as `map`, `filter` or a spread, work
on any variable:

```rune
let scores = [70, 95];
let more = [...scores, 80];
print(more);      // [70, 95, 80]
print(scores);    // [70, 95]
```

To change it in place, declare it `let mut`:

```rune
let mut scores = [70, 95];
scores.push(80);
print(scores);    // [70, 95, 80]
```

Since every value is [copied](values.md#copies) where it goes, no other
variable or function can change a `let` value behind your back either.
Copying a `let` array into a `let mut` variable gives you a separate array
you can change freely:

```rune
let scores = [70, 95];
let mut draft = scores;
draft.push(80);
print(draft);     // [70, 95, 80]
print(scores);    // [70, 95]
```

## Parameters and Loop Variables

Function parameters, `for` loop variables and `catch` variables are
immutable too, and so is a function's own name:

```rune
fun retry(attempts) {
    attempts -= 1;    // Error at 'attempts': Cannot assign to immutable variable 'attempts'.
}
```

Marking a parameter or loop variable `mut` lets you change it, and the
change is written back where the value came from: to the caller's
variable for a [`mut` parameter](functions.md#mut-parameters), to the
element of the collection for [`for mut`](loops.md#for-mut).

```rune
fun addTag(mut tags, tag) {
    tags.push(tag);
}

let mut tags = ["new"];
addTag(tags, "sale");
print(tags);    // ["new", "sale"]
```

```rune
let mut prices = [10, 20, 30];
for mut p in prices {
    p = p * 2;
}
print(prices);    // [20, 40, 60]
```

## Constants

`const` names a value fixed before the program runs. Use it for limits,
settings and lookup tables, so that a magic number has a name and lives in
one place:

```rune
const KB = 1 << 10;
const MAX_UPLOAD = 4 * KB;
const UNIT = "bytes";
const LIMITS = ["body": 8 * KB, "header": KB];

print("max upload: {MAX_UPLOAD} {UNIT}");    // max upload: 4096 bytes
print(LIMITS["body"]);                       // 8192
```

A constant's value must be computable by the compiler. It may be:

- a number, string, `true`, `false` or `nil`;
- another constant, or operators applied to constants (`4 * KB`);
- an enum variant;
- `set()`, or an array or map literal made of such values.

Reading a constant costs nothing at run time: each use is replaced by the
value itself.

```rune
let size = 4;
const LIMIT = size * 2;    // Error at 'LIMIT': Constant 'LIMIT' must be a constant: a literal, a const or operators on them, an enum variant, or an array, map or set() of them.
```

```rune
const LIMIT = 10;
LIMIT = 20;    // Error at 'LIMIT': Cannot assign to constant 'LIMIT'.
```

Constants are declared at the top level of a file. They can be used before
the line that declares them, serve as a struct field's default, and be
[exported](../programs/modules.md) from a module:

```rune
struct Request { url, retries = MAX_RETRIES }

print(Request { url: "/orders" });    // Request {url: "/orders", retries: 3}

const MAX_RETRIES = 3;
```

In a [pattern](pattern-matching.md#patterns), a constant's name matches its
value:

```rune
const ADMIN = "admin";

fun greet(role) {
    match role {
        ADMIN => "welcome back, boss",
        _ => "hello",
    }
}

print(greet("admin"));    // welcome back, boss
print(greet("guest"));    // hello
```

An array or map constant cannot be used as a pattern.

## Scopes

A variable lives in the block that declares it. An inner block may declare
a variable with the same name, which hides the outer one until the block
ends; it may even differ in mutability:

```rune
let total = 10;
{
    let mut total = 0;
    total += 5;
    print(total);    // 5
}
print(total);        // 10
```

Declaring the same name twice in one scope is an error:

```rune
let total = 10;
let total = 20;    // Error at 'total': Variable with this name already declared in this scope.
```

Built-in functions such as `int` and `print` are not declarations, so a
program may reuse their names. The built-in is hidden from then on:

```rune
let int = 3;
print(int);    // 3
```

## Discarding with _

`_` is a name that throws its value away. Use it for a value you must accept
but do not need: a parameter, a `for` index, a part of a pattern. It can
appear any number of times, and can never be read.

```rune
fun onEvent(_, payload) { payload["id"] }
print(onEvent("click", ["id": 7]));    // 7

for _, name in ["ann", "bob"] {
    print(name);
}
// ann
// bob
```

```rune
let _ = 5;
print(_);    // Error at '_': Cannot read '_'.
```
