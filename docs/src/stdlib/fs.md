# fs

Files are read and written whole, as strings; relative paths are taken from
the current directory, not the script's (use `module.dir` for those). For
large files see [Streams](streams.md).

The examples work in a fresh temporary directory:

```rune
let dir = fs.tempDir();
let notes = fs.join(dir, "notes", "todo.txt");
fs.mkdir(fs.dir(notes));
fs.write(notes, "buy milk\nwrite docs\n");
fs.append(notes, "ship v1\n");

print(fs.lines(notes));          // ["buy milk", "write docs", "ship v1"]
print(fs.read(notes).len());      // 28
print(fs.readBytes(notes).slice(0, 3)); // b"buy"
fs.removeAll(dir);
```

## Reading and writing

- `read(path)` — the contents as a string; `lines(path)` — its lines without `\n` or `\r\n`; `readBytes(path)` — the contents as bytes.
- `write(path, data)` (creates or truncates), `append(path, data)`, of a string or bytes.
- `open(path[, mode])` — a [Stream](streams.md) of the file: `"r"` (default) to read, `"w"` to write (creating or truncating), `"a"` to append.

```rune
let path = fs.join(fs.tempDir(), "settings.json");
fs.write(path, json.stringify({ theme: "dark", fontSize: 14 }));
let settings = json.parse(fs.read(path));
print(settings.theme);   // dark
fs.removeAll(fs.dir(path));
```

## Files and directories

- `exists(path)`; `stat(path)` — a `FileInfo { name, size, isDir, modified, mode }`, `modified` in seconds like `time.now()`.
- `list(dir)` — the entries as `FileInfo`s, sorted by name.
- `mkdir(path)` (with parents; an existing directory is fine), `remove(path)` (a file or an empty directory), `removeAll(path)`, `rename(from, to)`, `copy(from, to)`.
- `cwd()`; `tempDir()` — a new directory, for the program to remove.

```rune
let dir = fs.tempDir();
fs.write(fs.join(dir, "draft.md"), "# Hello\n");
fs.copy(fs.join(dir, "draft.md"), fs.join(dir, "backup.md"));
fs.rename(fs.join(dir, "draft.md"), fs.join(dir, "post.md"));
fs.mkdir(fs.join(dir, "images"));

for f in fs.list(dir) {
    print("{f.name} {f.isDir ? "dir" : f.size}");
}
// backup.md 8
// images dir
// post.md 8

let info = fs.stat(fs.join(dir, "post.md"));
print("%o".format(info.mode));              // 644
print(fs.exists(fs.join(dir, "draft.md"))); // false

fs.remove(fs.join(dir, "backup.md"));
fs.removeAll(dir);
print(fs.exists(dir));                      // false
```

## Paths

`join(a, ...)`, `base(p)`, `dir(p)`, `ext(p)` and `stem(p)`, the base
without its extension.

```rune
let p = "content/posts/hello.md";
print(fs.join("content", "posts", "hello.md")); // content/posts/hello.md
print(fs.base(p));   // hello.md
print(fs.dir(p));    // content/posts
print(fs.ext(p));    // .md
print(fs.stem(p));   // hello
print(fs.join("site", fs.stem(p) + ".html"));   // site/hello.html
print(fs.join("a", "../b", "./c"));             // b/c
```

## Errors

A failure is an `IOError` with `path` and `code`: `"notFound"`, `"exists"`,
`"permission"`, `"isDir"`, `"notDir"`, `"notEmpty"`, or `nil` for anything
else.

```rune
try {
    fs.read("missing.txt");
} catch e is IOError {
    print(e.code);     // notFound
    print(e.path);     // missing.txt
    print(e.message);  // fs.read() missing.txt: no such file or directory.
}
```

A missing file is often not an error:

```rune
fun loadTodos(path) {
    try {
        return fs.lines(path);
    } catch e is IOError {
        if e.code != "notFound" throw e;
        return [];
    }
}

print(loadTodos("todo.txt"));   // []
```
