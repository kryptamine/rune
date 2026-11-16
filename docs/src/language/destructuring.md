# Destructuring

Destructuring takes a value apart into variables in one step. Instead of
reading `user.name` and `user.age` into two variables by hand, you write
the shape you expect and Rune fills in the names. Patterns work in `let`,
in function parameters and in `for`-`in` variables.

These are the same patterns as in [`match`](pattern-matching.md) and
[`is`](values.md#type-tests-and-patterns). The difference is what happens
when the value does not fit: `match` tries the next arm, `is` gives
`false`, and destructuring fails with an error.

## Objects and Structs

An object pattern binds properties to variables of the same name:

```rune
let user = { name: "ann", age: 31, role: "admin" };
let { name, age } = user;

print(name);    // ann
print(age);     // 31
```

`key: name` binds a property under another name. A key that is not a
plain identifier, such as `"full name"`, needs one:

```rune
let user = { name: "ann", "full name": "Ann Lee" };
let { name: login, "full name": fullName } = user;

print(login);       // ann
print(fullName);    // Ann Lee
```

Variables from a pattern are immutable, like any `let`. Mark one with
`mut`, or write `let mut { ... }` to make them all mutable:

```rune
let { name, age: mut years } = { name: "ann", age: 31 };
years += 1;
print("{name} turns {years}");    // ann turns 32
```

The same pattern reads the fields of a struct.

### In parameters

A pattern in a parameter takes just the fields the function needs. It
accepts any value that has them, an object or a struct alike:

```rune
struct Point { x, y = 0 }

fun lengthSquared({ x, y }) { x * x + y * y }

print(lengthSquared(Point { x: 3, y: 4 }));    // 25
print(lengthSquared({ x: 1, y: 1 }));          // 2
```

This also makes arrow functions over records short:

```rune
let orders = [{ id: 1, total: 30 }, { id: 2, total: 120 }];
print(orders.map(({ total }) => total));    // [30, 120]
```

### In for-in

A pattern on the value variable unpacks each element:

```rune
let pages = { "/home": { hits: 10 }, "/about": { hits: 3 } };

for path, { hits } in pages {
    print("{path}: {hits}");
}
// /home: 10
// /about: 3
```

### Taking the rest

`...name` collects the remaining properties, or fields, into a new
object. It must come last. Use it to split off one part of a record:

```rune
let post = { id: 7, title: "Hello", draft: true };
let { id, ...data } = post;

print(id);      // 7
print(data);    // {title: "Hello", draft: true}
```

On a struct, the rest is a plain object with the other fields:

```rune
struct User { name, role = "member", age = 31 }

let { name, ...rest } = User { name: "ann" };
print(rest);    // {role: "member", age: 31}
```

A missing property or field is an error, just as `obj.email` would be.
To allow it, give it a [default](#defaults):

```rune
let { email } = { name: "ann" }; // KeyError: Undefined property 'email'.
```

## Arrays

An array pattern takes elements by position:

```rune
let [x, y] = [3, 4];
print(x + y);    // 7
```

`_` skips an element, and `...name` collects the rest into a new array. A
rest must come last; `..._` ignores it.

```rune
let [first, _, third, ...rest] = ["mon", "tue", "wed", "thu", "fri"];

print(first);    // mon
print(third);    // wed
print(rest);     // ["thu", "fri"]

let [day, ..._] = ["sat", "sun"];
print(day);      // sat
```

This is handy for splitting a line of text into its fields:

```rune
let line = "2026-11-16 ERROR disk full";
let [date, level, ...words] = line.split(" ");

print(date);               // 2026-11-16
print(level);              // ERROR
print(words.join(" "));    // disk full
```

A function can return several values as an array, and the caller picks
them apart:

```rune
fun minMax(xs) { [math.min(...xs), math.max(...xs)] }

let [lo, hi] = minMax([4, 9, 1]);
print("{lo}..{hi}");    // 1..9
```

Array patterns shine for pairs, such as the entries of an object:

```rune
let env = { HOME: "/home/ann", PORT: "8080" };
print(env.entries().map(([k, v]) => k + "=" + v));    // ["HOME=/home/ann", "PORT=8080"]

for [name, score] in [["ann", 90], ["bob", 72]] {
    print("{name}: {score}");
}
// ann: 90
// bob: 72
```

The array must have exactly as many elements as the pattern, as in
[match](pattern-matching.md). Only a rest takes more. A different length
is an `IndexError`:

```rune
let [host, port] = ["localhost"]; // IndexError: Expected 2 elements but got 1.
```

```rune
let [host, port] = ["localhost", 80, "tls"]; // IndexError: Expected 2 elements but got 3.
```

Anything that is not an array is a `TypeError`:

```rune
let [a, b] = "ab"; // TypeError: Cannot destructure string as an array.
```

## Nesting

A property or element can be a pattern itself, so one `let` reaches deep
into a value:

```rune
let resp = { user: { name: "ann", tags: ["admin", "ops", "dev"] }, status: 200 };
let { user: { name, tags: [first, ...others] }, status } = resp;

print(name);      // ann
print(first);     // admin
print(others);    // ["ops", "dev"]
print(status);    // 200
```

The same key may appear twice, to keep the whole value and take it apart
too:

```rune
let { pos, pos: { x } } = { pos: { x: 3, y: 4 } };
print(pos);    // {x: 3, y: 4}
print(x);      // 3
```

Nested patterns work in parameters and `for` loops as well:

```rune
fun label({ name, size: [w, h] }) { "{name} {w}x{h}" }
print(label({ name: "logo", size: [64, 32] }));    // logo 64x32

for [day, { hits }] in [["mon", { hits: 10 }], ["tue", { hits: 3 }]] {
    print("{day}: {hits}");
}
// mon: 10
// tue: 3
```

A pattern here only takes a value apart. Testing it, with a literal, a
range, `|` or a type, belongs in [`match`](pattern-matching.md) or
[`is`](values.md#type-tests-and-patterns):

```rune
let [cmd, "now"] = ["run", "now"]; // Error at '"now"': This pattern only takes a value apart; test the value with match or 'is'.
```

## Defaults

A variable may have a default, `name = value`, used when the property,
field or element is missing. This makes optional settings easy:

```rune
fun page({ title, root = "/" }) { "{title} at {root}" }

print(page({ title: "Home" }));                   // Home at /
print(page({ title: "Docs", root: "/docs" }));    // Docs at /docs
```

In an array pattern, defaults fill in missing positions at the end:

```rune
let [src = "content", out = "site"] = ["posts"];

print(src);    // posts
print(out);    // site
```

A default is used only when the value is missing, not when it is `nil`.
It is evaluated only when used, and may refer to the variables before
it:

```rune
let opts = { title: "Blog", root: nil };
let { title, theme = "light", root = ".", depth = if root == "." { 0 } else { 1 } } = opts;

print(title);    // Blog
print(theme);    // light
print(root);     // nil
print(depth);    // 1
```

In an array pattern, every element after one with a default needs a
default too. `_` and `...rest` take none.

```rune
let [a = 1, b] = [1, 2]; // Error at 'b': An element without a default cannot follow one with a default.
```
