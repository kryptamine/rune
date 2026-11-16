# Values

Every value in Rune is one of these kinds:

| Kind | Examples |
| --- | --- |
| `nil` | `nil`, the absence of a value |
| boolean | `true`, `false` |
| [number](numbers.md) | `42`, `19.99`, `0xff` |
| [string](strings.md) | `"ann"`, `"total: {sum}"` |
| [array](../data/arrays.md) | `[1, 2, 3]` |
| object | `{ name: "ann", age: 31 }` |
| function | `fun area(r) { ... }`, `x => x * 2` |
| [map and set](../data/map-and-set.md) | `map()`, `set(["a", "b"])` |
| [bytes](../data/bytes.md) | `b"\x89PNG"` |
| [struct](../data/structs.md) | `User { name: "ann" }` |
| [enum](../data/enums.md) variant | `Status.Active` |
| [stream](../stdlib/streams.md) | an open file, `io.stdin` |

An object holds data only: its properties cannot be functions. To give
data behavior, use a struct and its [`impl`](../data/methods.md).

## Copies

Assigning a value, passing it to a function, returning it, or storing it
in an array, object or field always gives the receiver its own copy. This
holds for arrays, objects, maps, sets and structs too, so changing a copy
never changes the original:

```rune
let order = { item: "tea", qty: 1 };
let mut draft = order;
draft.qty = 3;

print(order);    // {item: "tea", qty: 1}
print(draft);    // {item: "tea", qty: 3}
```

The same goes for a function argument. The function works on its own copy,
and the caller's value stays as it was:

```rune
fun discounted(order) {
    let mut copy = order;
    copy.price = copy.price * 0.9;
    copy
}

let order = { price: 100 };
print(discounted(order));    // {price: 90}
print(order);                // {price: 100}
```

Copies are cheap. Rune copies a container lazily, only when one of the two
is about to be changed, and not at all when the original is never read
again.

There are exactly two ways to change a value outside a function:

- a [`mut` parameter](functions.md#mut-parameters), which writes the
  function's value back to the caller's variable;
- a variable that the function captures as a closure.

```rune
fun applyDiscount(mut order) {
    order.price *= 0.9;
}

let mut order = { price: 100 };
applyDiscount(order);
print(order);    // {price: 90}
```

```rune
let mut seen = set();
fun visit(id) { seen.add(id); }

visit(1);
let before = seen;
visit(2);

print(seen);      // set {1, 2}
print(before);    // set {1}
```

`before` is a copy, so it keeps the value `seen` had when it was taken.

Because no value refers to another, a container can never hold itself. A
graph, or a tree with links back to parents, keeps its nodes in an array or
a map and links them by index or key, as
[the linked list example](https://github.com/kryptamine/rune/blob/master/examples/linked-list.rn)
does.

Functions and streams are not data and are never copied: a "copy" of one is
the same function or stream.

## Type Tests

`v is T` asks whether `v` has type `T`. `T` is a struct or enum you
declared, or one of the built-in type names:

`nil`, `boolean`, `number`, `string`, `array`, `object`, `function`,
`map`, `set`, `bytes`, `buffer`, `stream`, `struct`, `enum`

```rune
struct User { name }
let ann = User { name: "ann" };

print(42 is number);       // true
print("42" is number);     // false
print([1, 2] is array);    // true
print(ann is User);        // true
print(ann is object);      // false
print(nil is nil);         // true
print(User is struct);     // true
```

A struct value is not an object, and a struct type itself is a `struct`.

`is T | U` accepts any of several types, and `!is` negates the test:

```rune
fun describe(id) {
    if id is number | string { "id {id}" } else { "no id" }
}

print(describe(7));        // id 7
print(describe("a7"));     // id a7
print(describe(nil));      // no id
print("42" !is number);    // true
```

The same tests appear in [`match`](pattern-matching.md#types) and in
[`catch e is T`](error-handling.md#catching-by-type).

A variable named like a built-in type hides it, and the right side of `is`
must be a type:

```rune
let string = "text";
print("a" is string);    // TypeError: Right operand of 'is' must be a type, got string.
```

## Truthiness

`if`, `while`, `!`, `&&` and `||` accept any value, not only booleans. These
values count as false, or **falsy**:

- `nil` and `false`
- `0` and `NaN`
- anything empty: `""`, `[]`, `{}`, an empty map, set, bytes or buffer

Everything else is **truthy**, including `"0"`, `[0]` and `{ a: nil }`.

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

This makes emptiness checks short. There is no need for `.len() > 0`, and
`||` gives a fallback for an empty value:

```rune
let user = { name: "ann", nickname: "", tags: [] };
print(user.nickname || user.name);    // ann
print(user.tags || ["guest"]);        // ["guest"]

let mut cart = ["tea", "milk"];
while cart {
    print("packing {cart.pop()}");
}
// packing milk
// packing tea
```

## Membership

`x in xs` asks whether `xs` contains `x`, and `x !in xs` whether it does
not. What "contains" means depends on `xs`:

| `xs` | `x in xs` is true when |
| --- | --- |
| array | an element equals `x` |
| string | `x` is a substring |
| object, map | `x` is a key |
| set | `x` is a member |
| bytes, buffer | `x` is a byte or a run of bytes in it |
| range | `x` is one of its numbers |

```rune
let roles = ["admin", "ops"];
print("ops" in roles);                       // true
print("dev" !in roles);                      // true
print("ell" in "hello");                     // true
print("port" in { host: "db", port: 80 });   // true
print(404 in map([[404, "Not Found"]]));     // true
print(0x89 in b"\x89PNG");                   // true
```

A range answers `in` without building its array, so `x in 0..n` is just as
fast for any `n`. A `step` counts:

```rune
let status = 503;
print(status in 500..600);    // true
print(4 in 0..10 step 2);     // true
print(5 in 0..10 step 2);     // false
```

The left side must be something the right side can hold, so a number is
never "in" a string:

```rune
print(1 in "123");    // TypeError: Left operand of 'in' must be a string for string, got number.
```

## Equality

`==` and `!=` compare values by content:

- numbers, strings, bytes, booleans and `nil` by value; an integer equals
  the float with the same value, so `1 == 1.0`;
- arrays, objects, maps, sets and structs element by element, so two
  arrays with the same elements are equal;
- functions and streams by identity: a function equals only itself.

Values of different kinds are never equal, so `1 == "1"` is `false`. `NaN`
equals nothing, not even itself. `in` and `indexOf` use this same equality.

```rune
print(1 == 1.0);                     // true
print([1, [2, 3]] == [1, [2, 3]]);   // true
print({ role: "admin", tags: ["ops"] } == { tags: ["ops"], role: "admin" });    // true
print(1 == "1");                     // false
print(nil == false);                 // false

let cart = [{ id: 1 }, { id: 2 }];
print({ id: 2 } in cart);            // true
print(cart.indexOf({ id: 2 }));      // 1

let isValid = order => order.qty > 0;
let check = isValid;
print(check == isValid);                       // true
print((order => order.qty > 0) == isValid);    // false
```

## Printing

`print` writes a string on its own as is, but quotes strings inside arrays
and objects, so `["1", 1]` and `[1, 1]` look different. Objects print
their keys in the order they were added.

```rune
print("1");                         // 1
print(["1", 1]);                    // ["1", 1]
print([nil, true, 2.5]);            // [nil, true, 2.5]
print({ name: "ann", age: 31 });    // {name: "ann", age: 31}
```
