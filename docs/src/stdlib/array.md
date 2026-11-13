# Array Methods

Methods marked **in place** change the array, so it must be held by a
mutable variable, and return `nil`, unless they take an element out; the
others return a new value and leave the array as it is.

## Adding and removing

- `push(v1, ...)` — appends **in place** (amortized O(1) each).
- `pop()` — removes the last element in place and returns it; an empty array is a `ValueError`.
- `insert(i, v)` — inserts `v` before index `i` **in place** (`xs.len()` appends).
- `remove(i)` — removes element `i` in place and returns it. A negative `i` counts from the end; one out of bounds is an `IndexError`.
- `append(v1, ...)` — a new array with the values appended. It copies, so building an array with it in a loop is quadratic; use `push` for that.

```rune
let mut queue = ["build", "test"];
queue.push("deploy");
print(queue.pop());                 // deploy
queue.insert(0, "lint");
print(queue);                       // ["lint", "build", "test"]
print(queue.remove(-1));            // test
print(queue.append("release"));     // ["lint", "build", "release"]
print(queue);                       // ["lint", "build"]
```

```rune
let mut empty = [];
empty.pop(); // ValueError: array.pop() got an empty array.
```

## Transforming

`map(f)`, `filter(f)` and `reduce(f[, init])` return new values; `f` may be
any function, built-ins included. Without `init`, `reduce` starts from the
first element.

```rune
let orders = [
    { id: 1, total: 30 },
    { id: 2, total: 120 },
    { id: 3, total: 75 },
];

print(orders.map(o => o.total));                         // [30, 120, 75]
print(orders.filter(o => o.total > 50).map(o => o.id));  // [2, 3]
print(orders.reduce((sum, o) => sum + o.total, 0));      // 225
print([3, 1, 2].reduce((a, b) => a + b));                // 6
print(["10", "2"].map(int));                             // [10, 2]
```

## Sorting

`sort([cmp])` sorts **in place**, stably; `reverse()` and the rest return a
new array.
Without `cmp` the elements must be all numbers or all strings; with it,
`cmp(a, b) < 0` puts `a` first.

```rune
let mut scores = [5, 3, 10, 1];
scores.sort();
print(scores);                            // [1, 3, 5, 10]

let mut fruits = ["pear", "Apple", "fig"];
fruits.sort();
print(fruits);                            // ["Apple", "fig", "pear"]
fruits.sort((a, b) => a.len() - b.len());
print(fruits);                            // ["fig", "pear", "Apple"]

let mut people = [{ name: "bob", age: 25 }, { name: "ann", age: 31 }];
people.sort((a, b) => b.age - a.age);
print(people.map(({ name }) => name));    // ["ann", "bob"]
```

```rune
let mut mixed = [1, "a"];
mixed.sort(); // TypeError: array.sort() without a comparator needs all numbers or all strings.
```

## Searching

- `every(f)`, `some(f)` — whether `f` is truthy for all or any element, stopping once that is known.
- `find(f)` — the first element for which it is, or `nil`; `findIndex(f)` — its index, or `-1`.
- `contains(item)`, `indexOf(item)` — by `==`, so arrays and objects compare by content; `indexOf` is `-1` if missing.

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
print([10, 20, 30].contains(20));               // true
print([10, 20, 30].indexOf(99));                // -1
print([[1, 2], [3, 4]].indexOf([3, 4]));        // 1
```

## Slicing and joining

`slice(start[, end])` (negative bounds count from the end), `reverse()`,
`repeat(n)` and `join(sep)` return new values. `repeat` makes an array of a
given size; its elements are shared, not copied, so `[[]].repeat(3)` holds
one array three times.

```rune
let xs = [10, 20, 30, 40, 50];
print(xs.slice(1, 3));   // [20, 30]
print(xs.slice(-2));     // [40, 50]
print(xs.reverse());     // [50, 40, 30, 20, 10]
print([0].repeat(3));    // [0, 0, 0]
print(xs);               // [10, 20, 30, 40, 50]
print(["a", "b", "c"].join(", ")); // a, b, c
```
