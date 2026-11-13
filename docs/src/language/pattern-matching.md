# Pattern Matching

`match` tries its arms in order and runs the first whose pattern matches.

```rune
fun status(code) {
    return match code {
        200 | 201 | 204 => "ok",
        301 | 302 => "redirect",
        404 => "not found",
        c if c >= 500 => "server error {c}",
        _ => "unknown",
    };
}

print(status(201)); // ok
print(status(404)); // not found
print(status(503)); // server error 503
print(status(418)); // unknown
```

An arm is `pattern => expression`, ended by `,` or `;`, `pattern => { block }`,
or a bare statement starting with a keyword, `pattern => return x;`. As
after an arrow function's `=>`, braces that look like an object are one:
`1 => { a: 1 }` and `1 => { x }` give an object. A guard,
`pattern if condition`, adds a test after the pattern matches.

## Patterns

- `_` matches anything;
- a name binds the value (`mut name` mutably); a struct type's name is
  an error, test the type with `is`;
- `name is T` and `_ is T` test the value is a `T`, a struct type or a
  built-in one like `string` or `nil`, `_ is T | U` one of them, the same
  [test](values.md#type-tests) as `v is T` and `catch e is T`;
- literals: numbers, strings, `true`, `false`, `nil`, compared as by `==`;
- `p | q` matches if either does;
- `Point { x, y: 0 }` tests the type, then matches the listed fields;
- `{ key, "other key": p }` matches an object, or a struct, that has the
  keys; other keys are allowed and a missing one is no match.

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
    { type: "click", "target id": id } => print("clicked " + id); // clicked save
    { type: "key", key } => print("pressed " + key);
    _ => {}
}
```

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

As an expression its value is the arm's; a block or statement arm gives
`nil`. No arm matching is a `ValueError`:

```rune
let count = 3;
let label = match count {
    0 => "empty",
    1 => "one item",
    n => "{n} items",
};
print(label); // 3 items

print(match 7 { 1 => "a" }); // ValueError: No match arm for 7.
```

At the start of a statement it is a statement: values are dropped, and no
arm matching is a `ValueError` all the same, so add `_ => {}` to ignore the
rest. An arm may `return` from the enclosing function:

```rune
fun handle(cmd) {
    match cmd {
        "quit" => return "bye";
        "help" => print("usage: add | list | quit");
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

An arm after one that matches everything is an error:

```rune
match 1 { _ => 1, 2 => 2 } // Error at '2': Unreachable match arm.
```
