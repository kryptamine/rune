# Map and Set

A **map** stores values by key; a **set** stores unique values. Use a map
to count, group or index things, and a set to remember what you have seen.

Both differ from an object in two ways: their keys may be any hashable
value, not just strings, and they are built up at run time rather than
written out field by field. Like an object, they keep keys in the order
they were first added.

```rune
let mut stock = map();
stock["apple"] = 3;
stock["pear"] = 0;
print(stock["apple"]);    // 3
print(stock);             // map {"apple": 3, "pear": 0}

let mut visitors = set();
visitors.add("ann");
visitors.add("bob");
visitors.add("ann");
print(visitors);          // set {"ann", "bob"}
print(visitors.len());    // 2
```

A classic use: counting words.

```rune
let words = "the cat saw the dog and the cat".split(" ");

let mut counts = map();
for w in words {
    counts[w] = (counts?.[w] ?? 0) + 1;
}
print(counts);    // map {"the": 3, "cat": 2, "saw": 1, "dog": 1, "and": 1}
```

## Making One

`map()` makes an empty map. `map(src)` makes one from an object, a struct,
another map, or an array of `[key, value]` pairs. `set()` and `set(array)`
work the same way; duplicates in the array are dropped.

```rune
print(map({ host: "localhost", port: 8080 }));  // map {"host": "localhost", "port": 8080}
print(map([[200, "OK"], [404, "Not Found"]]));  // map {200: "OK", 404: "Not Found"}
print(set(["a", "b", "a"]));                    // set {"a", "b"}
```

## Keys

A key may be any hashable value:

- `nil` and booleans;
- numbers, where `1` and `1.0` are the same key and `NaN` cannot be one;
- strings and bytes;
- functions and struct types;
- struct values whose fields are all hashable, compared by value.

A struct makes a natural composite key, such as a grid position:

```rune
struct Pos { x, y }

let mut grid = map();
grid[Pos { x: 0, y: 1 }] = "wall";
print(grid[Pos { x: 0, y: 1 }]);                // wall
print(grid?.[Pos { x: 9, y: 9 }] ?? "floor");   // floor
```

Arrays, objects, maps and sets cannot be keys. To key by a pair of values,
use a struct as above.

```rune
let mut seen = map();
seen[[1, 2]] = true; // TypeError: Cannot use an array as a key.
```

## Reading and Writing

`m[k]` reads a key; a missing key is a `KeyError`. `m?.[k]` gives `nil`
instead, so `m?.[k] ?? fallback` reads with a default. `m[k] = v` adds or
replaces a key, which needs `m` to be mutable.

```rune
let prices = map({ apple: 1.5 });
print(prices?.["kiwi"]);         // nil
print(prices?.["kiwi"] ?? 0);    // 0
```

```rune
let prices = map({ apple: 1.5 });
print(prices["kiwi"]); // KeyError: Key not found: "kiwi".
```

A value stored in a map can be changed in place through its key:

```rune
let mut groups = map();
groups["admins"] = ["ann"];
groups["admins"].push("bob");
print(groups);    // map {"admins": ["ann", "bob"]}
```

## Map Methods

- `k in m` — whether the key is there;
- `delete(k)` — removes a key and returns whether it was there;
- `clear()` — removes every key;
- `keys()`, `values()`, `entries()` — new arrays, in insertion order;
- `len()` — the number of keys.

```rune
let mut stock = map({ apple: 3, pear: 0, plum: 7 });

print("pear" in stock);         // true
print(stock.delete("pear"));    // true
print(stock.delete("pear"));    // false
print(stock.keys());            // ["apple", "plum"]
print(stock.values());          // [3, 7]
print(stock.entries());         // [["apple", 3], ["plum", 7]]
print(stock.len());             // 2
```

## Set Methods

- `x in s` — whether the element is there;
- `add(x)`, `delete(x)`, `clear()` — change the set in place;
- `union(s)`, `intersection(s)`, `difference(s)` — new sets;
- `len()` — the number of elements.

```rune
let backend = set(["ann", "bob", "eve"]);
let frontend = set(["bob", "kim"]);

print("eve" in backend);                // true
print(backend.union(frontend));         // set {"ann", "bob", "eve", "kim"}
print(backend.intersection(frontend));  // set {"bob"}
print(backend.difference(frontend));    // set {"ann", "eve"}
```

`[...s]` turns a set into an array, which is the usual way to drop
duplicates while keeping the first-seen order:

```rune
let ids = [3, 1, 3, 2, 1];
print([...set(ids)]);    // [3, 1, 2]
```

## Iterating

`for k, v in m` walks a map's keys and values, `for k in m` its keys
alone. `for x in s` walks a set, and `for i, x in s` numbers its elements.
The order is always insertion order.

```rune
let stock = map({ apple: 3, plum: 7 });
for name, n in stock {
    print("{name}: {n}");
}
// apple: 3
// plum: 7
```

The loop walks the map or set as it was when the loop began, so the body
may change it freely:

```rune
let mut counts = map({ a: 1, b: 2 });
for k in counts {
    counts[k + "!"] = 0;
}
print(counts);    // map {"a": 1, "b": 2, "a!": 0, "b!": 0}
```

## Equality and Conversion

`==` compares maps and sets by their entries, in any order.

```rune
print(map({ a: 1, b: 2 }) == map({ b: 2, a: 1 }));    // true
print(set([1, 2]) == set([2, 1]));                    // true
```

`json.stringify` writes a set as an array, and a map with string keys as
an object, in insertion order.

```rune
print(json.stringify(set([3, 1, 2])));        // [3,1,2]
print(json.stringify(map({ a: 1, b: 2 })));   // {"a":1,"b":2}
```

Like every value, a map or a set is [copied](../language/values.md#copies)
when assigned or passed; a function changes the caller's one only through
a [`mut` parameter](../language/functions.md#mut-parameters).
