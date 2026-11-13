# Getting Started

## Installation

Download the archive for your platform from the
[releases page](https://github.com/kryptamine/rune/releases/latest). Builds
exist for Linux, macOS and Windows, on amd64 and arm64. Each archive holds a
single `rune` binary; put it somewhere on your `PATH`:

```sh
# macOS on Apple silicon; swap the version and platform for yours
curl -LO https://github.com/kryptamine/rune/releases/download/v0.1.0/rune-0.1.0-darwin-arm64.tar.gz
tar -xzf rune-0.1.0-darwin-arm64.tar.gz
sudo mv rune-0.1.0-darwin-arm64/rune /usr/local/bin/
rune version
```

`checksums.txt` on the same page lists the SHA-256 sum of every archive.

### From source

Building needs **Go** 1.27+. Clone the repository and run:

```sh
make build
```

This produces the `rune` binary in the project directory.

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

Arguments after the file go to the program's [`main`](programs/main.md):

```rune
fun main(args) {
    let name = args.len() > 0 ? args[0] : "world";
    print("hello, {name}!");
}
```

```sh
$ rune run greet.rn ann
hello, ann!
```

`make run` builds and runs `program.rn`, a scratch file ignored by git.

## Interpreter Commands

- `run` — runs the program in the file.
- `evaluate` — evaluates a single expression in the file and prints its value.
- `tokenize` — prints the tokens of the file.
- `disasm` — prints the bytecode the file compiles to.
- `version` — prints the interpreter's version.

```sh
$ echo '(2 + 3) * 4' > expr.rn
$ rune evaluate expr.rn
20
```
