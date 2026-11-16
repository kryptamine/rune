# Array Methods

Most methods return a new value and leave the array as it is. The ones
marked **in place** change the array itself. Those need an array held by a
`let mut` variable (or another mutable place), and return `nil` unless they
take an element out.

| Method | Does |
|---|---|
| `len()` | number of elements |
| `push(v, ...)` | appends, **in place** |
| `pop()` | removes and returns the last element, **in place** |
| `insert(i, v)` | inserts `v` before index `i`, **in place** |
| `remove(i)` | removes and returns element `i`, **in place** |
| `sort([cmp])` | sorts, **in place** |
| `reverse()` | reverses, **in place** |
| `map(f)`, `filter(f)`, `reduce(f[, init])` | transform |
| `every(f)`, `some(f)`, `find(f)`, `findIndex(f)`, `indexOf(v)` | search |
| `join(sep)`, `repeat(n)` | build strings and arrays |

Arrays are values, so changing one in place never changes a copy of it:

```rune
let mut todo = ["write docs", "fix bug"];
let snapshot = todo;
todo.push("ship");
print(todo);       // ["write docs", "fix bug", "ship"]
print(snapshot);   // ["write docs", "fix bug"]
```

## Adding and removing

- `push(v, ...)` appends one or more values. It is amortized O(1).
- `pop()` removes the last element and returns it. On an empty array it is
  a `ValueError`.
- `insert(i, v)` inserts `v` before index `i`; `xs.insert(xs.len(), v)`
  appends.
- `remove(i)` removes element `i` and returns it. A negative `i` counts
  from the end.

```rune
let mut queue = ["build", "test"];
queue.push("deploy", "notify");
print(queue);              // ["build", "test", "deploy", "notify"]
print(queue.pop());        // notify
queue.insert(0, "lint");
print(queue);              // ["lint", "build", "test", "deploy"]
print(queue.remove(1));    // build
print(queue.remove(-1));   // deploy
print(queue);              // ["lint", "test"]
```

To get a new, longer array and keep the old one, use
[spread](../language/spread-and-rest.md):

```rune
let steps = ["lint", "test"];
print([...steps, "release"]);   // ["lint", "test", "release"]
print(["setup", ...steps]);     // ["setup", "lint", "test"]
print(steps);                   // ["lint", "test"]
```

An empty array has nothing to pop:

```rune
let mut empty = [];
empty.pop(); // ValueError: array.pop() got an empty array.
```

An index past the end is an `IndexError`:

```rune
let mut xs = ["a"];
xs.remove(5); // IndexError: array.remove() index 5 out of bounds of 1 element.
```

## Transforming

- `map(f)` is a new array of `f(x)` for each element.
- `filter(f)` keeps the elements for which `f(x)` is truthy.
- `reduce(f, init)` folds the array into one value: it starts from `init`
  and calls `f(acc, x)` for each element. Without `init` it starts from the
  first element; an empty array then is a `ValueError`.

```rune
let orders = [
    { id: 1, customer: "ann", total: 30 },
    { id: 2, customer: "bob", total: 120 },
    { id: 3, customer: "ann", total: 75 },
];

print(orders.map(o => o.total));                          // [30, 120, 75]
print(orders.filter(o => o.total > 50).map(o => o.id));   // [2, 3]
print(orders.reduce((sum, o) => sum + o.total, 0));       // 225
print([3, 1, 2].reduce((a, b) => a + b));                 // 6
```

`reduce` can build any value, such as an object of totals by customer:

```rune
let orders = [
    { customer: "ann", total: 30 },
    { customer: "bob", total: 120 },
    { customer: "ann", total: 75 },
];

let byCustomer = orders.reduce((acc, o) => {
    let mut next = acc;
    next[o.customer] = (acc?.[o.customer] ?? 0) + o.total;
    next
}, {});
print(byCustomer);   // {ann: 105, bob: 120}
```

`f` can be any function, built-ins included. `map`, `filter`, `every`,
`some`, `find` and `findIndex` also pass the index as a second argument,
but only when `f` requires two. So `map(int)`, whose `base` is optional,
gets the element alone:

```rune
print(["10", "2", "33"].map(int));               // [10, 2, 33]
print(["a", "b"].map((x, i) => "{i}:{x}"));      // ["0:a", "1:b"]
print([5, 6, 7, 8].filter((_, i) => i % 2 == 0)); // [5, 7]
```

## Sorting

`sort()` sorts in place, and is stable: equal elements keep their order.
Without a comparator, the elements must be all numbers or all strings;
strings sort by code point, so uppercase comes before lowercase.

`sort(cmp)` takes a comparator: `cmp(a, b)` below 0 puts `a` first, above
0 puts `b` first.

`reverse()` reverses in place.

```rune
let mut scores = [5, 3, 10, 1];
scores.sort();
print(scores);                           // [1, 3, 5, 10]
scores.reverse();
print(scores);                           // [10, 5, 3, 1]

let mut fruits = ["pear", "Apple", "fig"];
fruits.sort();
print(fruits);                           // ["Apple", "fig", "pear"]
fruits.sort((a, b) => a.len() - b.len());
print(fruits);                           // ["fig", "pear", "Apple"]
```

Sorting records by a field, oldest first:

```rune
let mut people = [
    { name: "bob", age: 25 },
    { name: "ann", age: 31 },
    { name: "eve", age: 25 },
];
people.sort((a, b) => b.age - a.age);
print(people.map(p => p.name));   // ["ann", "bob", "eve"]
```

To keep the original, sort a copy. A copy is just another variable:

```rune
let prices = [30, 10, 20];
let mut sorted = prices;
sorted.sort();
print(sorted);   // [10, 20, 30]
print(prices);   // [30, 10, 20]
```

Mixed elements need a comparator:

```rune
let mut mixed = [1, "a"];
mixed.sort(); // TypeError: array.sort() without a comparator needs all numbers or all strings.
```

## Searching

- `every(f)`, `some(f)` — whether `f` is truthy for all elements, or for
  any. They stop as soon as the answer is known.
- `find(f)` — the first element for which `f` is truthy, or `nil`.
- `findIndex(f)` — the index of that element, or `nil`.
- `indexOf(v)` — the index of the first element equal to `v`, or `nil`.
  Arrays and objects compare by content.

To ask only whether a value is there, use [`in`](../language/values.md#membership).

```rune
let users = [
    { name: "ann", admin: true },
    { name: "bob", admin: false },
];

print(users.some(u => u.admin));                // true
print(users.every(u => u.admin));               // false
print(users.find(u => !u.admin));               // {name: "bob", admin: false}
print(users.find(u => u.name == "eve"));        // nil
print(users.findIndex(u => u.name == "bob"));   // 1

print(20 in [10, 20, 30]);                      // true
print([10, 20, 30].indexOf(30));                // 2
print([10, 20, 30].indexOf(99));                // nil
print([[1, 2], [3, 4]].indexOf([3, 4]));        // 1
```

`find` returns `nil` when nothing matches, so `??` gives a fallback:

```rune
let users = [{ name: "ann" }, { name: "bob" }];
let eve = users.find(u => u.name == "eve") ?? { name: "guest" };
print(eve.name);   // guest
```

## Joining, repeating and slicing

- `join(sep)` is a string of the elements, as `print` shows them, with
  `sep` between.
- `repeat(n)` is an array of `n` copies of the elements. Each copy is its
  own value, so changing one leaves the others.
- A [slice](../data/arrays.md), `xs[a..b]`, is a new array of part of
  `xs`.

```rune
print(["a", "b", "c"].join(", "));   // a, b, c
print([2026, 11, 12].join("-"));     // 2026-11-12
print([0].repeat(3));                // [0, 0, 0]
print(["-", "+"].repeat(2));         // ["-", "+", "-", "+"]

let xs = [10, 20, 30, 40, 50];
print(xs[1..3]);                     // [20, 30]
print(xs[-2..]);                     // [40, 50]
print(xs[..2]);                      // [10, 20]
```
