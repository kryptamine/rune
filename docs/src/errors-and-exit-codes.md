# Errors and Exit Codes

Errors name the file, line and column, show the source with the offending
token underlined, and, when the cause is certain, a hint on how to fix it.
On a terminal they are colored; set `NO_COLOR` to turn that off. A runtime
error also lists the calls it unwound through:

```rune
fun total(order) {
    return order.price * order.qty;
}

let orders = [{ price: 10, qty: 2 }, 5];
print(orders.map(total));
```

```
TypeError: Cannot read property 'price' of number.
 --> main.rn:2:18
  |
1 | fun total(order) {
2 |     return order.price * order.qty;
  |                  ^^^^^

  at total (main.rn:2)
  at array.map (native)
  at <script> (main.rn:6)
```

A hint follows the source:

```rune
let count = 0;
for n in [1, 2, 3] {
    count += n;
}
```

```
error: Cannot assign to immutable variable 'count'.
 --> main.rn:3:5
  |
2 | for n in [1, 2, 3] {
3 |     count += n;
  |     ^^^^^
  |
  = hint: declare it as 'mut count' on line 1
```

The parser recovers from a syntax error and reports all of them in one run:

```rune
let x = (1 + ;
print(x)
```

```
error: Expect expression.
 --> main.rn:1:14
  |
1 | let x = (1 + ;
  |              ^

error: Expect ';' after expression.
 --> main.rn:2:9
  |
1 | let x = (1 + ;
2 | print(x)
  |         ^
```

`run` exits with `65` on a syntax or compile error, `70` on an uncaught
runtime error, and otherwise with `0` or the code [`main`](programs/main.md)
returns.
