# Streams

A stream reads or writes data a piece at a time, so a large file is never
held in memory whole. Streams come from:

- [`fs.open(path[, mode])`](fs.md#reading-and-writing), a file opened to
  read (`"r"`), write (`"w"`) or append (`"a"`);
- [`fs.lock(path)`](fs.md#locks), a file opened for writing under a lock;
- [`http.request`](json-http.md#httprequest) with `stream: true`, a
  response body.

| Method | Does |
|---|---|
| `readLine()` | the next line, or `nil` at the end |
| `read(n)` | up to `n` bytes, or `nil` at the end |
| `readBytes()`, `readText()` | the rest, as bytes or as a string |
| `write(data)` | writes a string or bytes |
| `flush()` | writes out what is buffered |
| `close()` | flushes and closes |
| `readAt(offset, n)` | up to `n` bytes at `offset` (files only) |
| `seek(offset)` | moves to `offset` (files only) |
| `sync()` | flushes to disk (files only) |

## Reading

- `readLine()` is the next line, without `\n` or `\r\n`, or `nil` at the
  end.
- `read(n)` is the next bytes, at least one and at most `n`, or `nil` at
  the end.
- `readBytes()` and `readText()` are everything left, as bytes or as a
  string.

`for line in s` reads the lines to the end; `for i, line in s` numbers
them from 0.

```rune
let path = fs.join(fs.tempDir(), "access.log");
fs.write(path, "GET /a 200\nPOST /b 500\nGET /c 404\n");

let log = fs.open(path);
print(log.readLine());                // GET /a 200
print(log.read(4));                   // b"POST"
print(log.readLine());                //  /b 500
print("%q".format(log.readText()));   // "GET /c 404\n"
print(log.readLine());                // nil
log.close();
```

Counting failed requests in a log, one line at a time:

```rune
let path = fs.join(fs.tempDir(), "access.log");
fs.write(path, "GET /a 200\nPOST /b 500\nGET /c 404\n");

let mut failed = 0;
for i, line in fs.open(path) {
    let status = int(line.split(" ")[2]);
    if status >= 400 {
        print("line {i + 1}: {line}");
        failed += 1;
    }
}
print("{failed} failed");
// line 2: POST /b 500
// line 3: GET /c 404
// 2 failed
```

## Writing

`write(data)` writes a string or bytes. Writes are buffered: `flush()`
sends the buffer to the file, and `close()` flushes and closes the stream.
Closing twice does nothing, and streams still open when the program ends
are closed then.

Copying a file line by line, changing each line on the way:

```rune
let dir = fs.tempDir();
let src = fs.join(dir, "words.txt");
let dst = fs.join(dir, "upper.txt");
fs.write(src, "alpha\nbeta\n");

let out = fs.open(dst, "w");
for line in fs.open(src) {
    out.write(line.upper() + "\n");
}
out.close();
print(fs.lines(dst));   // ["ALPHA", "BETA"]

let log = fs.open(dst, "a");
log.write("GAMMA\n");
log.flush();
print(fs.lines(dst));   // ["ALPHA", "BETA", "GAMMA"]
log.close();
```

## Moving around a file

A stream of a file can also read at any position:

- `readAt(offset, n)` is up to `n` bytes at `offset` from the start, or
  `nil` at or past the end. It reads the file itself, so it sees what
  other streams wrote and flushed, and does not move this stream.
- `seek(offset)` moves to `offset` from the start, flushing writes first.
  A stream opened with `"a"` still appends at the end.
- `sync()` flushes, then makes the system write the file to disk
  (`fsync`), so it survives a power loss.

```rune
let path = fs.join(fs.tempDir(), "data.log");
let log = fs.open(path, "a");
log.write("hello world");
log.sync();

let r = fs.open(path);
print(r.readAt(6, 5));   // b"world"
print(r.read(5));        // b"hello"
r.seek(0);
print(r.readLine());     // hello world
r.close();
log.close();
fs.removeAll(fs.dir(path));
```

## Errors

A failure is an `IOError`, and so is using a stream after closing it:

```rune
let s = fs.open(fs.join(fs.tempDir(), "data.txt"), "w");
s.close();
try {
    s.readLine();
} catch e is IOError {
    print(e.message.endsWith("stream is closed."));   // true
}
```
