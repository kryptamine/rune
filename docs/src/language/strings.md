# Strings

A string is immutable Unicode text. Its methods, such as `split`, `upper`
and `replace`, are listed in [String Methods](../stdlib/string.md); this
page covers the syntax.

## Joining and Interpolation

`+` joins two strings. It does not convert other values, so the usual way
to build a string is interpolation: an expression in `{}` inside a string
literal is evaluated and inserted.

```rune
let user = "ann";
let ticket = 42;
print("hello, " + user);              // hello, ann
print("ticket #{ticket} for {user}"); // ticket #42 for ann
```

```rune
let ticket = 42;
print("ticket #" + ticket);    // TypeError: Operands must be two numbers, two strings or two bytes.
```

Any expression works inside `{}`: arithmetic, field access, method
calls, even an [`if` expression](conditionals.md#if-as-an-expression). The
value is shown the way [`print`](values.md#printing) shows it.

```rune
struct Order { item, qty, price, tags }

let order = Order { item: "tea", qty: 3, price: 2.5, tags: ["gift"] };

print("{order.qty} x {order.item}");                         // 3 x tea
print("total: {order.qty * order.price}");                   // total: 7.5
print("tags: {order.tags}");                                 // tags: ["gift"]
print("{order.qty} item{if order.qty == 1 { "" } else { "s" }}");    // 3 items
print("price: {"%.2f".format(order.price)}");                // price: 2.50
```

`string(x)` converts a single value to a string, as interpolation would:

```rune
print(string(42).len());    // 2
```

## Characters and Slices

A string is indexed by character, not by byte, so text with accents or
emoji works as expected. A negative index counts from the end, and
`s[a..b]` [slices](../data/arrays.md) out a substring. `s.len()` counts
characters.

```rune
let word = "héllo";
print(word[1]);        // é
print(word[-1]);       // o
print(word[1..3]);     // él
print(word[..-1]);     // héll
print(word.len());     // 5
```

`sub in s` tests for a substring. `<`, `<=`, `>` and `>=` compare strings
character by character, by code point, so every uppercase ASCII letter
sorts before every lowercase one:

```rune
print("ll" in "héllo");    // true
print("ann" < "bob");      // true
print("Zed" < "ann");      // true
```

Strings never change in place; build a new one instead:

```rune
let mut name = "ann";
name[0] = "A";    // TypeError: Strings are immutable.
```

```rune
let name = "ann";
let title = name[0].upper() + name[1..];
print(title);    // Ann
```

## Escapes

Inside `"..."` a backslash starts an escape:

| Escape | Character |
| --- | --- |
| `\n` | newline |
| `\t` | tab |
| `\r` | carriage return |
| `\"` | `"` |
| `\\` | `\` |
| `\{`, `\}` | a literal brace, not interpolation |
| `\u{e9}` | the code point U+00E9, here `é` |

```rune
print("say \"hi\"");               // say "hi"
print("C:\\temp");                 // C:\temp
print("caf\u{e9} \u{2713}");       // café ✓
print("set \{name} in the form");  // set {name} in the form
print("shipped\npending");
// shipped
// pending
```

## Raw Strings

A raw string, `r"..."`, is taken exactly as written: no escapes and no
interpolation. It is the natural choice for regular expressions and
Windows paths, which are full of backslashes:

```rune
print(r"C:\temp\new");                              // C:\temp\new
print(regex.test(r"^\d{3}-\d{4}$", "555-1234"));    // true
```

A raw string cannot contain `"`. Use `r"""..."""` for that, or write `\x22`
inside a regular expression.

## Multi-line Strings

`"""..."""` can span several lines and contain `"` without escaping.
Escapes and interpolation work as in `"..."`, and `r"""..."""` is the raw
form.

```rune
let user = "ann";
print("""she said "hi" to {user}""");    // she said "hi" to ann
```

Multi-line strings are written indented with the code around them. When
the opening `"""` ends its line and the closing `"""` starts its own, Rune
drops those two line breaks and removes the closing quotes' indentation
from every line. What is indented further stays indented:

```rune
let title = "Orders";
let page = """
    <h1 class="title">{title}</h1>
      <p>indented by two</p>
    """;
print(page);
// <h1 class="title">Orders</h1>
//   <p>indented by two</p>
```

A raw multi-line string is the easiest way to embed JSON, which is full of
`"`, `{` and `}`:

```rune
let body = r"""{"id": 7, "tags": ["new"]}""";
print(json.parse(body)["id"]);    // 7
```

In a plain string, the `{` of JSON would start interpolation, so it must be
written `\{`:

```rune
print(json.parse("\{\"ok\": true}")["ok"]);    // true
```

A line indented less than the closing quotes is an error, unless it is
blank:

```rune
let text = """
    shipped
  pending
    """;    // Error: Insufficient indentation in multi-line string.
```
