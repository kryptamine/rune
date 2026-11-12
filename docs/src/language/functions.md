# Functions

`fun name(params) { ... }` declares a function. `return` gives its value; a
function that ends without one returns `nil`.

```rune
fun total(items) {
    let mut sum = 0;
    for item in items { sum += item.price * item.qty; }
    return sum;
}

print total([{ price: 10, qty: 2 }, { price: 5, qty: 1 }]); // 25
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
print next(); // 2
```

Calling with the wrong number of arguments is an `ArgumentError`:

```rune
fun area(w, h) { return w * h; }
area(3); // ArgumentError: Expected 2 arguments but got 1.
```

## Anonymous Functions

`fun` without a name is an expression, so functions can be passed around inline.

```rune
let prices = [120, 45, 300];
print prices.map(fun (p) { return p * 2; }); // [240, 90, 600]
```

Arrow functions are shorthand: an expression body is returned, a block body works like `fun`'s.

```rune
let add = (a, b) => a + b;
let double = x => x * 2;
let log = msg => { print "[log] " + msg; };

print add(2, 3);  // 5
print double(21); // 42
log("started");   // [log] started
```

A single plain parameter needs no parentheses. Parameters may be `mut` or
[patterns](destructuring.md):

```rune
let users = [{ name: "ann", age: 31 }, { name: "bob", age: 17 }];

print users.map(({ name }) => name);                    // ["ann", "bob"]
print users.filter(u => u.age >= 18).map(u => u.name);  // ["ann"]

let bump = (mut n) => { n += 1; return n; };
print bump(1); // 2
```

A body in braces is an object when it looks like one: `() => { a: 1 }` and
`x => { x }` return an object.

```rune
let origin = () => { x: 0, y: 0 };
let wrap = name => { name };

print origin();     // {x: 0, y: 0}
print wrap("ann");  // {name: "ann"}
```

## Default Parameters

A parameter may have a default, `name = expression`, taken when the call
passes no argument for it.

```rune
fun connect(host, port = 5432) {
    return "{host}:{port}";
}

print connect("db.local");       // db.local:5432
print connect("db.local", 6379); // db.local:6379
```

The default is evaluated at each such call, after the parameters before it,
which it may use, so a mutable default is a new value every call. An
explicit `nil` is an argument and gets no default.

```rune
fun greet(name, greeting = "hi", punct = greeting == "hi" ? "!" : ".") {
    return "{greeting} {name}{punct}";
}

print greet("ann");          // hi ann!
print greet("ann", "hello"); // hello ann.
print greet("ann", nil);     // nil ann.

fun tag(item, mut tags = []) { tags.push(item); return tags; }
print tag("a"); // ["a"]
print tag("b"); // ["b"]
```

Every parameter after one with a default needs one too, except a rest
parameter; `self` and a rest parameter cannot have one.

```rune
fun f(a = 1, b) {} // Error at 'b': A parameter without a default cannot follow one with a default.
```

Arrow functions take defaults too:

```rune
let inc = (n, by = 1) => n + by;
print inc(5);     // 6
print inc(5, 10); // 15
```
