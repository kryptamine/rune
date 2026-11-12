# Streams

A `Stream` is an open file from `fs.open`, or a response body from
`http.request` with `stream: true`. It reads and writes a piece at a time,
so a large file is never held whole.

## Reading

- `readLine()` — the next line without `\n` or `\r\n`, or `nil` at the end;
- `read(n)` — bytes, at most `n` and at least one, or `nil` at the end;
- `readAll()` — the rest as bytes; `readText()` — the rest as a string.

`for line in s` (or `for i, line in s`) reads lines up to the end.

```rune
let path = fs.join(fs.tempDir(), "access.log");
fs.write(path, "GET /a 200\nPOST /b 500\nGET /c 404\n");

let log = fs.open(path);
print(log.readLine());   // GET /a 200
print(log.read(4));      // b"POST"
print(log.readLine());   //  /b 500
print("%q".format(log.readText())); // "GET /c 404\n"
print(log.readLine());   // nil
log.close();

let mut errors = 0;
for line in fs.open(path) {
    if !line.endsWith(" 200") errors++;
}
print(errors);           // 2
```

## Writing

`write(data)` writes a string or bytes, buffered; `flush()` writes the
buffer out. `close()` flushes and closes the stream; closing again does
nothing, and streams the program leaves open are closed when it ends.

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

## Errors

A failure, reading a closed stream included, is an `IOError`.

```rune
let s = fs.open(fs.join(fs.tempDir(), "data.txt"), "w");
s.close();
try {
    s.readLine();
} catch e is IOError {
    print(e.message.endsWith("stream is closed."));   // true
}
```
