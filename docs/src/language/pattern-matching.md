# Pattern Matching

`match` compares a value against a list of patterns, top to bottom, and
runs the first arm that fits. A pattern can test a value and pull it
apart at the same time, so one `match` often replaces a chain of `if`s
plus the code that reads fields out.

```rune
fun status(code) {
    match code {
        200 | 201 | 204 => "ok",
        301 | 302 => "redirect",
        404 => "not found",
        400..500 => "client error",
        c if c >= 500 => "server error {c}",
        _ => "unknown",
    }
}

print(status(201)); // ok
print(status(404)); // not found
print(status(418)); // client error
print(status(503)); // server error 503
print(status(100)); // unknown
```

Read the arms from the top:

- `200 | 201 | 204` matches any of three literals;
- `400..500` matches a number from 400 up to, but not including, 500;
- `c if c >= 500` binds the value to `c`, then the guard after `if`
  must also hold;
- `_` matches anything, so it catches the rest.

## Arms

An arm is `pattern => expression`, or `pattern => { block }`. A block
arm's value is its last expression, written without a `;`. Arms are
separated by `,`, and a trailing `,` after the last is fine.

```rune
struct Order { country, total }

fun shipping(order) {
    match order.country {
        "US" => 5,
        "CA" | "MX" => 12,
        _ => {
            let base = 20;
            if order.total > 100 { base / 2 } else { base }
        },
    }
}

print(shipping(Order { country: "US", total: 40 }));  // 5
print(shipping(Order { country: "DE", total: 40 }));  // 20
print(shipping(Order { country: "DE", total: 150 })); // 10
```

After `=>`, a `{` always starts a block, as it does after an arrow
function's `=>`.

`return`, `break`, `continue` and `throw` are expressions, so an arm can
leave the function or the loop: `"quit" => return "bye",`.

## Patterns

| Pattern | Matches |
|---|---|
| `_` | anything |
| `name`, `mut name` | anything, and binds it to `name` |
| `42`, `"ok"`, `true`, `nil` | a value equal to the literal, as by `==` |
| `LIMIT`, `m.LIMIT` | a value equal to the [constant](variables.md#constants) |
| `a..b`, `a..=b`, `a..`, `..b`, `..=b` | a number in the range |
| `p \| q` | a value either pattern matches |
| `_ is T`, `name is T \| U`, `name is P` | a value of the given [type](values.md#type-tests-and-patterns), or one `P` matches, bound to `name` |
| `Point { x, y: 0 }` | a `Point` whose fields match |
| `Color.White`, `Expr.Num { value }` | that [enum](../data/enums.md) variant |
| `{ field, field: p, field = d, ...rest }` | any struct that has the fields |
| `[p, q]`, `[p, q = d]`, `[p, ...rest]` | an array of that many elements, or at least that many |

Any pattern may be followed by a guard, `pattern if condition`, which is
checked after the pattern matches.

## Literals and Constants

Literals compare with `==`. A constant's name matches the constant's
value, the same as writing the literal, so limits live in one place:

```rune
const ADMIN = "admin";
const MAX_RETRIES = 3;

fun access(role) {
    match role {
        ADMIN => "everything",
        "editor" | "author" => "posts",
        nil => "nothing, not signed in",
        _ => "read only",
    }
}

fun next(attempt) {
    match attempt {
        MAX_RETRIES => "give up",
        _ => "retry #{attempt + 1}",
    }
}

print(access("admin"));  // everything
print(access("author")); // posts
print(access(nil));      // nothing, not signed in
print(next(1));          // retry #2
print(next(3));          // give up
```

A constant binds instead of matching in two spots: after `mut`, and as a
bare field in a record pattern, `{ LIMIT }`.

## Ranges

A range pattern sorts a number into a band:

- `a..b` from `a` up to `b`, `b` excluded;
- `a..=b` from `a` up to `b`, `b` included;
- `a..` from `a` on, `..b` and `..=b` up to `b`.

A bound is a number literal or a constant holding a number. Floats match
too. Anything that is not a number, `NaN` included, does not.

```rune
const FREEZING = 0;

fun weather(celsius) {
    match celsius {
        ..FREEZING => "freezing",
        FREEZING..15 => "cold",
        15..25 => "mild",
        25.. => "hot",
        _ => "not a temperature",
    }
}

print(weather(-3));   // freezing
print(weather(14.9)); // cold
print(weather(22));   // mild
print(weather(31));   // hot
print(weather("?"));  // not a temperature
```

Ranges also work inside other patterns, combine with `|` and take a
guard. To use the number, bind a name and test it in the guard:

```rune
const SLOW = 250;
struct Resp { status, ms }

fun grade(resp) {
    match resp {
        { status: 500.. } => "error",
        { status: 200..300, ms: ..50 } => "fast",
        { status: 200..300, ms: SLOW.. } => "slow",
        { status: 200..300 } => "ok",
        _ => "other",
    }
}

print(grade(Resp { status: 200, ms: 12.5 })); // fast
print(grade(Resp { status: 204, ms: 90 }));   // ok
print(grade(Resp { status: 201, ms: 400 }));  // slow
print(grade(Resp { status: 503, ms: 1 }));    // error
print(grade(Resp { status: 404, ms: 1 }));    // other
```

```rune
fun reading(v) {
    match v {
        -5..=-1 | 10..20 => "in a normal band",
        [..0, 1..] => "crossed zero",
        n if n is number && n % 2 == 0 => "even {n}",
        n is number => "odd {n}",
        _ => "not a reading",
    }
}

print(reading(15));      // in a normal band
print(reading([-1, 1])); // crossed zero
print(reading(8));       // even 8
print(reading(7));       // odd 7
print(reading("15"));    // not a reading
```

A range never makes a `match` complete on its own: even `..0 | 0..`
still needs `_ =>` for values that are not numbers.

## Types

`name is T` matches a value of type `T` and binds it; `_ is T` only
tests. `T` is a struct, an enum or a built-in type such as `string`,
`number` or `nil`, and `T | U` accepts either. It is the same test as
[`v is T`](values.md#type-tests-and-patterns).

```rune
fun describe(v) {
    match v {
        nil => "nothing",
        n is number => "the number {n}",
        s is string => "text of {s.len()} characters",
        _ is array | map => "a collection",
        _ => "something else",
    }
}

print(describe(nil));        // nothing
print(describe(4.5));        // the number 4.5
print(describe("hello"));    // text of 5 characters
print(describe([1, 2]));     // a collection
print(describe(["id": 1])); // a collection
print(describe(true));       // something else
```

After `is` comes any pattern, in which a bare name is a type. So `name is P`
binds the whole value once `P` matches it, which a guard would otherwise
have to check:

```rune
struct Point { x, y = 0 }

fun classify(v) {
    match v {
        n is 1..10 => "small {n}",
        p is Point { x: 0 } => "on the axis: {p}",
        s is "yes" | "no" => "answer {s}",
        _ => "other",
    }
}

print(classify(3));                   // small 3
print(classify(Point { x: 0, y: 2 })); // on the axis: Point {x: 0, y: 2}
print(classify("no"));                // answer no
print(classify(42));                  // other
```

## Structs

`Point { x, y: 0 }` first checks that the value is a `Point`, then
matches the listed fields. A field written alone, `x`, binds it; a field
with a pattern, `y: 0`, must match it. Fields left out are not checked.

```rune
struct Point { x, y = 0 }

fun locate(p) {
    match p {
        Point { x: 0, y: 0 } => "origin",
        Point { x: 0, y } => "on the y axis at {y}",
        Point { x, y } if x == y => "on the diagonal",
        Point { y: ..0 } => "below the x axis",
        _ is Point => "somewhere else",
        _ => "not a point",
    }
}

print(locate(Point { x: 0 }));        // origin
print(locate(Point { x: 0, y: 5 }));  // on the y axis at 5
print(locate(Point { x: 2, y: 2 }));  // on the diagonal
print(locate(Point { x: 1, y: -4 })); // below the x axis
print(locate(Point { x: 1, y: 2 }));  // somewhere else
print(locate(42));                    // not a point
```

## Records

`{ field, ... }` matches any struct that has the listed fields; extra
fields are ignored and a missing one means no match. This dispatches on
the shape of data without naming its type, as in a tiny calculator:

```rune
struct Bin { op, a, b }
struct Neg { a }

fun eval(e) {
    match e {
        { op: "+", a, b } => eval(a) + eval(b),
        { op: "*", a, b } => eval(a) * eval(b),
        Neg { a } => -eval(a),
        n => n,
    }
}

// 1 + 2 * 3
print(eval(Bin { op: "+", a: 1, b: Bin { op: "*", a: 2, b: 3 } })); // 7
```

```rune
struct Click { target }
struct KeyPress { key }
struct KeyRelease { key }
struct Scroll { dy }

fun handle(event) {
    match event {
        Click { target } => "clicked {target}",
        { key: "Enter" | "Return" } => "submitted",
        { key } => "key {key}",
        _ => "ignored",
    }
}

print(handle(Click { target: "save" }));     // clicked save
print(handle(KeyPress { key: "Enter" }));    // submitted
print(handle(KeyRelease { key: "q" }));      // key q
print(handle(Scroll { dy: 3 }));             // ignored
```

## Arrays

`[p, q]` matches an array of exactly two elements. `[p, ...rest]`
matches one of at least one element, and `rest` gets the others as a new
array; `..._` ignores them. Anything that is not an array, a string
included, does not match.

Array patterns suit command lines, tuples and `entries()` pairs:

```rune
fun run(args) {
    match args {
        [] => "usage: todo <command>",
        ["add", item] => "added {item}",
        ["rm", first, ...more] => "removed {first} and {more.len()} more",
        [cmd, ..._] => "unknown command {cmd}",
    }
}

print(run([]));                     // usage: todo <command>
print(run(["add", "milk"]));        // added milk
print(run(["rm", "milk", "eggs"])); // removed milk and 1 more
print(run(["ls", "-l"]));           // unknown command ls
```

```rune
let results = ["passed": 12, "failed": 0, "skipped": 2];

for pair in results.entries() {
    print(match pair {
        [name, 0] => "{name}: none",
        [name, n] => "{name}: {n}",
    });
}
// passed: 12
// failed: none
// skipped: 2
```

## Defaults and Rest

A field or element may have a default, `name = value`, used when it is
missing, so one arm covers data with optional parts. `...rest` in a
record pattern binds a map of the other fields. These work exactly as
in [destructuring](destructuring.md#defaults):

```rune
fun command(args) {
    match args {
        [cmd, target = "."] => "{cmd} {target}",
        _ => "usage: cmd [target]",
    }
}

print(command(["ls"]));            // ls .
print(command(["ls", "/tmp"]));    // ls /tmp
print(command(["ls", "a", "b"]));  // usage: cmd [target]

struct Guest { name }
struct Member { name, role, id }

fun user(v) {
    match v {
        { name, role = "member", ...extra } => "{name} ({role}) {extra}",
        _ => "not a user",
    }
}

print(user(Guest { name: "ann" }));                         // ann (member) [:]
print(user(Member { name: "bob", role: "admin", id: 7 }));  // bob (admin) ["id": 7]
print(user(5));                                             // not a user
```

## Expression or Statement

Used as a value, `match` gives the value of the arm that ran. A block
arm ending with `;` gives `nil`. If no arm matches, it throws a
`ValueError`:

```rune
let count = 3;
let label = match count {
    0 => "empty",
    1 => "one item",
    n => {
        let noun = if n < 10 { "items" } else { "many items" };
        "{n} {noun}"
    },
};
print(label); // 3 items
```

```rune
print(match 7 { 1 => "one" }); // ValueError: No match arm for 7.
```

At the start of a statement, `match` is a statement and the arms'
values are dropped (unless it is the last expression of a block that has
a value, such as a function body). No arm matching is still a
`ValueError`, so add `_ => {}` to ignore the rest. An arm can `return`
from the enclosing function:

```rune
fun handle(cmd) {
    match cmd {
        "quit" => return "bye",
        "help" => print("usage: add | list | quit"),
        _ => {}
    }
    "continue"
}

print(handle("help"));
print(handle("quit"));
print(handle("save"));
// usage: add | list | quit
// continue
// bye
// continue
```

The value is computed once. It may be a struct literal,
`match Point { x: 1 } { ... }`, except an empty one, which needs
parentheses: `match (Empty {}) { ... }`.

A `match` that names variants of an enum must cover all of them, unless
an arm matches everything; see
[Matching every variant](../data/enums.md#matching-every-variant).

## Mistakes the Compiler Catches

A bare struct name would bind any value, which is almost never what you
meant, so it is an error:

```rune
struct Point { x, y = 0 }
match 1 { Point => 1, _ => 2 } // Error at 'Point': 'Point' is a struct type; test it with '_ is Point'.
```

Alternatives cannot bind names, since only one side would set them:

```rune
match 1 { 1 | x => 1, _ => 2 } // Error at 'x': Cannot bind variables in '|' patterns.
```

An arm after one that matches everything can never run:

```rune
match 1 { _ => 1, 2 => 2 } // Error at '2': Unreachable match arm.
```

A range no number fits in, or a bound that is not a number, is an
error:

```rune
match 1 { 5..5 => 1, _ => 2 } // Error at '..': Range pattern matches no number.
```

```rune
let v = 1;
match 1 { v..3 => 1, _ => 2 } // Error at '..': A bound of a range pattern must be a number or a constant holding one.
```
