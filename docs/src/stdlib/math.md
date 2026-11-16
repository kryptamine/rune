# math

| Function | Gives |
|---|---|
| `math.floor(x)`, `math.ceil(x)`, `math.round(x)` | `x` rounded down, up, to nearest |
| `math.idiv(a, b)` | integer division, toward zero |
| `math.abs(x)` | absolute value |
| `math.sqrt(x)`, `math.pow(x, y)` | square root, `x` to the power `y` |
| `math.min(a, ...)`, `math.max(a, ...)` | smallest, largest argument |
| `math.isInt(x)` | whether `x` is an integer |
| `math.random()` | a random float in `[0, 1)` |

Constants: `math.PI`, `math.INF`, `math.NAN`, `math.MAX_INT`,
`math.MIN_INT`.

Rune numbers are either 64-bit integers or floats; see
[Numbers](../language/numbers.md).

## Rounding

`floor`, `ceil` and `round` give an integer whenever one can hold the
result. `round` takes halves away from zero.

`/` always divides exactly, so `7 / 2` is `3.5`. `math.idiv(a, b)` is
integer division, cutting toward zero, and `%` is its remainder.

```rune
print(math.floor(3.7));    // 3
print(math.floor(-3.2));   // -4
print(math.ceil(3.2));     // 4
print(math.round(2.5));    // 3
print(math.round(-2.5));   // -3

print(7 / 2);              // 3.5
print(math.idiv(7, 2));    // 3
print(math.idiv(-7, 2));   // -3
print(7 % 2);              // 1
```

Rounding to cents, and splitting minutes into hours:

```rune
let price = 19.987;
print(math.round(price * 100) / 100);   // 19.99

let minutes = 135;
print("{math.idiv(minutes, 60)}h {minutes % 60}m");   // 2h 15m
```

Dividing an integer by zero has no answer:

```rune
math.idiv(7, 0); // ValueError: math.idiv() integer division by zero.
```

## Arithmetic

`min` and `max` take any number of arguments and return one of them as it
is. To use them on an array, spread it: `math.max(...xs)`.

```rune
print(math.abs(-7));            // 7
print(math.sqrt(16));           // 4
print(math.sqrt(2));            // 1.4142135623730951
print(math.pow(2, 10));         // 1024
print(math.pow(2, -1));         // 0.5
print(math.min(3, 1, 2));       // 1
print(math.max(1, 2.5));        // 2.5

let temps = [18, 23, 21, 25, 19];
print(math.max(...temps) - math.min(...temps));   // 7
```

`isInt(x)` asks whether `x` is stored as an integer. Division makes a
float even when it comes out even:

```rune
print(math.isInt(7));       // true
print(math.isInt(7.5));     // false
print(math.isInt(6 / 2));   // false
print(math.isInt(math.idiv(6, 2)));   // true
```

## Random numbers

`random()` is a float from 0 up to, but not including, 1. Scale and floor
it for a whole number in a range:

```rune
fun roll(sides) { math.floor(math.random() * sides) + 1 }

let die = roll(6);
print(die >= 1 && die <= 6);   // true

let colors = ["red", "green", "blue"];
let pick = colors[math.floor(math.random() * colors.len())];
print(pick in colors);         // true
```

## Constants

```rune
print(math.PI);        // 3.141592653589793
print(math.INF);       // inf
print(-math.INF);      // -inf
print(math.NAN);       // nan
print(math.MAX_INT);   // 9223372036854775807
print(math.MIN_INT);   // -9223372036854775808

let r = 2;
print(math.PI * r * r);   // 12.566370614359172
```
