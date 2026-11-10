![ci](https://github.com/kryptamine/rune/actions/workflows/ci.yml/badge.svg)
![Go](https://img.shields.io/badge/Go-1.26+-blue?logo=go&logoColor=white)
![Python](https://img.shields.io/badge/Python-3.8+-yellow?logo=python&logoColor=white)

# Rune Interpreter

Rune is a dynamically typed interpreted programming language inspired by [Lox](https://github.com/munificent/craftinginterpreters) and built following concepts from the book [Crafting Interpreters](https://craftinginterpreters.com/) by Robert Nystrom. This interpreter is implemented in **Go** and is designed to be cross-platform, compiling into a statically linked binary.

## Features

- Dynamically typed language with a simple syntax
- Supports expressions, statements, variables, and functions
- Built-in functions for array manipulation, JSON parsing, and timing
- Recursive descent parser and tree-walk interpreter
- Parser recovers from syntax errors and reports all of them in one run
- String escapes: `\n`, `\t`, `\r`, `\"`, `\\`
- Number literals with exponents: `1e3`, `2.5E-2`
- Object property access with `obj.name` as sugar for `obj["name"]`
- Error handling with `try`/`catch` and `throw`
- `break` and `continue` in `while`, `for` and `for`-`in` loops
- `for (x in ...)` over arrays, strings and objects
- Anonymous functions: `fun (x) { return x * 2; }`
- Modulo `%` and compound assignment: `+=`, `-=`, `*=`, `/=`, `%=`
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

- **`len(value)`** — Returns the length of an array, object or string (in characters).
- **`append(arr, value1, value2, ...)`** — Returns a new array with the values appended; the original array is not modified.
- **`json(url)`** — Fetches and parses JSON from a URL (object, array or primitive); network, HTTP status and parse errors are reported as runtime errors.
- **`clock()`** — Returns the current time in seconds (with sub-second precision).

## Error Handling

Runtime errors (including those from built-ins like `json()`) can be caught with `try`/`catch`. The catch variable holds the error message as a string; it is optional (`catch { ... }`).

```javascript
try {
    var todos = json("https://dummyjson.com/todos");
    print len(todos.todos);
} catch (err) {
    print "request failed: " + err;
}
```

`throw` raises any value; `catch` receives it unchanged. An uncaught throw stops the program like any runtime error.

```javascript
fun parseAge(v) {
    if (v < 0) throw { "code": "bad_age", "value": v };
    return v;
}

try {
    parseAge(-1);
} catch (e) {
    print e.code; // bad_age
}
```

## Loops

`for (x in arr)` visits array elements, string characters or object keys
(objects are iterated in sorted key order). With two variables you also get
the index, or the key and the value. Each iteration has its own variable, so
closures capture the current element.

```javascript
for (i, name in ["ann", "bob"]) {
    if (name == "ann") continue;
    print name; // bob
}

var ages = { "ann": 31, "bob": 25 };
for (name, age in ages) {
    if (age < 30) break;
    print name + " is 30+";
}
```

## Anonymous Functions

`fun` without a name is an expression, so functions can be passed around inline.

```javascript
fun map(arr, f) {
    var out = [];
    for (x in arr) out = append(out, f(x));
    return out;
}

print map([1, 2, 3], fun (x) { return x * x; }); // [1, 4, 9]
```

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
