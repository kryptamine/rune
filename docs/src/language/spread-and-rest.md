# Spread and Rest

`...` works in two directions. In a literal or a call it *spreads*: it
unpacks a collection into separate elements. In a parameter list it
*collects*: it gathers the remaining arguments into one array.

## Spreading Arrays

Inside an array literal, `...xs` puts all the elements of `xs` in place.
That makes joining arrays a one-liner:

```rune
let morning = ["standup", "review"];
let evening = ["deploy"];
print([...morning, "lunch", ...evening]);    // ["standup", "review", "lunch", "deploy"]
```

In a call, it passes the elements as separate arguments:

```rune
let scores = [72, 95, 88];
print(math.max(...scores));    // 95
```

A method call takes no spread:

```rune
let mut xs = [1];
xs.push(...[2, 3]); // Error at '...': Cannot spread arguments into a method call.
```

Sets spread too, in their order, and bytes spread as integers:

```rune
let tags = set(["sale", "new"]);
print([...tags, "featured"]);    // ["sale", "new", "featured"]
print([...b"hi"]);               // [104, 105]
```

A map is not a list, so spreading it is an error. Spread its `keys()`,
`values()` or `entries()` instead.

## Spreading into Structs

In a struct literal, `...s` fills fields from a struct of the same type,
or from a map whose string keys are field names. Fields are applied left
to right and later ones win, which gives an easy way to merge settings
(see [Structs](../data/structs.md#updating-with-spread)):

```rune
struct Config { timeout = 30, retries = 3 }

let defaults = Config {};
print(Config { ...defaults, timeout: 5 });    // Config {timeout: 5, retries: 3}
print(Config { timeout: 5, ...defaults });    // Config {timeout: 30, retries: 3}
print(Config { ...["retries": 1] });          // Config {timeout: 30, retries: 1}
```

Spreading anything else is a `TypeError`:

```rune
print([..."abc"]); // TypeError: Cannot spread string into an array.
```

## Rest Parameters

A last parameter written `...name` collects every argument left over into
an array. Use it for functions that take any number of arguments:

```rune
fun log(level, ...parts) {
    print("[{level}] " + parts.join(" "));
}

log("info", "started", "in", 3, "ms");    // [info] started in 3 ms
log("warn", "disk", "almost", "full");    // [warn] disk almost full
```

With no extra arguments, the rest is an empty array:

```rune
let count = (...args) => args.len();
print(count(1, 2, 3));    // 3
print(count());           // 0
```

Rest and spread work together. Collect the arguments on one side, spread
an array into them on the other:

```rune
fun sum(...nums) { nums.reduce((a, b) => a + b, 0) }

let xs = [1, 2, 3];
print(sum(...xs, 10));    // 16
```

The parameters before the rest are still required:

```rune
fun log(level, ...parts) {}
log(); // ArgumentError: Expected at least 1 argument but got 0.
```

For `...rest` in patterns, see [Destructuring](destructuring.md).
