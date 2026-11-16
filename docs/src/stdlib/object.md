# Object Methods

An object is a record of named values: `{ host: "localhost", port: 8080 }`.
Its keys are strings and keep the order they were added in. Objects hold
data only; behavior belongs to a [struct](../data/structs.md) and its
methods. For keys of any type, use a [map](../data/map-and-set.md).

| Method | Does |
|---|---|
| `len()` | number of keys |
| `keys()` | array of the keys |
| `values()` | array of the values |
| `entries()` | array of `[key, value]` pairs |
| `delete(k)` | removes key `k`, **in place**; returns whether it was there |

```rune
let mut config = { port: 8080, host: "localhost", debug: false };

print(config.len());                 // 3
print(config.keys());                // ["port", "host", "debug"]
print(config.values());              // [8080, "localhost", false]
print(config.entries());             // [["port", 8080], ["host", "localhost"], ["debug", false]]
print(config.delete("debug"));       // true
print(config.delete("debug"));       // false
print(config);                       // {port: 8080, host: "localhost"}
```

## Reading keys

`o.k` and `o["k"]` read a key. A missing key is a `KeyError`; with `?.`
it is `nil` instead, so `?? fallback` gives a default. `"k" in o` asks
whether the key is there.

```rune
let config = { port: 8080 };

print(config.port);                  // 8080
print(config["port"]);               // 8080
print("host" in config);             // false
print(config?.host);                 // nil
print(config?.host ?? "localhost");  // localhost

let field = "port";
print(config[field]);                // 8080
```

```rune
let config = { port: 8080 };
print(config.host); // KeyError: Undefined property 'host'.
```

## Walking an object

`for` walks the keys in order; with two names it gets the key and the
value:

```rune
let prices = { apple: 1.2, pear: 0.8, fig: 2.5 };

for fruit, price in prices {
    print("{fruit}: {price}");
}
// apple: 1.2
// pear: 0.8
// fig: 2.5

let cheap = prices.entries().filter(([_, p]) => p < 2).map(([f, _]) => f);
print(cheap);   // ["apple", "pear"]
```

## Building objects

A key in brackets is computed: its value, a string, becomes the key.
[Spread](../language/spread-and-rest.md) copies the keys of another object,
and later keys win:

```rune
fun limits(name) { ({ [name]: 0, [name + "Max"]: 10 }) }
print(limits("retries"));   // {retries: 0, retriesMax: 10}

let defaults = { host: "localhost", port: 8080, debug: false };
let config = { ...defaults, port: 9000, debug: true };
print(config);              // {host: "localhost", port: 9000, debug: true}
```

To turn an array of pairs back into an object, use
[`object`](builtins.md#object):

```rune
let env = { HOST: "x.io", PORT: "443" };
let lower = object(env.entries().map(([k, v]) => [k.lower(), v]));
print(lower);   // {host: "x.io", port: "443"}
```

A computed key must be a string:

```rune
print({ [1]: "one" }); // TypeError: Object properties must be accessed with string keys.
```

## Keys named like methods

`o.k` reads the key `k` when there is one. A call `o.k()` always calls the
method:

```rune
let stats = { keys: 3 };
print(stats.keys);     // 3
print(stats.keys());   // ["keys"]
```
