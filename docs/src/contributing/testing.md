# Running Tests

```sh
make test                # all tests
make test filter=arrays  # scripts under test/arrays
make update              # rewrite golden files after an intended change
make cover               # coverage
make fuzz FUZZTIME=1m    # fuzz the whole pipeline
```

Each `test/**/*.rn` script runs in-process, in parallel, with a 5s timeout. Directives in comments:

- `// expect: <line>` — next line of stdout;
- `// [line: N] Error...` — compile error, in order; without `[line: N]` it is the comment's line;
- `// expect runtime error: [line: N] <message>` — uncaught runtime error.

`foo.stderr` next to `foo.rn` holds the full error report (snippet, caret, trace); a missing file means empty stderr.
`testdata/<command>/*.rn` are run through the CLI command (`tokenize`, `evaluate`, `run`, `disasm`) and compared with `.golden` files: exit code, stdout and stderr.
