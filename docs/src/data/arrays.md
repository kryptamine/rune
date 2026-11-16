# Arrays

An array is an ordered list of values of any type. `xs[i]` reads element
`i`, counting from `0`, and `xs.len()` is the number of elements.

```rune
let langs = ["go", "rust", "rune"];
print(langs[0]);     // go
print(langs.len());  // 3

let row = [1, 2.5, "three", true, nil, [4]];
print(row);          // [1, 2.5, "three", true, nil, [4]]

let grid = [[1, 2], [3, 4]];
print(grid[1][0]);   // 3
```

An index must be an integer from `-xs.len()` to `xs.len() - 1`; a negative one
counts from the end, as in strings, bytes, slices and `remove`. One out of
bounds is an `IndexError`, for reading and writing alike. To grow an array,
`push` to it.

```rune
let langs = ["go", "rust", "rune"];
print(langs[-1]); // rune
print(langs[3]);  // IndexError: Index out of bounds: 3 of 3
```

`xs[a..b]` is a new array of the elements from `a` up to `b`, `xs[a..=b]`
up to `b` included; either bound may be left out, and a negative one
counts from the end. Bounds past either end are clamped, so a slice is
never an error but may be empty. `x in xs` tests for an element (see
[Membership](../language/values.md#membership)).

```rune
let xs = [10, 20, 30, 40, 50];
print(xs[1..3]);    // [20, 30]
print(xs[1..=3]);   // [20, 30, 40]
print(xs[..2]);     // [10, 20]
print(xs[-2..]);    // [40, 50]
print(xs[..-1]);    // [10, 20, 30, 40]
print(xs[3..99]);   // [40, 50]
print(30 in xs);    // true
```

Writing an element needs a mutable variable (see [Variables](../language/variables.md)):

```rune
let mut scores = [70, 85, 90];
scores[0] = 75;
scores[1] += 5;
scores.push(100);
print(scores); // [75, 90, 90, 100]
```

`for x in xs` walks the elements, `for i, x in xs` the indices too. The
elements are taken when the loop starts, so pushing inside the loop does not
extend it.

```rune
let todo = ["write docs", "fix bug"];
for i, task in todo {
    print("{i + 1}. {task}");
}
// 1. write docs
// 2. fix bug
```

`==` compares arrays element by element:

```rune
print([1, [2, 3]] == [1, [2, 3]]); // true
print([1, 2] == [2, 1]);           // false
```

Arrays are values, like numbers and strings: assigning one, passing it to
a function, returning it or storing it in an element or a property gives
an independent copy. `push`, `pop` and index assignment change only the
array they are called on.

```rune
let a = [1, 2];
let mut b = a;
b.push(3);
print(a); // [1, 2]
print(b); // [1, 2, 3]

fun grow(xs) { let mut ys = xs; ys.push(0); return ys; }
print(grow(a)); // [1, 2, 0]
print(a);       // [1, 2]
```

A copy costs nothing until it is written: the array is copied on its first
change, and only if something else still holds it.

A write along a path, as in `grid[0][1] = 9` or `grid[0].push(9)`, changes
only the array at the end of it: a copy taken earlier of any array on the
way keeps its value.

```rune
let mut grid = [[0, 0], [0, 0]];
let row = grid[0];
let before = grid;
grid[0][1] = 9;
print(row);    // [0, 0]
print(before); // [[0, 0], [0, 0]]
print(grid);   // [[0, 9], [0, 0]]
```

A function changes the caller's array only through a
[`mut` parameter](../language/functions.md#mut-parameters).

`map`, `filter`, `sort`, `join` and the rest are
[array methods](../stdlib/array.md).

```rune
let prices = [12, 5, 30, 8];
print(prices.map(p => p * 2));           // [24, 10, 60, 16]
print(prices.filter(p => p > 10));       // [12, 30]
print(prices.reduce((a, b) => a + b));   // 55
```
