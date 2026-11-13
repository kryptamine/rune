# Variables

`let` declares an immutable variable, `let mut` a mutable one. Assigning to an
immutable variable (`=`, `+=`, `++`, ...) is a compile-time error. An immutable
variable must be initialized; `let mut x;` starts as `nil`.

```rune
let name = "ann";
let mut visits = 0;
visits += 1;
visits++;
print("{name}: {visits}");    // ann: 2

let mut lastSeen;
print(lastSeen);              // nil
```

```rune
let name = "ann";
name = "bob";    // Error at 'name': Cannot assign to immutable variable 'name'.
```

```rune
let limit;       // Error at 'limit': Immutable variable must be initialized.
```

## Immutable Arrays and Objects

Through an immutable variable the array, object or struct it holds cannot
be changed either: index and property assignment (`a[0] = 1`, `o.x += 1`,
`a[0]++`, also nested like `o.xs[0] = 1`) and calls of the in-place
methods (`push`, `pop`, `sort`, `map.set`, `set.add`, `delete`, `clear`,
...) and of `mut self` methods are compile errors. Taking an in-place
method as a value is a runtime `TypeError`. Methods that return a new value
(`append`, `map`, ...) work as usual.

```rune
let config = { db: { hosts: ["db1"] } };
config.db.hosts[0] = "db2";    // Error at 'config': Cannot mutate immutable variable 'config'.
```

```rune
let scores = [70, 95];
print(scores.append(80));       // [70, 95, 80]
print(scores);                  // [70, 95]
scores.push(80);                // Error at 'scores': Cannot mutate immutable variable 'scores'.
```

```rune
let mut scores = [70, 95];
scores.push(80);
print(scores);                  // [70, 95, 80]
```

## Parameters and Loop Variables

Function parameters, `for`-`in` variables, `catch` variables and functions are
immutable too. Parameters and `for`-`in` variables are made mutable with `mut`:

```rune
fun addTag(mut tags, tag) {
    tags.push(tag);
}

let mut tags = ["new"];
addTag(tags, "sale");
print(tags);    // ["new", "sale"]

let mut grid = [[0, 0], [0, 0]];
for i, mut row in grid { row[i] = 1; }
print(grid);    // [[1, 0], [0, 1]]
```

```rune
fun retry(attempts) {
    attempts -= 1;    // Error at 'attempts': Cannot assign to immutable variable 'attempts'.
}
```

The check is static and follows the variable, not the value: an array held by
an immutable variable still changes when it is mutated through another,
mutable reference (`let mut b = a;` or a `mut` parameter).

```rune
let scores = [70, 95];
let mut alias = scores;
alias.push(80);
print(scores);    // [70, 95, 80]
```

## Constants

`const` names a value known before the program runs: a number, string,
`true`, `false` or `nil`, written as a literal or computed from other
constants with operators, or an array of such values. A constant costs
nothing to read: each use is the value itself. Constants are declared at
the top level only, can be used before their declaration, as a struct
field's default, and [exported](../programs/modules.md).

```rune
const KB = 1 << 10;
const LIMIT = 4 * KB;
const UNIT = "bytes";
const SIZES = [KB, LIMIT, [0, -KB]];

print("{LIMIT} {UNIT}");    // 4096 bytes
print(SIZES);               // [1024, 4096, [0, -1024]]
```

```rune
let size = 4;
const LIMIT = size * 2;    // Error at 'LIMIT': Constant 'LIMIT' must be a constant expression: literals, other constants, operators on them and arrays of these.
```

```rune
const LIMIT = 10;
LIMIT = 20;    // Error at 'LIMIT': Cannot assign to constant 'LIMIT'.
```

A constant array is one array, shared by every read. Like an array held
by a `let` it cannot be changed through its name, but still changes
through a mutable reference to it. An object is no constant; hold it in
a `let`.

In a [pattern](pattern-matching.md#patterns) a constant's name tests for
its value; an array constant cannot be one.

## Scopes

Redeclaring a variable in the same scope is an error, at the top level too.
A new scope may shadow a name with a different mutability.
Built-ins are not declarations, so a program may still redefine them.

```rune
let total = 10;
{
    let mut total = 0;
    total += 5;
    print(total);    // 5
}
print(total);        // 10

let type = 3;        // shadows the built-in type
print(type);         // 3
```

```rune
let total = 10;
let total = 20;    // Error at 'total': Variable with this name already declared in this scope.
```

## Discarding with _

`_` declares nothing: in `let`, a parameter, a `for`-`in` or `catch`
variable or a pattern it takes a value and drops it, so it may appear any
number of times. It cannot be read, assigned or exported.

```rune
fun second(_, b, _) { return b; }
print(second(1, 2, 3));    // 2

let _ = 1;
let _ = 2;
for _, x in ["a"] { print(x); }    // a
```

```rune
let _ = 5;
print(_);    // Error at '_': Cannot read '_'.
```
