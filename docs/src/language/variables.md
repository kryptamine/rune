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
methods (`push`, `pop`, `sort`, `reverse`, `set.add`, `delete`, `clear`,
...) and of `mut self` methods are compile errors. Taking an in-place
method as a value is a runtime `TypeError`. Methods that return a new value
(`map`, `filter`, ...) work as usual. As every value is
[copied](values.md#copies), nothing else can change it either: what a
`let` holds never changes.

```rune
let config = { db: { hosts: ["db1"] } };
config.db.hosts[0] = "db2";    // Error at 'config': Cannot mutate immutable variable 'config'.
```

```rune
let scores = [70, 95];
print([...scores, 80]);         // [70, 95, 80]
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
immutable too. A `mut` parameter or `for`-`in` variable writes its value back
where it came from: the caller's argument (see
[Functions](functions.md#mut-parameters)) or the element of the collection
(see [Loops](loops.md#for-mut)).

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

A container is a value too, so `let mut b = a;` gives `b` a copy of its
own: changing `b` leaves `a` as it was (see [Copies](values.md#copies)).

```rune
let scores = [70, 95];
let mut copy = scores;
copy.push(80);
print(scores);    // [70, 95]
```

## Constants

`const` names a value known before the program runs: a number, string,
`true`, `false` or `nil`, written as a literal or computed from other
constants with operators, an enum variant, `map()` or `set()`, or an array
or object of such values. A constant costs
nothing to read: each use is the value itself. Constants are declared at
the top level only, can be used before their declaration, as a struct
field's default, and [exported](../programs/modules.md).

```rune
const KB = 1 << 10;
const LIMIT = 4 * KB;
const UNIT = "bytes";
const SIZES = [KB, LIMIT, [0, -KB]];
const UNITS = { KB, MB: KB * KB, limits: { body: 8 * KB } };

print("{LIMIT} {UNIT}");    // 4096 bytes
print(SIZES);               // [1024, 4096, [0, -1024]]
print(UNITS.limits.body);   // 8192
```

```rune
let size = 4;
const LIMIT = size * 2;    // Error at 'LIMIT': Constant 'LIMIT' must be a constant: a literal, a const or operators on them, an enum variant, or an array, object, map() or set() of them.
```

```rune
const LIMIT = 10;
LIMIT = 20;    // Error at 'LIMIT': Cannot assign to constant 'LIMIT'.
```

A constant never changes: each read is a copy of it. Spreads and computed
keys are not constant.

In a [pattern](pattern-matching.md#patterns) a constant's name tests for
its value; an array or object constant cannot be one.

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

let int = 3;         // shadows the built-in int
print(int);          // 3
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
