# Conditionals

## if and else

The condition needs no parentheses; it is tested for
[truthiness](values.md#truthiness).

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

A body is always a block, here and in `else`, `while`, `for` and `defer`:

```rune
fun classify(age) {
    if age < 0 { return "invalid"; }
    if age < 18 { return "minor"; }
    return "adult";
}

print(classify(-1));    // invalid
print(classify(30));    // adult
```

## &&, || and !

`&&` and `||` stop as soon as the result is known and give one of their
operands, not a boolean; `!` negates. `!` binds tightly, like `-`, so
`!a == b` is `(!a) == b`; write `!(a == b)` or `a != b`.

```rune
let user = nil;
if !user { print("guest"); }       // guest
print(!(1 == 2));                  // true
print(!1 == 2);                    // false

print(nil || "default");           // default
print(0 || 5);                     // 5
print("ann" && "admin");           // admin
```

## if as an Expression

A block's value is its last expression when that has no `;` after it;
a block ending with `;` gives `nil`. So `if` is an expression too: its
value is that of the branch taken, `nil` when no branch is. `else if`
chains as usual.

```rune
let stock = 3;
let label = if stock > 10 { "in stock" } else if stock > 0 { "low" } else { "sold out" };
print(label);    // low

let cart = ["tea", "milk"];
print("{cart.len()} item{if cart.len() == 1 { "" } else { "s" }}");    // 2 items

print(if stock > 10 { "many" });    // nil
```

A branch may run statements before its value:

```rune
let score = 72;
let grade = if score >= 90 {
    "A"
} else {
    let gap = 90 - score;
    "B, {gap} to A"
};
print(grade);    // B, 18 to A
```

After `if` a `{` always starts a block; an object to give goes in
parentheses: `if ok { ({ code: 200 }) } else { nil }`.

The bodies of `while`, `for` and `defer` have no value, so each of their
statements ends with `;`.
