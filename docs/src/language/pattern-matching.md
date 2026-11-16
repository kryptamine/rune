# Pattern Matching

`match` tries its arms in order and runs the first whose pattern matches.

```rune
fun status(code) {
    return match code {
        200 | 201 | 204 => "ok",
        301 | 302 => "redirect",
        404 => "not found",
        400..500 => "client error",
        c if c >= 500 => "server error {c}",
        _ => "unknown",
    };
}

print(status(201)); // ok
print(status(404)); // not found
print(status(418)); // client error
print(status(503)); // server error 503
print(status(100)); // unknown
```

An arm is `pattern => expression` or `pattern => { block }`, whose value
is the block's last expression, written without a `;`. Arms are
separated by `,`, and the last may have one too. `return`,
`break`, `continue` and `throw` are expressions, so `pattern => return x,`
works. As after an arrow function's `=>`, `{` starts a block: an object is
`1 => ({ a: 1 })`. A guard, `pattern if condition`, adds a test after the
pattern matches.

## Patterns

- `_` matches anything;
- a name binds the value (`mut name` mutably); a struct type's name is
  an error, test the type with `is`;
- the name of a [constant](variables.md#constants), `LIMIT` or `m.LIMIT`,
  matches its value, as the literal would; as a bare key in `{ LIMIT }`,
  or after `mut`, it still binds;
- `name is T` and `_ is T` test the value is a `T`, a struct type or a
  built-in one like `string` or `nil`, `_ is T | U` one of them, the same
  [test](values.md#type-tests) as `v is T` and `catch e is T`;
- literals: numbers, strings, `true`, `false`, `nil`, compared as by `==`;
- `a..b` matches a number from `a` up to `b`, `b` excluded, `a..=b` up to
  `b` included, `a..` one from `a` on and `..b` or `..=b` one up to `b`; a
  bound is a number or the name of a constant. A float within the bounds
  matches, and anything but a number, `NaN` included, is no match;
- `p | q` matches if either does;
- `Color.White` matches that [enum](../data/enums.md) variant, and
  `Expr.Num { value }` a variant with fields, as a struct pattern;
- `Point { x, y: 0 }` tests the type, then matches the listed fields;
- `{ key, "other key": p }` matches an object, or a struct, that has the
  keys; other keys are allowed and a missing one is no match;
- `[p, q]` matches an array of exactly that many elements, `[p, ...rest]`
  one of at least as many, `rest` taking the others as a new array
  (`..._` ignores them).

Struct patterns test the type and the fields at once:

```rune
struct Point { x, y = 0 }

fun describe(v) {
    return match v {
        nil => "nothing",
        Point { x: 0, y: 0 } => "origin",
        Point { x: 0, y } => "on the y axis at {y}",
        Point { x, y } if x == y => "diagonal",
        _ is Point => "a point",
        _ => "other",
    };
}

print(describe(nil));                  // nothing
print(describe(Point { x: 0 }));       // origin
print(describe(Point { x: 0, y: 5 })); // on the y axis at 5
print(describe(Point { x: 2, y: 2 })); // diagonal
print(describe(Point { x: 1, y: 2 })); // a point
print(describe(42));                   // other
```

Range patterns sort a number into bands, on their own or inside another
pattern:

```rune
const SLOW = 250;

fun grade(resp) {
    return match resp {
        { status: 500.. } => "error",
        { status: 200..300, ms: ..50 } => "fast",
        { status: 200..300, ms: SLOW.. } => "slow",
        { status: 200..300 } => "ok",
        _ => "other",
    };
}

print(grade({ status: 200, ms: 12.5 })); // fast
print(grade({ status: 204, ms: 90 }));   // ok
print(grade({ status: 201, ms: 400 }));  // slow
print(grade({ status: 503, ms: 1 }));    // error
print(grade({ status: 404, ms: 1 }));    // other
```

Ranges combine with `|`, sit in array patterns and take a guard; to use
the number, bind it with a name and test it in the guard:

```rune
const LOW = 10;
const HIGH = 20;

fun where(v) {
    return match v {
        -5..=-1 | LOW..HIGH => "in",
        [..0, 1..] => "sign change",
        4.. if v % 2 == 0 => "even, 4 or more",
        n if n is number => "number {n}",
        _ => "other",
    };
}

print(where(15));      // in
print(where([-1, 1])); // sign change
print(where(8));       // even, 4 or more
print(where(7));       // number 7
print(where("15"));    // other
```

A range no number is in is an error, and so is a bound that is not a
number or a constant holding one:

```rune
match 1 { 5..5 => 1, _ => 2 } // Error at '..': Range pattern matches no number.
```

```rune
let v = 1;
match 1 { v..3 => 1, _ => 2 } // Error at '..': A bound of a range pattern must be a number or a constant holding one.
```

Ranges never make a match exhaustive: `..0 | 0..` still needs `_ =>`
for values that are not numbers.

Object patterns pick data apart by shape, which makes small interpreters
and event handlers short:

```rune
fun eval(e) {
    return match e {
        { op: "+", a, b } => eval(a) + eval(b),
        { op: "*", a, b } => eval(a) * eval(b),
        { op: "neg", a } => -eval(a),
        n => n,
    };
}

print(eval({ op: "+", a: 1, b: { op: "*", a: 2, b: 3 } })); // 7
```

```rune
let event = { type: "click", "target id": "save" };

match event {
    { type: "click", "target id": id } => print("clicked " + id), // clicked save
    { type: "key", key } => print("pressed " + key),
    _ => {}
}
```

Array patterns suit tuples, command lines and `entries()` pairs:

```rune
fun run(args) {
    return match args {
        [] => "usage: tool <command>",
        ["add", name] => "adding " + name,
        ["rm", first, ...more] => "removing {first} and {more.len()} more",
        [cmd, ..._] => "unknown command " + cmd,
    };
}

print(run([]));                   // usage: tool <command>
print(run(["add", "milk"]));      // adding milk
print(run(["rm", "a", "b"]));     // removing a and 1 more
print(run(["ls", "-l"]));         // unknown command ls

for pair in map({ ok: 3, failed: 0 }).entries() {
    print(match pair { [k, 0] => "{k}: none", [k, n] => "{k}: {n}" });
}
// ok: 3
// failed: none
```

Anything but an array, a string included, is no match.

A bare struct name would bind, so it is an error:

```rune
struct Point { x, y = 0 }
match 1 { Point => 1, _ => 2 } // Error at 'Point': 'Point' is a struct type; test it with '_ is Point'.
```

Alternatives cannot bind:

```rune
match 1 { 1 | x => 1, _ => 2 } // Error at 'x': Cannot bind variables in '|' patterns.
```

## Expression or Statement

As an expression its value is the arm's; a block arm ending with `;`
gives `nil`. No arm matching is a `ValueError`:

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

print(match 7 { 1 => "a" }); // ValueError: No match arm for 7.
```

At the start of a statement it is a statement: values are dropped, unless
it ends a block that has a value, as a function body, and no
arm matching is a `ValueError` all the same, so add `_ => {}` to ignore the
rest. An arm may `return` from the enclosing function:

```rune
fun handle(cmd) {
    match cmd {
        "quit" => return "bye",
        "help" => print("usage: add | list | quit"),
        _ => {}
    }
    return "continue";
}

print(handle("help")); // continue
print(handle("quit")); // bye
print(handle("save")); // continue
```

The first call prints `usage: add | list | quit` before `continue`.

## Details

The value is evaluated once. It may be a struct literal,
`match Point { x: 1 } { ... }`, except an empty one, which needs
parentheses: `match (Empty {}) { ... }`.

A match naming variants of an enum must cover them all, unless an arm
matches everything; see [Matching every variant](../data/enums.md#matching-every-variant).

An arm after one that matches everything is an error:

```rune
match 1 { _ => 1, 2 => 2 } // Error at '2': Unreachable match arm.
```
