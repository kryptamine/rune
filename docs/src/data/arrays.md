# Arrays

An array is an ordered list of values. Write one with square brackets; the
elements may be of any type, mixed freely.

```rune
let langs = ["go", "rust", "rune"];
print(langs);          // ["go", "rust", "rune"]
print(langs.len());    // 3

let row = [1, 2.5, "three", true, nil, [4]];
print(row);            // [1, 2.5, "three", true, nil, [4]]
```

## Indexing

`xs[i]` reads the element at index `i`, counting from `0`. A negative index
counts from the end, so `xs[-1]` is the last element. Arrays nest, and
indexes chain.

```rune
let langs = ["go", "rust", "rune"];
print(langs[0]);       // go
print(langs[-1]);      // rune

let grid = [[1, 2], [3, 4]];
print(grid[1][0]);     // 3
```

An index must be an integer from `-xs.len()` to `xs.len() - 1`. Anything
outside is an `IndexError`, for reading and writing alike:

```rune
let langs = ["go", "rust", "rune"];
print(langs[3]); // IndexError: Index out of bounds: 3 of 3
```

When an element may be missing, read it with `?.[i]`, which gives `nil`
instead, and pick a fallback with `??` (see [Nil Safety](../language/nil-safety.md)):

```rune
let args = ["report.csv"];
print(args?.[0] ?? "stdin");    // report.csv
print(args?.[1] ?? "out.txt");  // out.txt
```

## Slices

`xs[a..b]` is a new array of the elements from index `a` up to, but not
including, `b`; `xs[a..=b]` includes `b`. Either bound may be left out,
and a negative bound counts from the end. Bounds past either end are
clamped, so a slice is never an error, though it may be empty.

```rune
let xs = [10, 20, 30, 40, 50];
print(xs[1..3]);    // [20, 30]
print(xs[1..=3]);   // [20, 30, 40]
print(xs[..2]);     // [10, 20]
print(xs[-2..]);    // [40, 50]
print(xs[..-1]);    // [10, 20, 30, 40]
print(xs[3..99]);   // [40, 50]
```

`x in xs` tests whether an element is in the array, comparing with `==`
(see [Membership](../language/values.md#membership)):

```rune
let admins = ["ann", "bob"];
print("ann" in admins);    // true
print("eve" in admins);    // false
```

## Changing an Array

Changing an array needs a mutable variable (see
[Variables](../language/variables.md)). Assign to an index to replace an
element, and `push` to add one at the end; an index cannot grow the array.

```rune
let mut scores = [70, 85, 90];
scores[0] = 75;
scores[1] += 5;
scores.push(100);
print(scores);    // [75, 90, 90, 100]
print(scores.pop());    // 100
print(scores);    // [75, 90, 90]
```

`insert`, `remove`, `sort`, `reverse` and the other
[array methods](../stdlib/array.md) are listed in the standard library.

## Iterating

`for x in xs` walks the elements; `for i, x in xs` gives the index too.

```rune
let todo = ["write docs", "fix bug", "release"];
for i, task in todo {
    print("{i + 1}. {task}");
}
// 1. write docs
// 2. fix bug
// 3. release
```

The loop walks the array as it was when the loop began, so pushing inside
the body does not make it run longer:

```rune
let mut queue = ["build"];
for job in queue {
    queue.push("after " + job);
}
print(queue);    // ["build", "after build"]
```

## Equality

`==` compares arrays element by element, nested ones included. Order
matters.

```rune
print([1, [2, 3]] == [1, [2, 3]]);    // true
print([1, 2] == [2, 1]);              // false
```

## Arrays Are Values

An array behaves like a number or a string: assigning it, passing it to a
function, returning it or storing it somewhere gives an independent copy.
Changing the copy never changes the original.

```rune
let defaults = ["read"];
let mut perms = defaults;
perms.push("write");
print(defaults);    // ["read"]
print(perms);       // ["read", "write"]
```

So a function cannot change the caller's array by accident. It works on
its own copy and returns the result:

```rune
fun withTax(prices) {
    let mut out = prices;
    for i, p in out {
        out[i] = p * 1.2;
    }
    out
}

let prices = [10, 25];
print(withTax(prices));    // [12, 30]
print(prices);             // [10, 25]
```

When a function should change the caller's array, mark the parameter
[`mut`](../language/functions.md#mut-parameters):

```rune
fun addTag(mut tags, tag) {
    if !(tag in tags) {
        tags.push(tag);
    }
}

let mut tags = ["new"];
addTag(tags, "sale");
addTag(tags, "sale");
print(tags);    // ["new", "sale"]
```

Copies are cheap: an array is copied only when it is written and something
else still holds it, so passing a large array around costs nothing.

A write along a path, as in `grid[0][1] = 9` or `grid[0].push(9)`, changes
only the array it goes through. Copies taken earlier of any part keep their
values:

```rune
let mut grid = [[0, 0], [0, 0]];
let row = grid[0];
let before = grid;
grid[0][1] = 9;
print(grid);      // [[0, 9], [0, 0]]
print(row);       // [0, 0]
print(before);    // [[0, 0], [0, 0]]
```

## Transforming

`map`, `filter`, `reduce` and friends return new arrays, which makes
chains of them read like a pipeline. All of them are in
[Array Methods](../stdlib/array.md).

```rune
let orders = [
    { id: 1, total: 30 },
    { id: 2, total: 120 },
    { id: 3, total: 75 },
];

let big = orders.filter(o => o.total > 50).map(o => o.id);
print(big);                                          // [2, 3]
print(orders.reduce((sum, o) => sum + o.total, 0));  // 225
```
