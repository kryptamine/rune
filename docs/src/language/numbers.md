# Numbers

A number is a 64-bit integer or a float; `type` says `"number"` for both
and `math.isInt` tells them apart. A literal without a fraction or an
exponent is an integer; `0x`, `0o` and `0b` prefix hex, octal and binary
ones, and `_` may separate digits. An integer literal past `math.MAX_INT`
is a compile error.

```rune
print(1_000_000);    // 1000000
print(0xff);         // 255
print(0o17);         // 15
print(0b1010);       // 10
print(1e3);          // 1000
print(2.5e-3);       // 0.0025
print(math.isInt(42));    // true
print(math.isInt(1e3));   // false
```

## Arithmetic

`+`, `-`, `*` and `%` on two integers give an integer, and an integer that
would overflow is a `ValueError`; with a float they give a float. `/`
always gives a float: `6 / 2` is `3`, a float, and `math.idiv(7, 2)` is the
integer `3`. A float with an integer value prints without a fraction, and
one outside [1e-6, 1e21) with an exponent: `1e+21`, `1e-7`.

```rune
let price = 19.99;
let qty = 3;
print(qty * 2);                // 6
print(price * qty);            // 59.97
print(10 / 4);                 // 2.5
print(math.isInt(10 / 5));     // false

let items = 17;
let perPage = 5;
print(math.idiv(items, perPage));       // 3
print(items % perPage);                 // 2
print(math.ceil(items / perPage));      // 4
```

`%` takes the sign of its left operand, and an integer `% 0` is a
`ValueError`. A float divided by zero is `inf`, `-inf` or `nan`.

```rune
print(-7 % 3);     // -1
print(7 / 0);      // inf
```

```rune
print(math.MAX_INT);        // 9223372036854775807
print(math.MAX_INT + 1);    // ValueError: Integer overflow.
```

```rune
print(7 % 0);               // ValueError: Integer modulo by zero.
```

Integers compare exactly with floats, so `1 == 1.0`, and
`9007199254740993 == 9007199254740992.0` is `false`.

```rune
print(1 == 1.0);                                   // true
print(9007199254740993 == 9007199254740992.0);     // false
print(0.1 + 0.2);                                  // 0.30000000000000004
```

## Bitwise Operators

The bitwise operators `&`, `|`, `^`, `~`, `<<` and `>>` take integers, or
floats with an integer value, and give an integer; `>>` keeps the sign, a
shift by 64 or more gives `0` (or `-1`), and a negative one is a
`ValueError`. They bind tighter than comparisons and looser than `+`, with
`|` loosest, then `^`, `&` and the shifts: `1 + 2 & 3` is `3`, and
`flags & MASK == 0` needs no parentheses. Each has a compound assignment:
`&=`, `|=`, `^=`, `<<=`, `>>=`.

```rune
let READ = 0b001;
let WRITE = 0b010;
let EXEC = 0b100;

let mut perms = READ | WRITE;
print(perms);                     // 3
print(perms & WRITE == WRITE);    // true
perms &= ~WRITE;
perms |= EXEC;
print(perms);                     // 5
print(1 << 10);                   // 1024
print(-16 >> 2);                  // -4
```

```rune
print(1.5 & 1);     // TypeError: Operands must be integers.
```

## Where Integers Come From

Lengths, indices, `int`, ranges of integers and JSON numbers without a
fraction or an exponent are integers; `math.floor`, `ceil` and `round` give
integers when one holds the result.

```rune
print(math.isInt("abc".len()));              // true
print(math.isInt(math.floor(3.7)));         // true
print(math.isInt(json.parse("2")));         // true
print(math.isInt(json.parse("2.0")));       // false
print(int(-3.9));                           // -3
```
