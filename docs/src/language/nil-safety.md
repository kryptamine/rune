# Nil Safety

Data from the outside world is often incomplete: a user without a
profile, a config without a port, a JSON response without the field you
expected. Rune gives you three tools for that: `?.` reads through a
missing value, `??` supplies a default, and `return`, `break` or
`continue` can bail out right where a value is missing.

## Optional Chaining

Reading a property of `nil` is an error. With `?.` instead of `.`, the
read gives `nil` when the left side is `nil`, and the rest of the chain
is skipped:

```rune
let user = { name: "ann", profile: nil };

print(user.profile?.city);            // nil
print(user.profile?.address.city);    // nil
```

The second line shows the skip: once `user.profile` is `nil`, neither
`.address` nor `.city` is read, so one `?.` guards the whole chain after
it.

`?.` also reads properties, fields and elements that are missing, instead
of failing. `?.[i]` is the indexing form:

```rune
let users = [{ name: "ann" }];

print(users?.[0]?.name);    // ann
print(users?.[5]?.name);    // nil
print(users[0]?.email);     // nil
```

Without `?.` the same read is an error:

```rune
let user = { name: "ann" };
print(user.email); // KeyError: Undefined property 'email'.
```

`f?.(args)` calls `f` only if it is not `nil`. It suits optional
callbacks:

```rune
let onDone = nil;
let notify = msg => "sent " + msg;

print(onDone?.("ok"));    // nil
print(notify?.("ok"));    // sent ok
```

`?.` only forgives `nil` and missing keys. Reading a property of a value
that has no properties at all, such as a number, is still a `TypeError`:

```rune
print(5?.x); // TypeError: Cannot read property 'x' of number.
```

A chain with `?.` reads; it cannot be assigned to:

```rune
let mut user = { profile: nil };
user?.profile = {}; // Error at '=': Invalid assignment target.
```

## Default Values

`a ?? b` gives `a`, or `b` when `a` is `nil`. Together with `?.` it reads
a deeply nested value with a fallback in one line:

```rune
let resp = json.parse(r"""{"data": {"items": [1, 2]}}""");
let empty = json.parse("\{}");

print(resp?.data?.items ?? []);     // [1, 2]
print(empty?.data?.items ?? []);    // []
```

Unlike `||`, `??` keeps `false`, `0` and `""`. Only `nil` is replaced, so
a setting that is explicitly off stays off:

```rune
let opts = { retries: 0, verbose: false };

print(opts.retries ?? 3);       // 0
print(opts.retries || 3);       // 3
print(opts.verbose ?? true);    // false
print(opts?.timeout ?? 30);     // 30
```

The right side runs only when it is needed, so an expensive default
costs nothing when there is a value. `??` also chains, taking the first
value that is not `nil`:

```rune
fun loadDefault() {
    print("loading default");
    1
}

print(5 ?? loadDefault());               // 5
print(nil ?? nil ?? "last resort");      // last resort
```

`??` binds looser than `||`, so `a ?? b || c` means `a ?? (b || c)`.

## Leaving Early

`return`, `break` and `continue` are expressions, like
[`throw`](error-handling.md#throw). That lets a missing value end the
function or skip the iteration right where it is read, without a
separate `if`:

```rune
fun port(cfg) {
    let p = cfg?.port ?? return 80;
    int(p)
}

print(port({ port: "8080" }));    // 8080
print(port(nil));                 // 80
```

In a loop, `?? continue` skips the rows that lack a value:

```rune
let rows = [{ id: 1 }, {}, { id: 3 }];
let mut ids = [];
for row in rows {
    ids.push(row?.id ?? continue);
}
print(ids);    // [1, 3]
```

They also work as `match` arms:

```rune
fun describe(n) {
    let size = match n {
        0 => return "nothing",
        1..10 => "a few",
        _ => "many",
    };
    "{size} items"
}

print(describe(0));     // nothing
print(describe(4));     // a few items
print(describe(40));    // many items
```

`return` takes a value unless `;`, `,`, `:` or a closing bracket follows
it, so `?? return` alone returns `nil`.
