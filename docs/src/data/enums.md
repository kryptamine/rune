# Enums

An enum is a type with a fixed set of named values, its variants. A
variant is not a number or a string: it equals only itself, prints as
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

## Methods

`impl` adds methods, as for a struct; `self` is the variant. A variant
has no fields to change, so methods cannot take `mut self`.

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
