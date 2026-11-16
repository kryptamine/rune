# Structs

A struct is a named record with a fixed set of fields. Use one when a
value has a known shape, such as a user, an order or a config: the
compiler checks field names in literals, the type can be tested with
`is` and matched in [patterns](../language/pattern-matching.md#structs),
and [methods](methods.md) can be attached to it.

```rune
struct User { name, role = "member", tags = [] }

let ann = User { name: "ann" };
print(ann);         // User {name: "ann", role: "member", tags: []}
print(ann.name);    // ann
print(ann["role"]); // member
```

A field with a default, `role = "member"`, may be left out of a literal;
a field without one is required. Structs are declared at the top level
only and can be used before their declaration.

When a variable has the same name as a field, write it once:
`User { name }` is short for `User { name: name }`.

```rune
struct User { name, role = "member" }

let name = "bob";
let role = "admin";
print(User { name, role }); // User {name: "bob", role: "admin"}
```

## Defaults

A default must be a constant: a literal or operators on literals
(`60 * 60`), a [`const`](../language/variables.md#constants), a variant
of an [enum](enums.md) in the same file, an array or map literal of
these, or `set()`.

```rune
const DAY = 24 * 60 * 60;

struct Session { user, ttl = DAY, scopes = ["read"] }

print(Session { user: "ann" });                    // Session {user: "ann", ttl: 86400, scopes: ["read"]}
print(Session { user: "bob", scopes: ["admin"] }); // Session {user: "bob", ttl: 86400, scopes: ["admin"]}
```

Like every value in Rune, a default is [copied](../language/values.md#copies),
so two structs never share the same `[]`:

```rune
struct User { name, tags = [] }

impl User {
    fun tag(mut self, t) { self.tags.push(t); }
}

let mut ann = User { name: "ann" };
let bob = User { name: "bob" };
ann.tag("admin");
print(ann.tags); // ["admin"]
print(bob.tags); // []
```

## Typos Are Caught Early

A misspelled or unknown field, or a required one left out, is a compile
error in a literal, so the program does not even start:

```rune
struct User { name, role = "member" }
User { name: "ann", email: "ann@example.com" }; // Error at 'email': User has no field 'email'.
```

```rune
struct User { name, role = "member" }
User { role: "admin" }; // Error at 'User': Missing field 'name' of User.
```

Reading a field that does not exist fails when it runs:

```rune
struct User { name, role = "member" }
let ann = User { name: "ann" };
print(ann.email); // KeyError: User has no field 'email'.
```

## Setting Fields

A struct guards its own data: its fields are set only inside its
[mutating methods](methods.md#mutating-methods), those that take
`mut self`. To change a struct, call one of them:

```rune
struct User { name, role = "member" }

impl User {
    fun promote(mut self, role) { self.role = role; }
}

let mut ann = User { name: "ann" };
ann.promote("admin");
print(ann); // User {name: "ann", role: "admin"}

let mut team = [User { name: "bob" }];
team[0].promote("lead");
print(team[0].role); // lead
```

Setting a field anywhere else, through a variable, an array element or
another struct, is an error:

```rune
struct User { name, role = "member" }
let mut ann = User { name: "ann" };
ann.role = "admin"; // TypeError: Cannot set field 'role' of User outside its mut self methods.
```

A struct also owns what its fields hold. An array, map, set or
another struct in a field is part of it, so changing it in place,
`ann.tags.push(x)`, counts as setting the field:

```rune
struct User { name, tags = [] }
let mut ann = User { name: "ann" };
ann.tags.push("x"); // TypeError: Cannot set field 'tags' of User outside its mut self methods.
```

Inside a `mut self` method all of this is allowed: `self.tags.push(x)`,
`self.pos.x = 1`, `self.pos.move(1)`.

## Updating with Spread

A struct literal may spread a struct of its own type, or a map, then
list the fields to change: `Config { ...base, debug: true }`.
Items apply in order, so later ones win.

```rune
struct Config { host, port = 8080, debug = false }

let base = Config { host: "localhost" };
let dev = Config { ...base, debug: true };
let prod = Config { ...base, host: "example.com", port: 443 };

print(dev);  // Config {host: "localhost", port: 8080, debug: true}
print(prod); // Config {host: "example.com", port: 443, debug: false}
print(base); // Config {host: "localhost", port: 8080, debug: false}
```

A struct of another type is a runtime error.

## From a Map

Spreading a map with string keys into a struct literal is how data from
`json.parse` becomes a struct:

```rune
struct Config { host, port = 8080 }

let cfg = Config { ...json.parse(r"""{"host": "example.com"}""") };
print(cfg);      // Config {host: "example.com", port: 8080}
print(cfg.port); // 8080
```

Keys that name no field are skipped, so a struct takes only what it
needs from a larger document. Every field without a default must be
given, by the map or by the literal; the check runs when the literal is
built:

```rune
struct Config { host, port = 8080 }
print(Config { ...["host": "example.com", "timeout": 5] }); // Config {host: "example.com", port: 8080}
```

```rune
struct Config { host, port = 8080 }
Config { ...["port": 80] }; // KeyError: Missing field 'host' of Config.
```

## Type Tests and Equality

`v is User` is `true` only for a `User`, never for a map or another
struct with the same fields; see
[Type Tests](../language/values.md#type-tests-and-patterns).

`==` compares two structs of the same type field by field. A struct
never equals a map or a struct of another type.

```rune
struct User { name, role = "member" }
let ann = User { name: "ann" };

print(ann is User);                            // true
print(["name": "ann"] is User);                  // false
print(ann == User { name: "ann" });              // true
print(ann == ["name": "ann", "role": "member"]); // false
```

## Walking the Fields

`map(s)` copies a struct into a map, its fields in declaration order.
Use it to loop over the fields or count them:

```rune
struct User { name, role = "member" }
let ann = User { name: "ann" };

for field, value in map(ann) {
    print("{field}: {value}");
}
// name: ann
// role: member

print(map(ann).len()); // 2
```
