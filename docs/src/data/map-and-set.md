# Map and Set

A map holds values by key, a set unique values. Unlike an object, whose
keys are strings, their keys may be any hashable value; like an object,
they keep keys in the order they were first added.

```rune
let mut stock = map();
stock["apple"] = 3;
stock["pear"] = 0;
print(stock["apple"]);   // 3
print(stock);            // map {"apple": 3, "pear": 0}

let mut visitors = set();
visitors.add("ann");
visitors.add("bob");
visitors.add("ann");
print(visitors);         // set {"ann", "bob"}
print(visitors.len());   // 2
```

## Making one

`map()` makes an empty map and `map(src)` one from an object, a struct, another map
or an array of `[key, value]` pairs; `set()` and `set(array)` likewise.

```rune
print(map({ host: "localhost", port: 8080 }));  // map {"host": "localhost", "port": 8080}
print(map([[200, "OK"], [404, "Not Found"]]));  // map {200: "OK", 404: "Not Found"}
print(set(["a", "b", "a"]));                    // set {"a", "b"}
```

## Keys

A key is any hashable value: `nil`, a boolean, a number (`1` and `1.0` are
one key; `NaN` is none), a string, bytes, a function, a struct type, or a
struct whose fields are hashable, compared by value. A struct makes a
natural composite key:

```rune
struct Pos { x, y }

let mut grid = map();
grid[Pos { x: 0, y: 1 }] = "wall";
print(grid[Pos { x: 0, y: 1 }]);                // wall
print(grid.get(Pos { x: 9, y: 9 }, "floor"));   // floor
```

An array, object, map or set is not hashable, as it can change:

```rune
let mut seen = map();
seen[[1, 2]] = true; // TypeError: Cannot use an array as a key.
```

## Reading and writing

`m[k]` reads a key and a missing one is a `KeyError`; `m?.[k]` gives `nil`
instead. `m[k] = v` sets one, like an array element, so `m` must be
mutable.

```rune
let mut prices = map({ apple: 1.5 });
print(prices?.["kiwi"]); // nil
print(prices["kiwi"]);   // KeyError: Key not found: "kiwi".
```

Map methods: `get(k[, default])`, `set(k, v)`, `has(k)`, `delete(k)`
(whether it was there), `clear()`, and `keys()`, `values()`, `entries()`,
new arrays. Like an array's in-place methods, `set` and `clear` return
`nil`.

```rune
let mut stock = map();
stock.set("apple", 3);
stock.set("pear", 0);
stock.set("plum", 7);

print(stock.get("kiwi"));     // nil
print(stock.get("kiwi", 0));  // 0
print(stock.has("pear"));     // true
print(stock.delete("pear"));  // true
print(stock.delete("pear"));  // false
print(stock.keys());          // ["apple", "plum"]
print(stock.values());        // [3, 7]
print(stock.entries());       // [["apple", 3], ["plum", 7]]
```

A set has `add(x)`, `contains(x)`, `delete`, `clear`, `values()`, and
`union(s)`, `intersection(s)`, `difference(s)`, new sets. As everywhere,
`has` tests a key (of an object or a map) and `contains` a value (of an
array, a string or a set).

```rune
let backend = set(["ann", "bob", "eve"]);
let frontend = set(["bob", "kim"]);

print(backend.contains("eve"));          // true
print(backend.union(frontend));          // set {"ann", "bob", "eve", "kim"}
print(backend.intersection(frontend));   // set {"bob"}
print(backend.difference(frontend));     // set {"ann", "eve"}
```

## Iterating

`m.len()` counts the entries; `for k, v in m` walks a map's keys and values,
`for k in m` its keys, and `for x in s` or `for i, x in s` a set, always
in insertion order.

```rune
let stock = map({ apple: 3, plum: 7 });
for name, n in stock {
    print("{name}: {n}");
}
// apple: 3
// plum: 7
```

A loop may set existing keys and delete them, but adding one is a
`ValueError`; walk `m.keys()` to change it freely.

```rune
let mut counts = map({ a: 1 });
for k in counts {
    counts[k + "!"] = 0; // ValueError: Cannot add to a map while iterating over it.
}
```

```rune
let mut counts = map({ a: 1, b: 2 });
for k in counts.keys() {
    counts[k + "!"] = 0;
}
print(counts); // map {"a": 1, "b": 2, "a!": 0, "b!": 0}
```

## Equality and conversion

`==` compares entries in any order. `[...s]` spreads a set into an array;
`json.stringify` writes a set as an array and a map of string keys as an
object, in order. Both are references, like arrays.

```rune
print(map({ a: 1, b: 2 }) == map({ b: 2, a: 1 })); // true

let ids = set([3, 1, 2]);
print([...ids]);                      // [3, 1, 2]
print(json.stringify(ids));           // [3,1,2]
print(json.stringify(map({ a: 1 }))); // {"a":1}
```
