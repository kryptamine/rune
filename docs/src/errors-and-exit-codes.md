# Errors and Exit Codes

Errors show the offending source line with a caret under the column.
A runtime error also lists the calls it unwound through:

```rune
fun total(order) {
    return order.price * order.qty;
}

let orders = [{ price: 10, qty: 2 }, 5];
print(orders.map(total));
```

```
[line: 2] TypeError: Cannot read property 'price' of number.
    2 |     return order.price * order.qty;
      |                  ^
  at total (line 2)
  at array.map (native)
  at <script> (line 6)
```

The parser recovers from a syntax error and reports all of them in one run:

```rune
let x = (1 + ;
print(x)
```

```
[line: 1] Error at ';': Expect expression.
    1 | let x = (1 + ;
      |              ^
[line: 2] Error at end: Expect ';' after expression.
```

`run` exits with `65` on a syntax or compile error, `70` on an uncaught
runtime error, and otherwise with `0` or the code [`main`](programs/main.md)
returns.
