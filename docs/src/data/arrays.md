# Arrays

An array is an ordered list of values of any type. `xs[i]` reads element
`i`, counting from `0`, and `xs.len()` is the number of elements.

```rune
let langs = ["go", "rust", "rune"];
print(langs[0]);     // go
print(langs.len());   // 3

let row = [1, 2.5, "three", true, nil, [4]];
print(row);          // [1, 2.5, "three", true, nil, [4]]

let grid = [[1, 2], [3, 4]];
print(grid[1][0]);   // 3
```

An index must be an integer from `-xs.len()` to `xs.len() - 1`; a negative one
counts from the end, as in strings, bytes, `slice` and `remove`. One out of
bounds is an `IndexError`, for reading and writing alike. To grow an array,
`push` to it.

```rune
let langs = ["go", "rust", "rune"];
print(langs[-1]); // rune
print(langs[3]);  // IndexError: Index out of bounds: 3 of 3
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

Arrays are references: assigning an array or passing it to a function does
not copy it, so `push`, `pop` and index assignment are visible through every
reference. `[...xs]` or `xs.slice(0)` makes a copy.

```rune
let a = [1, 2];
let mut b = a;
b.push(3);
print(a); // [1, 2, 3]

let mut c = [...a];
c.push(4);
print(a); // [1, 2, 3]
print(c); // [1, 2, 3, 4]
```

`slice`, `map`, `filter`, `sort`, `join` and the rest are
[array methods](../stdlib/array.md).

```rune
let prices = [12, 5, 30, 8];
print(prices.slice(1, 3));               // [5, 30]
print(prices.filter(p => p > 10));       // [12, 30]
print(prices.reduce((a, b) => a + b));   // 55
```
