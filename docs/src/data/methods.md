# Methods

An `impl` block adds methods to a struct declared in the same file. A
method taking `self` first is called on an instance, `self` being the
instance; one without is static and called on the type.

```rune
struct Money { cents, currency = "USD" }

impl Money {
  fun of(amount, currency = "USD") {
    return Money { cents: math.round(amount * 100), currency };
  }
  fun plus(self, other) {
    return Money { ...self, cents: self.cents + other.cents };
  }
  fun format(self) {
    return "%.2f %s".format(self.cents / 100, self.currency);
  }
}

let price = Money.of(12.5);
let total = price.plus(Money.of(0.99));
print total.format(); // 13.49 USD
print price.format(); // 12.50 USD
```

Impl blocks are top-level only, may come before the struct and there may be
several; a method cannot share a name with a field or another method of the
type. A static method is not reachable from an instance:

```rune
struct Money { cents }
impl Money {
  fun zero() { return Money { cents: 0 }; }
}
Money.zero().zero(); // TypeError: 'zero' is a static method of Money; call it as Money.zero().
```

## Mutating methods

`mut self` makes a mutating method, the only code that can set fields:
those of `self` (`self.x += 1`, `self["x"] = 1`), or `self` as a whole.
Without `mut`, `self` is immutable.

```rune
struct Account { owner, balance = 0 }

impl Account {
  fun deposit(mut self, amount) {
    self.balance += amount;
  }
  fun withdraw(mut self, amount) {
    self.balance -= amount;
    if self.balance < 0 throw ValueError { message: "insufficient funds" };
  }
}

let mut acc = Account { owner: "ann" };
acc.deposit(100);
acc.withdraw(30);
print acc.balance; // 70
```

Structs are values, so the method works on a copy, which replaces the
receiver when it returns; nothing else holding the old struct sees its
fields change, and if the method throws, the receiver keeps its old value.

```rune
struct Account { owner, balance = 0 }
impl Account {
  fun deposit(mut self, amount) { self.balance += amount; }
  fun withdraw(mut self, amount) {
    self.balance -= amount;
    if self.balance < 0 throw ValueError { message: "insufficient funds" };
  }
}

let mut acc = Account { owner: "ann", balance: 50 };
let before = acc;
acc.deposit(10);
print acc.balance;    // 60
print before.balance; // 50

try {
    acc.withdraw(1000);
} catch (e) {
    print e.message;  // insufficient funds
}
print acc.balance;    // 60
```

The copy is shallow: an array, object, map or set in a field is shared
with the old struct, so changing it in place (`self.items.push(x)`) shows
through both. Assign a new one (`self.items = self.items.append(x)`) to
keep the old struct as it was.

```rune
struct Cart { items = [] }
impl Cart {
  fun add(mut self, x) { self.items.push(x); }
  fun with(mut self, x) { self.items = self.items.append(x); }
}

let mut cart = Cart {};
let before = cart;
cart.add("tea");
print before.items; // ["tea"]

let snapshot = cart;
cart.with("milk");
print snapshot.items; // ["tea"]
print cart.items;     // ["tea", "milk"]
```

The receiver must be a mutable variable, or an element or property of an
array or object one holds; a struct's field changes only through that
struct's own `mut self` method.

```rune
struct Account { owner, balance = 0 }
impl Account {
  fun deposit(mut self, amount) { self.balance += amount; }
}

let mut accounts = [Account { owner: "ann" }, Account { owner: "bob" }];
accounts[1].deposit(25);
print accounts[1]; // Account {owner: "bob", balance: 25}
```

```rune
struct Account { owner, balance = 0 }
impl Account {
  fun deposit(mut self, amount) { self.balance += amount; }
}

let acc = Account { owner: "ann" };
acc.deposit(10); // TypeError: Cannot call mutating method 'deposit' of Account: the receiver is not a mutable variable, element or property.
```

## Methods as values

`acc.format` without a call is the method bound to `acc`, a function taking
the rest of the arguments; `Money.format` is the plain function, taking the
instance first, so `prices.map(Money.format)` works. A mutating method
cannot be taken as a value.

```rune
struct Money { cents }
impl Money {
  fun format(self) { return "$%.2f".format(self.cents / 100); }
}

let prices = [Money { cents: 499 }, Money { cents: 1250 }];
print prices.map(Money.format); // ["$4.99", "$12.50"]

let show = prices[0].format;
print show(); // $4.99
```

## Methods on built-in types

`impl` also takes `string`, `number`, `boolean`, `array`, `object`,
`function`, `Map`, `Set` and `bytes`. Their methods must take `self`, not
`mut self`, and are seen by the whole program; an object's own property of
the same name wins, and a method of the program over a built-in one.
Built-in structs, such as `IOError`, take no methods.

```rune
impl string {
  fun shout(self) { return self.upper() + "!"; }
}

impl array {
  fun sum(self) { return self.reduce((a, b) => a + b, 0); }
}

print "hi".shout();      // HI!
print [3, 4, 5].sum();   // 12
```
