# Conditionals

## if and else

The condition needs no parentheses; it is tested for
[truthiness](values.md#truthiness).

```rune
let temp = 22;

if temp > 30 {
    print "hot";
} else if temp > 15 {
    print "mild";
} else {
    print "cold";
}
// mild
```

A body is a block, or a bare statement starting with a name or keyword, which
suits early returns:

```rune
fun classify(age) {
    if age < 0 return "invalid";
    if age < 18 return "minor";
    return "adult";
}

print classify(-1);    // invalid
print classify(30);    // adult
```

## and, or, !

`and` and `or` stop as soon as the result is known and give one of their
operands, not a boolean; `!` negates. `&&` and `||` are an error with a hint.

```rune
let user = nil;
if !user { print "guest"; }       // guest

print nil or "default";           // default
print 0 or 5;                     // 5
print "ann" and "admin";          // admin
```

## Conditional Expressions

`c ? a : b` is `a` if `c` is truthy, else `b`; only the chosen branch runs.
It binds looser than ranges and `??`, tighter than assignment, and nests to
the right: `a ? b : c ? d : e` is `a ? b : (c ? d : e)`.

```rune
let stock = 3;
let label = stock > 10 ? "in stock" : stock > 0 ? "low" : "sold out";
print label;    // low

let cart = ["tea", "milk"];
print "{len(cart)} item{len(cart) == 1 ? "" : "s"}";    // 2 items

let parity = n => n % 2 == 0 ? "even" : "odd";
print parity(7);    // odd
```
