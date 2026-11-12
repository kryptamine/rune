# io

- `readLine()` — the next line of stdin without `\n` or `\r\n`, or `nil` at the end.
- `read()` — the rest of stdin.
- `write(v)` — writes `v` to stdout as `print` does, without a newline.
- `eprint(v)`, `ewrite(v)` — the same to stderr, with and without a newline.

Reading stdin or writing stderr first flushes what was printed.

A filter that sums the numbers it is given, one per line, and warns about
the lines that are not numbers:

```rune
// sum.rn
let mut total = 0;
let mut line = io.readLine();
while line != nil {
    try {
        total += number(line);
    } catch e is ValueError {
        io.eprint("skipping '{line}'");
    }
    line = io.readLine();
}
io.write("total: ");
print(total);
```

```sh
$ printf '10\n20\nabc\n12\n' | ./rune run sum.rn
skipping 'abc'
total: 42
```

`read()` takes all of stdin at once, for input that is not line by line:

```rune
// count.rn
let words = regex.split(r"\s+", io.read().trim());
print("{len(words)} words");
```

```sh
$ echo "the quick brown fox" | ./rune run count.rn
4 words
```
