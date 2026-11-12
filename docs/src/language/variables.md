# Variables

`let` declares an immutable variable, `let mut` a mutable one. Assigning to an
immutable variable (`=`, `+=`, `++`, ...) is a compile-time error. An immutable
variable must be initialized; `let mut x;` starts as `nil`.

```rune
let name = "ann";
let mut visits = 0;
visits += 1;
visits++;
print("{name}: {visits}");    // ann: 2

let mut lastSeen;
print(lastSeen);              // nil
```

```rune
let name = "ann";
name = "bob";    // Error at 'name': Cannot assign to immutable variable 'name'.
```

```rune
let limit;       // Error at 'limit': Immutable variable must be initialized.
```

## Immutable Arrays and Objects

Through an immutable variable the array, object or struct it holds cannot
be changed either: index and property assignment (`a[0] = 1`, `o.x += 1`,
`a[0]++`, also nested like `o.xs[0] = 1`) and calls of the in-place
methods (`push`, `pop`, `sort`, `Map.set`, `Set.add`, `delete`, `clear`,
...) and of `mut self` methods are compile errors. Taking an in-place
method as a value is a runtime `TypeError`. Methods that return a new value
(`append`, `map`, ...) work as usual.

```rune
let config = { db: { hosts: ["db1"] } };
config.db.hosts[0] = "db2";    // Error at 'config': Cannot mutate immutable variable 'config'.
```

```rune
let scores = [70, 95];
print(scores.append(80));       // [70, 95, 80]
print(scores);                  // [70, 95]
scores.push(80);                // Error at 'scores': Cannot mutate immutable variable 'scores'.
```

```rune
let mut scores = [70, 95];
scores.push(80);
print(scores);                  // [70, 95, 80]
```

## Parameters and Loop Variables

Function parameters, `for`-`in` variables, `catch` variables and functions are
immutable too. Parameters and `for`-`in` variables are made mutable with `mut`:

```rune
fun addTag(mut tags, tag) {
    tags.push(tag);
}

let mut tags = ["new"];
addTag(tags, "sale");
print(tags);    // ["new", "sale"]

let mut grid = [[0, 0], [0, 0]];
for i, mut row in grid { row[i] = 1; }
print(grid);    // [[1, 0], [0, 1]]
```

```rune
fun retry(attempts) {
    attempts -= 1;    // Error at 'attempts': Cannot assign to immutable variable 'attempts'.
}
```

The check is static and follows the variable, not the value: an array held by
an immutable variable still changes when it is mutated through another,
mutable reference (`let mut b = a;` or a `mut` parameter).

```rune
let scores = [70, 95];
let mut alias = scores;
alias.push(80);
print(scores);    // [70, 95, 80]
```

## Scopes

Redeclaring a variable in the same scope is an error, at the top level too.
A new scope may shadow a name with a different mutability.
Built-ins are not declarations, so a program may still redefine them.

```rune
let total = 10;
{
    let mut total = 0;
    total += 5;
    print(total);    // 5
}
print(total);        // 10

let len = 3;         // shadows the built-in len
print(len);          // 3
```

```rune
let total = 10;
let total = 20;    // Error at 'total': Variable with this name already declared in this scope.
```
