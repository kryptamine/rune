# Loops

Rune has two loops: `for x in ...` walks a collection or a range, and
`while` repeats while a condition holds. There is no C-style
`for (i = 0; i < n; i++)`; count with a [range](#ranges) instead.

## for-in

`for x in xs { ... }` runs the body once per element:

```rune
let tasks = ["write tests", "fix bug", "deploy"];

for task in tasks {
    print("- {task}");
}
// - write tests
// - fix bug
// - deploy
```

With two variables you get the position as well. Over an array that is the
index, over an object the key and the value, in the order they were added:

```rune
for i, name in ["ann", "bob"] {
    print("{i + 1}. {name}");
}
// 1. ann
// 2. bob

let stock = { pears: 0, apples: 5 };
for fruit, count in stock {
    print("{fruit}: {count}");
}
// pears: 0
// apples: 5
```

A string is walked character by character, not byte by byte:

```rune
for ch in "hé!" {
    print(ch);
}
// h
// é
// !
```

`for` also walks [maps and sets](../data/map-and-set.md),
[bytes](../data/bytes.md) and [streams](../stdlib/streams.md). To walk a
struct's fields, spread it into an object first: `for k, v in { ...s }`.

## break and continue

`continue` skips the rest of the body and goes to the next element.
`break` leaves the loop. Both work in `for` and `while`.

```rune
let orders = [
    { id: 1, total: 30 },
    { id: 2, total: 0 },
    { id: 3, total: 120 },
    { id: 4, total: 15 },
];

for order in orders {
    if order.total == 0 { continue }
    if order.total > 100 {
        print("order {order.id} needs review, stopping");
        break;
    }
    print("order {order.id}: {order.total}");
}
// order 1: 30
// order 3 needs review, stopping
```

## Ranges

`a..b` counts from `a` up to `b`, leaving `b` out. `a..=b` includes `b`.

```rune
for i in 1..4 {
    print(i);
}
// 1
// 2
// 3

for i in 1..=3 {
    print("lap {i}");
}
// lap 1
// lap 2
// lap 3
```

Add `step n` to skip ahead, or count down with a negative step. Steps may
be fractions:

```rune
for i in 10..0 step -3 {
    print(i);
}
// 10
// 7
// 4
// 1

for x in 0..1 step 0.25 {
    print(x);
}
// 0
// 0.25
// 0.5
// 0.75
```

`for` and `in` walk a range without building an array, so even a huge
range costs nothing. Anywhere else a range is just an array:

```rune
print(1..4);              // [1, 2, 3]
print(0..10 step 5);      // [0, 5]
print([1, 2] == 1..3);    // true
print(5 in 1..10);        // true
print(3 in 0..10 step 2); // false
```

Ranges show up in two more places: in brackets they
[slice](../data/arrays.md#slices), `xs[1..]`, and in a `match` they are
[patterns](pattern-matching.md#patterns), `200..300 => "ok"`.

`..` binds looser than arithmetic and tighter than comparisons, so
`0..n + 1`, `x in 0..10` and `xs == 1..3` need no parentheses. A `??` in a
bound does: `0..(limit ?? 10)`.

## What a Loop Sees

A `for` loop walks the collection as it was when the loop started.
Pushing to the array inside the loop does not make the loop longer:

```rune
let mut queue = [1, 2];
for x in queue {
    queue.push(x * 10);
}
print(queue);    // [1, 2, 10, 20]
```

Each iteration gets a fresh variable, so a closure made in the loop keeps
the element of its own iteration:

```rune
let mut handlers = [];
for name in ["save", "load"] {
    handlers.push(() => "clicked " + name);
}
print(handlers[0]());    // clicked save
print(handlers[1]());    // clicked load
```

## for mut

The loop variable is a copy, so changing it does not change the
collection. `for mut x in xs` makes it change: each iteration gets its own
copy of the element, and the copy is written back into `xs` when the
iteration ends, whether by reaching the end of the body, `continue`,
`break`, `return` or a throw. It works on arrays and on the values of
objects and maps.

```rune
let mut prices = { apple: 1.2, pear: 2.5 };
for fruit, mut price in prices {
    price *= 2;
}
print(prices);    // {apple: 2.4, pear: 5}

let mut grid = [[0, 0, 0], [0, 0, 0], [0, 0, 0]];
for i, mut row in grid {
    row[i] = 1;
}
print(grid);    // [[1, 0, 0], [0, 1, 0], [0, 0, 1]]
```

The collection must be a mutable place, the same rule as for a
[`mut` parameter](functions.md#mut-parameters). Writing back into a `let`
variable is a compile error:

```rune
let rows = [[0]];
for mut row in rows {} // Error at 'rows': Cannot write 'for mut' variable 'row' back into immutable variable 'rows'.
```

A range is not a place, so there is nothing to write back into:

```rune
for mut i in 0..3 {} // Error at 'i': A 'for mut' loop walks a mutable variable, element or property, which it writes back into.
```

The body cannot replace the collection it walks:

```rune
let mut xs = [[0]];
for mut x in xs { xs = []; } // Error at 'xs': Cannot assign to 'xs' inside a 'for mut' loop over it.
```

Only the value can be `mut`, never the index or key.

## while

`while` repeats its body as long as the condition holds. Use it when you
don't know the number of steps in advance:

```rune
let mut stack = ["home", "settings", "profile"];
while stack {
    print("closing {stack.pop()}");
}
// closing profile
// closing settings
// closing home
```

The condition is tested for [truthiness](values.md#truthiness), so
`while stack` runs until the array is empty. With a pattern,
`while queue.pop() is { id } { ... }` runs as long as the value matches;
see [Binding in Conditions](values.md#binding-in-conditions). Here is a loop that counts
the steps of the Collatz sequence from 27 to 1:

```rune
let mut n = 27;
let mut steps = 0;
while n != 1 {
    n = if n % 2 == 0 { math.idiv(n, 2) } else { 3 * n + 1 };
    steps += 1;
}
print(steps);    // 111
```
