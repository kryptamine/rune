# Strings

Strings are immutable UTF-8 text. `+` joins two strings; anything else
must be converted first, with `str` or interpolation. Methods such as
`split` and `upper` are in [String Methods](../stdlib/string.md).

```rune
let ticket = 42;
print("ticket #" + str(ticket));    // ticket #42
print("ticket #" + ticket);         // TypeError: Operands must be two numbers, two strings or two bytes.
```

A string indexes by character, a negative index from the end, `len`
counts characters, and `<`, `<=`, `>`, `>=` compare strings by their
characters' code points:

```rune
print("héllo"[1]);       // é
print("héllo"[-1]);      // o
print(len("héllo"));     // 5
print("ann" < "bob");    // true
print("Zed" < "ann");    // true
```

## Escapes

`\n`, `\t`, `\r`, `\"`, `\\`, `\{` and `\}`:

```rune
print("say \"hi\"");             // say "hi"
print("C:\\temp");               // C:\temp
print("shipped\npending");
// shipped
// pending
```

## Interpolation

Any expression in `{}` inside a string literal is evaluated and inserted as
`print` would show it; `\{` is a literal brace.

```rune
let user = { name: "ann", roles: ["admin", "ops"] };
let total = 59.97;

print("hi {user.name}");                                  // hi ann
print("roles: {user.roles}");                             // roles: ["admin", "ops"]
print("{len(user.roles)} roles, total {total.fixed(1)}"); // 2 roles, total 60.0
print("order is {total > 50 ? "big" : "small"}");         // order is big
print("set \{name} in the template");                     // set {name} in the template
```

So a `{` meant literally, as in JSON, needs `\{` in a plain string. A raw
string is simpler:

```rune
json.parse("{\"ok\": true}");                  // Error: Unexpected character: \
```

```rune
print(json.parse("\{\"ok\": true}").ok);        // true
let body = r"""{"id": 7, "tags": ["new"]}""";
print(json.parse(body).id);                     // 7
```

## Raw Strings

A raw string, `r"..."`, is taken as written: no escapes and no
interpolation, so it suits regular expressions and Windows paths. It cannot
hold `"`; in a pattern write it `\x22`, or use `r"""..."""`.

```rune
print(r"C:\temp\{x}");                              // C:\temp\{x}
print(regex.test(r"^\d{3}-\d{4}$", "555-1234"));    // true
```

## Multi-line Strings

`"""..."""` is a string that may hold `"` and span lines, with escapes and
interpolation; `r"""..."""` is its raw form. When the opening quotes end
their line and the closing ones start theirs, those two newlines are
dropped and the whitespace before the closing quotes is taken off every
line; a line indented less, unless blank, is an error.

```rune
let user = "ann";
print("""say "hi" to {user}""");    // say "hi" to ann

let title = "Orders";
let page = """
    <h1 class="title">{title}</h1>
      <p>indented by two</p>
    """;
print(page);
// <h1 class="title">Orders</h1>
//   <p>indented by two</p>

let css = r"""
    a { color: red; }
    """;
print(css);    // a { color: red; }
```

```rune
let text = """
    shipped
  pending
    """;    // Error: Insufficient indentation in multi-line string.
```
