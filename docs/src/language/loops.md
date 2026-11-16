# Loops

There are two loops: `for x in ...` and `while`. There is no C-style `for`:
count with a range.

## for-in

`for x in arr { ... }` visits array elements, string characters or object keys,
in the order they were added. With two variables you also get the index, or
the key and the value.

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

for ch in "hé!" { print(ch); }
// h
// é
// !
```

It also walks [maps and sets](../data/map-and-set.md),
[bytes](../data/bytes.md) and [streams](../stdlib/streams.md); a
struct's fields are walked as an object, `for k, v in { ...s }`.

## break and continue

`continue` skips to the next iteration, `break` leaves the loop; both work in
`for` and `while`.

```rune
let orders = [{ id: 1, total: 30 }, { id: 2, total: 0 }, { id: 3, total: 120 }];

for order in orders {
    if order.total == 0 { continue; }
    if order.total > 100 {
        print("big order {order.id}, stopping");
        break;
    }
    print("order {order.id}: {order.total}");
}
// order 1: 30
// big order 3, stopping
```

A body is always a block: `if order.total == 0 { continue; }`.

## Ranges

Count with `a..b` (end excluded) or `a..=b` (end included), optionally
`step n`; a `for`-`in` walks it, and `x in a..b` tests it, without building
the array. Anywhere else a range is an array. In brackets it
[slices](../data/arrays.md), `xs[1..]`, and in a `match` it is a
[pattern](pattern-matching.md#patterns), `200..300 => "ok"`. `..` binds looser than arithmetic
and bitwise operators and tighter than comparisons, so `0..n + 1`,
`x in 0..10` and `xs == 1..3` need no parentheses; a `??` or `? :` in a
bound does.

```rune
for i in 1..4 { print(i); }              // 1 2 3
for i in 1..=3 { print(i); }             // 1 2 3
for i in 10..0 step -3 { print(i); }     // 10 7 4 1
for x in 0..1 step 0.25 { print(x); }    // 0 0.25 0.5 0.75

print(5 in 1..10);      // true
print(1..4);            // [1, 2, 3]
print([1, 2] == 1..3);  // true
print(0..10 step 5);    // [0, 5]
```

## What a Loop Sees

Array elements are taken when the loop starts, so pushing to the array
inside the loop does not extend it. Each iteration has its own variable, so
closures capture the current element.

```rune
let mut queue = [1, 2];
for x in queue { queue.push(x * 10); }
print(queue);    // [1, 2, 10, 20]

let mut handlers = [];
for name in ["save", "load"] {
    handlers.push(() => "clicked " + name);
}
print(handlers[0]());    // clicked save
print(handlers[1]());    // clicked load
```

## for mut

`for mut x in xs` gives each iteration its own copy of the element, and
writes it back into `xs` when the iteration ends: at the end of the body,
on `continue`, `break`, `return` and on a throw. It walks arrays and the
values of objects and maps.

```rune
let mut grid = [[0, 0], [0, 0]];
for i, mut row in grid { row[i] = 1; }
print(grid);    // [[1, 0], [0, 1]]

let mut prices = { apple: 1, pear: 2 };
for k, mut v in prices { v *= 10; }
print(prices);  // {apple: 10, pear: 20}
```

The collection must be a mutable place, as for a
[`mut` parameter](functions.md#mut-parameters), found once before the
loop. The loop walks it as it was when the loop started; a write the body
makes to it directly is overwritten by the write-back of that element.
The body cannot assign the variable the collection is read from, and a
key cannot be `mut`.

```rune
let mut xs = [[0]];
for mut x in xs { xs = []; } // Error at 'xs': Cannot assign to 'xs' inside a 'for mut' loop over it.
let rows = [[0]];
for mut row in rows {}  // Error at 'rows': Cannot write 'for mut' variable 'row' back into immutable variable 'rows'.
for mut i in 0..3 {}    // Error at 'i': A 'for mut' loop walks a mutable variable, element or property, which it writes back into.
```

## while

Anything else is a `while`:

```rune
let mut n = 27;
let mut steps = 0;
while n != 1 {
    n = n % 2 == 0 ? math.idiv(n, 2) : 3 * n + 1;
    steps++;
}
print(steps);    // 111
```
