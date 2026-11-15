# Rune

Rune is a small, dynamically typed scripting language, inspired by
JavaScript, Python, Rust and Swift. It is written in Go: the AST is compiled
to register bytecode run by a VM, and the whole thing ships as a single
static binary.

## A Taste

Classifying the lines of an access log, with a struct, destructuring,
pattern matching and a `map`:

```rune
struct Request { method, path, status, ms }

fun parse(line) {
    let [method, path, status, ms] = line.split(" ");
    return Request { method, path, status: int(status), ms: number(ms) };
}

fun kind(r) {
    return match r {
        Request { status } if status >= 500 => "error",
        Request { ms } if ms > 100 => "slow",
        _ => "ok",
    };
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

New here? Start with [Getting Started](getting-started.md), then read the
Language chapters in order.

## Features

**Language**

- [values](language/values.md): `nil`, booleans, 64-bit [numbers](language/numbers.md) (integers and floats), Unicode [strings](language/strings.md) with interpolation `"x = {x}"`
- [variables](language/variables.md) immutable by default: `let`, `let mut`, compile-time `const`
- `if`/`else`, `? :`, [loops](language/loops.md) `while` and `for x in ...` over arrays, strings, objects and ranges `a..b`
- [functions](language/functions.md): closures, defaults, arrows `x => x * 2`, expression bodies `fun sq(x) => x * x;`
- [spread and rest](language/spread-and-rest.md), [destructuring](language/destructuring.md): `let [head, ...tail] = xs;`
- [nil safety](language/nil-safety.md): `resp?.data?.items ?? []`
- [pattern matching](language/pattern-matching.md) on literals, types, structs, arrays and enums, exhaustive over enum variants
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
