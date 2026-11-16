# Getting Started

## Installation

Download the archive for your platform from the
[releases page](https://github.com/kryptamine/rune/releases/latest). There
are builds for Linux, macOS and Windows, on amd64 and arm64. Each archive
holds a single `rune` binary; put it somewhere on your `PATH`:

```sh
# macOS on Apple silicon; swap the version and platform for yours
curl -LO https://github.com/kryptamine/rune/releases/download/v0.3.0/rune-0.3.0-darwin-arm64.tar.gz
tar -xzf rune-0.3.0-darwin-arm64.tar.gz
sudo mv rune-0.3.0-darwin-arm64/rune /usr/local/bin/
rune version
```

`checksums.txt` on the same page lists the SHA-256 sum of every archive.

### From source

Building needs Go 1.27 or newer. Clone the repository and run:

```sh
git clone https://github.com/kryptamine/rune && cd rune
make build
```

This puts the `rune` binary in the project directory.

## Your First Program

Save this as `hello.rn`:

```rune
let name = "world";
print("hello, {name}!");
```

and run it:

```sh
$ rune run hello.rn
hello, world!
```

`{name}` inside the string is [interpolation](language/strings.md#joining-and-interpolation):
the value of `name` is inserted in its place.

## Command-Line Arguments

Words after the file name go to the program's [`main`](programs/main.md)
function as an array of strings. An empty array is falsy, so
`if args { ... }` asks "were any arguments given?":

```rune
// greet.rn
fun main(args) {
    let name = if args { args[0] } else { "world" };
    print("hello, {name}!");
}
```

```sh
$ rune run greet.rn
hello, world!
$ rune run greet.rn ann
hello, ann!
```

## Commands

| Command | What it does |
| --- | --- |
| `rune run <file> [args...]` | Runs the program. |
| `rune fmt <file>` | Prints the file formatted. |
| `rune fmt -w <path>...` | Formats files and directories in place. |
| `rune fmt --check <path>...` | Lists the files that are not formatted and exits with `1` if there are any. |
| `rune disasm <file>` | Prints the bytecode the file compiles to. |
| `rune version` | Prints the interpreter's version. |

`rune fmt` gives every program one layout: 4-space indents, spaces around
operators, one statement per line. A block that holds another block goes
on lines of its own, and so do the blocks chained to it by `else` or
`catch`, and each arm of such a `match`.

```sh
$ cat messy.rn
let total=19.99*3;
print( total );
$ rune fmt messy.rn
let total = 19.99 * 3;
print(total);
```

When working on Rune itself, `make run` builds the interpreter and runs
`program.rn`, a scratch file that git ignores.

## Next Steps

- [Errors and Exit Codes](errors-and-exit-codes.md) shows what a failing
  program prints.
- The Language chapters, starting with [Values](language/values.md), cover
  the language from the ground up.
- [Examples](programs/examples.md) lists complete programs to read.
