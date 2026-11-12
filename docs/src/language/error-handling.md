# Error Handling

## try and catch

Runtime errors, including those from built-ins like `http.get()`, are
caught with `try`/`catch`:

```rune
fun parsePort(s) {
    try {
        return int(s);
    } catch (e) {
        print "bad port: " + e.message;
        return 80;
    }
}

print parsePort("8080"); // 8080
print parsePort("http"); // 80
```

The second call first prints `bad port: int() cannot parse "http" as an integer in base 10.`

The catch variable is optional:

```rune
let mut queue = [];
try { queue.pop(); } catch { print "queue is empty"; } // queue is empty
```

## Error Types

An error raised by the language is a struct of one of the built-in error
types, each with fields `{ message, line = nil, cause = nil }`:

- `TypeError` — a value of the wrong type (`1 + nil`, calling a number)
- `ArgumentError` — the wrong number of arguments
- `ValueError` — the right type but a bad value (`step 0`, `[].pop()`)
- `IndexError` — an index out of bounds
- `KeyError` — a missing property or struct field
- `NameError` — an undefined variable
- `IOError` — a file system call or an `http` request failing; it also has
  `path` and `code` (see [fs](../stdlib/fs.md))

```rune
try {
    [].pop();
} catch (e) {
    print type(e);  // ValueError
    print e.message; // array.pop() got an empty array.
    print e.line;    // 2
}
```

A stack overflow is fatal and cannot be caught, and `os.exit` is not an
error at all: `try` does not stop it.

## Catching by Type

A catch clause can name the struct types it catches. Clauses are tried in
order, and an error none of them catches goes on up with its trace intact.
A clause without types catches everything and must come last.

```rune
let config = { port: 8080 };

fun get(key) {
    try {
        return config[key];
    } catch (e: KeyError) {
        return nil;
    }
}

print get("port"); // 8080
print get("host"); // nil
```

```rune
fun load(raw) {
    try {
        return json.parse(raw).port + 1;
    } catch (e: KeyError, IndexError) {
        return "missing: " + e.message;
    } catch (e: ValueError) {
        return "invalid JSON";
    } catch (e) {
        return type(e);
    }
}

print load(r"""{"port": 80}""");   // 81
print load("\{}");                 // missing: Undefined property 'port'.
print load("not json");            // invalid JSON
print load(r"""{"port": "80"}"""); // TypeError
```

## throw

`throw` raises a struct instance, built-in or your own; `catch` receives it
unchanged, with all its fields.

```rune
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

An uncaught throw stops the program like any runtime error, reported as
`Name: message` when the struct has a string `message`:

```rune
struct AgeError { message, value }
throw AgeError { message: "negative age", value: -1 }; // AgeError: negative age
```

`throw` is also an expression, whose operand extends as far as it can, so it
fits where a value is expected: after `??`, in `? :`, or in a match arm.

```rune
struct ConfigError { message, key }

fun require(cfg, key) {
    return cfg?.[key] ?? throw ConfigError { message: "missing " + key, key };
}

fun positive(n) {
    return n > 0 ? n : throw ValueError { message: "not positive: {n}" };
}

let cfg = { port: 8080 };
print require(cfg, "port"); // 8080

try { require(cfg, "host"); } catch (e: ConfigError) { print e.message; } // missing host
try { positive(-2); } catch (e: ValueError) { print e.message; }          // not positive: -2
```

Wrap a lower-level error in `cause` to keep its details:

```rune
fun readConfig(raw) {
    try {
        return json.parse(raw);
    } catch (e) {
        throw ValueError { message: "bad config", cause: e };
    }
}

try {
    readConfig("port=80");
} catch (e) {
    print e.message;       // bad config
    print type(e.cause);   // ValueError
}
```

Throwing anything but a struct is a `TypeError`:

```rune
throw "oops"; // TypeError: Can only throw struct instances.
```
