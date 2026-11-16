# Modules

As a program grows, split it into files. Every `.rn` file is a module
with its own top level: the names it declares are private to it unless
marked `export`. Another file loads it with `import` and reaches its
exports through the module's name, as `money.format(...)`.

## A Small Shop

Three files: money handling, a cart that uses it, and the entry file.

```
shop/
├── main.rn
└── lib/
    ├── cart.rn
    └── money.rn
```

`lib/money.rn` exports a struct, an error type and two functions. `check`
has no `export`, so it stays an implementation detail:

```rune
// lib/money.rn
export struct Money { cents, currency = "USD" }
export struct CurrencyError { message }

export fun format(m) {
    "{m.cents / 100} {m.currency}"
}

export fun add(a, b) {
    check(a, b);
    Money { cents: a.cents + b.cents, currency: a.currency }
}

fun check(a, b) {
    if a.currency != b.currency {
        throw CurrencyError { message: "{a.currency} vs {b.currency}" };
    }
}
```

`lib/cart.rn` imports its neighbour with a leading dot and keeps the items
in a module-level variable:

```rune
// lib/cart.rn
import .money;

export let mut items = [];

export fun add(name, cents) {
    items.push({ name, price: money.Money { cents } });
}

export fun total() {
    items.reduce((sum, it) => money.add(sum, it.price), money.Money { cents: 0 })
}
```

The entry file imports both. `as m` binds `lib/money.rn` under a shorter
name:

```rune
// main.rn
import lib.cart;
import lib.money as m;

cart.add("book", 1250);
cart.add("pen", 199);

print(cart.items.len());          // 2
print(m.format(cart.total()));    // 14.49 USD

let eur = m.Money { cents: 500, currency: "EUR" };
print(eur is m.Money);            // true

try {
    m.add(cart.total(), eur);
} catch e is m.CurrencyError {
    print("cannot add: " + e.message);    // cannot add: USD vs EUR
}
```

```sh
$ rune run shop/main.rn
2
14.49 USD
true
cannot add: USD vs EUR
```

It runs the same from any directory. Each module has its own names, so
`cart.add` and `money.add` do not clash.

## Exports

`export` goes in front of a `fun`, `let`, `const`, `struct` or `enum`.
Everything else in a module is visible only inside it.

## Import Paths

`import a.b;` loads `a/b.rn` from the directory of the entry file, the one
passed to `rune run`, and binds it as `b`. `import a.b as m;` binds it as
`m` instead.

A leading dot makes the path relative to the importing file:

| Import | Loads |
|---|---|
| `import lib.money;` | `lib/money.rn` under the entry file's directory |
| `import .money;` | `money.rn` next to the importing file |
| `import ..util.text;` | `util/text.rn` one directory up |

Each extra dot goes up one more level, but never above the entry file's
directory:

```rune
import ...shared.util; // Error at '...': Module '...shared.util' is outside the program's directory.
```

Imports come first in a file:

```rune
print("start");
import lib.money; // Error at 'import': Imports must come before other statements.
```

## Finding Files: module.dir and module.file

`module.dir` is the absolute path of the directory holding the current
file, and `module.file` the file itself. Use them to find data files that
ship with the program, wherever it is run from:

```
shop/
├── main.rn
└── data/prices.txt
```

```rune
// main.rn
let prices = fs.readText(fs.join(module.dir, "data", "prices.txt"));
print(prices.trim()); // book 12.50
```

Each file has its own `module`: inside `lib/cart.rn`, `module.dir` is the
`lib` directory.

## Mistakes Caught Early

Module members are resolved when the program is compiled, so these
mistakes are reported before anything runs. With the shop above:

Using a name the module does not export:

```rune
import lib.money;
money.check(1, 2); // Error at 'check': Module 'money' has no export 'check'.
```

Using a module as a value. Like a [namespace](../stdlib/builtins.md#namespaces),
a module is only a way to reach its members:

```rune
import lib.money;
print(money); // Error at 'money': Module 'money' is not a value; use its members, as in money.name.
```

Assigning to a module's variable from outside:

```rune
import lib.cart;
cart.items = []; // Error at 'items': Cannot assign to a member of module 'cart'.
```

The value held by an exported `let mut` may be changed from outside, as
in `cart.items.push(...)`; a value held by a plain `let` may not:

```rune
// lib/consts.rn
export let tags = ["sale"];
```

```rune
import lib.consts;
consts.tags.push("new"); // Error at 'tags': Cannot mutate immutable variable 'consts.tags'.
```

## Shadowing a Built-in Namespace

An import named like a built-in namespace hides it in that file:
`import lib.math;` makes `math.floor` refer to the module. Import it under
another name to keep both:

```rune
// lib/math.rn
export fun floor(x) { "mine" }
```

```rune
import lib.math as mymath;

print(mymath.floor(1.5)); // mine
print(math.floor(1.5));   // 1
```

## Running Order

Each file runs once, after the files it imports. So a module's top-level
variables are ready by the time the importing file uses them.

An imported file may only declare things at its top level: `fun`, `let`,
`const`, `struct`, `enum`, `impl`. Statements such as `print(...)` belong
in the entry file:

```rune
// lib/noisy.rn
print("loaded"); // Error at 'print': Only declarations can be at the top level of a module.
```

Two modules may not import each other, directly or through others:

```rune
// lib/a.rn
import .b;
export let x = 1;
```

```rune
// lib/b.rn
import .a;
export let y = 1;
```

```rune
// main.rn
import lib.a; // Error at '.': Import cycle: lib/a.rn -> lib/b.rn -> lib/a.rn.
```

To break a cycle, move what both need into a third module that both
import.
