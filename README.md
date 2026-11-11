![ci](https://github.com/kryptamine/rune/actions/workflows/ci.yml/badge.svg)
![Go](https://img.shields.io/badge/Go-1.26+-blue?logo=go&logoColor=white)
![Python](https://img.shields.io/badge/Python-3.8+-yellow?logo=python&logoColor=white)

# Rune Interpreter

Rune is a dynamically typed interpreted programming language inspired by [Lox](https://github.com/munificent/craftinginterpreters) and built following concepts from the book [Crafting Interpreters](https://craftinginterpreters.com/) by Robert Nystrom. This interpreter is implemented in **Go** and is designed to be cross-platform, compiling into a statically linked binary.

## Features

- Dynamically typed language with a simple syntax
- Supports expressions, statements, variables, and functions
- Variables are immutable by default (`let`); `let mut` makes them mutable
- Built-in functions for array manipulation, JSON parsing, and timing
- Recursive descent parser; the resolved AST is compiled into Go closures before it runs
- Parser recovers from syntax errors and reports all of them in one run
- String escapes: `\n`, `\t`, `\r`, `\"`, `\\`
- Number literals with exponents: `1e3`, `2.5E-2`
- Object property access with `obj.name` as sugar for `obj["name"]`
- Strings compare with `<`, `<=`, `>`, `>=` and index by character: `"héllo"[1]` is `"é"`
- Error handling with `try`/`catch` and `throw`
- Two loops only: `while` and `for (x in ...)` over arrays, strings, objects and ranges (`a..b`)
- `break` and `continue` in both
- Anonymous functions: `fun (x) { return x * 2; }`
- Modulo `%` and compound assignment: `+=`, `-=`, `*=`, `/=`, `%=`
- Increment and decrement, prefix and postfix: `++i`, `i++`, `--i`, `i--` (on variables, array elements and properties)
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

## Errors and Exit Codes

Errors show the offending source line with a caret under the column;
runtime errors also list the calls they unwound through:

```
[line: 2] Cannot read property 'y' of number.
    2 |   return x.y;
      |            ^
  at g (line 2)
  at map (native)
  at <script> (line 5)
```

`run` exits with `65` on a syntax error and `70` on an uncaught runtime error.
`&&` and `||` are reported with a hint to use `and` / `or`.

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

## Values

Falsy values are `nil`, `false`, `0` and `""`; everything else, including empty
arrays and objects, is truthy.

`==` compares numbers, strings, booleans and `nil` by value, functions by identity,
and arrays and objects structurally (element by element, cycles included), so
`[1, 2] == [1, 2]` is `true`. `contains` and `indexOf` use the same equality.
Values of different types are never equal: `1 == "1"` is `false`.

`print` shows strings inside arrays and objects quoted, so `["1", 1]` and `[1, 1]`
look different; a string printed on its own is written as is.

## Variables

`let` declares an immutable variable, `let mut` a mutable one. Assigning to an
immutable variable (`=`, `+=`, `++`, ...) is a compile-time error. An immutable
variable must be initialized; `let mut x;` starts as `nil`.

```javascript
let mut count = 0;
count += 1;

let name = "ann";
name = "bob"; // Error at 'name': Cannot assign to immutable variable 'name'.
```

Through an immutable variable the array or object it holds cannot be changed
either: index and property assignment (`a[0] = 1`, `o.x += 1`, `a[0]++`, also
nested like `o.xs[0] = 1`) and the in-place built-ins `push`, `pop` and `sort`
are errors. Built-ins that return a new value (`append`, `reverse`, `map`, ...)
work as usual.

```javascript
let a = [1];
a[0] = 3;   // Error at 'a': Cannot mutate immutable variable 'a'.
push(a, 2); // Error at 'a': Cannot mutate immutable variable 'a'.

let mut b = [1];
push(b, 2); // ok
```

Function parameters, `for`-`in` variables, `catch` variables and functions are
immutable too. Parameters and `for`-`in` variables are made mutable with `mut`:

```javascript
fun fill(mut arr, n) {
    push(arr, n);
}

let mut grid = [[0, 0], [0, 0]];
for (i, mut row in grid) row[i] = 1;
```

The check is static and follows the variable, not the value: an array held by
an immutable variable still changes when it is mutated through another,
mutable reference (`let mut b = a;` or a `mut` parameter).

Redeclaring a variable in the same scope is an error, at the top level too.
A new scope may shadow a name with a different mutability.
Built-ins are not declarations, so a program may still redefine them.

## Arrays

Arrays are references: assigning an array or passing it to a function does not copy it, so `push`, `pop` and index assignment are visible through every reference.

```javascript
let a = [1, 2];
let mut b = a;
push(b, 3);
print a; // [1, 2, 3]
```

## Built-in Functions

Rune provides several built-in functions:

- **`len(value)`** — Returns the length of an array, object or string (in characters).
- **`append(arr, value1, value2, ...)`** — Returns a new array with the values appended; the original array is not modified. It copies the array, so building an array with it in a loop takes quadratic time; use `push` for that.
- **`push(arr, value1, value2, ...)`** — Appends the values to the array in place (amortized O(1) each) and returns its new length.
- **`pop(arr)`** — Removes the last element of the array in place and returns it; an empty array is a runtime error.
- **`json(url)`** — Fetches and parses JSON from a URL (object, array or primitive); network, HTTP status and parse errors are reported as runtime errors.
- **`clock()`** — Returns the current time in seconds (with sub-second precision).

### Math

`floor(x)`, `ceil(x)`, `round(x)`, `abs(x)`, `sqrt(x)`, `pow(x, y)`, `min(a, ...)`, `max(a, ...)`, `random()` (in `[0, 1)`), constants `PI` and `INF`.

### Strings

`split(s, sep)` (`""` splits into characters), `join(arr, sep)`, `upper(s)`, `lower(s)`, `trim(s)`, `startsWith(s, p)`, `endsWith(s, p)`, `replace(s, old, new)` (all occurrences), `repeat(s, n)`.

### Arrays and strings

- `map(arr, f)`, `filter(arr, f)`, `reduce(arr, f[, init])` — return new values; `f` may be any function, built-ins included.
- `sort(arr[, cmp])` — sorts **in place** and returns `arr`; stable. Without `cmp` the elements must be all numbers or all strings; `cmp(a, b) < 0` puts `a` first.
- `slice(x, start[, end])`, `reverse(x)`, `contains(x, item)`, `indexOf(x, item)` — work on arrays and strings; string indices count characters, negative slice bounds count from the end.

```javascript
let mut people = [{name: "bob", age: 25}, {name: "ann", age: 31}];
sort(people, fun (a, b) { return a.age - b.age; });
print join(map(people, fun (p) { return p.name; }), ", "); // bob, ann
```

Built-ins are ordinary globals, so a program may redefine them.

## Performance

Before running, the resolved AST is compiled into a tree of Go closures, so each node costs one indirect call and the decisions that depend only on the source are taken once. In particular:

- local variables live in slots resolved ahead of time, and calls to functions that create no closures reuse pooled environments instead of allocating;
- conditions of `if` and loops yield a Go bool directly; constant operands and literal expressions are folded; `i = i + 1`, `i += 1` and `i++` are single nodes;
- numbers, booleans and strings are stored inline in values and never allocate;
- `print` output is buffered and flushed when the program ends (also on a runtime error).

Benchmarks: `make bench`.

## Error Handling

Runtime errors (including those from built-ins like `json()`) can be caught with `try`/`catch`. The catch variable is optional (`catch { ... }`); for an error raised by the language it holds an object `{message, line}`.

```javascript
try {
    let todos = json("https://dummyjson.com/todos");
    print len(todos.todos);
} catch (err) {
    print "request failed: " + err.message;
}
```

`throw` raises any value; `catch` receives it unchanged, so a thrown value is never confused with a language error. An uncaught throw stops the program like any runtime error.

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
closures capture the current element. Array elements are taken when the loop
starts, so pushing to the array inside the loop does not extend it.
There is no C-style `for`: count with `a..b` (end excluded) or `a..=b` (end
included), optionally `step n`; a `for`-`in` walks it without building the
array. Outside a loop a range is an array. Anything else is a `while`.

```javascript
for (i, name in ["ann", "bob"]) {
    if (name == "ann") continue;
    print name; // bob
}

let ages = { "ann": 31, "bob": 25 };
for (name, age in ages) {
    if (age < 30) break;
    print name + " is 30+";
}

for (i in 1..4) print i;  // 1 2 3
for (i in 1..=3) print i; // 1 2 3
for (i in 10..0 step -2) print i; // 10 8 6 4 2
```

## Anonymous Functions

`fun` without a name is an expression, so functions can be passed around inline.

```javascript
fun map(arr, f) {
    let mut out = [];
    for (x in arr) push(out, f(x));
    return out;
}

print map([1, 2, 3], fun (x) { return x * x; }); // [1, 4, 9]
```

## Example Program

Here’s a simple Rune script that calculates the sum of an array:

```javascript
fun sumArray(arr) {
    let mut total = 0;
    for (x in arr) total += x;
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
