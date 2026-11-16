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
print(total.format()); // 13.49 USD
print(price.format()); // 12.50 USD
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

`mut self` makes a mutating method, which can set the fields of `self`
(`self.x += 1`, `self["x"] = 1`), or `self` as a whole, as a mutable
variable can. Without `mut`, `self` is immutable. These methods are the
only place a struct's fields are set; see
[Setting fields](structs.md#setting-fields).

```rune
struct Account { owner, balance = 0 }

impl Account {
  fun deposit(mut self, amount) {
    self.balance += amount;
  }
  fun withdraw(mut self, amount) {
    if amount > self.balance { throw ValueError { message: "insufficient funds" }; }
    self.balance -= amount;
  }
}

let mut acc = Account { owner: "ann" };
acc.deposit(100);
acc.withdraw(30);
print(acc.balance); // 70
```

`self` is the receiver itself, as a
[`mut` parameter](../language/functions.md#mut-parameters) is the
caller's variable: the method changes it in place, and it goes back where
the receiver was read from when the method returns or throws. A copy
taken before the call keeps its value.

```rune
struct Account { owner, balance = 0 }
impl Account {
  fun deposit(mut self, amount) { self.balance += amount; }
  fun withdraw(mut self, amount) {
    if amount > self.balance { throw ValueError { message: "insufficient funds" }; }
    self.balance -= amount;
  }
}

let mut acc = Account { owner: "ann", balance: 50 };
let before = acc;
acc.deposit(10);
print(acc.balance);    // 60
print(before.balance); // 50

try {
    acc.withdraw(1000);
} catch e {
    print(e.message);  // insufficient funds
}
print(acc.balance);    // 60
```

A method that throws keeps what it changed before the throw, so check
first and change after, as `withdraw` does.

The fields' containers are part of the struct too:

```rune
struct Cart { items = [] }
impl Cart {
  fun add(mut self, x) { self.items.push(x); }
}

let mut cart = Cart {};
let before = cart;
cart.add("tea");
print(before.items); // []
print(cart.items);   // ["tea"]
```

The receiver must be a mutable variable, or an element, property or
field of a value one holds; through an immutable variable, or on a
struct literal or conversion, the call is a compile error. A struct in a
field of another struct is such a receiver only in the outer struct's
own `mut self` methods, as `self.pos.move(1)`; see
[Setting fields](structs.md#setting-fields). A method called as a value,
as `Account.deposit`, takes no receiver to write back, so calling it is
a `TypeError`. On another
value it is a runtime error, or a compile error when no method of that
name but a mutating one exists.

```rune
struct Account { owner, balance = 0 }
impl Account {
  fun deposit(mut self, amount) { self.balance += amount; }
}

let mut accounts = [Account { owner: "ann" }, Account { owner: "bob" }];
accounts[1].deposit(25);
print(accounts[1]); // Account {owner: "bob", balance: 25}
```

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
print(prices.map(Money.format)); // ["$4.99", "$12.50"]

let show = prices[0].format;
print(show()); // $4.99
```

## Functions in fields

`x.f(args)` always calls a method. A function held in a field is called
as `(x.f)(args)`, or `x["f"](args)`:

```rune
struct Button { label, onClick }
let save = Button { label: "save", onClick: () => "saved" };
print((save.onClick)()); // saved
print(save["onClick"]()); // saved
save.onClick();          // TypeError: Button has no method 'onClick'; call the function in its field as (x.onClick)().
```

An object holds data only: a function as the value of one of its
properties is an error. Behavior belongs to a struct and its `impl`; a
table of functions by name is a [map](map-and-set.md):

```rune
let handlers = { greet: name => "hi " + name }; // Error at 'greet': Property 'greet' of an object cannot hold a function.

let greeters = map([["greet", name => "hi " + name]]);
print(greeters["greet"]("ann")); // hi ann
```

Members of a namespace, as in `math.floor(x)`, are no methods and are
called as they are.

## Methods on built-in types

`impl` also takes the built-in types `string`, `number`, `boolean`,
`array`, `object`, `function`, `map`, `set`, `bytes` and `stream`. Their methods must take `self`, not
`mut self`, and are seen by the whole program; a method of the program
wins over a built-in one.
Built-in structs, such as `IOError`, take no methods.

```rune
impl string {
  fun shout(self) { return self.upper() + "!"; }
}

impl array {
  fun sum(self) { return self.reduce((a, b) => a + b, 0); }
}

print("hi".shout());      // HI!
print([3, 4, 5].sum());   // 12
```
