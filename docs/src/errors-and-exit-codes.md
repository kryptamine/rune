# Errors and Exit Codes

When something goes wrong, Rune tells you where and why. Every error names
the file, line and column, shows the surrounding source with the offending
token underlined, and, when the cause is certain, adds a hint on how to
fix it. On a terminal the report is colored; set `NO_COLOR` to turn that
off.

Rune finds errors at two moments: when it compiles the program, before any
of it runs, and while it runs.

## Compile Errors

A compile error stops the program before its first line runs. Assigning to
a variable declared without `mut` is one:

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

After a syntax error the parser carries on, so one run lists all of them:

```rune
let price = (19.99 * ;
print(price)
```

```
error: Expect expression.
 --> main.rn:1:22
  |
1 | let price = (19.99 * ;
  |                      ^

error: Expect ';' after expression.
 --> main.rn:2:13
  |
1 | let price = (19.99 * ;
2 | print(price)
  |             ^
```

## Runtime Errors

A runtime error happens while the program runs, for example reading a
property of a number. Unless a [`try`](language/error-handling.md) catches
it, the program stops, and the report also lists the calls the error
passed through, innermost first:

```rune
struct Order { price, qty }

fun total(order) {
    order.price * order.qty
}

let orders = [Order { price: 10, qty: 2 }, 5];
print(orders.map(total));
```

```
TypeError: Cannot read property 'price' of number.
 --> main.rn:4:11
  |
3 | fun total(order) {
4 |     order.price * order.qty
  |           ^^^^^

  at total (main.rn:4)
  at array.map (native)
  at <script> (main.rn:8)
```

The first word, here `TypeError`, is the error's type. [Error
Handling](language/error-handling.md) lists the types and shows how to
catch them by type.

## Exit Codes

`rune run` tells the shell how the program ended:

| Code | Meaning |
| --- | --- |
| `0` | The program finished. |
| `65` | A syntax or compile error; nothing ran. |
| `70` | An uncaught runtime error. |
| any other | The number [`main`](programs/main.md) returned. |

So a program can report its own failure by returning a code from `main`:

```rune
// main.rn
fun main(args) {
    if !args {
        print("usage: rune run main.rn <file>");
        return 2;
    }
    print("processing {args[0]}");
}
```

```sh
$ rune run main.rn; echo $?
usage: rune run main.rn <file>
2
```
