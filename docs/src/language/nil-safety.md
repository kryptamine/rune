# Nil Safety

## Optional Chaining

`a?.b`, `a?.[i]` and `a?.(args)` are `nil` when `a` is, and the rest of the
chain is skipped: `a?.b.c` is `nil` for a `nil` `a`, without reading `.c`.

```rune
let user = { name: "ann", profile: nil };

print user.profile?.city;         // nil
print user.profile?.address.city; // nil
```

A missing property, field or element read with `?.` is `nil` too, so
`o?.name` never fails on an object or struct:

```rune
let users = [{ name: "ann" }];

print users?.[0]?.name; // ann
print users?.[5]?.name; // nil
print users[0]?.email;  // nil
```

Without `?.` the same read is an error:

```rune
let user = { name: "ann" };
print user.email; // KeyError: Undefined property 'email'.
```

`f?.(args)` calls `f` only if it is not `nil`, which suits optional callbacks:

```rune
let onDone = nil;
let notify = msg => "sent " + msg;

print onDone?.("ok"); // nil
print notify?.("ok"); // sent ok
```

A value that has no properties at all is still a `TypeError`:

```rune
print 5?.x; // TypeError: Cannot read property 'x' of number.
```

A chain with `?.` cannot be assigned to:

```rune
let mut user = { profile: nil };
user?.profile = {}; // Error at '=': Invalid assignment target.
```

## Default Values

`a ?? b` is `a` unless it is `nil`, then `b`, which runs only then.

```rune
let resp = json.parse(r"""{"data": {"items": [1, 2]}}""");
let empty = json.parse("\{}");

print resp?.data?.items ?? [];  // [1, 2]
print empty?.data?.items ?? []; // []
```

Unlike `or`, it keeps `false`, `0` and `""`, so a setting that is
explicitly off stays off:

```rune
let opts = { retries: 0, verbose: false };

print opts.retries ?? 3;      // 0
print opts.retries or 3;      // 3
print opts.verbose ?? true;   // false
print opts?.timeout ?? 30;    // 30
```

The right side is evaluated only when needed, and `??` chains:

```rune
fun loadDefault() { print "loading"; return 1; }

print 5 ?? loadDefault();    // 5
print nil ?? nil ?? "last";  // last
```

`??` binds looser than `or` and tighter than ranges and `? :`.
