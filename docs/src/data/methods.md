# Methods

An `impl` block attaches functions to a [struct](structs.md), so the
behavior lives next to the data it works on. A method whose first
parameter is `self` is called on an instance, and `self` is that
instance. A method without `self` is a static method, called on the
type itself; it is the usual place for constructors.

```rune
struct Money { cents, currency = "USD" }

impl Money {
    fun of(amount, currency = "USD") {
        Money { cents: math.round(amount * 100), currency }
    }

    fun plus(self, other) {
        Money { ...self, cents: self.cents + other.cents }
    }

    fun format(self) {
        "{"%.2f".format(self.cents / 100)} {self.currency}"
    }
}

let price = Money.of(12.5);
let total = price.plus(Money.of(0.99));
print(total.format()); // 13.49 USD
print(price.format()); // 12.50 USD
```

`Money.of` is static, `plus` and `format` are instance methods. Note that
`plus` does not change `price`: it returns a new `Money`.

`impl` blocks are top-level only. They may come before the struct, and a
type may have several. A method cannot share its name with a field or
with another method of the type.

A static method is called only on the type, never on an instance:

```rune
struct Money { cents }

impl Money {
    fun zero() { Money { cents: 0 } }
}

Money.zero().zero(); // TypeError: 'zero' is a static method of Money; call it as Money.zero().
```

## Mutating Methods

A method that takes `mut self` may change the instance: set its fields
(`self.balance += 1`), change what they hold
(`self.items.push(x)`) or replace `self` as a whole. Without `mut`,
`self` is read-only. Mutating methods are the only place a struct's
fields can be set; see [Setting Fields](structs.md#setting-fields).

```rune
struct Account { owner, balance = 0 }

impl Account {
    fun deposit(mut self, amount) {
        self.balance += amount;
    }

    fun withdraw(mut self, amount) {
        if amount > self.balance {
            throw ValueError { message: "insufficient funds" };
        }
        self.balance -= amount;
    }
}

let mut acc = Account { owner: "ann" };
acc.deposit(100);
acc.withdraw(30);
print(acc.balance); // 70
```

The call changes the variable it was made on, as a
[`mut` parameter](../language/functions.md#mut-parameters) changes the
caller's variable. A copy taken before the call keeps the old value:

```rune
struct Cart { items = [] }

impl Cart {
    fun add(mut self, item) { self.items.push(item); }
}

let mut cart = Cart {};
let before = cart;
cart.add("tea");
cart.add("milk");
print(cart.items);   // ["tea", "milk"]
print(before.items); // []
```

If a mutating method throws, whatever it changed before the throw stays
changed. So check first and change after, as `withdraw` does; then a
failed call leaves the instance as it was:

```rune
struct Account { owner, balance = 0 }

impl Account {
    fun withdraw(mut self, amount) {
        if amount > self.balance {
            throw ValueError { message: "insufficient funds" };
        }
        self.balance -= amount;
    }
}

let mut acc = Account { owner: "ann", balance: 50 };

try {
    acc.withdraw(1000);
} catch e {
    print(e.message); // insufficient funds
}
print(acc.balance);   // 50
```

### Where a Mutating Method Can Be Called

The receiver must be something that can be changed: a `let mut`
variable, or an element, key or field reached from one. That includes
structs stored in arrays and maps:

```rune
struct Account { owner, balance = 0 }

impl Account {
    fun deposit(mut self, amount) { self.balance += amount; }
}

let mut accounts = [Account { owner: "ann" }, Account { owner: "bob" }];
accounts[1].deposit(25);
print(accounts[1]); // Account {owner: "bob", balance: 25}

let mut byOwner = ["ann": Account { owner: "ann" }];
byOwner["ann"].deposit(10);
print(byOwner["ann"].balance); // 10
```

On an immutable variable, a struct literal or the result of a call, there
is nothing to write the change back to, so it is a compile error:

```rune
struct Account { owner, balance = 0 }

impl Account {
    fun deposit(mut self, amount) { self.balance += amount; }
}

let acc = Account { owner: "ann" };
acc.deposit(10); // Error at 'acc': Cannot mutate immutable variable 'acc'.
```

```rune
struct Account { owner, balance = 0 }

impl Account {
    fun deposit(mut self, amount) { self.balance += amount; }
}

Account { owner: "ann" }.deposit(10); // Error at 'deposit': Cannot call mutating method 'deposit' of Account: the receiver is not a mutable variable, element or property.
```

A struct held in a field of another struct can be changed only from the
outer struct's own `mut self` methods, as in `self.pos.move(1)`; see
[Setting Fields](structs.md#setting-fields).

## Calling Only

A method taking `self` is only called, on an instance. To pass one
along, wrap the call in an arrow function:

```rune
struct Money { cents }

impl Money {
    fun format(self) { "$%.2f".format(self.cents / 100) }
}

let prices = [Money { cents: 499 }, Money { cents: 1250 }];
print(prices.map(m => m.format())); // ["$4.99", "$12.50"]
```

```rune
struct Money { cents }

impl Money {
    fun format(self) { "$%.2f".format(self.cents / 100) }
}

let show = Money { cents: 499 }.format; // TypeError: Cannot take method 'format' of Money as a value.
```

A static method is a plain function, so `Money.of` may be passed as is.

## Functions in Fields

A struct field can hold a function. Since `x.f(args)` always calls a
method, call a function stored in a field as `(x.f)(args)` or
`x["f"](args)`:

```rune
struct Button { label, onClick }

let save = Button { label: "save", onClick: () => "saved" };
print((save.onClick)());  // saved
print(save["onClick"]()); // saved
```

```rune
struct Button { label, onClick }

let save = Button { label: "save", onClick: () => "saved" };
save.onClick(); // TypeError: Button has no method 'onClick'; call the function in its field as (x.onClick)().
```

For a lookup table of functions by name, use a [map](map-and-set.md):

```rune
let greeters = [
    "en": name => "hello, " + name,
    "es": name => "hola, " + name,
];

print(greeters["es"]("ann")); // hola, ann
```

Functions of a namespace, such as `math.floor(x)` or `fs.join(a, b)`, are
not methods; they are plain functions grouped under a name.

## Methods on Built-in Types

`impl` also works on the built-in types `string`, `number`, `boolean`,
`array`, `function`, `map`, `set`, `bytes` and `stream`. This
lets you add the helpers your program keeps needing:

```rune
impl string {
    fun shout(self) { self.upper() + "!" }
}

impl array {
    fun sum(self) { self.reduce((a, b) => a + b, 0) }
    fun average(self) { self.sum() / self.len() }
}

print("hi".shout());           // HI!
print([3, 4, 5].sum());        // 12
print([3, 4, 5, 8].average()); // 5
```

Methods on built-in types take `self`, never `mut self`, and are visible
to the whole program. If one has the same name as a built-in method, the
program's own wins. Built-in structs, such as `IOError`, cannot get
methods.
