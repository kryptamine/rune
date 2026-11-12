# Getting Started

## Installation

Building needs **Go**. Clone the repository and run:

```sh
make build
```

This produces a single binary, `rune`, in the project directory.

## Your First Program

Save this as `hello.rn`:

```rune
let name = "world";
print("hello, {name}!");
```

and run it:

```sh
$ ./rune run hello.rn
hello, world!
```

Arguments after the file go to the program's [`main`](programs/main.md):

```rune
fun main(args) {
    let name = len(args) > 0 ? args[0] : "world";
    print("hello, {name}!");
}
```

```sh
$ ./rune run greet.rn ann
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
$ ./rune evaluate expr.rn
20
```
