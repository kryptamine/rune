# Values

A value is one of `nil`, a boolean, a [number](numbers.md), a
[string](strings.md), an [array](../data/arrays.md), an object, a function,
a [`map` or `set`](../data/map-and-set.md), [bytes](../data/bytes.md), a
[struct](../data/structs.md) or a [stream](../stdlib/streams.md);
[`is`](#type-tests) tests which.

An object holds data: its properties cannot be functions. Behavior goes in
a struct's [`impl`](../data/methods.md#functions-in-fields).

## Copies

Every value is copied where it goes: assigning it, passing it to a
function, returning it, capturing it in a closure, storing it in an element,
a property or a field. Arrays, objects, maps, sets, buffers and structs
are no exception, so two values never hold one changeable container:
changing one leaves the other as it was. A copy costs nothing until one of them is
written, and none at all if the source is not read again.

```rune
let a = { tags: ["new"] };
let mut b = a;
b.tags.push("sale");
b.price = 10;
print(a); // {tags: ["new"]}
print(b); // {tags: ["new", "sale"], price: 10}
```

A function changes the caller's variable only through a
[`mut` parameter](functions.md#mut-parameters). The one way to share
changeable state is a variable that functions capture; a copy taken from
it keeps its value when they change it later:

```rune
let mut seen = set();
fun visit(id) { seen.add(id); }
visit(1);
let before = seen;
visit(2);
print(seen);   // set {1, 2}
print(before); // set {1}
```

No value refers to another, so no container holds itself: a graph or a
tree with links back is built on indices or keys, as in
[the linked list](https://github.com/kryptamine/rune/blob/master/examples/linked-list.rn).

Functions and streams are not data and are not copied: a copy of one is
the same function or stream.

## Type Tests

`v is T` tests that `v` is of type `T`: a struct or enum type, or one of
the built-in `nil`, `boolean`, `number`, `string`, `array`, `object`,
`function`, `map`, `set`, `bytes`, `buffer` and `stream`, or `struct` and
`enum`, which a struct type and an enum type themselves are.
`v is T | U` tests that it is one of them, `v !is T | U` that it is none.
Both have the precedence of `==`; the same test is `name is T` in
[match](pattern-matching.md) and `catch e is T`.

```rune
struct User { name }

print(42 is number);                  // true
print("42" is number);                // false
print([1, 2] is array);               // true
print(User { name: "ann" } is User);  // true
print(User { name: "ann" } is object); // false
print(nil is nil);                    // true
print("42" !is number);               // true
print("42" is number | string);       // true
print(nil !is number | string);       // true
print(User is struct);                // true
```

A variable named like a built-in type hides it, and testing against
anything but a type is a `TypeError`:

```rune
let string = "text";
print("a" is string); // TypeError: Right operand of 'is' must be a type, got string.
```

## Truthiness

Falsy values are `nil`, `false`, `0`, `NaN` and anything empty: `""`,
`[]`, `{}`, an empty map, set, bytes or buffer. Everything else is truthy.

```rune
for v in [nil, false, 0, "", [], {}, map(), "0", [0], { a: nil }] {
    print("%q is %s".format(v, if v { "truthy" } else { "falsy" }));
}
// nil is falsy
// false is falsy
// 0 is falsy
// "" is falsy
// [] is falsy
// {} is falsy
// map {} is falsy
// "0" is truthy
// [0] is truthy
// {a: nil} is truthy
```

So an emptiness check needs no `.len()`, and `||` gives a fallback for an
empty value:

```rune
let user = { name: "ann", nickname: "", tags: [] };
print(user.nickname || user.name);   // ann
print(user.tags || ["guest"]);       // ["guest"]

let mut cart = [];
if !cart { print("cart is empty"); } // cart is empty
cart.push("tea");
while cart { print(cart.pop()); }    // tea
```

## Membership

`x in xs` tests that `xs` holds `x`: an element of an array (by `==`), a
substring of a string, a key of an object, a map or a set, a byte or
bytes in bytes or a buffer. `x !in xs` is its negation. `in` has the
precedence of `==`.

```rune
let roles = ["admin", "ops"];
print("ops" in roles);                  // true
print("dev" !in roles);                 // true
print("ell" in "hello");                // true
print("port" in { port: 80 });          // true
print(404 in map([[404, "Not Found"]])); // true
print(0x89 in b"\x89PNG");              // true
```

A range is tested without making its array, so `x in 0..n` costs the same
for any `n`; the step counts:

```rune
let code = 503;
print(code in 500..600);         // true
print(4 in 0..10 step 2);        // true
print(2.5 in 1..=3);             // false
```

Anything else on the right is a `TypeError`, and so is a left side the
right one cannot hold, such as a number in a string:

```rune
print(1 in "123"); // TypeError: Left operand of 'in' must be a string for string, got number.
```

## Equality

`==` compares numbers (an integer with a float too), strings, bytes, booleans and `nil` by value, functions and streams by identity,
and arrays, objects, maps, sets and structs structurally (element by element), so
`[1, 2] == [1, 2]` is `true`. `in` and `indexOf` use the same equality.
`NaN` equals nothing, itself included.
Values of different types are never equal: `1 == "1"` is `false`.

```rune
print(1 == 1.0);                                    // true
print([1, [2, 3]] == [1, [2, 3]]);                  // true
print({ role: "admin", tags: ["ops"] } == { tags: ["ops"], role: "admin" }); // true
print(1 == "1");                                    // false
print(nil == false);                                // false

let seen = [{ id: 1 }, { id: 2 }];
print({ id: 2 } in seen);                    // true
print(seen.indexOf({ id: 2 }));                     // 1

let validate = order => order.qty > 0;
let check = validate;
print(check == validate);                           // true
print((order => order.qty > 0) == validate);        // false
```

## Printing

`print` shows strings inside arrays and objects quoted, so `["1", 1]` and `[1, 1]`
look different; a string printed on its own is written as is. Objects print
their keys in the order they were added.

```rune
print("1");                          // 1
print(["1", 1]);                     // ["1", 1]
print([nil, true, 2.5]);             // [nil, true, 2.5]
print({ name: "ann", age: 31 });     // {name: "ann", age: 31}
```
