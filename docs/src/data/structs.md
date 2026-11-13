# Structs

A struct is a named record with a fixed set of fields. Fields without a
default are required. Structs are declared at the top level only and can
be used before their declaration.

```rune
struct User { name, role = "member", tags = [] }

let ann = User { name: "ann" };
print(ann);          // User {name: "ann", role: "member", tags: []}
print(ann.name);     // ann
print(ann["role"]);  // member

let name = "bob";
print(User { name, role: "admin" }); // User {name: "bob", role: "admin", tags: []}
```

`User { name }` is shorthand for `User { name: name }`.

A default is a constant: a literal or operators on literals (`60 * 60`), a
[`const`](../language/variables.md#constants), a variant of an
[enum](enums.md) of the file, an array or object of them, `map()` or `set()`. Each struct made gets its own copy of an array, object, map or
set, so `tags = []` is never shared:

```rune
struct User { name, tags = [] }

let mut ann = User { name: "ann" };
let bob = User { name: "bob" };
ann.tags.push("admin");
print(ann.tags); // ["admin"]
print(bob.tags); // []
```

A field that does not exist, or a required one left out, is a compile
error in a literal:

```rune
struct User { name, role = "member" }
User { name: "ann", email: "ann@example.com" }; // Error at 'email': User has no field 'email'.
```

```rune
struct User { name, role = "member" }
User { role: "admin" }; // Error at 'User': Missing field 'name' of User.
```

Reading a field that does not exist is a runtime error:

```rune
struct User { name, role = "member" }
let ann = User { name: "ann" };
print(ann.email); // KeyError: User has no field 'email'.
```

## Setting fields

Fields are read-only outside the struct's own
[mutating methods](methods.md#mutating-methods): only `self.x = ...` (or
`self["x"]`, `+=`, `++`, ...) in a `mut self` method sets one. Anywhere
else setting a field of a struct, through a variable, an element or
another struct's field, is a runtime error. To change a struct from
outside, call one of its `mut self` methods or make an updated copy
with [spread](#updating-with-spread).

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

```rune
struct User { name, role = "member" }
let mut ann = User { name: "ann" };
ann.role = "admin"; // TypeError: Cannot set field 'role' of User outside its mut self methods.
```

A struct held in another struct's field is changed by the outer
struct's methods, calling the inner one's: `self.pos.move(1)` in a
method of the outer struct; `self.pos.x = 1` there, or `acc.pos.move(1)`
outside it, is an error.

The rule is shallow: an array, object, map or set in a field is not
part of the struct, so `ann.tags.push("x")` changes it in place.

## Updating with spread

A literal may spread a struct of the same type, or an object of some of
its fields, as `T { ...t, x: 1 }` does an update without changing `t`.
Items apply in order, later ones winning, and a field none of them gives
takes its default; one with no default, a key that is not a field, or a
struct of another type is a runtime error.

```rune
struct Config { host, port = 8080, debug = false }

let base = Config { host: "localhost" };
print(Config { ...base, debug: true });  // Config {host: "localhost", port: 8080, debug: true}
print(Config { debug: true, ...base });  // Config {host: "localhost", port: 8080, debug: false}
print(Config { ...{ host: "example.com" } }); // Config {host: "example.com", port: 8080, debug: false}
print(base);                             // Config {host: "localhost", port: 8080, debug: false}
```

## Converting

Calling a struct type converts to it. From an object the keys must be
exactly the fields, the defaulted ones optional; a struct of the same
type is returned as is, and any other value is an error. There are no
implicit conversions, and none between struct types. This is how data
from `json.parse` becomes a struct:

```rune
struct Config { host, port = 8080 }

let cfg = Config(json.parse(r"""{"host": "example.com"}"""));
print(cfg);      // Config {host: "example.com", port: 8080}
print(cfg.port); // 8080
```

```rune
struct Config { host, port = 8080 }
Config({ host: "example.com", timeout: 5 }); // KeyError: Config has no field 'timeout'.
```

```rune
struct Config { host, port = 8080 }
Config({ port: 80 }); // KeyError: Missing field 'host' of Config.
```

## Type checks, equality, iteration

`v is User` is `true` only for a `User`, never for an object; see
[Type Tests](../language/values.md#type-tests).

`==` compares structs field by field and only if they are of the same
struct, so a struct never equals an object. `for k, v in s` walks the
fields in declaration order. `object(s)` copies a struct into a new
object, `object(s).len()` its number of fields and `type(s)` its name.

```rune
struct User { name, role = "member" }
let ann = User { name: "ann" };

print(ann is User);                            // true
print({ name: "ann" } is User);                // false
print(ann == User { name: "ann" });            // true
print(ann == { name: "ann", role: "member" }); // false

for field, value in ann {
    print("{field}: {value}");
}
// name: ann
// role: member

print(object(ann)); // {name: "ann", role: "member"}
print(object(ann).len()); // 2
print(type(ann));   // User
print(type(User));  // struct
```
