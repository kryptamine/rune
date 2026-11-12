# Values

A value is one of `nil`, a boolean, a [number](numbers.md), a
[string](strings.md), an [array](../data/arrays.md), an object, a function,
a [`Map` or `Set`](../data/map-and-set.md), [bytes](../data/bytes.md), a
[struct](../data/structs.md) or a [stream](../stdlib/streams.md). `type`
names it:

```rune
print(type(nil));              // nil
print(type(42));               // number
print(type("ann"));            // string
print(type([1, 2]));           // array
print(type({ name: "ann" }));  // object
print(type(x => x * 2));       // function
```

## Truthiness

Falsy values are `nil`, `false`, `0` and `""`; everything else, including empty
arrays and objects, is truthy.

```rune
for v in [nil, false, 0, "", [], {}, "0"] {
    print("%q is %s".format(v, v ? "truthy" : "falsy"));
}
// nil is falsy
// false is falsy
// 0 is falsy
// "" is falsy
// [] is truthy
// {} is truthy
// "0" is truthy
```

So `or` gives a fallback for an empty string, and an empty array needs `len`:

```rune
let user = { name: "ann", nickname: "" };
print(user.nickname || user.name);   // ann

let cart = [];
if cart { print("has a cart"); }     // has a cart
if len(cart) == 0 { print("cart is empty"); }  // cart is empty
```

## Equality

`==` compares numbers (an integer with a float too), strings, bytes, booleans and `nil` by value, functions and streams by identity,
and arrays, objects, maps and sets structurally (element by element, cycles included), so
`[1, 2] == [1, 2]` is `true`. `contains` and `indexOf` use the same equality.
Values of different types are never equal: `1 == "1"` is `false`.

```rune
print(1 == 1.0);                                    // true
print([1, [2, 3]] == [1, [2, 3]]);                  // true
print({ role: "admin", tags: ["ops"] } == { tags: ["ops"], role: "admin" }); // true
print(1 == "1");                                    // false
print(nil == false);                                // false

let seen = [{ id: 1 }, { id: 2 }];
print(seen.contains({ id: 2 }));                    // true
print(seen.indexOf({ id: 2 }));                     // 1

let validate = order => order.qty > 0;
let check = validate;
print(check == validate);                           // true
print((order => order.qty > 0) == validate);        // false
```

## Printing

`print` shows strings inside arrays and objects quoted, so `["1", 1]` and `[1, 1]`
look different; a string printed on its own is written as is. Objects print
their keys in sorted order.

```rune
print("1");                          // 1
print(["1", 1]);                     // ["1", 1]
print([nil, true, 2.5]);             // [nil, true, 2.5]
print({ name: "ann", age: 31 });     // {age: 31, name: "ann"}
```
