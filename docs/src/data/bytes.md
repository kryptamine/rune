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

A byte cannot be changed; build new bytes, or use a [buffer](#buffers):

```rune
let mut buf = b"ab";
buf[0] = 120; // TypeError: Bytes are immutable.
```

## The bytes namespace

Besides `bytes(v)` itself, `alloc(n[, fill])` (`n` bytes of `fill`,
default `0`), `concat(b, ...)`, `fromHex(s)`, `fromBase64(s)` and
`buffer([data])`, a new [buffer](#buffers):

```rune
print(bytes.alloc(3));             // b"\x00\x00\x00"
print(bytes.alloc(2, 255));        // b"\xff\xff"
print(bytes.concat(b"a", b"b"));   // b"ab"
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

## Binary data

These read numbers at an offset; a read past the end is an `IndexError`.
`order` is `"le"` (the default) or `"be"`.

- `uint(at, size[, order])`, `int(at, size[, order])` — an integer of 1, 2, 4 or 8 bytes; an 8-byte `uint` keeps all 64 bits, as two's complement;
- `float(at[, size[, order]])` — a float of 4 or 8 bytes (the default);
- `uvarint(at)`, `varint(at)` — a varint as `[value, next offset]`, or `nil` if the bytes end first.

`crc32()` is the IEEE checksum, an integer; `md5()`, `sha1()` and
`sha256()` give the digest as bytes.

```rune
let b = b"\x2a\x00\xff\xff\xac\x02";
print(b.uint(0, 2));          // 42
print(b.uint(0, 2, "be"));    // 10752
print(b.int(2, 2));           // -1
print(b.uvarint(4));          // [300, 6]
print(b"hello".crc32());      // 907060870
print(b"abc".sha256().hex().slice(0, 16)); // ba7816bf8f01cfea
```

## Buffers

A buffer is binary data that changes in place: `bytes.buffer()` makes an
empty one and `bytes.buffer(data)` one holding a copy of a string, bytes or
another buffer. `buf[i]` reads a byte and `buf[i] = x` sets one; it
iterates, spreads and compares like bytes, and every reading method of
bytes works on it too. Like an array, it needs a `let mut` to change.

- `write(data)` — appends a string, bytes or a buffer;
- `writeUint(x, size[, order])`, `writeInt(x, size[, order])`, `writeFloat(x[, size[, order]])`, `writeUvarint(x)`, `writeVarint(x)` — append a number, the counterparts of the reads above; a number that does not fit is a `ValueError`;
- `clear()` empties it; `len()`;
- `bytes([start[, end]])` — a copy as bytes, which later writes leave alone.

`fs.write`, `fs.append` and a stream's `write` take a buffer as they
take bytes.

```rune
let mut rec = bytes.buffer();
rec.writeUint(1, 1);
rec.writeUvarint(300);
rec.write("key");
print(rec);                  // buffer(b"\x01\xac\x02key")
rec[0] = 2;
print(rec.uvarint(1));       // [300, 3]

let mut out = bytes.buffer();
out.writeUint(rec.len(), 4, "be");
out.write(rec);
print(out.bytes().hex());    // 0000000602ac026b6579
```
