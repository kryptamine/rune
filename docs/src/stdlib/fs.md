# fs

Files and directories. These functions read and write a file whole, as a
string or bytes. For files too large to hold in memory, open a
[stream](streams.md).

Relative paths are taken from the current directory, not from the script.
To find a file next to the script, join it with `module.dir`; see
[Modules](../programs/modules.md).

| Function | Does |
|---|---|
| `fs.readText(path)` | the contents as a string |
| `fs.lines(path)` | the lines, without line endings |
| `fs.readBytes(path)` | the contents as bytes |
| `fs.write(path, data)` | writes the file, creating or replacing it |
| `fs.append(path, data)` | adds to the end of the file |
| `fs.truncate(path, size)` | cuts or pads the file to `size` bytes |
| `fs.open(path[, mode])` | opens a [stream](streams.md) |
| `fs.lock(path)` | opens the file with an exclusive lock |
| `fs.exists(path)` | whether the path exists |
| `fs.stat(path)` | a `FileInfo` |
| `fs.list(dir)` | the entries of a directory, as `FileInfo`s |
| `fs.mkdir(path)` | makes a directory and its parents |
| `fs.remove(path)` | removes a file or an empty directory |
| `fs.removeAll(path)` | removes a path and everything under it |
| `fs.rename(from, to)`, `fs.copy(from, to)` | moves, copies a file |
| `fs.cwd()`, `fs.tempDir()` | the current directory; a new temporary one |
| `fs.join`, `fs.base`, `fs.dir`, `fs.ext`, `fs.stem` | work with [paths](#paths) |

A quick tour, in a fresh temporary directory:

```rune
let dir = fs.tempDir();
let todo = fs.join(dir, "notes", "todo.txt");
fs.mkdir(fs.dir(todo));

fs.write(todo, "buy milk\nwrite docs\n");
fs.append(todo, "ship v1\n");

print(fs.lines(todo));               // ["buy milk", "write docs", "ship v1"]
print(fs.readText(todo).len());      // 28
print(fs.readBytes(todo)[..3]);      // b"buy"

fs.removeAll(dir);
```

## Reading and writing

- `readText(path)` is the contents as a string; `lines(path)` splits it
  into lines, without `\n` or `\r\n`; `readBytes(path)` is the contents as
  [bytes](../data/bytes.md).
- `write(path, data)` creates the file, or replaces what it held.
  `append(path, data)` adds to its end, creating it if needed. `data` is a
  string, bytes or a [buffer](../data/bytes.md#buffers).
- `truncate(path, size)` cuts the file to `size` bytes, or pads it with
  zero bytes up to `size`.
- `open(path[, mode])` opens a [stream](streams.md): `"r"` (the default)
  to read, `"w"` to write, creating or replacing the file, and `"a"` to
  append.

Saving and loading settings as JSON:

```rune
let path = fs.join(fs.tempDir(), "settings.json");

struct Settings { theme, fontSize }

fs.write(path, json.stringify(Settings { theme: "dark", fontSize: 14 }, 2));
let settings = Settings { ...json.parse(fs.readText(path)) };
print(settings.theme);      // dark
print(settings.fontSize);   // 14

fs.removeAll(fs.dir(path));
```

Summing a column of a CSV file:

```rune
let path = fs.join(fs.tempDir(), "orders.csv");
fs.write(path, "id,total\n1,30\n2,120\n3,75\n");

let rows = fs.lines(path)[1..];
let total = rows.map(r => number(r.split(",")[1])).reduce((a, b) => a + b, 0);
print(total);   // 225

fs.removeAll(fs.dir(path));
```

`truncate` keeps the start of a file:

```rune
let path = fs.join(fs.tempDir(), "app.log");
fs.write(path, "hello world");
fs.truncate(path, 5);
print(fs.readText(path));   // hello

fs.removeAll(fs.dir(path));
```

## Files and directories

- `exists(path)` is whether anything is at `path`.
- `stat(path)` is a `FileInfo { name, size, isDir, modified, mode }`:
  `size` in bytes, `modified` in seconds like [`time.now()`](time.md),
  `mode` the permission bits.
- `list(dir)` is the entries of `dir` as `FileInfo`s, sorted by name.
- `mkdir(path)` makes a directory and any missing parents; one that already
  exists is fine.
- `remove(path)` removes a file or an empty directory; `removeAll(path)`
  removes a path and everything under it.
- `rename(from, to)` moves a file or directory; `copy(from, to)` copies a
  file.
- `cwd()` is the current directory. `tempDir()` makes a new, empty
  directory; the program should remove it when done.

```rune
let dir = fs.tempDir();
fs.write(fs.join(dir, "draft.md"), "# Hello\n");
fs.copy(fs.join(dir, "draft.md"), fs.join(dir, "backup.md"));
fs.rename(fs.join(dir, "draft.md"), fs.join(dir, "post.md"));
fs.mkdir(fs.join(dir, "images"));

for f in fs.list(dir) {
    let kind = if f.isDir { "dir" } else { "{f.size} bytes" };
    print("{f.name}: {kind}");
}
// backup.md: 8 bytes
// images: dir
// post.md: 8 bytes

let info = fs.stat(fs.join(dir, "post.md"));
print("%o".format(info.mode));                // 644
print(fs.exists(fs.join(dir, "draft.md")));   // false

fs.removeAll(dir);
print(fs.exists(dir));                        // false
```

## Locks

`lock(path)` opens `path` for writing, creating it but keeping what it
holds, and takes an exclusive lock on it. The lock lasts until the stream
is closed or the program ends. While another stream or process holds the
lock, `lock` fails at once with an `IOError` of code `"locked"`.

It keeps two copies of a program from working on the same data at once:

```rune
let path = fs.join(fs.tempDir(), "LOCK");
let held = fs.lock(path);

try {
    fs.lock(path);
} catch e is IOError {
    print(e.code);   // locked
}

held.close();
fs.lock(path).close();   // free again
fs.removeAll(fs.dir(path));
```

## Paths

- `join(a, ...)` joins parts with the path separator, and cleans up `..`,
  `.` and doubled separators.
- `base(p)` is the last element, `dir(p)` everything before it.
- `ext(p)` is the extension, from the last dot; `stem(p)` is the base
  without it.

```rune
let p = "content/posts/hello.md";
print(fs.join("content", "posts", "hello.md"));   // content/posts/hello.md
print(fs.base(p));   // hello.md
print(fs.dir(p));    // content/posts
print(fs.ext(p));    // .md
print(fs.stem(p));   // hello

print(fs.join("site", fs.stem(p) + ".html"));     // site/hello.html
print(fs.join("a", "../b", "./c"));               // b/c
print(fs.ext("backup.tar.gz"));                   // .gz
```

## Errors

A failure is an `IOError` with a `path` and a `code`. The code is one of
`"notFound"`, `"exists"`, `"permission"`, `"isDir"`, `"notDir"`,
`"notEmpty"` and `"locked"`, or `nil` for anything else.

```rune
try {
    fs.readText("missing.txt");
} catch e is IOError {
    print(e.code);      // notFound
    print(e.path);      // missing.txt
    print(e.message);   // fs.readText() missing.txt: no such file or directory.
}
```

Check `code` to treat one failure as normal, here a missing file as an
empty list, and pass on the rest:

```rune
fun loadTodos(path) {
    try {
        return fs.lines(path);
    } catch e is IOError {
        if e.code != "notFound" { throw e; }
        return [];
    }
}

print(loadTodos("todo.txt"));   // []
```
