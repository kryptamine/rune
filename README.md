![ci](https://github.com/kryptamine/rune/actions/workflows/ci.yml/badge.svg)
![Go](https://img.shields.io/badge/Go-1.26+-blue?logo=go&logoColor=white)

# Rune

A dynamically typed scripting language, inspired by [Lox](https://github.com/munificent/craftinginterpreters) from [Crafting Interpreters](https://craftinginterpreters.com/). Written in Go: the AST is compiled to register bytecode run by a VM, shipped as a single static binary.

**Documentation: [kryptamine.github.io/rune](https://kryptamine.github.io/rune/)**

```js
struct Point { x, y = 0 }

impl Point {
  fun len(self) { return math.sqrt(self.x * self.x + self.y * self.y); }
}

let points = [Point { x: 3, y: 4 }, Point { x: 1 }];
for i, p in points {
    print("{i}: {p} has length {p.len()}");
}
```

- immutable by default (`let` / `let mut`), compile-time `const`, structs and enums with `impl` methods
- pattern matching, destructuring, spread, nil-safe chains `a?.b ?? c`
- `try`/`catch` with typed errors, modules with `export` / `import`
- standard library: `string`, `regex`, `math`, `json`, `http`, `time`, `fs`, `io`, `os`

## Quick start

```sh
make build
./rune run examples/fib.rn
```

More programs in [`examples/`](examples). Tests: `make test`; docs locally: `mdbook serve docs`.

## License

MIT
