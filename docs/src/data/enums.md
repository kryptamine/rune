# Enums

An enum is a type with a fixed set of named values, called variants.
Use one when a value is one of a few known choices: an order status, a
direction, a log level. Unlike strings, a typo is a compile error, and a
`match` can check that every case is handled.

```rune
enum Status { Pending, Paid, Shipped, Cancelled }

let s = Status.Paid;
print(s);                   // Status.Paid
print(s == Status.Paid);    // true
print(s == Status.Shipped); // false
print(s is Status);         // true
```

A variant is not a number or a string: it equals only itself and prints
as `Type.Variant`. Enums are declared at the top level only and can be
used before their declaration.

A name that is not a variant is caught before the program runs:

```rune
enum Status { Pending, Paid, Shipped, Cancelled }
print(Status.Payed); // Error at 'Payed': Status has no variant 'Payed'.
```

Variants can also [carry fields](#variants-with-fields), which makes
enums a natural fit for values that come in several shapes.

## Matching

A variant is a pattern, so `match` turns a variant into whatever it
stands for:

```rune
enum Status { Pending, Paid, Shipped, Cancelled }

fun label(s) {
    match s {
        Status.Pending => "waiting for payment",
        Status.Paid | Status.Shipped => "on its way",
        Status.Cancelled => "cancelled",
    }
}

print(label(Status.Pending)); // waiting for payment
print(label(Status.Shipped)); // on its way
```

This `match` has no `_` arm, and it does not need one: it lists every
variant. If a variant is later added to `Status`, the compiler points at
each `match` that forgot it; see [Matching every variant](#matching-every-variant).

When every arm names a variant of one enum, say so once in the head,
`match s is Status`, and name the variants bare. The value must then be a
`Status`; anything else is a `ValueError`, as when no arm matches:

```rune
enum Status { Pending, Paid, Shipped, Cancelled }

fun label(s) {
    match s is Status {
        Pending => "waiting for payment",
        Paid | Shipped => "on its way",
        Cancelled => "cancelled",
    }
}

print(label(Status.Shipped)); // on its way
print(label("paid"));         // ValueError: No match arm for "paid".
```

A bare name in such a match is always a variant, so a typo is caught:

```rune
enum Status { Pending, Paid }
match Status.Paid is Status { Pendin => 1, _ => 2 }; // Error at 'Pendin': Status has no variant 'Pendin'.
```

To match on whether a value is a `Status`, put the test in parentheses:
`match (s is Status) { true => ..., false => ... }`.

## Numbers and Names

Variants are numbered from 0 in declaration order. `int(v)` gives a
variant's number and `v.name` its name. Calling the enum converts back,
from a number or from a name, and `values()` lists all variants in order.

```rune
enum Dir { North, East, South, West }

print(int(Dir.South));  // 2
print(Dir.South.name);  // South
print(Dir(3));          // Dir.West
print(Dir("East"));     // Dir.East
print(Dir.values());    // [Dir.North, Dir.East, Dir.South, Dir.West]
```

That is enough to turn right by stepping to the next variant and
wrapping around:

```rune
enum Dir { North, East, South, West }

fun turnRight(d) { Dir((int(d) + 1) % Dir.values().len()) }

print(turnRight(Dir.North)); // Dir.East
print(turnRight(Dir.West));  // Dir.North
```

A number or name with no variant is a `ValueError`:

```rune
enum Dir { North, East, South, West }
Dir(4); // ValueError: Dir has no variant 4.
```

`json.stringify` writes a variant as its name, so calling the enum on
the parsed name reads it back. Variants can also be map keys and set
elements.

```rune
enum Dir { North, East, South, West }

let saved = json.stringify(["facing": Dir.East]);
print(saved);                            // {"facing":"East"}
print(Dir(json.parse(saved)["facing"])); // Dir.East

let visits = [Dir.North: 2, Dir.West: 1];
print(visits[Dir.North]); // 2
```

## Variants with Fields

A variant may have fields, declared like a struct's. Each such variant
is a struct type of its own: build it with `Enum.Variant { ... }`, read
its fields with `.`, and test it with `is`. `v is Enum` holds for every
variant.

This fits data that comes in several shapes, such as payment methods:

```rune
enum Payment {
    Card { last4, brand = "visa" },
    Transfer { iban },
    Cash,
}

let p = Payment.Card { last4: "4242" };
print(p);                    // Payment.Card {last4: "4242", brand: "visa"}
print(p.last4);              // 4242
print(p is Payment.Card);    // true
print(p is Payment);         // true
print(Payment.Cash is Payment); // true
```

Match on the variant and take its fields apart in one step:

```rune
enum Payment {
    Card { last4, brand = "visa" },
    Transfer { iban },
    Cash,
}

fun describe(p) {
    match p {
        Payment.Card { brand, last4 } => "{brand} card ending in {last4}",
        Payment.Transfer { iban } => "bank transfer from {iban[..4]}…",
        Payment.Cash => "cash on delivery",
    }
}

print(describe(Payment.Card { last4: "4242", brand: "amex" })); // amex card ending in 4242
print(describe(Payment.Transfer { iban: "DE89370400440532013000" })); // bank transfer from DE89…
print(describe(Payment.Cash));                                  // cash on delivery
```

Variants that refer to the enum itself describe trees. Here is a small
expression evaluator:

```rune
enum Expr {
    Num { value },
    Add { left, right },
    Mul { left, right },
    Neg { expr },
}

fun eval(e) {
    match e is Expr {
        Num { value } => value,
        Add { left, right } => eval(left) + eval(right),
        Mul { left, right } => eval(left) * eval(right),
        Neg { expr } => -eval(expr),
    }
}

// -(2 + 3) * 4
let e = Expr.Mul {
    left: Expr.Neg { expr: Expr.Add { left: Expr.Num { value: 2 }, right: Expr.Num { value: 3 } } },
    right: Expr.Num { value: 4 },
};
print(eval(e)); // -20
```

Fields work as in a [struct](structs.md): they are read-only, defaults
and spread work, `Expr.Num { ...obj }` included. An enum that has
variants with fields has no numbers, so `values()`, `Expr(n)` and
`int(v)` are a `TypeError`:

```rune
enum Expr { Num { value }, Zero }
Expr.values(); // TypeError: Expr has variants with fields; values(), Expr(n) and int() are for enums without.
```

## Matching Every Variant

A `match` that names variants of an enum, and has no arm that matches
everything, must cover all of them. A variant counts as covered by an
arm without a guard whose pattern is the variant itself,
`_ is Enum.Variant`, or `Enum.Variant { ... }`. A missing one is a
compile error that names it:

```rune
enum Shape { Circle { r }, Square { side }, Dot }

fun area(s) {
    match s {
        Shape.Circle { r } => math.PI * r * r,
        Shape.Square { side } => side * side,
    }
} // Error at 'match': Match does not cover Shape.Dot; add arms or '_ =>'.
```

This is the main reason to prefer an enum over strings: adding a variant
shows you every place that has to handle it.

## Methods

`impl` adds methods to an enum, as to a struct; `self` is the variant.
Variants never change in place, so methods cannot take `mut self`, and
they cannot share a name with a field of a variant.

```rune
enum Side { White, Black }

impl Side {
    fun other(self) {
        if self == Side.White { Side.Black } else { Side.White }
    }

    fun parse(s) {
        match s {
            "w" => Side.White,
            "b" => Side.Black,
            _ => throw ValueError { message: "unknown side '{s}'" },
        }
    }
}

print(Side.White.other());            // Side.Black
print(Side.parse("b"));               // Side.Black
print(Side.values().map(s => s.other())); // [Side.Black, Side.White]
```

An enum is not a struct, so `Side {}` is an error. No variant, and no
method taking `self`, may be named `values` or `name`.
