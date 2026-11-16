# Functions

`fun name(params) { ... }` declares a function. Its result is the value of
its body: the last expression, written without a `;` after it.

```rune
fun greet(name) {
    "hello, {name}!"
}

print(greet("ann"));    // hello, ann!
```

A body can run any statements before that final value:

```rune
struct Item { price, qty }

fun total(items) {
    let mut sum = 0;
    for item in items {
        sum += item.price * item.qty;
    }
    sum
}

let cart = [Item { price: 10, qty: 2 }, Item { price: 5, qty: 1 }];
print(total(cart));    // 25
```

A body that ends with `;` gives `nil`, which suits functions called only
for what they do, such as printing.

Calling with the wrong number of arguments is an `ArgumentError`:

```rune
fun area(w, h) { w * h }
area(3); // ArgumentError: Expected 2 arguments but got 1.
```

## Body Values

Short functions fit on one line:

```rune
fun square(x) { x * x }
fun isEven(n) { n % 2 == 0 }

print(square(7));    // 49
print(isEven(10));   // true
```

When the body ends with an `if` or a `match`, the branch taken gives the
result. `return` leaves early, which keeps guard checks flat:

```rune
struct Order { total }

fun shipping(order) {
    if order.total <= 0 { return "nothing to ship" }
    if order.total >= 50 { "free" } else { "$4.99" }
}

print(shipping(Order { total: 0 }));     // nothing to ship
print(shipping(Order { total: 80 }));    // free
print(shipping(Order { total: 20 }));    // $4.99

fun httpStatus(code) {
    match code {
        200 => "ok",
        404 => "not found",
        500.. => "server error",
        _ => "other",
    }
}

print(httpStatus(404));    // not found
print(httpStatus(503));    // server error
```

Methods work the same way:

```rune
struct Money { cents }

impl Money {
    fun of(amount) { Money { cents: math.round(amount * 100) } }
    fun format(self) { "$%.2f".format(self.cents / 100) }
}

print(Money.of(4.99).format());    // $4.99
```

## Closures

A function can use the variables around it. It keeps them alive even
after the function that made them has returned. Here each counter keeps
its own `n`:

```rune
fun counter() {
    let mut n = 0;
    () => {
        n += 1;
        n
    }
}

let tickets = counter();
let orders = counter();
tickets();
tickets();
print(tickets());    // 3
print(orders());     // 1
```

## Arrow Functions

An arrow function is a function without a name, written as an expression.
Use it to pass behavior inline, as in `map` or `filter`. A single
parameter needs no parentheses:

```rune
let add = (a, b) => a + b;
let double = x => x * 2;

print(add(2, 3));                     // 5
print(double(21));                    // 42
print([120, 45, 300].map(double));    // [240, 90, 600]
```

A longer body goes in braces and gives its last expression, like `fun`:

```rune
let describe = n => {
    let kind = if n % 2 == 0 { "even" } else { "odd" };
    "{n} is {kind}"
};
print(describe(7));    // 7 is odd
```

Parameters may be [patterns](destructuring.md), which takes just the
fields you need:

```rune
struct User { name, age }

let users = [User { name: "ann", age: 31 }, User { name: "bob", age: 17 }];

print(users.map(({ name }) => name));                    // ["ann", "bob"]
print(users.filter(u => u.age >= 18).map(u => u.name));  // ["ann"]
```

After `=>` a `{` starts a block, both here and in a `match` arm.

## Default Parameters

A parameter may have a default, `name = value`, used when the call leaves
that argument out:

```rune
fun connect(host, port = 5432) {
    "{host}:{port}"
}

print(connect("db.local"));          // db.local:5432
print(connect("db.local", 6379));    // db.local:6379
```

The default is evaluated at each call that needs it, so it may depend on
the parameters before it:

```rune
fun greet(name, greeting = "hi", punct = if greeting == "hi" { "!" } else { "." }) {
    "{greeting} {name}{punct}"
}

print(greet("ann"));             // hi ann!
print(greet("ann", "hello"));    // hello ann.
```

Because it is evaluated fresh, a default `[]` is a new array every time,
never one shared between calls:

```rune
fun tag(item, tags = []) {
    let mut all = tags;
    all.push(item);
    all
}

print(tag("a"));    // ["a"]
print(tag("b"));    // ["b"]
```

Passing `nil` explicitly is passing a value, so the default is not used:

```rune
fun greet(name, greeting = "hi") { "{greeting} {name}" }
print(greet("ann", nil));    // nil ann
```

Arrow functions take defaults too:

```rune
let inc = (n, by = 1) => n + by;
print(inc(5));        // 6
print(inc(5, 10));    // 15
```

Every parameter after one with a default needs a default too, except a
rest parameter. `self` and rest parameters cannot have one.

```rune
fun f(a = 1, b) {} // Error at 'b': A parameter without a default cannot follow one with a default.
```

## Mut Parameters

Arguments are passed as copies, so a function cannot change the caller's
variable by accident:

```rune
fun addTag(tags, tag) {
    let mut copy = tags;
    copy.push(tag);
    copy
}

let tags = ["new"];
print(addTag(tags, "sale"));    // ["new", "sale"]
print(tags);                    // ["new"]
```

When changing the caller's value is the point, mark the parameter `mut`.
The function works on its own copy, and when it ends, whether by a
`return`, a throw or the end of the body, the copy is written back to the
place the caller passed:

```rune
fun addTag(mut tags, tag) {
    tags.push(tag);
}

let mut tags = ["new"];
addTag(tags, "sale");
print(tags);         // ["new", "sale"]

let mut user = ["name": "ann", "tags": []];
addTag(user["tags"], "admin");
print(user["tags"]);    // ["admin"]
```

The argument must be a mutable place: a `let mut` variable, a `mut`
parameter, or an element or property reached from one. A value with no
place to write back to is a compile error:

```rune
fun addTag(mut tags, tag) { tags.push(tag); }
addTag(["x"], "y"); // Error at ')': Mut parameter 'tags' takes a mutable variable, element or property.
```

So is a `let` variable, which cannot change:

```rune
fun addTag(mut tags, tag) { tags.push(tag); }
let fixed = [];
addTag(fixed, "y"); // Error at 'fixed': Cannot pass immutable variable 'fixed' to mut parameter 'tags'.
```

A call through a function value, a method or a callback is checked when it
runs instead, and fails with a `TypeError`. Natives such as `map` never
pass a place, so they cannot call a function with a `mut` parameter:

```rune
fun addTag(mut tags, tag) { tags.push(tag); }
let f = [addTag][0];
f([], "y"); // TypeError: Mut parameter 'tags' of addTag takes a mutable variable, element or property.
```

A rest parameter cannot be `mut`. A `mut` parameter left to its default
writes nothing back.
