# Functions

`fun name(params) { ... }` declares a function. It returns the value of its
body: the last expression, written without a `;`, or an `if` or `match`
that ends it and has values. A body ending with `;` returns `nil`.
`return` leaves early.

```rune
fun total(items) {
    let mut sum = 0;
    for item in items { sum += item.price * item.qty; }
    return sum;
}

print(total([{ price: 10, qty: 2 }, { price: 5, qty: 1 }])); // 25
```

A function captures the variables around it, and keeps them after the
enclosing call returns:

```rune
fun counter() {
    let mut n = 0;
    return () => { n += 1; return n; };
}

let next = counter();
next();
print(next()); // 2
```

Calling with the wrong number of arguments is an `ArgumentError`:

```rune
fun area(w, h) { return w * h; }
area(3); // ArgumentError: Expected 2 arguments but got 1.
```

## Body Values

```rune
fun square(x) { x * x }
fun origin() { ({ x: 0, y: 0 }) }

fun sign(n) {
    if n < 0 { return "negative"; }
    if n == 0 { "zero" } else { "positive" }
}

struct Money { cents }

impl Money {
    fun of(amount) { Money { cents: math.round(amount * 100) } }
    fun format(self) { "$%.2f".format(self.cents / 100) }
}

print(square(7));                // 49
print(origin());                 // {x: 0, y: 0}
print(sign(-2), sign(0));        // negative zero
print(Money.of(4.99).format());  // $4.99
```

## Arrow Functions

A function without a name is an arrow function, an expression, so
functions can be passed around inline. An expression body is returned; a
block body works like `fun`'s, giving its last expression.

```rune
let add = (a, b) => a + b;
let double = x => x * 2;
let log = msg => { print("[log] " + msg); };

print(add(2, 3));                     // 5
print(double(21));                    // 42
print([120, 45, 300].map(double));    // [240, 90, 600]
log("started");                       // [log] started
```

A single plain parameter needs no parentheses. Parameters may be
[`mut`](#mut-parameters) or [patterns](destructuring.md):

```rune
let users = [{ name: "ann", age: 31 }, { name: "bob", age: 17 }];

print(users.map(({ name }) => name));                    // ["ann", "bob"]
print(users.filter(u => u.age >= 18).map(u => u.name));  // ["ann"]
```

After `=>` a `{` always starts a block, here and in a match arm; an
object to return goes in parentheses:

```rune
let origin = () => ({ x: 0, y: 0 });
let wrap = name => ({ name });

print(origin());     // {x: 0, y: 0}
print(wrap("ann"));  // {name: "ann"}
```

```rune
let wrap = name => { name: name }; // Error at '{': A '{' here starts a block; wrap an object in parentheses: ({ ... }).
```

## Default Parameters

A parameter may have a default, `name = expression`, taken when the call
passes no argument for it.

```rune
fun connect(host, port = 5432) {
    return "{host}:{port}";
}

print(connect("db.local"));       // db.local:5432
print(connect("db.local", 6379)); // db.local:6379
```

The default is evaluated at each such call, after the parameters before it,
which it may use, so a mutable default is a new value every call. An
explicit `nil` is an argument and gets no default.

```rune
fun greet(name, greeting = "hi", punct = if greeting == "hi" { "!" } else { "." }) {
    return "{greeting} {name}{punct}";
}

print(greet("ann"));          // hi ann!
print(greet("ann", "hello")); // hello ann.
print(greet("ann", nil));     // nil ann.

fun tag(item, tags = []) { let mut all = tags; all.push(item); return all; }
print(tag("a")); // ["a"]
print(tag("b")); // ["b"]
```

Every parameter after one with a default needs one too, except a rest
parameter; `self` and a rest parameter cannot have one.

```rune
fun f(a = 1, b) {} // Error at 'b': A parameter without a default cannot follow one with a default.
```

Arrow functions take defaults too:

```rune
let inc = (n, by = 1) => n + by;
print(inc(5));     // 6
print(inc(5, 10)); // 15
```

## Mut Parameters

A parameter is a copy of its argument, so a function cannot change the
caller's value. A `mut` parameter is the one way to: the function works on
its own value, and when the call ends, by a return, a throw or the end of
the body, the value goes back to the place the caller passed.

```rune
fun addTag(mut tags, tag) { tags.push(tag); }

let mut tags = ["new"];
addTag(tags, "sale");
print(tags); // ["new", "sale"]

let mut user = { name: "ann", tags: [] };
addTag(user.tags, "admin");
print(user.tags); // ["admin"]
```

The argument must be a mutable place: a `let mut` variable, a `mut`
parameter, or an element or property on a path from one. The place is
found once, before the call. A copy taken from it earlier keeps its value.

```rune
addTag(["x"], "y"); // Error at ')': Mut parameter 'tags' takes a mutable variable, element or property.

let fixed = [];
addTag(fixed, "y"); // Error at 'fixed': Cannot pass immutable variable 'fixed' to mut parameter 'tags'.
```

A call through a function value, a method or a callback checks this when
it runs: a function whose `mut` parameter gets no mutable place fails with
a `TypeError`. A native such as `map` passes none.

```rune
let f = [addTag][0];
let mut xs = [];
f(xs, 1);   // xs is [1]
f([], 2);   // TypeError: Mut parameter 'tags' of addTag takes a mutable variable, element or property.
```

A rest parameter cannot be `mut`. A `mut` parameter left to its default
writes back nothing.
