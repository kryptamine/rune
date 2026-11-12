# Map and Set

A `Map` holds values by key, a `Set` unique values. Unlike an object, whose
keys are strings, their keys may be any hashable value, and both keep keys
in the order they were first added.

```rune
let mut stock = Map();
stock["apple"] = 3;
stock["pear"] = 0;
print(stock["apple"]);   // 3
print(stock);            // Map {"apple": 3, "pear": 0}

let mut visitors = Set();
visitors.add("ann");
visitors.add("bob");
visitors.add("ann");
print(visitors);         // Set {"ann", "bob"}
print(len(visitors));    // 2
```

## Making one

`Map()` makes an empty map and `Map(src)` one from an object, another map
or an array of `[key, value]` pairs; `Set()` and `Set(array)` likewise.

```rune
print(Map({ host: "localhost", port: 8080 }));  // Map {"host": "localhost", "port": 8080}
print(Map([[200, "OK"], [404, "Not Found"]]));  // Map {200: "OK", 404: "Not Found"}
print(Set(["a", "b", "a"]));                    // Set {"a", "b"}
```

## Keys

A key is any hashable value: `nil`, a boolean, a number (`1` and `1.0` are
one key; `NaN` is none), a string, bytes, a function, a struct type, or a
struct whose fields are hashable, compared by value. A struct makes a
natural composite key:

```rune
struct Pos { x, y }

let mut grid = Map();
grid[Pos { x: 0, y: 1 }] = "wall";
print(grid[Pos { x: 0, y: 1 }]);                // wall
print(grid.get(Pos { x: 9, y: 9 }, "floor"));   // floor
```

An array, object, map or set is not hashable, as it can change:

```rune
let mut seen = Map();
seen[[1, 2]] = true; // TypeError: Cannot use an array as a key.
```

## Reading and writing

`m[k]` reads a key and a missing one is a `KeyError`; `m?.[k]` gives `nil`
instead. `m[k] = v` sets one, like an array element, so `m` must be
mutable.

```rune
let mut prices = Map({ apple: 1.5 });
print(prices?.["kiwi"]); // nil
print(prices["kiwi"]);   // KeyError: Key not found: "kiwi".
```

Map methods: `get(k[, default])`, `set(k, v)` (returns the map, so calls
chain), `has(k)`, `delete(k)` (whether it was there), `clear()`, and
`keys()`, `values()`, `entries()`, new arrays.

```rune
let mut stock = Map();
stock.set("apple", 3).set("pear", 0).set("plum", 7);

print(stock.get("kiwi"));     // nil
print(stock.get("kiwi", 0));  // 0
print(stock.has("pear"));     // true
print(stock.delete("pear"));  // true
print(stock.delete("pear"));  // false
print(stock.keys());          // ["apple", "plum"]
print(stock.values());        // [3, 7]
print(stock.entries());       // [["apple", 3], ["plum", 7]]
```

A set has `add(x)` (returns the set), `has`, `delete`, `clear`, `values()`,
and `union(s)`, `intersection(s)`, `difference(s)`, new sets.

```rune
let backend = Set(["ann", "bob", "eve"]);
let frontend = Set(["bob", "kim"]);

print(backend.has("eve"));               // true
print(backend.union(frontend));          // Set {"ann", "bob", "eve", "kim"}
print(backend.intersection(frontend));   // Set {"bob"}
print(backend.difference(frontend));     // Set {"ann", "eve"}
```

## Iterating

`len` counts the entries; `for k, v in m` walks a map's keys and values,
`for k in m` its keys, and `for x in s` or `for i, x in s` a set, always
in insertion order.

```rune
let stock = Map({ apple: 3, plum: 7 });
for name, n in stock {
    print("{name}: {n}");
}
// apple: 3
// plum: 7
```

A loop may set existing keys and delete them, but adding one is a
`ValueError`; walk `m.keys()` to change it freely.

```rune
let mut counts = Map({ a: 1 });
for k in counts {
    counts[k + "!"] = 0; // ValueError: Cannot add to a Map while iterating over it.
}
```

```rune
let mut counts = Map({ a: 1, b: 2 });
for k in counts.keys() {
    counts[k + "!"] = 0;
}
print(counts); // Map {"a": 1, "b": 2, "a!": 0, "b!": 0}
```

## Equality and conversion

`==` compares entries in any order. `[...s]` spreads a set into an array;
`json.stringify` writes a set as an array and a map of string keys as an
object, in order. Both are references, like arrays.

```rune
print(Map({ a: 1, b: 2 }) == Map({ b: 2, a: 1 })); // true

let ids = Set([3, 1, 2]);
print([...ids]);                      // [3, 1, 2]
print(json.stringify(ids));           // [3,1,2]
print(json.stringify(Map({ a: 1 }))); // {"a":1}
```
