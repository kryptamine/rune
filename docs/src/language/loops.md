# Loops

There are two loops: `for x in ...` and `while`. There is no C-style `for`:
count with a range.

## for-in

`for x in arr { ... }` visits array elements, string characters or object keys
(objects are iterated in sorted key order). With two variables you also get
the index, or the key and the value.

```rune
for i, name in ["ann", "bob"] {
    print "{i + 1}. {name}";
}
// 1. ann
// 2. bob

let stock = { pears: 0, apples: 5 };
for fruit, count in stock {
    print "{fruit}: {count}";
}
// apples: 5
// pears: 0

for ch in "hé!" { print ch; }
// h
// é
// !
```

It also walks [maps and sets](../data/map-and-set.md),
[bytes](../data/bytes.md), [struct fields](../data/structs.md) and
[streams](../stdlib/streams.md).

## break and continue

`continue` skips to the next iteration, `break` leaves the loop; both work in
`for` and `while`.

```rune
let orders = [{ id: 1, total: 30 }, { id: 2, total: 0 }, { id: 3, total: 120 }];

for order in orders {
    if order.total == 0 continue;
    if order.total > 100 {
        print "big order {order.id}, stopping";
        break;
    }
    print "order {order.id}: {order.total}";
}
// order 1: 30
// big order 3, stopping
```

A body is a block, or a bare statement starting with a name or keyword:
`if order.total == 0 continue;`, `for x in xs out.push(x);`.

## Ranges

Count with `a..b` (end excluded) or `a..=b` (end included), optionally
`step n`; a `for`-`in` walks it without building the array. Outside a loop
a range is an array.

```rune
for i in 1..4 { print i; }              // 1 2 3
for i in 1..=3 { print i; }             // 1 2 3
for i in 10..0 step -3 { print i; }     // 10 7 4 1
for x in 0..1 step 0.25 { print x; }    // 0 0.25 0.5 0.75

print 1..4;            // [1, 2, 3]
print 0..10 step 5;    // [0, 5]
```

## What a Loop Sees

Array elements are taken when the loop starts, so pushing to the array
inside the loop does not extend it. Each iteration has its own variable, so
closures capture the current element.

```rune
let mut queue = [1, 2];
for x in queue { queue.push(x * 10); }
print queue;    // [1, 2, 10, 20]

let mut handlers = [];
for name in ["save", "load"] {
    handlers.push(() => "clicked " + name);
}
print handlers[0]();    // clicked save
print handlers[1]();    // clicked load
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
print steps;    // 111
```
