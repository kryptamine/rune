# Destructuring

Destructuring takes a value apart into variables in one step. Instead of
reading `user.name` and `user.age` into two variables by hand, you write
the shape you expect and Rune fills in the names. Patterns work in `let`,
in function parameters and in `for`-`in` variables.

These are the same patterns as in [`match`](pattern-matching.md) and
[`is`](values.md#type-tests-and-patterns). The difference is what happens
when the value does not fit: `match` tries the next arm, `is` gives
`false`, and destructuring fails with an error.

## Structs

A record pattern binds fields to variables of the same name:

```rune
struct User { name, age, role = "member" }

let user = User { name: "ann", age: 31 };
let { name, age } = user;

print(name);    // ann
print(age);     // 31
```

`field: name` binds a field under another name. Variables from a pattern
are immutable, like any `let`. Mark each one that may change with `mut`:

```rune
struct User { name, age }

let { name: login, age: mut years } = User { name: "ann", age: 31 };
years += 1;
print("{login} turns {years}");    // ann turns 32
```

### In parameters

A pattern in a parameter takes just the fields the function needs. It
accepts any struct that has them:

```rune
struct Point { x, y }
struct Pixel { x, y, color }

fun lengthSquared({ x, y }) { x * x + y * y }

print(lengthSquared(Point { x: 3, y: 4 }));                // 25
print(lengthSquared(Pixel { x: 1, y: 1, color: "red" }));  // 2
```

This also makes arrow functions over records short:

```rune
struct Order { id, total }

let orders = [Order { id: 1, total: 30 }, Order { id: 2, total: 120 }];
print(orders.map(({ total }) => total));    // [30, 120]
```

### In for-in

A pattern on the value variable unpacks each element:

```rune
struct Stats { hits }

let pages = ["/home": Stats { hits: 10 }, "/about": Stats { hits: 3 }];

for path, { hits } in pages {
    print("{path}: {hits}");
}
// /home: 10
// /about: 3
```

### Taking the rest

`...name` collects the remaining fields into a map. It must come last.
Use it to split off one part of a record:

```rune
struct Post { id, title, draft = true }

let { id, ...data } = Post { id: 7, title: "Hello" };

print(id);      // 7
print(data);    // ["title": "Hello", "draft": true]
```

A missing field is an error, just as `user.email` would be. To allow it,
give it a [default](#defaults):

```rune
struct User { name }
let { email } = User { name: "ann" }; // KeyError: User has no field 'email'.
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

Array patterns shine for pairs, such as the entries of a map:

```rune
let env = ["HOME": "/home/ann", "PORT": "8080"];
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

A field or element can be a pattern itself, so one `let` reaches deep
into a value:

```rune
struct User { name, tags }
struct Resp { user, status }

let resp = Resp { user: User { name: "ann", tags: ["admin", "ops", "dev"] }, status: 200 };
let { user: { name, tags: [first, ...others] }, status } = resp;

print(name);      // ann
print(first);     // admin
print(others);    // ["ops", "dev"]
print(status);    // 200
```

The same key may appear twice, to keep the whole value and take it apart
too:

```rune
struct Point { x, y }
struct Shape { pos }

let { pos, pos: { x } } = Shape { pos: Point { x: 3, y: 4 } };
print(pos);    // Point {x: 3, y: 4}
print(x);      // 3
```

Nested patterns work in parameters and `for` loops as well:

```rune
struct Image { name, size }
struct Stats { hits }

fun label({ name, size: [w, h] }) { "{name} {w}x{h}" }
print(label(Image { name: "logo", size: [64, 32] }));    // logo 64x32

for [day, { hits }] in [["mon", Stats { hits: 10 }], ["tue", Stats { hits: 3 }]] {
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

A variable may have a default, `name = value`, used when the field or
element is missing. This lets one function take structs of several
shapes:

```rune
struct Page { title }
struct Section { title, root }

fun page({ title, root = "/" }) { "{title} at {root}" }

print(page(Page { title: "Home" }));                     // Home at /
print(page(Section { title: "Docs", root: "/docs" }));   // Docs at /docs
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
struct Opts { title, root }

let opts = Opts { title: "Blog", root: nil };
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
