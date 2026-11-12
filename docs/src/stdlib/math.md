# math

## Rounding

`floor(x)`, `ceil(x)` and `round(x)` give integers when one holds the
result; `round` takes halves away from zero. `idiv(a, b)` is integer
division, toward zero.

```rune
print(math.floor(3.7));    // 3
print(math.floor(-3.2));   // -4
print(math.ceil(3.2));     // 4
print(math.round(2.5));    // 3
print(math.round(-2.5));   // -3
print(7 / 2);              // 3.5
print(math.idiv(7, 2));    // 3
print(math.idiv(-7, 2));   // -3
```

## Arithmetic

`abs(x)`, `sqrt(x)`, `pow(x, y)`, `min(a, ...)` and `max(a, ...)`, which
return the argument itself, and `isInt(x)`, whether `x` is an integer.

```rune
print(math.abs(-7));           // 7
print(math.sqrt(16));          // 4
print(math.sqrt(2));           // 1.4142135623730951
print(math.pow(2, 10));        // 1024
print(math.min(3, 1, 2));      // 1
print(math.max(...[4, 9, 2])); // 9
print(math.isInt(7));          // true
print(math.isInt(7.5));        // false
print(math.isInt(6 / 2));      // false
```

## Random numbers

`random()` is a float in `[0, 1)`.

```rune
let die = math.floor(math.random() * 6) + 1;
print(die >= 1 && die <= 6);   // true

let colors = ["red", "green", "blue"];
let pick = colors[math.floor(math.random() * colors.len())];
print(colors.contains(pick));   // true
```

## Constants

```rune
print(math.PI);        // 3.141592653589793
print(math.INF);       // inf
print(-math.INF);      // -inf
print(math.MAX_INT);   // 9223372036854775807
print(math.MIN_INT);   // -9223372036854775808
```
