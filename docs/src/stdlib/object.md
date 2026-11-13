# Object Methods

- `has(k)` — whether the object has the key `k`.
- `get(k[, default])` — the value of `k`, or `default` (`nil` if not given) when missing; `obj[k]` and `obj.k` are a `KeyError` instead.
- `keys()`, `values()`, `entries()` — new arrays of the keys, the values, or `[key, value]` pairs, in the order the keys were added, as `for` walks them.
- `delete(k)` — removes `k` **in place** and returns whether it was there.

```rune
let mut config = { port: 8080, host: "localhost", debug: false };

print(config.has("port"));             // true
print(config.get("user"));             // nil
print(config.get("user", "admin"));    // admin
print(config.keys());                  // ["port", "host", "debug"]
print(config.values());                // [8080, "localhost", false]
print(config.entries());               // [["port", 8080], ["host", "localhost"], ["debug", false]]
print(config.delete("debug"));         // true
print(config.delete("debug"));         // false
print(config);                         // {port: 8080, host: "localhost"}
```

`entries()` pairs well with array destructuring:

```rune
let prices = { apple: 1.2, pear: 0.8 };
for [fruit, price] in prices.entries() {
    print("{fruit}: {price}");
}
// apple: 1.2
// pear: 0.8
```

Reading `o.k` gives an own property of that name first; a call `o.k()`
always calls the method:

```rune
let stats = { keys: 3 };
print(stats.keys);          // 3
print(stats.keys());        // ["keys"]
```
