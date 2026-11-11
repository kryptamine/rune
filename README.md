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
- Recursive descent parser; the resolved AST is compiled to register bytecode run by a VM
- Parser recovers from syntax errors and reports all of them in one run
- String escapes: `\n`, `\t`, `\r`, `\"`, `\\`
- Number literals with exponents: `1e3`, `2.5E-2`
- Object property access with `obj.name` as sugar for `obj["name"]`
- Immutable structs with required and default fields: `struct Point { x, y = 0 }`
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
[line: 2] TypeError: Cannot read property 'y' of number.
    2 |   return x.y;
      |            ^
  at g (line 2)
  at map (native)
  at <script> (line 5)
```

`run` exits with `65` on a syntax error and `70` on an uncaught runtime error.
`&&` and `||` are reported with a hint to use `and` / `or`.

## Running Tests

```sh
make test                # all tests
make test filter=arrays  # scripts under test/arrays
make update              # rewrite golden files after an intended change
make cover               # coverage
make fuzz FUZZTIME=1m    # fuzz the whole pipeline
```

Each `test/**/*.rn` script runs in-process, in parallel, with a 5s timeout. Directives in comments:

- `// expect: <line>` — next line of stdout;
- `// [line: N] Error...` — compile error, in order; without `[line: N]` it is the comment's line;
- `// expect runtime error: [line: N] <message>` — uncaught runtime error.

`foo.stderr` next to `foo.rn` holds the full error report (snippet, caret, trace); a missing file means empty stderr.
`testdata/<command>/*.rn` are run through the CLI command (`tokenize`, `evaluate`, `run`, `disasm`) and compared with `.golden` files: exit code, stdout and stderr.

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

## Structs

A struct is a named record with a fixed set of fields. Fields without a
default are required; a default must be a literal. Structs are declared at
the top level only and can be used before their declaration.

```javascript
struct Point { x, y = 0 }

let p = Point { x: 1 };
print p;      // Point {x: 1, y: 0}
print p.x;    // 1
print p["y"]; // 0

Point { x: 1, z: 2 }; // Error at 'z': Point has no field 'z'.
Point { y: 2 };       // Error at 'Point': Missing field 'x' of Point.
```

Fields are immutable, whatever the variable: `p.x = 2` is a runtime error.
To change a field, build a new struct; `let mut` only lets the variable be
rebound to it. An array or object held in a field stays mutable.

```javascript
let mut q = Point { x: 1, y: 2 };
q = Point { x: q.x + 1, y: q.y };
q.x = 5; // Cannot set field 'x' of Point: structs are immutable.
```

`with` copies a struct, changing the fields it lists; it binds tighter
than unary operators and looser than calls, so `(p with { x: 2 }).x`
needs the parentheses. The fields are checked at run time.

```javascript
let q = p with { x: 3 }; // Point {x: 3, y: 0}; p is unchanged
p with { z: 1 };         // Point has no field 'z'.
```

`v is Point` is `true` only for a `Point`; it has the precedence of `==`,
and its right operand must be a struct type.

Calling a struct type converts to it. From an object the keys must be
exactly the fields, the defaulted ones optional; a struct of the same
type is returned as is, and any other value is an error. There are no
implicit conversions, and none between struct types.

```javascript
Point({x: 1});       // Point {x: 1, y: 0}
Point({x: 1, z: 2}); // Point has no field 'z'.
Point({y: 1});       // Missing field 'x' of Point.
```

`==` compares structs field by field and only if they are of the same
struct, so `Point { x: 1, y: 2 } == { "x": 1, "y": 2 }` is `false`.
`for (k, v in p)` walks the fields in declaration order. Reading a field
that does not exist is a runtime error. `object(p)` copies a struct into
a new object, `len(p)` is its number of fields and `type(p)` its name.

## Imports

`import a.b;` loads `a/b.rn`, named from the directory of the file passed to `rune`. Imports come first in a file.

```
import lib.math;

print square(3);
```

All files share one top level: a name declared in one file is visible in every other, and declaring it twice is an error. Each file runs once, after the files it imports; an import cycle is an error.
An imported file may only declare things at its top level (`fun`, `let`, `struct`); statements such as `print` belong in the entry file.

## main

If the entry file declares `fun main`, it runs after the top level, with the program's arguments as an array of strings if it takes a parameter. It returns `nil` to exit with 0, or an exit code from 0 to 255.

```
fun main(args) {
  if (len(args) == 0) {
    print "usage: greet <name>";
    return 2;
  }

  print "hi " + args[0];
}
```

`./rune run greet.rn bob` prints `hi bob`.

## Built-in Functions

Rune provides several built-in functions:

- **`len(value)`** — Returns the length of an array, object or string (in characters), or the number of fields of a struct.
- **`type(value)`** — Returns `"nil"`, `"boolean"`, `"number"`, `"string"`, `"array"`, `"object"`, `"function"`, a struct's name, or `"struct"` for a struct type.
- **`object(s)`** — Returns a new object with the fields of struct `s`.
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

Before running, the resolved AST is compiled to bytecode for a register machine, in the style of Lua. In particular:

- every local, top-level variables included, lives in a register of its function's frame; a variable some nested function captures is kept in a heap cell that the closures share;
- a call pushes a frame onto one shared register stack, so it allocates nothing;
- comparisons in `if` and loop conditions jump directly; constant operands and literal expressions are folded; loops test their condition at the bottom;
- a value is two words: numbers, booleans and strings are stored inline and never allocate;
- a struct is one allocation of its fields; the compiler interns field names, so `p.x` finds the field by pointer, with no hashing or string comparison;
- `print` output is buffered and flushed when the program ends (also on a runtime error).

`rune disasm file.rn` prints the bytecode. Benchmarks: `make bench`.

## Error Handling

Runtime errors (including those from built-ins like `json()`) can be caught with `try`/`catch`. The catch variable is optional (`catch { ... }`). An error raised by the language is a struct of one of the built-in error types, each with fields `{ message, line = nil, cause = nil }`:

- `TypeError` — a value of the wrong type (`1 + nil`, calling a number)
- `ArgumentError` — the wrong number of arguments
- `ValueError` — the right type but a bad value (`step 0`, `pop([])`)
- `IndexError` — an index out of bounds
- `KeyError` — a missing property or struct field
- `NameError` — an undefined variable
- `IOError` — `json()` failing to fetch or parse

A stack overflow is fatal and cannot be caught.

```javascript
try {
    let todos = json("https://dummyjson.com/todos");
    print len(todos.todos);
} catch (err) {
    print "request failed: " + err.message;
}
```

A catch clause can name the struct types it catches; clauses are tried in
order, and an error none of them catches goes on up with its trace intact.
A clause without types catches everything and must come last.

```javascript
try {
    parse(s);
} catch (e: IndexError, KeyError) {
    print "lookup failed: " + e.message;
} catch (e: ParseError) {
    print "at " + e.pos;
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
