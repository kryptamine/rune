![ci](https://github.com/kryptamine/rune/actions/workflows/ci.yml/badge.svg)
![Go](https://img.shields.io/badge/Go-1.20+-blue?logo=go&logoColor=white)
![Python](https://img.shields.io/badge/Python-3.8+-yellow?logo=python&logoColor=white)

# Rune Interpreter

Rune is a dynamically typed interpreted programming language inspired by [Lox](https://github.com/munificent/craftinginterpreters) and built following concepts from the book [Crafting Interpreters](https://craftinginterpreters.com/) by Robert Nystrom. This interpreter is implemented in **Go** and is designed to be cross-platform, compiling into a statically linked binary.

## Features

- Dynamically typed language with a simple syntax
- Supports expressions, statements, variables, and functions
- Built-in functions for array manipulation, JSON parsing, and timing
- Recursive descent parser and tree-walk interpreter
- Cross-platform, compiles to a single binary

## Installation

To build the interpreter, you need to have **Go** installed. Clone the repository and run:

```sh
make build
```

This will generate an executable binary named `rune` in the project directory.

## Running the Interpreter

You can execute Rune scripts using the `run` command:

```sh
./rune run script.rn
```

Alternatively, you can specify the file in the `Makefile` and use:

```sh
make run
```

## Interpreter Commands

The Rune interpreter supports the following commands:

- `tokenize` — Tokenizes the given input file.
- `evaluate` — Evaluates a single expression from the file.
- `run` — Executes the entire program from the input file.

Example usage:

```sh
./rune run program.rn
```

## Running Tests

Test Framework with tests suites is stolen from [Ben Hoyt](https://github.com/benhoyt/loxlox) and patched.

```sh
make test
```

To run a specific test or filter tests by category:

```sh
python3 test.py <filter>
```

For example, to run only the `arrays` tests:

```sh
python3 test.py arrays
```

## Built-in Functions

Rune provides several built-in functions:

- **`len(arr)`** — Returns the length of an array.
- **`append(arr, value1, value2, ...)`** — Appends values to an array and returns the new array.
- **`json(url)`** — Fetches and parses JSON from a URL, error handling is not implemented.
- **`clock()`** — Returns the current time in seconds.

## Example Program

Here’s a simple Rune script that calculates the sum of an array:

```javascript
fun sumArray(arr) {
    var total = 0;
    for (var i = 0; i < len(arr); i = i + 1) {
        total = total + arr[i];
    }
    return total;
}

print sumArray([1, 2, 3, 4, 5]);
```

## License

This project is open-source and follows the MIT license.

## References

- [Crafting Interpreters](https://craftinginterpreters.com/) by Robert Nystrom
- [Lox Programming Language](https://github.com/munificent/craftinginterpreters)
- [Test Framework](https://github.com/benhoyt/loxlox)
