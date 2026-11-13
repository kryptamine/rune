# Rune

Rune is a dynamically typed interpreted programming language inspired by [Lox](https://github.com/munificent/craftinginterpreters) and built following concepts from the book [Crafting Interpreters](https://craftinginterpreters.com/) by Robert Nystrom. The interpreter is implemented in **Go**: the resolved AST is compiled to register bytecode run by a VM, and the whole thing ships as a single statically linked binary.

```rune
struct Point { x, y = 0 }

impl Point {
  fun len(self) { return math.sqrt(self.x * self.x + self.y * self.y); }
}

let points = [Point { x: 3, y: 4 }, Point { x: 1 }];
for i, p in points {
    print("{i}: {p} has length {p.len()}");
}
// 0: Point {x: 3, y: 4} has length 5
// 1: Point {x: 1, y: 0} has length 1
```

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
    counts[k] = counts.get(k, 0) + 1;
}
print(counts);    // map {"ok": 2, "slow": 1, "error": 1}
```

New here? Start with [Getting Started](getting-started.md), then read the
Language chapters in order.

## Features


- Dynamically typed language with a simple syntax
- Supports expressions, statements, variables, and functions
- Variables are immutable by default (`let`); `let mut` makes them mutable
- Standard library in namespaces: `array`, `string`, `regex`, `math`, `json`, `http`, `time`, `fs`, `io`, `os`
- Modules with their own top level, `export` and `import a.b as m;`, relative `import ..lib.x;`
- Recursive descent parser; the resolved AST is compiled to register bytecode run by a VM
- Parser recovers from syntax errors and reports all of them in one run
- String escapes: `\n`, `\t`, `\r`, `\"`, `\\`, `\{`, `\}`; raw strings take neither: `r"\d{3}"`
- Multi-line strings, dedented, with quotes inside: `"""..."""`, raw `r"""..."""`
- String interpolation: `"x = {x}, sum = {a + b}"`
- 64-bit integers and floats, both of type `number`: `7 / 2` is `3.5`, `7 % 2` is `1`, and overflow is an error
- Number literals with exponents, hex, octal, binary and `_`: `1e3`, `0xff`, `0o17`, `0b1010`, `1_000_000`
- Bitwise operators on integers: `&`, `|`, `^`, `~`, `<<`, `>>`
- Objects, `map` and `set` keep their keys in insertion order; a `map` or `set` key may be any hashable value
- `bytes`: immutable binary data, with literals `b"\x00\xff"`
- Streams: files and HTTP bodies read line by line or in chunks, `for line in fs.open(path)`
- Object property access with `obj.name` as sugar for `obj["name"]`; shorthand `{ ip, hits }` for `{ ip: ip, hits: hits }`
- Structs with required and default fields, values that copy on write: `struct Point { x, y = 0 }`, `p.x = 1`
- Methods in `impl` blocks, on structs and built-in types: `p.len()`, `Point.origin()`, `fun move(mut self)`; a function in a property is called as `(o.f)()`
- Destructuring of objects, structs and arrays, in `let`, parameters and `for`, with defaults: `let { x, y = 0 } = p;`, `let [head, ...tail] = xs;`
- Nil-safe access and defaults: `resp?.data?.items ?? []`
- Conditional expressions: `n > 0 ? "pos" : "neg"`
- Spread and rest: `[...a, ...b]`, `{...o, x: 1}`, `Point { ...p, x: 1 }`, `f(...args)`, `fun log(level, ...parts)`
- Strings compare with `<`, `<=`, `>`, `>=` and index by character: `"héllo"[1]` is `"é"`; a negative index counts from the end: `xs[-1]`
- Pattern matching with `match`, as an expression or a statement
- Logical operators `&&`, `||` and `!`, type tests with `is`: `v is Point`, `catch e is KeyError`, `p is Point =>`
- Error handling with `try`/`catch` and `throw`, also as an expression: `x ?? throw E { ... }`
- Two loops only: `while` and `for x in ...` over arrays, strings, objects and ranges (`a..b`)
- `break` and `continue` in both
- `if`/`else`, `while` and `for` bodies are blocks, or a bare statement starting with a name or keyword: `if v < 0 return;`, `for x in xs out.push(x);`
- Anonymous functions: `fun (x) { return x * 2; }`, or arrow functions: `x => x * 2`
- Default parameters, evaluated at each call: `fun greet(name, greeting = "hi")`
- Modulo `%` and compound assignment: `+=`, `-=`, `*=`, `/=`, `%=`
- Increment and decrement, prefix and postfix: `++i`, `i++`, `--i`, `i--` (on variables, array elements and properties)
- Cross-platform, compiles to a single binary

## References

- [Crafting Interpreters](https://craftinginterpreters.com/) by Robert Nystrom
- [Lox Programming Language](https://github.com/munificent/craftinginterpreters)
