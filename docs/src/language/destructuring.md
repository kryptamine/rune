# Destructuring

A pattern takes a value apart into variables. It works in `let`, in function
parameters and in `for`-`in` variables.

## Objects and Structs

An object pattern binds properties of an object, or fields of a struct, to
variables of the same name:

```rune
let user = { name: "ann", age: 31 };
let { name, age } = user;

print(name); // ann
print(age);  // 31
```

`key: name` binds under another name; a string key needs one. `mut x` and
`key: mut name` make a variable mutable, `let mut { ... }` makes them all
mutable.

```rune
let user = { name: "ann", age: 31, "full name": "Ann Lee" };
let { name: login, "full name": fullName, age: mut years } = user;
years += 1;

print(login);    // ann
print(fullName); // Ann Lee
print(years);    // 32
```

In a parameter, a pattern takes the fields it needs, from an object or a
struct alike:

```rune
struct Point { x, y = 0 }

fun length({ x, y }) {
    return x * x + y * y;
}

print(length(Point { x: 3, y: 4 })); // 25
print(length({ x: 1, y: 1 }));       // 2
```

In `for`-`in` the pattern takes the value, not the key:

```rune
let pages = { "/home": { hits: 10 }, "/about": { hits: 3 } };

for path, { hits } in pages {
    print("{path}: {hits}");
}
// /home: 10
// /about: 3
```

`...name` takes the other properties, or fields, as a new object and must
come last:

```rune
struct User { name, role = "member", age = 31 }

let { id, ...data } = { id: 7, title: "Hello", draft: true };
print(id);   // 7
print(data); // {title: "Hello", draft: true}

let { name, ...rest } = User { name: "ann" };
print(rest); // {role: "member", age: 31}
```

A missing property or field is a runtime error, as with `obj.name`:

```rune
let { email } = { name: "ann" }; // KeyError: Undefined property 'email'.
```

## Arrays

An array pattern takes elements by position: `_` skips one, `...name`
takes the rest as a new array and must come last. The array may be longer
than the pattern.

```rune
let [first, _, third, ...rest] = ["mon", "tue", "wed", "thu", "fri"];

print(first); // mon
print(third); // wed
print(rest);  // ["thu", "fri"]
```

It shines in parameters, for pairs and tuples:

```rune
let env = { HOME: "/home/ann", PORT: "8080" };
print(env.entries().map(([k, v]) => k + "=" + v)); // ["HOME=/home/ann", "PORT=8080"]

fun swap([a, b]) { return [b, a]; }
print(swap([1, 2])); // [2, 1]

let mut [lo, hi] = [1, 9];
lo = 2;
print([lo, hi]); // [2, 9]
```

A shorter array is an `IndexError`, anything but an array a `TypeError`:

```rune
let [host, port] = ["localhost"]; // IndexError: Expected at least 2 elements but got 1.
```

```rune
let [a, b] = "ab"; // TypeError: Cannot destructure string as an array.
```

## Defaults

A variable may have a default, `name = expression`, taken when the
property, field or element is missing, not when it is `nil`. It is
evaluated only then, and may use the variables before it.

```rune
let opts = { title: "Blog", root: nil };
let { title, theme = "light", root = ".", depth = root == "." ? 0 : 1 } = opts;

print(title); // Blog
print(theme); // light
print(root);  // nil
print(depth); // 1
```

```rune
let [src = "content", out = "site"] = ["posts"];

print(src); // posts
print(out); // site

fun page({ title, root = "." }) { return "{title} at {root}"; }

print(page({ title: "Home" }));                // Home at .
print(page({ title: "Docs", root: "/docs" })); // Docs at /docs
```

In an array pattern every element after one with a default needs one too;
`_` and `...rest`, in either pattern, take none.

```rune
let [a = 1, b] = [1, 2]; // Error at 'b': An element without a default cannot follow one with a default.
```
