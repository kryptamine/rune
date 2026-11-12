![ci](https://github.com/kryptamine/rune/actions/workflows/ci.yml/badge.svg)
![Go](https://img.shields.io/badge/Go-1.26+-blue?logo=go&logoColor=white)
![Python](https://img.shields.io/badge/Python-3.8+-yellow?logo=python&logoColor=white)

# Rune Interpreter

Rune is a dynamically typed interpreted programming language inspired by [Lox](https://github.com/munificent/craftinginterpreters) and built following concepts from the book [Crafting Interpreters](https://craftinginterpreters.com/) by Robert Nystrom. This interpreter is implemented in **Go** and is designed to be cross-platform, compiling into a statically linked binary.

## Features

- Dynamically typed language with a simple syntax
- Supports expressions, statements, variables, and functions
- Variables are immutable by default (`let`); `let mut` makes them mutable
- Standard library in namespaces: `array`, `string`, `math`, `json`, `http`, `time`, `fs`
- Recursive descent parser; the resolved AST is compiled to register bytecode run by a VM
- Parser recovers from syntax errors and reports all of them in one run
- String escapes: `\n`, `\t`, `\r`, `\"`, `\\`, `\{`, `\}`
- String interpolation: `"x = {x}, sum = {a + b}"`
- Number literals with exponents: `1e3`, `2.5E-2`
- Object property access with `obj.name` as sugar for `obj["name"]`
- Immutable structs with required and default fields: `struct Point { x, y = 0 }`
- Destructuring of objects and structs, in `let` and parameters: `let { x, y } = p;`
- Strings compare with `<`, `<=`, `>`, `>=` and index by character: `"héllo"[1]` is `"é"`
- Pattern matching with `match`, as an expression or a statement
- Error handling with `try`/`catch` and `throw`
- Two loops only: `while` and `for x in ...` over arrays, strings, objects and ranges (`a..b`)
- `break` and `continue` in both
- `if`/`else`, `while` and `for` bodies are blocks, or a bare statement starting with a name or keyword: `if v < 0 return;`, `for x in xs array.push(out, x);`
- Anonymous functions: `fun (x) { return x * 2; }`, or arrow functions: `x => x * 2`
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
`[1, 2] == [1, 2]` is `true`. `array.contains` and `array.indexOf` use the same equality.
Values of different types are never equal: `1 == "1"` is `false`.

`print` shows strings inside arrays and objects quoted, so `["1", 1]` and `[1, 1]`
look different; a string printed on its own is written as is.

## String Interpolation

Any expression in `{}` inside a string literal is evaluated and inserted as
`print` would show it; `\{` is a literal brace.

```javascript
let name = "ann";
let xs = [1, "two"];
print "hi {name}, {len(xs)} items: {xs}"; // hi ann, 2 items: [1, "two"]
print "\{name}";                          // {name}
```

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
nested like `o.xs[0] = 1`) and the in-place built-ins `array.push`, `array.pop` and `array.sort`
are errors. Built-ins that return a new value (`array.append`, `array.map`, ...)
work as usual.

```javascript
let a = [1];
a[0] = 3;   // Error at 'a': Cannot mutate immutable variable 'a'.
array.push(a, 2); // Error at 'a': Cannot mutate immutable variable 'a'.

let mut b = [1];
array.push(b, 2); // ok
```

Function parameters, `for`-`in` variables, `catch` variables and functions are
immutable too. Parameters and `for`-`in` variables are made mutable with `mut`:

```javascript
fun fill(mut arr, n) {
    array.push(arr, n);
}

let mut grid = [[0, 0], [0, 0]];
for i, mut row in grid { row[i] = 1; }
```

The check is static and follows the variable, not the value: an array held by
an immutable variable still changes when it is mutated through another,
mutable reference (`let mut b = a;` or a `mut` parameter).

Redeclaring a variable in the same scope is an error, at the top level too.
A new scope may shadow a name with a different mutability.
Built-ins are not declarations, so a program may still redefine them.

## Arrays

Arrays are references: assigning an array or passing it to a function does not copy it, so `array.push`, `array.pop` and index assignment are visible through every reference.

```javascript
let a = [1, 2];
let mut b = a;
array.push(b, 3);
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
`for k, v in p` walks the fields in declaration order. Reading a field
that does not exist is a runtime error. `object(p)` copies a struct into
a new object, `len(p)` is its number of fields and `type(p)` its name.

## Destructuring

A pattern binds properties of an object or fields of a struct to variables,
in `let` and in function parameters. `key: name` renames, `mut x` and
`key: mut name` make a variable mutable, `let mut { ... }` makes them all
mutable. A string key needs a name. A missing property or field is a runtime
error, as with `obj.name`.

```javascript
let { x, y } = { x: 1, y: 2 };
let { x: px, y: mut py, "full name": name } = person;

fun length({ x, y }) {
    return x * x + y * y;
}

length(Point { x: 3, y: 4 }); // 25
```

## Pattern Matching

`match` tries its arms in order and runs the first whose pattern matches;
an arm is `pattern => expression`, `pattern => { block }` or a bare
statement starting with a keyword, `pattern => return x;`, and may have a
guard, `pattern if condition`. As an expression its value is the arm's (a
block or statement gives `nil`) and no arm matching is a `ValueError`. At the start of
a statement it is a statement: values are dropped and no arm matching
does nothing.

```javascript
struct Point { x, y = 0 }

let label = match v {
    nil => "nothing",
    0 | 1 => "bit",
    Point { x: 0, y } => "on the y axis",
    Point { x, y } if x == y => "diagonal",
    Point => "a point",
    { op: "+", a, b } => a + b,
    _ => "other",
};

match cmd {
    "quit" => return;
    "help" => print usage;
    _ => {}
}
```

Patterns:

- `_` matches anything; a name binds the value (`mut name` mutably), unless
  it names a struct type, then it tests the value is of it;
- literals: numbers, strings, `true`, `false`, `nil`, compared as by `==`;
- `p | q` matches if either does; alternatives cannot bind;
- `Point { x, y: 0 }` tests the type, then matches the listed fields;
- `{ key, "other key": p }` matches an object, or a struct, that has the
  keys; other keys are allowed and a missing one is no match.

The value is evaluated once; it may be a struct literal, `match Point { x: 1 } { ... }`,
except an empty one, which needs parentheses. An arm after one that
matches everything is an error.

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
  if len(args) == 0 {
    print "usage: greet <name>";
    return 2;
  }

  print "hi " + args[0];
}
```

`./rune run greet.rn bob` prints `hi bob`.

## Built-in Functions

`len`, `type` and `object` are global; everything else lives in a namespace,
used as `math.floor(x)`:

- **`len(value)`** — Returns the length of an array, object or string (in characters), or the number of fields of a struct.
- **`type(value)`** — Returns `"nil"`, `"boolean"`, `"number"`, `"string"`, `"array"`, `"object"`, `"function"`, a struct's name, `"struct"` for a struct type, or `"namespace"`.
- **`object(s)`** — Returns a new object with the fields of struct `s`.

A namespace is an immutable struct of its members. Unless a program declares
a variable of its name, `ns.member` is bound when the program is compiled:
it costs nothing at run time, and a misspelled member is a compile error.

```javascript
math.flor(1); // Error at 'flor': math has no member 'flor'.
```

### array

- `push(arr, v1, ...)` — appends **in place** (amortized O(1) each) and returns the new length.
- `pop(arr)` — removes the last element in place and returns it; an empty array is a runtime error.
- `append(arr, v1, ...)` — returns a new array with the values appended. It copies, so building an array with it in a loop is quadratic; use `push` for that.
- `map(arr, f)`, `filter(arr, f)`, `reduce(arr, f[, init])` — return new values; `f` may be any function, built-ins included.
- `sort(arr[, cmp])` — sorts **in place** and returns `arr`; stable. Without `cmp` the elements must be all numbers or all strings; `cmp(a, b) < 0` puts `a` first.
- `slice(arr, start[, end])` (negative bounds count from the end), `reverse(arr)`, `contains(arr, item)`, `indexOf(arr, item)`, `join(arr, sep)`.

```javascript
let mut people = [{name: "bob", age: 25}, {name: "ann", age: 31}];
array.sort(people, (a, b) => a.age - b.age);
print array.join(array.map(people, ({ name }) => name), ", "); // bob, ann
```

### string

`split(s, sep)` (`""` splits into characters), `upper(s)`, `lower(s)`, `trim(s)`, `startsWith(s, p)`, `endsWith(s, p)`, `replace(s, old, new)` (all occurrences), `repeat(s, n)`, and `slice`, `reverse`, `contains`, `indexOf` as for arrays, counting characters.

### math

`floor(x)`, `ceil(x)`, `round(x)`, `abs(x)`, `sqrt(x)`, `pow(x, y)`, `min(a, ...)`, `max(a, ...)`, `random()` (in `[0, 1)`), constants `PI` and `INF`.

### json, http, time

- `json.parse(s)` — parses JSON (object, array or primitive); malformed input is a `ValueError`.
- `json.stringify(v)` — encodes `nil`, booleans, numbers, strings, arrays, objects (keys sorted) and structs (fields in order) compactly; a function is a `TypeError`, a non-finite number or a cycle a `ValueError`.
- `http.get(url)` — fetches a URL and returns the body; network errors and a status other than 200 are an `IOError`.
- `time.now()` — the current time in seconds, with sub-second precision.

### fs

Files are read and written whole, as strings; relative paths are taken from the
current directory, not the script's.

- `read(path)`, `lines(path)` — the contents, or its lines without `\n` or `\r\n`.
- `write(path, s)` (creates or truncates), `append(path, s)`.
- `exists(path)`, `stat(path)` — a `FileInfo { name, size, isDir, modified, mode }`, `modified` in seconds like `time.now()`.
- `list(dir)` — the entries as `FileInfo`s, sorted by name.
- `mkdir(path)` (with parents; an existing directory is fine), `remove(path)` (a file or an empty directory), `removeAll(path)`, `rename(from, to)`, `copy(from, to)`.
- `cwd()`, `tempDir()` — a new directory, for the program to remove.
- `join(a, ...)`, `base(p)`, `dir(p)`, `ext(p)`.

A failure is an `IOError` with `path` and `code`: `"notFound"`, `"exists"`,
`"permission"`, `"isDir"`, `"notDir"`, `"notEmpty"`, or `nil` for anything else.

```javascript
try {
    for line in fs.lines("todo.txt") { print line; }
} catch (e: IOError) {
    if e.code != "notFound" throw e;
}
```

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

Runtime errors (including those from built-ins like `http.get()`) can be caught with `try`/`catch`. The catch variable is optional (`catch { ... }`). An error raised by the language is a struct of one of the built-in error types, each with fields `{ message, line = nil, cause = nil }`:

- `TypeError` — a value of the wrong type (`1 + nil`, calling a number)
- `ArgumentError` — the wrong number of arguments
- `ValueError` — the right type but a bad value (`step 0`, `array.pop([])`)
- `IndexError` — an index out of bounds
- `KeyError` — a missing property or struct field
- `NameError` — an undefined variable
- `IOError` — a file system call or `http.get()` failing; it also has `path` and `code` (see `fs`)

A stack overflow is fatal and cannot be caught.

```javascript
try {
    let todos = json.parse(http.get("https://dummyjson.com/todos"));
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

`throw` raises a struct instance, built-in or your own; `catch` receives it
unchanged. Throwing anything else is a `TypeError`. An uncaught throw stops the program like any runtime error,
reported as `Name: message` when the struct has a string `message`.

```javascript
struct AgeError { message, value }

fun parseAge(v) {
    if v < 0 throw AgeError { message: "negative age", value: v };
    return v;
}

try {
    parseAge(-1);
} catch (e: AgeError) {
    print e.value; // -1
}
```

## Loops

`for x in arr { ... }` visits array elements, string characters or object keys
(objects are iterated in sorted key order). With two variables you also get
the index, or the key and the value. Each iteration has its own variable, so
closures capture the current element. Array elements are taken when the loop
starts, so pushing to the array inside the loop does not extend it.
There is no C-style `for`: count with `a..b` (end excluded) or `a..=b` (end
included), optionally `step n`; a `for`-`in` walks it without building the
array. Outside a loop a range is an array. Anything else is a `while`.

```javascript
for i, name in ["ann", "bob"] {
    if name == "ann" continue;
    print name; // bob
}

let ages = { "ann": 31, "bob": 25 };
for name, age in ages {
    if age < 30 break;
    print name + " is 30+";
}

for i in 1..4 { print i; }  // 1 2 3
for i in 1..=3 { print i; } // 1 2 3
for i in 10..0 step -2 { print i; } // 10 8 6 4 2
```

## Anonymous Functions

`fun` without a name is an expression, so functions can be passed around inline.

```javascript
fun map(arr, f) {
    let mut out = [];
    for x in arr { array.push(out, f(x)); }
    return out;
}

print map([1, 2, 3], fun (x) { return x * x; }); // [1, 4, 9]
```

Arrow functions are shorthand: an expression body is returned, a block body works like `fun`'s. Parameters may be `mut` or patterns; a single plain one needs no parentheses. `() => { a: 1 }` returns an object.

```javascript
let add = (a, b) => a + b;
let names = array.map(people, ({ name }) => name);
let log = msg => { print msg; };
```

## Example Program

Here’s a simple Rune script that calculates the sum of an array:

```javascript
fun sumArray(arr) {
    let mut total = 0;
    for x in arr { total += x; }
    return total;
}

print sumArray([1, 2, 3, 4, 5]);
```

## License

This project is open-source and follows the MIT license.

## References

- [Crafting Interpreters](https://craftinginterpreters.com/) by Robert Nystrom
- [Lox Programming Language](https://github.com/munificent/craftinginterpreters)
