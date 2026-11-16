# Error Handling

In Rune an error is an ordinary [struct](../data/structs.md) value.
The runtime throws one when something goes wrong, such as a file that is
missing or a string that is not a number, and your code throws its own
with `throw`. There are two tools for dealing with them:

- `try`/`catch` runs code and handles the error if one is thrown;
- `defer` runs cleanup code however a block is left.

## try and catch

Put the code that may fail in `try`; if it throws, the `catch` block
runs with the error, and the program carries on after it.

```rune
fun parsePort(s) {
    try {
        return number(s);
    } catch e {
        print("bad port: " + e.message);
        return 80;
    }
}

print(parsePort("8080"));
print(parsePort("http"));
// 8080
// bad port: number() cannot parse "http" as a number.
// 80
```

When you do not need the error, leave out its name:

```rune
let mut queue = [];

try {
    queue.pop();
} catch {
    print("queue is empty"); // queue is empty
}
```

An error nothing catches stops the program and prints the message with
the source line and a trace; see
[Errors and Exit Codes](../errors-and-exit-codes.md).

## Error Types

Errors raised by the language are structs of these built-in types. Each
has the fields `message`, `line` (the line it was thrown on) and
`cause` (an underlying error, or `nil`):

| Type | Thrown for | Example |
|---|---|---|
| `TypeError` | a value of the wrong type | `1 + nil`, calling a number |
| `ArgumentError` | the wrong number of arguments | `math.sqrt()` |
| `ValueError` | the right type but a bad value | `[].pop()`, `number("x")` |
| `IndexError` | an index out of bounds | `[1, 2][5]` |
| `KeyError` | a missing map key or struct field | `["a": 1]["b"]` |
| `NameError` | an undefined variable | |
| `IOError` | a failed [file](../stdlib/fs.md) or `http` call | `fs.readText("nope")` |

`IOError` also has `path` and `code`, described in [fs](../stdlib/fs.md).

```rune
try {
    [].pop();
} catch e {
    print(e);         // ValueError: array.pop() got an empty array.
    print(e.message); // array.pop() got an empty array.
    print(e.line);    // 2
}
```

Not everything is an error you can catch: a stack overflow ends the
program at once, and `os.exit` exits even from inside `try`.

## Catching by Type

`catch e is T` catches only errors of type `T`; `catch e is T | U`
catches either. This is the same [test](values.md#type-tests-and-patterns) as
`v is T`. Clauses are tried in order. An error that no clause takes goes
on up to the caller, with its trace intact. A clause without a type
catches everything, so it must come last.

```rune
fun loadPort(raw) {
    try {
        return json.parse(raw)["port"] + 1;
    } catch e is KeyError | IndexError {
        return "missing: " + e.message;
    } catch e is ValueError {
        return "invalid JSON";
    } catch e {
        return "other: " + string(e);
    }
}

print(loadPort(r"""{"port": 80}"""));   // 81
print(loadPort("\{}"));                 // missing: Key not found: "port".
print(loadPort("not json"));            // invalid JSON
print(loadPort(r"""{"port": "80"}""")); // other: TypeError: Operands must be two numbers, two strings or two bytes.
```

Catch only what you expect. Here a missing key gives a default, but a
bug, such as indexing something that is not a map, still
surfaces:

```rune
fun setting(config, key) {
    try {
        return config[key];
    } catch e is KeyError {
        return nil;
    }
}

let config = ["port": 8080];
print(setting(config, "port")); // 8080
print(setting(config, "host")); // nil
```

## try Gives a Value

Like [`if`](conditionals.md#if-as-an-expression), `try` is an
expression: it gives the last expression of the block that ran, the
`try` block or the `catch` that handled the error. That turns "parse
this, or use a default" into one line:

```rune
let port = try { number("http") } catch { 80 };
print(port); // 80
```

```rune
fun field(record, key) {
    try { record[key] } catch e is KeyError { "-" } catch e is TypeError { "not a record" }
}

print(field(["id": 7], "id"));   // 7
print(field(["id": 7], "name")); // -
print(field(5, "id"));           // not a record

print(try { json.parse("[1,") } catch e { e is ValueError }); // true
```

A `catch` block may also `return`, `break`, `continue` or `throw`. This
loop keeps only the numbers it can parse:

```rune
let mut ports = [];
for s in ["80", "http", "443"] {
    ports.push(try { number(s) } catch { continue });
}
print(ports); // [80, 443]
```

## throw

`throw` raises a struct instance, either a built-in error or one of your
own. `catch` receives the very same value, with all its fields, so an
error can carry whatever the handler needs:

```rune
struct StockError { message, item, left }

fun reserve(stock, item, qty) {
    let left = stock?.[item] ?? 0;
    if qty > left {
        throw StockError { message: "only {left} {item} left", item, left };
    }
    left - qty
}

let stock = ["apple": 3];

try {
    reserve(stock, "apple", 5);
} catch e is StockError {
    print(e.message); // only 3 apple left
    print(e.left);    // 3
}
```

A struct with a string `message` field prints as `Type: message`. An
uncaught `throw` is reported the same way and stops the program like any
runtime error:

```rune
struct StockError { message, item }
throw StockError { message: "no pears left", item: "pear" }; // StockError: no pears left
```

Only structs can be thrown:

```rune
throw "oops"; // TypeError: Can only throw struct instances.
```

`throw` is an expression too, so it fits wherever a value is expected:
after `??`, in an `if` branch or in a `match` arm.

```rune
struct ConfigError { message, key }

fun require(config, key) {
    config?.[key] ?? throw ConfigError { message: "missing " + key, key }
}

fun positive(n) {
    if n > 0 { n } else { throw ValueError { message: "not positive: {n}" } }
}

let config = ["port": 8080];
print(require(config, "port")); // 8080

try {
    require(config, "host");
} catch e is ConfigError {
    print(e.message); // missing host
}

try {
    positive(-2);
} catch e is ValueError {
    print(e.message); // not positive: -2
}
```

## Wrapping Errors

When you turn a low-level error into one that makes sense to your
caller, put the original in `cause` so its details are not lost:

```rune
fun readConfig(raw) {
    try {
        return json.parse(raw);
    } catch e {
        throw ValueError { message: "bad config", cause: e };
    }
}

try {
    readConfig("port=80");
} catch e {
    print(e.message); // bad config
    print(e.cause);   // ValueError: json.parse() could not parse JSON: invalid character 'p' looking for beginning of value
}
```

## defer

`defer expression;` schedules an expression to run when the enclosing
block is left: at its end, by `return`, `break` or `continue`, or by an
error, which then carries on up. It keeps cleanup next to the setup it
undoes, so it cannot be forgotten on an early exit. For cleanup of
several statements, defer a block: `defer { ... }`.

```rune
fun save(path, lines) {
    let out = fs.open(path, "w");
    defer out.close();

    for line in lines {
        out.write(line + "\n");
    }
}

let dir = fs.tempDir();
defer fs.removeAll(dir);

let path = fs.join(dir, "todo.txt");
save(path, ["milk", "tea"]);
print(fs.lines(path)); // ["milk", "tea"]
```

Defers run even when an error is thrown, and several run in reverse
order, last registered first:

```rune
fun step(n) {
    defer print("step {n}: cleanup");
    if n == 0 {
        throw ValueError { message: "step 0 failed" };
    }
    print("step {n}: work");
}

step(1);
try {
    step(0);
} catch e {
    print(e.message);
}
// step 1: work
// step 1: cleanup
// step 0: cleanup
// step 0 failed
```

```rune
fun pipeline() {
    defer print("close file");
    defer print("flush buffer");
    print("write data");
}

pipeline();
// write data
// flush buffer
// close file
```

A single expression is deferred without braces:

```rune
fun f() {
    defer { print("bye"); } // Error at 'defer': Defer one expression without braces: defer expression;
}
```

A `return` computes its value before the defers run. A defer's body
cannot itself `return`, nor `break` or `continue` out of itself:

```rune
fun f() {
    defer { return 1; } // Error at 'return': Cannot return from a 'defer' body.
}
```
