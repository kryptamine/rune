# Values

Every value in Rune is one of these kinds:

| Kind | Examples |
| --- | --- |
| `nil` | `nil`, the absence of a value |
| boolean | `true`, `false` |
| [number](numbers.md) | `42`, `19.99`, `0xff` |
| [string](strings.md) | `"ann"`, `"total: {sum}"` |
| [array](../data/arrays.md) | `[1, 2, 3]` |
| function | `fun area(r) { ... }`, `x => x * 2` |
| [map and set](../data/map-and-set.md) | `["a": 1]`, `set(["a", "b"])` |
| [bytes](../data/bytes.md) | `b"\x89PNG"` |
| [struct](../data/structs.md) | `User { name: "ann" }` |
| [enum](../data/enums.md) variant | `Status.Active` |
| [stream](../stdlib/streams.md) | an open file, `io.stdin` |

## Copies

Assigning a value, passing it to a function, returning it, or storing it
in an array, map or field always gives the receiver its own copy. This
holds for arrays, maps, sets and structs too, so changing a copy never
changes the original:

```rune
let cart = ["tea"];
let mut draft = cart;
draft.push("milk");

print(cart);     // ["tea"]
print(draft);    // ["tea", "milk"]
```

The same goes for a function argument. The function works on its own copy,
and the caller's value stays as it was:

```rune
fun discounted(prices) {
    let mut copy = prices;
    copy["tea"] = copy["tea"] * 0.9;
    copy
}

let prices = ["tea": 100];
print(discounted(prices));    // ["tea": 90]
print(prices);                // ["tea": 100]
```

Copies are cheap. Rune copies a container lazily, only when one of the two
is about to be changed, and not at all when the original is never read
again.

There are exactly two ways to change a value outside a function:

- a [`mut` parameter](functions.md#mut-parameters), which writes the
  function's value back to the caller's variable;
- a variable that the function captures as a closure.

```rune
fun applyDiscount(mut prices) {
    prices["tea"] *= 0.9;
}

let mut prices = ["tea": 100];
applyDiscount(prices);
print(prices);    // ["tea": 90]
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

## Type Tests and Patterns

`v is T` asks whether `v` has type `T`. `T` is a struct or enum you
declared, or one of the built-in type names:

`nil`, `boolean`, `number`, `string`, `array`, `function`,
`map`, `set`, `bytes`, `buffer`, `stream`, `struct`, `enum`

```rune
struct User { name }
let ann = User { name: "ann" };

print(42 is number);       // true
print("42" is number);     // false
print([1, 2] is array);    // true
print(ann is User);        // true
print(nil is nil);         // true
print(User is struct);     // true
```

A struct type itself is a `struct`.

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

After `is` may come any [pattern](pattern-matching.md#patterns), in which
a bare name is a type. `v is P` asks whether the pattern `P` matches `v`:

```rune
struct Point { x, y = 0 }
struct Resp { status, body = nil }
enum Dir { North, South }

let resp = Resp { status: 204, body: "" };
print(resp is { status: 200..300 });    // true
print(7 is 1..10);                      // true
print("b" is "a" | "b");                // true
print([1, 2, 3] is [1, ..._]);          // true
print(Point { x: 0 } is Point { x: 0 }); // true
print(Dir.South is Dir.North);          // false
```

### Binding in Conditions

In the condition of an `if` or a `while`, a pattern after `is` may bind
variables. They hold in the body, and on the right of an `&&` after the
test:

```rune
struct Resp { status, body = nil }

fun handle(resp) {
    if resp is { status: 200..300, body } && body {
        "got {body}"
    } else if resp is { status } {
        "status {status}"
    } else {
        "no response"
    }
}

print(handle(Resp { status: 200, body: "hi" }));    // got hi
print(handle(Resp { status: 404 }));                // status 404
print(handle(nil));                            // no response
```

A `while` loop runs as long as the pattern matches, here until the queue
gives something that is not a job:

```rune
struct Job { id }

let mut queue = [nil, Job { id: 2 }, Job { id: 1 }];
while queue.pop() is { id } {
    print("job {id}");
}
// job 1
// job 2
```

To bind the whole value, write `name is P`, as in a
[match arm](pattern-matching.md#types): `while queue.pop() is n is number`.

Anywhere else, and after `!is` or `||`, a test cannot bind:

```rune
let ok = [1] is [a]; // Error at 'a': Only an if or while condition can bind variables with 'is'.
```

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
- anything empty: `""`, `[]`, `[:]`, an empty set, bytes or buffer

Everything else is **truthy**, including `"0"`, `[0]` and `["a": nil]`.

```rune
for v in [nil, false, 0, "", [], [:], set(), "0", [0], ["a": nil]] {
    print("{"%q".format(v)} is {if v { "truthy" } else { "falsy" }}");
}
// nil is falsy
// false is falsy
// 0 is falsy
// "" is falsy
// [] is falsy
// [:] is falsy
// set {} is falsy
// "0" is truthy
// [0] is truthy
// ["a": nil] is truthy
```

This makes emptiness checks short. There is no need for `.len() > 0`, and
`||` gives a fallback for an empty value:

```rune
struct User { name, nickname = "", tags = [] }

let user = User { name: "ann" };
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
| map | `x` is a key |
| set | `x` is a member |
| bytes, buffer | `x` is a byte or a run of bytes in it |
| range | `x` is one of its numbers |

```rune
let roles = ["admin", "ops"];
print("ops" in roles);                       // true
print("dev" !in roles);                      // true
print("ell" in "hello");                     // true
print(404 in [404: "Not Found"]);            // true
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
- arrays, maps, sets and structs element by element, so two
  arrays with the same elements are equal;
- functions and streams by identity: a function equals only itself.

Values of different kinds are never equal, so `1 == "1"` is `false`. `NaN`
equals nothing, not even itself. `in` uses this same equality.

```rune
print(1 == 1.0);                     // true
print([1, [2, 3]] == [1, [2, 3]]);   // true
print(["role": "admin", "tags": ["ops"]] == ["tags": ["ops"], "role": "admin"]);    // true
print(1 == "1");                     // false
print(nil == false);                 // false

struct Item { id }
let cart = [Item { id: 1 }, Item { id: 2 }];
print(Item { id: 2 } in cart);       // true

let isValid = order => order.qty > 0;
let check = isValid;
print(check == isValid);                       // true
print((order => order.qty > 0) == isValid);    // false
```

## Printing

`print` writes a string on its own as is, but quotes strings inside arrays
and maps, so `["1", 1]` and `[1, 1]` look different. A map prints as its
literal, keys in the order they were added.

```rune
print("1");                         // 1
print(["1", 1]);                    // ["1", 1]
print([nil, true, 2.5]);            // [nil, true, 2.5]
print(["name": "ann", "age": 31]);  // ["name": "ann", "age": 31]
```
