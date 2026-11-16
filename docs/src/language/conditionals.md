# Conditionals

## if and else

`if` runs a block when its condition holds. The condition needs no
parentheses, and the body always needs braces, even for one statement.
Chain more cases with `else if`, and catch the rest with `else`.

```rune
let temp = 22;

if temp > 30 {
    print("hot");
} else if temp > 15 {
    print("mild");
} else {
    print("cold");
}
// mild
```

The condition does not have to be a boolean. Any value works, and it is
tested for [truthiness](values.md#truthiness): `nil`, `false`, `0`, `""`
and empty containers count as false, everything else as true.

```rune
let cart = [];
let coupon = "SPRING10";

if !cart { print("cart is empty"); }      // cart is empty
if coupon { print("coupon: {coupon}"); }  // coupon: SPRING10
```

A common shape is a series of early exits, one check per line:

```rune
fun classify(age) {
    if age < 0 { return "invalid" }
    if age < 18 { return "minor" }
    "adult"
}

print(classify(-1));    // invalid
print(classify(12));    // minor
print(classify(30));    // adult
```

## &&, || and !

`&&` and `||` combine conditions. They stop as soon as the result is
known, so the right side runs only when needed:

```rune
let user = { name: "ann", banned: false, orders: 3 };

if user && !user.banned && user.orders > 0 {
    print("{user.name} can leave a review");
}
// ann can leave a review
```

They give back one of their operands, not a plain `true` or `false`. `||`
gives the first truthy operand, which makes it a quick fallback; `&&`
gives the first falsy one, or the last:

```rune
print(nil || "guest");       // guest
print("" || "anonymous");    // anonymous
print(0 || 5);               // 5
print("ann" && "admin");     // admin
print(0 && "never");         // 0
```

`||` treats `0`, `""` and `false` as missing. When those are real values,
use [`??`](nil-safety.md#default-values), which only replaces `nil`.

`!` negates. It binds tightly, like a minus sign, so `!a == b` means
`(!a) == b`. To negate a comparison, write `!(a == b)` or simply `a != b`:

```rune
print(!(1 == 2));    // true
print(!1 == 2);      // false
print(1 != 2);       // true
```

## if as an Expression

A block gives a value: its last expression, written without a `;` after
it. A block that ends with `;` gives `nil`. So `if` is an expression too,
and gives the value of the branch it takes. Use it to pick a value without
a temporary `let mut`:

```rune
let stock = 3;
let label = if stock > 10 { "in stock" } else if stock > 0 { "low" } else { "sold out" };
print(label);    // low
```

It fits anywhere a value does, even inside string interpolation:

```rune
let cart = ["tea", "milk"];
print("{cart.len()} item{if cart.len() == 1 { "" } else { "s" }}");    // 2 items
```

With no `else`, an `if` whose condition fails gives `nil`:

```rune
let stock = 3;
print(if stock > 10 { "many" });    // nil
```

A branch may run statements before its value. Only the last line, the one
without `;`, is the result:

```rune
let score = 72;
let grade = if score >= 90 {
    "A"
} else {
    let gap = 90 - score;
    "B, {gap} points to A"
};
print(grade);    // B, 18 points to A
```

After `if`, a `{` always starts a block. To give an object, wrap it in
parentheses:

```rune
let ok = true;
let response = if ok { ({ code: 200 }) } else { ({ code: 500 }) };
print(response);    // {code: 200}
```

The bodies of `while`, `for` and `defer` give no value, so every statement
in them ends with `;`. The exception is `return`, `break` and `continue`
right before a `}`: they leave the block, so they need no `;` anywhere:

```rune
let mut paid = [];
for amount in [20, -5, 0, 40, 999] {
    if amount <= 0 { continue }
    if amount > 500 { break }
    paid.push(amount);
}
print(paid);    // [20, 40]
```

To test the shape of a value and take it apart in one step, as in
`if resp is { status: 200, body } { ... }`, see
[Binding in Conditions](values.md#binding-in-conditions). To choose
between many values, see [`match`](pattern-matching.md).
