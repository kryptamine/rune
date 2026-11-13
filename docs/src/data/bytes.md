# Bytes

`bytes` are immutable binary data: a `b"..."` literal, which takes
`\xNN` escapes and no interpolation, or `bytes(v)` of a string (its UTF-8),
an array of integers from 0 to 255, or bytes. `b[i]` is the integer byte,
`b.len()` the size; `+` joins and `<` compares bytes, `==` by content, and
`for i, x in b` walks them. Bytes are hashable, and print as a literal.

```rune
let header = b"\x89PNG";
print(header);           // b"\x89PNG"
print(header[0]);        // 137
print(header.len());     // 4
print(header.hex());     // 89504e47

print(bytes("é"));                // b"\xc3\xa9"
print(bytes([104, 105]));         // b"hi"
print(b"hi" + b"!");              // b"hi!"
print(b"hi" == bytes("hi"));      // true

for i, x in b"hi" {
    print("{i}: {x}");
}
// 0: 104
// 1: 105
```

A byte cannot be changed; build new bytes instead:

```rune
let mut buf = b"ab";
buf[0] = 120; // TypeError: Bytes are immutable.
```

## The bytes namespace

Besides `bytes(v)` itself, `alloc(n[, fill])` (`n` bytes of `fill`,
default `0`), `fromHex(s)` and `fromBase64(s)`:

```rune
print(bytes.alloc(3));             // b"\x00\x00\x00"
print(bytes.alloc(2, 255));        // b"\xff\xff"
print(bytes.fromHex("cafe"));      // b"\xca\xfe"
print(bytes.fromBase64("aGk="));   // b"hi"
```

## Methods

`text()` (the bytes as UTF-8 text, as a stream's `readText()` reads it;
anything else is a `ValueError`), `toArray()`, `slice(start[, end])`
(sharing the data), `indexOf(sub)` (`-1` if missing), `concat(b, ...)`,
`hex()` and `base64()`.

```rune
let data = b"hello";
print(data.text());                  // hello
print(data.toArray());               // [104, 101, 108, 108, 111]
print(data.slice(1, 3));             // b"el"
print(data.indexOf(b"ll"));          // 2
print(data.concat(b", ", b"world")); // b"hello, world"
print(data.base64());                // aGVsbG8=
```

```rune
print(b"\xff".text()); // ValueError: bytes.text() got bytes that are not valid UTF-8.
```
