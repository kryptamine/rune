# io

Standard input, output and error.

| Function | Does |
|---|---|
| `io.readLine()` | the next line of stdin, or `nil` at the end |
| `io.readText()` | the rest of stdin, as a string |
| `io.write(v, ...)` | writes to stdout like `print`, without a newline |
| `io.eprint(v, ...)` | writes to stderr like `print` |
| `io.ewrite(v, ...)` | writes to stderr like `print`, without a newline |

`readLine()` gives a line without its `\n` or `\r\n`. Reading stdin or
writing to stderr first flushes what was printed, so prompts and messages
come out in order.

## Reading lines

A filter that sums the numbers it is given, one per line, and warns on
stderr about lines that are not numbers. `?? break` ends the loop when
`readLine` gives `nil`:

```rune
// sum.rn
let mut total = 0;
while true {
    let line = io.readLine() ?? break;
    try {
        total += number(line);
    } catch e is ValueError {
        io.eprint("skipping '{line}'");
    }
}
io.write("total: ");
print(total);
```

```sh
$ printf '10\n20\nabc\n12\n' | rune run sum.rn
skipping 'abc'
total: 42
```

## Reading everything

`readText()` takes all of stdin at once, for input that is not line by
line:

```rune
// count.rn
let words = regex.split(r"\s+", io.readText().trim());
print("{words.len()} words");
```

```sh
$ echo "the quick brown fox" | rune run count.rn
4 words
```

## Prompts

`io.write` leaves the cursor on the same line, which suits a prompt:

```rune
// ask.rn
io.write("name: ");
let name = io.readLine() ?? "stranger";
print("hello, {name}!");
```

```sh
$ echo ann | rune run ask.rn
name: hello, ann!
```
