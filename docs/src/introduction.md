# Rune

Rune is a small, dynamically typed scripting language. It borrows freely
from JavaScript, Python, Rust and Swift: familiar syntax, values that are
immutable unless you say otherwise, structs and enums, and pattern
matching. It is written in Go: your code is compiled to register bytecode
and run by a VM, and the whole interpreter ships as one static binary.

## A Taste

This program reads an access log, turns each line into a `Request` struct,
sorts requests into "ok", "slow" and "error" with `match`, and counts each
kind in a `map`:

```rune
struct Request { method, path, status, ms }

fun parse(line) {
    let [method, path, status, ms] = line.split(" ");
    Request { method, path, status: number(status), ms: number(ms) }
}

fun kind(req) {
    match req {
        Request { status: 500.. } => "error",
        Request { ms } if ms > 100 => "slow",
        _ => "ok",
    }
}

let log = """
    GET /users 200 12.5
    POST /orders 201 340
    GET /health 503 1
    GET /users 200 8
    """;

let mut counts = map();

for line in log.split("\n") {
    let k = kind(parse(line));
    counts[k] = (counts?.[k] ?? 0) + 1;
}

print(counts);    // map {"ok": 2, "slow": 1, "error": 1}
```

A few things to notice:

- `let [method, path, status, ms] = ...` [destructures](language/destructuring.md)
  an array into four variables.
- A function gives the value of its last expression, so `parse` and `kind`
  need no `return`.
- `Request { status: 500.. }` is a [pattern](language/pattern-matching.md):
  it matches any request whose status is 500 or more.
- `counts` is declared `let mut` because it changes; plain `let` variables
  never do.
- `counts?.[k] ?? 0` reads a key that may be missing and falls back to `0`.

New here? Start with [Getting Started](getting-started.md), then read the
Language chapters in order.

## Features

**Language**

- [values](language/values.md): `nil`, booleans, 64-bit [numbers](language/numbers.md) (integers and floats), Unicode [strings](language/strings.md) with interpolation `"x = {x}"`
- [variables](language/variables.md) immutable by default: `let`, `let mut`, compile-time `const`
- every value is [copied](language/values.md#copies) where it goes, containers too, lazily; `mut` parameters change the caller's variable
- `if`/`else` as statement or expression, [loops](language/loops.md) `while` and `for x in ...` over arrays, strings, objects and ranges `a..b`
- [functions](language/functions.md): closures, defaults, arrows `x => x * 2`, bodies give their last expression `fun sq(x) { x * x }`
- [spread and rest](language/spread-and-rest.md), [destructuring](language/destructuring.md): `let [head, ...tail] = xs;`
- [nil safety](language/nil-safety.md): `resp?.data?.items ?? []`
- [pattern matching](language/pattern-matching.md) on literals, ranges, types, structs, arrays and enums, exhaustive over enum variants; the same nested patterns in `let`, parameters and `if x is { status: 200, body }`
- [errors](language/error-handling.md): `throw`, `try`/`catch` by type, `try` as an expression, `defer`
- `return`, `break`, `continue` and `throw` as expressions: `let v = x ?? return nil;`

**Data**

- [arrays](data/arrays.md), objects, [map and set](data/map-and-set.md) in insertion order, [bytes](data/bytes.md)
- [structs](data/structs.md) with default fields, [methods](data/methods.md) in `impl` blocks, also on built-in types
- [enums](data/enums.md), whose variants may carry fields

**Programs**

- [modules](programs/modules.md) with `export` and `import`, [`main`](programs/main.md) with command-line arguments
- [standard library](stdlib/builtins.md): `string`, `regex`, `math`, `json`, `http`, `time`, `fs`, [streams](stdlib/streams.md), `io`, `os`
- errors with source snippets and traces; the parser reports every syntax error in one run
- `rune fmt` formats code, `rune disasm` shows the bytecode

## References

- [Crafting Interpreters](https://craftinginterpreters.com/) by Robert Nystrom, where Rune started
