# Spread and Rest

## Spread

`...` spreads an array into an array literal or into a call's arguments:

```rune
let morning = ["standup", "review"];
let evening = ["deploy"];
print([...morning, "lunch", ...evening]); // ["standup", "review", "lunch", "deploy"]

let scores = [72, 95, 88];
print(math.max(...scores)); // 95
```

Both also take a set, in its order, a map as `[key, value]` pairs and bytes
as integers: `[...tags]`, `f(...counts)`, `[...b"hi"]` is `[104, 105]`.

An object or a struct spreads into an object literal. Keys are applied in
order and later ones win, so put the defaults first to override them, or
last to fill in only what is missing:

```rune
let defaults = { timeout: 30, retries: 3 };

print({ ...defaults, timeout: 5 }); // {timeout: 5, retries: 3}
print({ timeout: 5, ...defaults }); // {timeout: 30, retries: 3}

struct Point { x, y = 0 }
let p = Point { x: 1, y: 2 };
print({ ...p, z: 3 });              // {x: 1, y: 2, z: 3}
```

A struct literal takes a struct of its own type, or an object of some of
its fields, which makes an updated copy (see [Structs](../data/structs.md)):

```rune
struct Point { x, y = 0 }
let p = Point { x: 1, y: 2 };
print(Point { ...p, y: 5 }); // Point {x: 1, y: 5}
```

Spreading anything else is a `TypeError`:

```rune
print([..."abc"]); // TypeError: Cannot spread string into an array.
```

## Rest Parameters

A last parameter `...name` (or `...mut name`) takes the arguments after the
others as an array, possibly empty.

```rune
fun log(level, ...parts) {
    print("[{level}] " + parts.join(" "));
}

log("info", "started", "in", 3, "ms"); // [info] started in 3 ms

let count = (...args) => args.len();
print(count(1, 2, 3)); // 3
print(count());        // 0
```

Rest and spread go together to forward arguments:

```rune
fun sum(...nums) { return nums.reduce((a, b) => a + b, 0); }

let xs = [1, 2, 3];
print(sum(...xs, 10)); // 16
```

Fewer arguments than the parameters before the rest is an `ArgumentError`:

```rune
fun log(level, ...parts) {}
log(); // ArgumentError: Expected at least 1 argument but got 0.
```
