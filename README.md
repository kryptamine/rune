![ci](https://github.com/kryptamine/rune/actions/workflows/ci.yml/badge.svg)
![Go](https://img.shields.io/badge/Go-1.26+-blue?logo=go&logoColor=white)
![Python](https://img.shields.io/badge/Python-3.8+-yellow?logo=python&logoColor=white)

# Rune Interpreter

Rune is a dynamically typed interpreted programming language inspired by [Lox](https://github.com/munificent/craftinginterpreters) and built following concepts from the book [Crafting Interpreters](https://craftinginterpreters.com/) by Robert Nystrom. This interpreter is implemented in **Go** and is designed to be cross-platform, compiling into a statically linked binary.

## Features

- Dynamically typed language with a simple syntax
- Supports expressions, statements, variables, and functions
- Variables are immutable by default (`let`); `let mut` makes them mutable
- Standard library in namespaces: `array`, `string`, `regex`, `math`, `json`, `http`, `time`, `fs`, `io`, `os`
- Modules with their own top level, `export` and `import a.b as m;`
- Recursive descent parser; the resolved AST is compiled to register bytecode run by a VM
- Parser recovers from syntax errors and reports all of them in one run
- String escapes: `\n`, `\t`, `\r`, `\"`, `\\`, `\{`, `\}`
- String interpolation: `"x = {x}, sum = {a + b}"`
- 64-bit integers and floats, both of type `number`: `7 / 2` is `3.5`, `7 % 2` is `1`, and overflow is an error
- Number literals with exponents, hex, octal, binary and `_`: `1e3`, `0xff`, `0o17`, `0b1010`, `1_000_000`
- Bitwise operators on integers: `&`, `|`, `^`, `~`, `<<`, `>>`
- `Map` and `Set` with keys of any hashable value, in insertion order
- `bytes`: immutable binary data, with literals `b"\x00\xff"`
- Streams: files and HTTP bodies read line by line or in chunks, `for line in fs.open(path)`
- Object property access with `obj.name` as sugar for `obj["name"]`
- Structs with required and default fields, read-only outside their `mut self` methods: `struct Point { x, y = 0 }`
- Methods in `impl` blocks, on structs and built-in types: `p.len()`, `Point.origin()`, `fun move(mut self)`
- Destructuring of objects, structs and arrays, in `let` and parameters: `let { x, y } = p;`, `let [head, ...tail] = xs;`
- Nil-safe access and defaults: `resp?.data?.items ?? []`
- Spread and rest: `[...a, ...b]`, `{...o, x: 1}`, `f(...args)`, `fun log(level, ...parts)`
- Strings compare with `<`, `<=`, `>`, `>=` and index by character: `"héllo"[1]` is `"é"`
- Pattern matching with `match`, as an expression or a statement
- Error handling with `try`/`catch` and `throw`
- Two loops only: `while` and `for x in ...` over arrays, strings, objects and ranges (`a..b`)
- `break` and `continue` in both
- `if`/`else`, `while` and `for` bodies are blocks, or a bare statement starting with a name or keyword: `if v < 0 return;`, `for x in xs out.push(x);`
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

`==` compares numbers (an integer with a float too), strings, bytes, booleans and `nil` by value, functions and streams by identity,
and arrays, objects, maps and sets structurally (element by element, cycles included), so
`[1, 2] == [1, 2]` is `true`. `contains` and `indexOf` use the same equality.
Values of different types are never equal: `1 == "1"` is `false`.

`print` shows strings inside arrays and objects quoted, so `["1", 1]` and `[1, 1]`
look different; a string printed on its own is written as is.

## Numbers

A number is a 64-bit integer or a float; `type` says `"number"` for both
and `math.isInt` tells them apart. A literal without a fraction or an
exponent is an integer; `0x`, `0o` and `0b` prefix hex, octal and binary
ones, and `_` may separate digits. An integer literal past `math.MAX_INT`
is a compile error.

`+`, `-`, `*` and `%` on two integers give an integer, and an integer that
would overflow is a `ValueError`; with a float they give a float. `/`
always gives a float: `6 / 2` is `3`, a float, and `math.idiv(7, 2)` is the
integer `3`. `%` takes the sign of its left operand, and an integer `% 0`
is a `ValueError`. Integers compare exactly with floats, so `1 == 1.0`,
and `9007199254740993 == 9007199254740992.0` is `false`.

```javascript
print math.MAX_INT;        // 9223372036854775807
print math.MAX_INT + 1;    // ValueError: Integer overflow.
print 0xff & 0b1010;       // 10
```

The bitwise operators `&`, `|`, `^`, `~`, `<<` and `>>` take integers, or
floats with an integer value, and give an integer; `>>` keeps the sign, a
shift by 64 or more gives `0` (or `-1`), and a negative one is a
`ValueError`. They bind tighter than comparisons and looser than `+`, with
`|` loosest, then `^`, `&` and the shifts: `1 + 2 & 3` is `3`, and
`flags & MASK == 0` needs no parentheses. Each has a compound assignment:
`&=`, `|=`, `^=`, `<<=`, `>>=`.

Lengths, indices, `int`, ranges of integers and JSON numbers without a
fraction or an exponent are integers; `math.floor`, `ceil` and `round` give
integers when one holds the result.

## Map and Set

`Map()` makes an empty map and `Map(src)` one from an object, another map
or an array of `[key, value]` pairs; `Set()` and `Set(array)` likewise. A
key is any hashable value: `nil`, a boolean, a number (`1` and `1.0` are
one key; `NaN` is none), a string, bytes, a function, a struct type, or a
struct whose fields are hashable, compared by value. An array, object, map
or set is not, as it can change. Both keep their keys in the order they
were first added.

```javascript
let mut seen = Map();
seen[[1, 2]];                      // TypeError: Cannot use an array as a key.

struct Pos { x, y }
let mut grid = Map();
grid[Pos { x: 0, y: 1 }] = "wall";
print grid[Pos { x: 0, y: 1 }];    // wall
print grid.get(Pos { x: 9, y: 9 }, "floor"); // floor

let tags = Set(["a", "b", "a"]);
tags.add("c");
print tags;                        // Set {"a", "b", "c"}
print tags.has("b");               // true
```

`m[k]` reads a key and a missing one is a `KeyError`; `m?.[k]` gives `nil`
instead. `m[k] = v` sets one, like an array element, so `m` must be
mutable. Methods: `get(k[, default])`, `set(k, v)` (returns the map),
`has(k)`, `delete(k)` (whether it was there), `clear()`, and `keys()`,
`values()`, `entries()`, new arrays. A set has `add(x)` (returns the set),
`has`, `delete`, `clear`, `values()`, and `union(s)`, `intersection(s)`,
`difference(s)`, new sets.

`len` counts the entries; `for k, v in m` walks a map's keys and values,
`for k in m` its keys, and `for x in s` or `for i, x in s` a set. A loop
may set and delete keys, but adding one is a `ValueError`; walk `m.keys()`
to change it freely. `==` compares entries in any order. `[...s]` spreads a
set into an array; `json.stringify` writes a set as an array and a map of
string keys as an object, in order. Both are references, like arrays.

## Bytes

`bytes` are immutable binary data: a `b"..."` literal, which takes
`\xNN` escapes and no interpolation, or `bytes.from` a string (its UTF-8),
an array of integers from 0 to 255, or bytes. `b[i]` is the integer byte,
`len(b)` the size; `+` joins and `<` compares bytes, `==` by content, and
`for i, x in b` walks them. Bytes are hashable, and print as a literal.

```javascript
let header = b"\x89PNG";
print header[0];             // 137
print header.hex();          // 89504e47
print bytes.from("é");       // b"\xc3\xa9"
print b"hi".toString();      // hi
```

The `bytes` namespace has `from(v)`, `alloc(n[, fill])`, `fromHex(s)` and
`fromBase64(s)`; bytes have the methods `toString()` (UTF-8; anything else is a `ValueError`), `toArray()`,
`slice(start[, end])` (sharing the data), `indexOf(sub)`, `concat(b, ...)`,
`hex()` and `base64()`.

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
nested like `o.xs[0] = 1`) is a compile error, and the in-place methods
`push`, `pop` and `sort` a runtime `TypeError`, as is taking one as a value.
Methods that return a new value (`append`, `map`, ...) work as usual.

```javascript
let a = [1];
a[0] = 3;   // Error at 'a': Cannot mutate immutable variable 'a'.
a.push(2);  // TypeError: Cannot call in-place method 'push' of array: its receiver is in an immutable variable.

let mut b = [1];
b.push(2); // ok
```

Function parameters, `for`-`in` variables, `catch` variables and functions are
immutable too. Parameters and `for`-`in` variables are made mutable with `mut`:

```javascript
fun fill(mut arr, n) {
    arr.push(n);
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

Arrays are references: assigning an array or passing it to a function does not copy it, so `push`, `pop` and index assignment are visible through every reference.

```javascript
let a = [1, 2];
let mut b = a;
b.push(3);
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

let x = 5;
Point { x };          // shorthand for Point { x: x }

Point { x: 1, z: 2 }; // Error at 'z': Point has no field 'z'.
Point { y: 2 };       // Error at 'Point': Missing field 'x' of Point.
```

Fields are read-only: only a `mut self` method of the struct can set them,
on `self` (see [Methods](#methods)). Anywhere else `p.x = 2` is a runtime
error, whatever the variable. An array or object held in a field stays
mutable.

```javascript
let mut q = Point { x: 1, y: 2 };
q.x = 5; // Cannot set field 'x' of Point: only a mut self method can set the fields of self.
q = Point { x: q.x + 1, y: q.y }; // rebinding is fine
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

## Methods

An `impl` block adds methods to a struct declared in the same file. A
method taking `self` first is called on an instance, `self` being the
instance; one without is static and called on the type. Impl blocks are
top-level only, may come before the struct and there may be several; a
method cannot share a name with a field or another method of the type.

```javascript
struct Point { x, y = 0 }

impl Point {
  fun origin() { return Point { x: 0 }; }
  fun len(self) { return math.sqrt(self.x * self.x + self.y * self.y); }
  fun moved(self, dx) { return Point { x: self.x + dx, y: self.y }; }
}

let p = Point { x: 3, y: 4 };
p.len();           // 5
p.moved(1).len();  // methods chain; p is unchanged
Point.origin();    // Point {x: 0, y: 0}
Point.origin().origin(); // 'origin' is a static method of Point; call it as Point.origin().
```

`mut self` makes a mutating method, the only code that can set fields:
those of `self` (`self.x += 1`, `self["x"] = 1`), or `self` as a whole.
Structs are values, so the method works on a copy, which replaces the
receiver when it returns; nothing else holding the old struct sees the
change, and if the method throws, the receiver keeps its old value. The
receiver must be a mutable variable, or an element or property of an array
or object one holds; a struct's field changes only through that struct's
own `mut self` method. Without `mut`, `self` is immutable.

```javascript
struct Counter { n = 0 }

impl Counter {
  fun incr(mut self) { self.n++; }
}

let mut c = Counter {};
let old = c;
c.incr();         // c is Counter {n: 1}; old is unchanged
let mut cs = [Counter {}];
cs[0].incr();     // cs is [Counter {n: 1}]
c.n = 5;          // Cannot set field 'n' of Counter: only a mut self method can set the fields of self.

let d = Counter {};
d.incr(); // Cannot call mutating method 'incr' of Counter: the receiver is not a mutable variable, element or property.
```

`p.len` without a call is the method bound to `p`, a function taking the
rest of the arguments; `Point.len` is the plain function, taking the
instance first, so `points.map(Point.len)` works. A mutating method
cannot be taken as a value.

`impl` also takes `string`, `number`, `boolean`, `array`, `object`,
`function`, `Map`, `Set` and `bytes`. Their methods must take `self`, not
`mut self`, and are seen by the whole program; an object's own property of
the same name wins, and a method of the program over a built-in one.
Built-in structs, such as `IOError`, take no methods.

```javascript
impl string {
  fun shout(self) { return self.upper() + "!"; }
}

print "hi".shout(); // HI!
```

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

An array pattern takes elements by position: `_` skips one, `...name`
takes the rest as a new array and must come last. The array may be longer
than the pattern; a shorter one is an `IndexError`, anything but an array
a `TypeError`.

```javascript
let [a, _, mut c, ...rest] = [1, 2, 3, 4, 5]; // a = 1, c = 3, rest = [4, 5]
let mut [x, y] = [1, 2];

fun swap([p, q]) { return [q, p]; }
pairs.map(([k, v]) => k + "=" + v);
```

## Nil Safety

`a?.b`, `a?.[i]` and `a?.(args)` are `nil` when `a` is, and the rest of the
chain is skipped: `a?.b.c` is `nil` for a `nil` `a`, without reading `.c`.
A missing property, field or element read with `?.` is `nil` too, so
`o?.name` never fails on an object or struct; a value that has no
properties at all (`5?.x`) is still a `TypeError`. A chain with `?.` cannot
be assigned to.

`a ?? b` is `a` unless it is `nil`, then `b`, which runs only then. Unlike
`or`, it keeps `false`, `0` and `""`. It binds looser than `or` and tighter
than ranges.

```javascript
let resp = json.parse(body);
let items = resp?.data?.items ?? [];
let name = user?.profile?.name ?? "anonymous";
let n = opts?.retries ?? 3; // 0 stays 0
cb?.(result);
```

## Spread and Rest

`...` spreads an array into an array literal or a call's arguments, and an
object or struct into an object literal, where later keys win. Spreading
anything else is a `TypeError`.

```javascript
let all = [...xs, 0, ...ys];
let opts = { ...defaults, timeout: 5 };
let p2 = { ...p, z: 0 };       // a struct's fields into an object
print math.max(...xs);
```

A last parameter `...name` (or `...mut name`) takes the arguments after the
others as an array, possibly empty; fewer than the others is an `ArgumentError`.

```javascript
fun log(level, ...parts) {
    io.eprint(level + ": " + parts.join(" "));
}

log("info", "started", "in", 3, "ms");
let count = (...args) => len(args);
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

## Modules

`import a.b;` loads `a/b.rn`, named from the directory of the file passed to `rune`, and binds it as `b`;
`import a.b as m;` binds it as `m`. Imports come first in a file.

Each file has its own top level: names it declares are private unless marked `export` (`fun`, `let`, `struct`),
and another file reaches them only through the module, as `m.name`. Two modules may declare the same name.

```
// lib/geo.rn
export struct Point { x, y = 0 }
export let origin = Point { x: 0 };
export fun dist(p) { return math.sqrt(p.x * p.x + p.y * p.y); }
fun helper() {} // private

// main.rn
import lib.geo;
import lib.geo as g;

print geo.dist(geo.Point { x: 3, y: 4 }); // 5
let p = g.Point({x: 1});
print p is g.Point;                         // true
match p { g.Point { x, y } => print x; _ => {} }
try { ... } catch (e: g.ParseError) { ... }
```

Members are bound when the program is compiled: a member a module does not export is a compile error,
and so is using a module as a value (`print geo;`, `geo[k]`). A module's variables cannot be assigned from
outside (`geo.origin = ...`); the array or object a `let mut` export holds may be changed, one held by an
immutable `let` may not. Importing a module shadows a built-in namespace of the same name in that file,
so `import lib.math;` hides `math.floor`; import it `as` another name to keep both.

Each file runs once, after the files it imports; an import cycle is an error.
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

`len`, `type`, `object`, `str`, `number`, `int`, `Map` and `Set` are global;
the operations on a value are its methods, as `xs.map(f)` or `s.split(",")`;
everything else lives in a namespace, used as `math.floor(x)`:

- **`len(value)`** — Returns the length of an array, object or string (in characters), the number of fields of a struct, the entries of a Map or Set, or the size of bytes.
- **`type(value)`** — Returns `"nil"`, `"boolean"`, `"number"`, `"string"`, `"array"`, `"object"`, `"function"`, `"Map"`, `"Set"`, `"bytes"`, `"Stream"`, a struct's name, `"struct"` for a struct type, or `"namespace"`.
- **`object(s)`** — Returns a new object with the fields of struct `s`.
- **`str(v)`** — `v` as `print` shows it.
- **`number(v)`** — a number as is, or a string holding a number literal, optionally negated and surrounded by spaces (`"42"`, `" -2.5e1 "`), an integer unless it has a fraction or an exponent or int64 cannot hold it; anything else, `".5"`, `"+1"` or `"inf"` included, is a `ValueError`.
- **`int(v[, base])`** — a number truncated toward zero (one past the integers is a `ValueError`), or a string parsed as an integer in `base` (2 to 36, default 10): `int("ff", 16)` is `255`.

A namespace is an immutable struct of its members. Unless a program declares
a variable of its name, `ns.member` is bound when the program is compiled:
it costs nothing at run time, and a misspelled member is a compile error.

```javascript
math.flor(1); // Error at 'flor': math has no member 'flor'.
```

### Array methods

- `push(v1, ...)` — appends **in place** (amortized O(1) each) and returns the new length.
- `pop()` — removes the last element in place and returns it; an empty array is a runtime error.
- `append(v1, ...)` — returns a new array with the values appended. It copies, so building an array with it in a loop is quadratic; use `push` for that.
- `map(f)`, `filter(f)`, `reduce(f[, init])` — return new values; `f` may be any function, built-ins included.
- `sort([cmp])` — sorts **in place** and returns the array; stable. Without `cmp` the elements must be all numbers or all strings; `cmp(a, b) < 0` puts `a` first.
- `slice(start[, end])` (negative bounds count from the end), `reverse()`, `contains(item)`, `indexOf(item)`, `join(sep)`.

```javascript
let mut people = [{name: "bob", age: 25}, {name: "ann", age: 31}];
people.sort((a, b) => a.age - b.age);
print people.map(({ name }) => name).join(", "); // bob, ann
```

### String and number methods

`split(sep)` (`""` splits into characters), `upper()`, `lower()`, `trim()`, `startsWith(p)`, `endsWith(p)`, `replace(old, new)` (all occurrences), `repeat(n)`, and `slice`, `reverse`, `contains`, `indexOf` as for arrays, counting characters.

- `charCode([i])` — the code point of character `i` (default 0); `string.fromCharCode(c, ...)` — the string of those code points.
- `padStart(width[, pad])`, `padEnd(width[, pad])` — the string filled to `width` characters with `pad` (default `" "`), repeated and cut.
- `format(...)` — printf, the string being the format: `%[flags][width][.precision]verb`, flags `-+ 0#`. `%v` and `%s` show a value as `print` does, `%q` as inside an array; `%d %x %X %o %b` take integers, `%f %e %E %g %G` numbers; `%%` is `%`. Widths count characters; a missing or extra argument is an `ArgumentError`.

```javascript
"%-6s|%5.2f|%03d".format("ab", 3.14159, 7); // "ab    | 3.14|007"
```

A number has `fixed(digits)`, with 0 to 100 decimals: `3.14159.fixed(2)` is `"3.14"`.

### regex

Patterns are RE2 syntax (no backreferences; matching is linear in the input). A bad pattern is a `ValueError`.
Indices count characters.

- `test(re, s)` — whether `re` matches anywhere in `s`.
- `find(re, s)` — the first `Match { text, index, groups, named }`, or `nil`; `groups` holds each group's text, `nil` if it did not take part, `named` the named ones (`(?P<name>...)`).
- `findAll(re, s[, n])` — all matches, at most `n`.
- `replace(re, s, with)` — every match replaced; `with` is a string, where `$1` or `${name}` is a group, or a function from a `Match` to a string.
- `split(re, s[, n])` — the parts between matches, at most `n`; `escape(s)` — `s` with the special characters quoted.

```javascript
let m = regex.find("(\\d+)-(\\d+)", "pages 10-20");
print m.groups; // ["10", "20"]
print regex.replace("\\s+", "a  b   c", " "); // a b c
```

### math

`floor(x)`, `ceil(x)`, `round(x)` (integers when one holds the result), `abs(x)`, `sqrt(x)`, `pow(x, y)`, `min(a, ...)`, `max(a, ...)` (the argument itself), `idiv(a, b)` (integer division, toward zero), `isInt(x)`, `random()` (in `[0, 1)`), constants `PI`, `INF`, `MAX_INT` and `MIN_INT`.

### json, http, time

- `json.parse(s)` — parses JSON (object, array or primitive); malformed input is a `ValueError`.
- `json.stringify(v)` — encodes `nil`, booleans, numbers, strings, arrays, objects (keys sorted) and structs (fields in order) compactly; a function is a `TypeError`, a non-finite number or a cycle a `ValueError`.
- `http.get(url)` — fetches a URL and returns the body; network errors and a status other than 200 are an `IOError`.
- `http.request(method, url[, opts])` — sends a request and returns `HttpResponse { status, headers, body }`, header names lowercased and repeated ones joined with `, `. Any status is a response; a network error or timeout is an `IOError`. `opts` is an `HttpOptions { headers, body, timeout, stream }`, or an object converted to one: `headers` an object of strings, `body` a string or bytes, `timeout` seconds (default 30). With `stream: true`, `body` is a `Stream` to read and close, and the timeout covers only getting the response.

```javascript
let r = http.request("POST", "https://example.com/api", {
    headers: { "Content-Type": "application/json" },
    body: json.stringify({ name: "ann" }),
});
if r.status != 201 throw IOError { message: "status {r.status}" };
```
- `time.now()` — the current time in seconds, with sub-second precision.

### fs

Files are read and written whole, as strings; relative paths are taken from the
current directory, not the script's.

- `read(path)`, `lines(path)` — the contents, or its lines without `\n` or `\r\n`; `readBytes(path)` — the contents as bytes.
- `write(path, data)` (creates or truncates), `append(path, data)`, of a string or bytes.
- `open(path[, mode])` — a `Stream` of the file: `"r"` (default) to read, `"w"` to write (creating or truncating), `"a"` to append.
- `exists(path)`, `stat(path)` — a `FileInfo { name, size, isDir, modified, mode }`, `modified` in seconds like `time.now()`.
- `list(dir)` — the entries as `FileInfo`s, sorted by name.
- `mkdir(path)` (with parents; an existing directory is fine), `remove(path)` (a file or an empty directory), `removeAll(path)`, `rename(from, to)`, `copy(from, to)`.
- `cwd()`, `tempDir()` — a new directory, for the program to remove.

### Streams

A `Stream` is an open file from `fs.open`, or a response body from
`http.request` with `stream: true`. Reading:

- `readLine()` — the next line without `\n` or `\r\n`, or `nil` at the end;
- `read(n)` — bytes, at most `n` and at least one, or `nil` at the end;
- `readAll()` — the rest as bytes; `readText()` — the rest as a string.

`for line in s` (or `for i, line in s`) reads lines up to the end, so a
large file is never held whole. Writing: `write(data)` of a string or bytes,
buffered, and `flush()`. `close()` flushes and closes it, and closing again
does nothing; streams the program leaves open are closed when it ends. A
failure, reading a closed stream included, is an `IOError`.

```javascript
let out = fs.open("upper.txt", "w");
for line in fs.open("words.txt") {
    out.write(line.upper() + "\n");
}
out.close();
```

### io

- `readLine()` — the next line of stdin without `\n` or `\r\n`, or `nil` at the end.
- `read()` — the rest of stdin.
- `write(v)` — writes `v` to stdout as `print` does, without a newline.
- `eprint(v)`, `ewrite(v)` — the same to stderr, with and without a newline.

Reading stdin or writing stderr first flushes what was printed.

### os

- `env(name)` — the variable's value, or `nil` if unset; `setEnv(name, v)`, `unsetEnv(name)`; `environ()` — all as an object.
- `exit([code])` — exits with `code` (0 to 255, default 0) at once; `try` does not catch it.
- `exec(cmd[, args[, opts]])` — runs `cmd` with the array of string `args` and waits; returns `ExecResult { code, stdout, stderr }`. A non-zero exit is not an error; a command not found is an `IOError` with code `notFound`. `opts` is an `ExecOptions { input, dir, env }`, or an object converted to one: `input` is stdin, `env` an object added to the current environment; all are optional.
- `platform` — `"darwin"`, `"linux"`, `"windows"`, ...

```javascript
let r = os.exec("git", ["status", "--short"], { dir: "repo" });
if r.code != 0 {
    io.eprint(r.stderr);
    os.exit(r.code);
}
```
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
- `ValueError` — the right type but a bad value (`step 0`, `[].pop()`)
- `IndexError` — an index out of bounds
- `KeyError` — a missing property or struct field
- `NameError` — an undefined variable
- `IOError` — a file system call or an `http` request failing; it also has `path` and `code` (see `fs`)

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
    for x in arr { out.push(f(x)); }
    return out;
}

print map([1, 2, 3], fun (x) { return x * x; }); // [1, 4, 9]
```

Arrow functions are shorthand: an expression body is returned, a block body works like `fun`'s. Parameters may be `mut` or patterns; a single plain one needs no parentheses. `() => { a: 1 }` returns an object.

```javascript
let add = (a, b) => a + b;
let names = people.map(({ name }) => name);
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
