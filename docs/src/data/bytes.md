# Bytes

`bytes` are immutable binary data: file headers, network packets, hashes,
anything that is not text. A string holds characters; bytes hold raw
values from 0 to 255.

## Making Bytes

Write a `b"..."` literal, where `\xNN` gives a byte by its hex value (no
interpolation), or convert with `bytes(v)`: a string gives its UTF-8
encoding, an array gives one byte per integer.

```rune
let magic = b"\x89PNG";
print(magic);                 // b"\x89PNG"
print(bytes("é"));            // b"\xc3\xa9"
print(bytes([104, 105]));     // b"hi"
```

Bytes print as a literal: printable ASCII as is, the rest escaped as
`\n`, `\xNN` and so on.

The `bytes` namespace has a few more constructors: `alloc(n[, fill])`
makes `n` bytes of `fill` (default `0`); `fromHex(s)` and `fromBase64(s)`
decode text.

```rune
print(bytes.alloc(3));             // b"\x00\x00\x00"
print(bytes.alloc(2, 255));        // b"\xff\xff"
print(bytes.fromHex("cafe"));      // b"\xca\xfe"
print(bytes.fromBase64("aGk="));   // b"hi"
```

## Working With Bytes

Bytes work much like a string or an array:

- `b[i]` is the byte at `i`, an integer; a negative index counts from the end;
- `b[a..b]` is a slice, which shares the data instead of copying it;
- `b.len()` is the size;
- `x in b` finds a byte or a run of bytes;
- `+` joins, `==` compares content, `<` compares byte by byte;
- `for i, x in b` walks the bytes, and `[...b]` spreads them into an array of integers.

```rune
let file = b"\x89PNG\r\n\x1a\nIHDR";
print(file.len());                  // 12
print(file[0]);                     // 137
print(file[..4] == b"\x89PNG");     // true
print(b"IHDR" in file);             // true
print(file[-4..]);                  // b"IHDR"
```

Bytes are hashable, so they can be [map](map-and-set.md) keys and set
elements.

## Methods

- `text()` — decodes the bytes as UTF-8; invalid UTF-8 is a `ValueError`;
- `indexOf(sub)` — where `sub` starts, or `nil`;
- `hex()`, `base64()` — encode as text.

```rune
let data = b"hello";
print(data.text());           // hello
print([...data]);             // [104, 101, 108, 108, 111]
print(data.indexOf(b"ll"));   // 2
print(data + b", world");     // b"hello, world"
print(data.hex());            // 68656c6c6f
print(data.base64());         // aGVsbG8=
```

```rune
print(b"\xff".text()); // ValueError: bytes.text() got bytes that are not valid UTF-8.
```

A byte cannot be changed in place. Build new bytes, or use a
[buffer](#buffers):

```rune
let mut header = b"ab";
header[0] = 120; // TypeError: Bytes are immutable.
```

## Reading Numbers

Binary formats store numbers as fixed-size or variable-size runs of bytes.
These methods read one at an offset `at`; a read past the end is an
`IndexError`. `order` is `"le"` (little-endian, the default) or `"be"`.

- `uint(at, size[, order])`, `int(at, size[, order])` — an integer of 1, 2, 4 or 8 bytes; an 8-byte `uint` keeps all 64 bits, as two's complement;
- `float(at[, size[, order]])` — a float of 4 or 8 bytes (the default);
- `uvarint(at)`, `varint(at)` — a varint, as `[value, next offset]`, or `nil` if the bytes end first.

```rune
let b = b"\x2a\x00\xff\xff\xac\x02";
print(b.uint(0, 2));          // 42
print(b.uint(0, 2, "be"));    // 10752
print(b.int(2, 2));           // -1
print(b.uvarint(4));          // [300, 6]
```

## Checksums and Hashes

`crc32()` is the IEEE checksum, as an integer. `md5()`, `sha1()` and
`sha256()` give the digest as bytes; `hex()` makes it readable.

```rune
print(b"hello".crc32());                  // 907060870
print(bytes("hello").sha256().hex()[..16]);  // 2cf24dba5fb0a30e
```

## Buffers

A buffer is binary data that changes in place, the way to build bytes
piece by piece. `bytes.buffer()` makes an empty one; `bytes.buffer(data)`
starts with a copy of a string, bytes or another buffer. Like an array, it
needs a `let mut` to change, and like every value it is
[copied](../language/values.md#copies) when assigned or passed.

Writing:

- `write(data)` — appends a string, bytes or a buffer;
- `writeUint(x, size[, order])`, `writeInt(x, size[, order])`, `writeFloat(x[, size[, order]])`, `writeUvarint(x)`, `writeVarint(x)` — append a number, the counterparts of the reads above; a number that does not fit is a `ValueError`;
- `buf[i] = x` — sets one byte;
- `clear()` — empties it.

Reading works as on bytes: `buf[i]`, `len()`, `for`, spread, `==`,
`text()`, `indexOf()`, `hex()`, `base64()`, the number reads (`uint`,
`varint`, ...) and the checksums. `buf[a..b]` is a copy as bytes, which
later writes leave alone; `buf[..]` copies it all.

A length-prefixed record, as a binary protocol might send it:

```rune
let mut body = bytes.buffer();
body.writeUvarint(300);
body.write("key");

let mut packet = bytes.buffer();
packet.writeUint(body.len(), 4, "be");
packet.write(body);

print(packet);              // buffer(b"\x00\x00\x00\x05\xac\x02key")
print(packet.hex());        // 00000005ac026b6579

let size = packet.uint(0, 4, "be");
let [id, next] = packet.uvarint(4);
print(size, id);                       // 5 300
print(packet[next..next + 3].text());  // key
```

`fs.write`, `fs.append` and a stream's `write` take a buffer as they take
bytes.
