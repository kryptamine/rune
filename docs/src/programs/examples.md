# Examples

Complete programs live in [`examples/`](https://github.com/kryptamine/rune/tree/master/examples).
Each one runs as is from the repository root.

## Real-world programs

### [`blog/`](https://github.com/kryptamine/rune/tree/master/examples/blog) — static blog generator

Markdown posts with front matter to an HTML site. Split into
[modules](modules.md) (`site/`, `md/`), finds its content with `module.dir`,
renders pages with `"""` templates, parses Markdown with `regex`.

```sh
rune run examples/blog/main.rn                    # build the bundled posts into a temp dir
rune run examples/blog/main.rn examples/blog/content site --drafts
```

### [`todo.rn`](https://github.com/kryptamine/rune/blob/master/examples/todo.rn) — task tracker

Tasks with tags, priorities and due dates, kept in a JSON file. Shows
`fun main` with arguments, structs with `impl`, a `Priority` enum read
back from JSON by name, `match` on commands,
`time` for dates, and writes the file atomically with `fs.rename`.
Tasks live in `$TODO_FILE`, or `~/.todo.json`.

```sh
rune run examples/todo.rn                                  # demo session against a temp file
rune run examples/todo.rn add "pay rent" +home !high due:fri
rune run examples/todo.rn list +home --overdue
```

### [`log-report.rn`](https://github.com/kryptamine/rune/blob/master/examples/log-report.rn) — access log report

Per-endpoint latency percentiles, status classes, top clients and slow
requests. Reads the log line by line with a [stream](../stdlib/streams.md),
groups with `map`, writes JSON.

```sh
rune run examples/log-report.rn                 # generate a sample log and analyze it
rune run examples/log-report.rn access.log -j   # your own log, plus a JSON report
```

### [`chess.rn`](https://github.com/kryptamine/rune/blob/master/examples/chess.rn) — chess engine

Move generation on a 10×12 mailbox board, perft, alpha-beta search with
quiescence, a transposition table on Zobrist keys and iterative deepening.
Moves are packed into integers with bitwise operators; the position is a
struct changed in place by `mut self` methods `make` and `unmake`; table
entries record their bound as an enum.

```sh
rune run examples/chess.rn                     # perft check, a mate puzzle, self-play
rune run examples/chess.rn perft 4             # leaf nodes per root move
rune run examples/chess.rn go -t 3             # best move from the start position
rune run examples/chess.rn play black -t 2     # play against the engine
```

### [`calc.rn`](https://github.com/kryptamine/rune/blob/master/examples/calc.rn) — calculator

Tokenizer, Pratt parser and evaluator with variables and functions. A good
read for structs, an enum of token kinds matched inside struct patterns,
`match` on AST nodes and `throw` for parse errors.

```sh
rune run examples/calc.rn        # demo
rune run examples/calc.rn repl   # read lines from stdin
```

### [`fetch.rn`](https://github.com/kryptamine/rune/blob/master/examples/fetch.rn) — HTTP and JSON

Fetches a URL, parses the JSON and handles network errors with `try`.

```sh
rune run examples/fetch.rn
```

## Basics

- [`fib.rn`](https://github.com/kryptamine/rune/blob/master/examples/fib.rn) — Fibonacci numbers, recursion
- [`bubble-sort.rn`](https://github.com/kryptamine/rune/blob/master/examples/bubble-sort.rn) — loops and in-place array updates
- [`linked-list.rn`](https://github.com/kryptamine/rune/blob/master/examples/linked-list.rn) — a linked list built from objects
