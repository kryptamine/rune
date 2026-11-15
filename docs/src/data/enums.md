# Enums

An enum is a type with a fixed set of named values, its variants, which
may [have fields](#variants-with-fields). A variant is not a number or a string: it equals only itself, prints as
`Type.Variant` and is tested with `is`. Enums are declared at the top
level only and can be used before their declaration.

```rune
enum Color { White, Black }

let c = Color.White;
print(c);                // Color.White
print(c is Color);       // true
print(c == Color.Black); // false
print(type(c));          // Color
print(type(Color));      // enum
print(Color is enum);    // true
```

A name that is not a variant is a compile error:

```rune
enum Color { White, Black }
print(Color.Red); // Error at 'Red': Color has no variant 'Red'.
```

## Ordinals and conversion

Variants are numbered from 0 in declaration order. `int(v)` is a variant's
number and `v.name` its name; calling the enum converts back, from the
number or from the name. `Color.values()` lists the variants in order.

```rune
enum Dir { North, East, South, West }

print(int(Dir.South));         // 2
print(Dir.South.name);         // South
print(Dir(3));                 // Dir.West
print(Dir("East"));            // Dir.East
print(Dir.values());           // [Dir.North, Dir.East, Dir.South, Dir.West]
print(Dir((int(Dir.West) + 1) % 4)); // Dir.North
```

```rune
enum Dir { North, East, South, West }
Dir(4); // ValueError: Dir has no variant 4.
```

`json.stringify` writes a variant as its name, so `Dir(name)` reads it
back. Variants are hashable, as map keys and set elements.

```rune
enum Dir { North, East }

let saved = json.stringify({ facing: Dir.East });
print(saved);                             // {"facing":"East"}
print(Dir(json.parse(saved).facing));     // Dir.East
```

## Matching

A variant is a pattern:

```rune
enum Piece { Pawn, Knight, Bishop, Rook, Queen, King }

fun worth(p) {
    return match p {
        Piece.Pawn => 1,
        Piece.Knight | Piece.Bishop => 3,
        Piece.Rook => 5,
        Piece.Queen => 9,
        Piece.King => 0,
    };
}

print(worth(Piece.Bishop)); // 3
```

## Variants with fields

A variant may have fields, declared as a struct's are. `Enum.Variant { ... }`
makes one, and it is a struct of that type: its fields are read with `.`,
matched with `Enum.Variant { ... }` and tested with `is`; `v is Enum` holds
for every variant, and `type(v)` is the variant's type, `"Enum.Variant"`.

```rune
enum Expr {
    Num { value },
    Add { left, right },
    Neg { expr },
    Zero,
}

let e = Expr.Add { left: Expr.Num { value: 1 }, right: Expr.Neg { expr: Expr.Num { value: 3 } } };
print(e.left.value);    // 1
print(e is Expr.Add);   // true
print(e is Expr);       // true
print(type(e));         // Expr.Add
print(Expr.Num { value: 5 }); // Expr.Num {value: 5}
```

Fields are read-only, defaults and spread work as for structs, and
`Expr.Num(obj)` converts. An enum with such variants has no ordinals:
`values()`, `Expr(n)` and `int(v)` are a `TypeError`.

```rune
enum Expr { Num { value }, Zero }
print(Expr.Num); // <struct Expr.Num>
Expr.values();   // TypeError: Expr has variants with fields; values(), Expr(n) and int() are for enums without.
```

## Matching every variant

A `match` that names variants of an enum and has no catch-all arm must
cover them all: a variant is covered by an arm without a guard whose
pattern is the variant, `_ is Enum.Variant`, or `Enum.Variant { ... }`
binding its fields. A missing one is a compile error.

```rune
enum Shape { Circle { r }, Square { side }, Dot }

fun area(s) {
    return match s {
        Shape.Circle { r } => 3 * r * r,
        Shape.Square { side } => side * side,
        Shape.Dot => 0,
    };
}

print(area(Shape.Square { side: 2 })); // 4
```

```rune
enum Shape { Circle { r }, Square { side }, Dot }
match Shape.Dot { Shape.Dot => print("dot") } // Error at 'match': Match does not cover Shape.Circle, Shape.Square; add arms or '_ =>'.
```

## Methods

`impl` adds methods, as for a struct; `self` is the variant. Variants
are not changed in place, so methods cannot take `mut self`, nor share a
name with a field of a variant.

```rune
enum Side { White, Black }

impl Side {
    fun other(self) {
        return self == Side.White ? Side.Black : Side.White;
    }
    fun parse(s) {
        return s == "w" ? Side.White : Side.Black;
    }
}

print(Side.White.other());              // Side.Black
print(Side.parse("b"));                 // Side.Black
print(Side.values().map(Side.other));   // [Side.Black, Side.White]
```

An enum is no struct, so `Color { }` is an error. A variant cannot be
named `values` or `name`, nor can a method taking `self`.
