# Running Tests

```sh
make test                # all tests
make test filter=arrays  # only the scripts under test/arrays
make update              # rewrite golden files after an intended change
make cover               # coverage
make fuzz FUZZTIME=1m    # fuzz the whole pipeline
```

## Script tests

Most tests are Rune scripts, `test/**/*.rn`. Each runs in-process, in
parallel with the others, with a 5 second timeout. Comments in the script
say what it must do:

- `// expect: <text>` — the next line of stdout.
- `// Error...` — a compile error on this line. Compile errors are
  checked in order; `// [line: N] Error...` puts one on line `N`.
- `// expect runtime error: [line: N] <message>` — the uncaught runtime
  error the script ends with.

```rune
let prices = [30, 120];
print(prices.len());   // expect: 2
print(prices[5]);      // expect runtime error: [line: 3] IndexError: Index out of bounds: 5 of 2
```

A file `name.stderr` next to `name.rn` holds the full error report the
script prints: the snippet, the caret and the trace. No such file means
stderr must be empty. `make update` writes these files.

## CLI tests

`testdata/<command>/*.rn` run through the CLI command of their directory,
`run`, `disasm` or `fmt`. Each is compared with its `.golden` file, which
holds the exit code, stdout and stderr.
