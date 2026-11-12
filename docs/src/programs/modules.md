# Modules

A program can be split into files. Each file has its own top level: names
it declares are private unless marked `export` (`fun`, `let`, `struct`),
and another file reaches them only through the module, as `m.name`.

A small shop, in three files:

```
shop/
├── main.rn
├── data/prices.txt
└── lib/
    ├── cart.rn
    └── money.rn
```

```rune
// lib/money.rn
export struct Money { cents, currency = "USD" }
export struct CurrencyError { message }

export fun format(m) {
    return "{m.cents / 100} {m.currency}";
}

export fun add(a, b) {
    check(a, b);
    return Money { cents: a.cents + b.cents, currency: a.currency };
}

fun check(a, b) {
    if a.currency != b.currency throw CurrencyError { message: "{a.currency} vs {b.currency}" };
}
```

```rune
// lib/cart.rn
import .money;

export let mut items = [];

export fun add(name, cents) {
    items.push({ name, price: money.Money { cents } });
}

export fun total() {
    return items.reduce((sum, it) => money.add(sum, it.price), money.Money { cents: 0 });
}
```

```rune
// main.rn
import lib.cart;
import lib.money as m;

cart.add("book", 1250);
cart.add("pen", 199);

print(cart.items.len());        // 2
print(m.format(cart.total())); // 14.49 USD

let eur = m.Money { cents: 500, currency: "EUR" };
print(eur is m.Money);         // true

try {
    m.add(cart.total(), eur);
} catch e is m.CurrencyError {
    print("cannot add: " + e.message); // cannot add: USD vs EUR
}
```

`./rune run shop/main.rn` runs it from any directory. Two modules may
declare the same name: `cart.add` and `money.add` do not clash.

## Import Paths

`import a.b;` loads `a/b.rn`, named from the directory of the file passed
to `rune`, and binds it as `b`; `import a.b as m;` binds it as `m`.

Leading dots make it relative to the importing file: `import .money;` is
`money.rn` next to it, `import ..c.d;` is `c/d.rn` in its parent directory,
one more dot per level up. The program's directory cannot be left:

```rune
import ...shared.util; // Error at '...': Module '...shared.util' is outside the program's directory.
```

Imports come first in a file:

```rune
print("start");
import lib.money; // Error at 'import': Imports must come before other statements.
```

## module.dir and module.file

`module.dir` is the directory of the file it is in, and `module.file` its
path, both absolute. A program uses them to find the files it comes with
wherever it is run from:

```rune
// main.rn
let prices = fs.read(fs.join(module.dir, "data", "prices.txt"));
print(prices.trim()); // book 12.50
```

`module` is a namespace of each file's own: in `lib/cart.rn`,
`module.dir` is the `lib` directory.

## What a Module Allows

Members are bound when the program is compiled, so a mistake is caught
before anything runs. A member the module does not export is an error:

```rune
import lib.money;
money.check(1, 2); // Error at 'check': Module 'money' has no export 'check'.
```

A module is not a value:

```rune
import lib.money;
print(money); // Error at 'money': Module 'money' is not a value; use its members, as in money.name.
```

A module's variables cannot be assigned from outside:

```rune
import lib.cart;
cart.items = []; // Error at 'items': Cannot assign to a member of module 'cart'.
```

The array or object a `let mut` export holds may be changed, as
`cart.items.push(...)` would; one held by an immutable `let` may not:

```rune
// lib/consts.rn
export let tags = ["sale"];
```

```rune
import lib.consts;
consts.tags.push("new"); // Error at 'tags': Cannot mutate immutable variable 'consts.tags'.
```

Importing a module shadows a built-in namespace of the same name in that
file, so `import lib.math;` hides `math.floor`. Import it `as` another name
to keep both:

```rune
// lib/math.rn
export fun floor(x) { return "mine"; }
```

```rune
import lib.math as mymath;

print(mymath.floor(1.5)); // mine
print(math.floor(1.5));   // 1
```

## Running Order

Each file runs once, after the files it imports. An import cycle is an
error:

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
import lib.a; // Error at '.': Import cycle: lib/a.rn -> lib/b.rn -> lib/a.rn.
```

An imported file may only declare things at its top level (`fun`, `let`,
`struct`, `impl`); statements such as `print(...)` belong in the entry file:

```rune
// lib/noisy.rn
print("loaded"); // Error at 'print': Only declarations can be at the top level of a module.
```
