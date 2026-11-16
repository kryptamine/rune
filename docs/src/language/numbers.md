# Numbers

Rune has one number type, `number`, which holds either a 64-bit integer or
a 64-bit float. Integers stay exact, so counts, ids and money in cents
never pick up rounding errors; floats cover everything with a fraction.
`math.isInt` tells the two apart.

## Literals

A literal without a fraction or an exponent is an integer. `_` may
separate digits for readability, and `0x`, `0o` and `0b` write hex, octal
and binary integers:

```rune
print(1_000_000);         // 1000000
print(0xff);              // 255
print(0o17);              // 15
print(0b1010);            // 10
print(1e3);               // 1000
print(2.5e-3);            // 0.0025
print(math.isInt(42));    // true
print(math.isInt(1e3));   // false
```

`1e3` has an exponent, so it is a float, even though it prints as `1000`.

A decimal literal larger than `math.MAX_INT` is a compile error. A hex,
octal or binary literal may use all 64 bits, read as two's complement, so
`0xFFFF_FFFF_FFFF_FFFF` is `-1`:

```rune
print(0xFFFF_FFFF_FFFF_FFFF);                     // -1
print(0x8000_0000_0000_0000 == math.MIN_INT);     // true
```

```rune
print(99999999999999999999);    // Error: Integer literal is too large.
```

## Arithmetic

`+`, `-`, `*` and `%` on two integers give an integer. If either side is a
float, the result is a float. `/` always gives a float, even when the
division is exact; for integer division use `math.idiv`, which drops the
fraction:

```rune
let price = 19.99;
let qty = 3;
print(qty * 2);                 // 6
print(price * qty);             // 59.97
print(10 / 4);                  // 2.5
print(10 / 5);                  // 2
print(math.isInt(10 / 5));      // false
```

Splitting 17 items into pages of 5 gives 3 full pages, 2 items left over,
and 4 pages in total:

```rune
let items = 17;
let perPage = 5;
print(math.idiv(items, perPage));     // 3
print(items % perPage);               // 2
print(math.ceil(items / perPage));    // 4
```

A float with a whole value prints without a fraction, as `2` above. A very
large or very small float prints with an exponent: `1e+21`, `1e-7`. To
control the digits, use [`format`](../stdlib/string.md):

```rune
print(1 / 3);                    // 0.3333333333333333
print("%.2f".format(1 / 3));     // 0.33
```

`%` takes the sign of its left operand:

```rune
print(7 % 3);     // 1
print(-7 % 3);    // -1
```

## Division by Zero and Overflow

Dividing a float by zero does not fail. It gives `inf`, `-inf` or `nan`,
available as `math.INF`, `-math.INF` and `math.NAN`. `nan` is falsy and
equals nothing, not even itself:

```rune
print(7 / 0);                   // inf
print(-7 / 0);                  // -inf
print(0 / 0);                   // nan
print(0 / 0 == math.NAN);       // false
print(!(0 / 0));                // true
```

Integers never wrap around silently. A result that does not fit in 64 bits,
or an integer `% 0`, is a `ValueError`:

```rune
print(math.MAX_INT);        // 9223372036854775807
print(math.MAX_INT + 1);    // ValueError: Integer overflow.
```

```rune
print(7 % 0);    // ValueError: Integer modulo by zero.
```

## Comparing Integers and Floats

An integer equals the float with the same value, and comparisons are exact
even past the range where floats can hold every integer:

```rune
print(1 == 1.0);                                  // true
print(9007199254740993 == 9007199254740992.0);    // false
```

Floats are binary fractions, so decimal sums can be slightly off. Compare
them with a tolerance, or count money in integer cents:

```rune
print(0.1 + 0.2);                               // 0.30000000000000004
print(0.1 + 0.2 == 0.3);                        // false
print(math.abs(0.1 + 0.2 - 0.3) < 1e-9);        // true
print(10 + 20 == 30);                           // true
```

## Bitwise Operators

| Operator | Meaning |
| --- | --- |
| `a & b` | and |
| `a \| b` | or |
| `a ^ b` | exclusive or |
| `~a` | not |
| `a << n` | shift left |
| `a >> n` | shift right, keeping the sign |
| `a >>> n` | shift right, filling with zeros |

They take integers, or floats with a whole value, and give an integer.
Each has a compound assignment: `&=`, `|=`, `^=`, `<<=`, `>>=`, `>>>=`.

Bit flags are their classic use. Each permission is one bit; `|` combines
them, `&` tests one, and `& ~` clears one:

```rune
const READ = 0b001;
const WRITE = 0b010;
const EXEC = 0b100;

let mut perms = READ | WRITE;
print(perms);                     // 3
print(perms & WRITE == WRITE);    // true

perms &= ~WRITE;
perms |= EXEC;
print(perms);                     // 5
print(perms & WRITE == WRITE);    // false
```

The bitwise operators bind tighter than comparisons, so
`perms & WRITE == WRITE` needs no parentheses. Among themselves, `|` binds
loosest, then `^`, then `&`, then the shifts; all of them bind looser than
`+` and `-`.

Shifts never overflow: bits shifted out are lost, and a shift by 64 or more
gives `0` (or `-1` for `>>` of a negative number). A negative shift count
is a `ValueError`.

```rune
print(1 << 10);       // 1024
print(-16 >> 2);      // -4
print(-16 >>> 60);    // 15
print(1 << 64);       // 0
```

```rune
print(1.5 & 1);    // TypeError: Operands must be integers.
```

## Where Integers Come From

Lengths, indices, ranges of integers, `int(...)`, and JSON numbers without
a fraction or exponent are integers. `math.floor`, `math.ceil` and
`math.round` return integers when the result fits. `int` drops the
fraction, toward zero:

```rune
print(math.isInt("abc".len()));             // true
print(math.isInt(math.floor(3.7)));         // true
print(math.isInt(json.parse("2")));         // true
print(math.isInt(json.parse("2.0")));       // false
print(int(3.9));                            // 3
print(int(-3.9));                           // -3
print(int("42") + 1);                       // 43
```
