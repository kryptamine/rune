# os

## Environment

`env(name)` is the variable's value, or `nil` if unset; `setEnv(name, v)`
and `unsetEnv(name)` change it, and `environ()` is all of it as an object.
`platform` is `"darwin"`, `"linux"`, `"windows"`, ...

```rune
let port = int(os.env("PORT") ?? "8080");
print(port);                    // 8080

os.setEnv("APP_MODE", "test");
print(os.env("APP_MODE"));      // test
os.unsetEnv("APP_MODE");
print(os.env("APP_MODE"));      // nil

print(os.environ().has("PATH")); // true
let shell = os.platform == "windows" ? "cmd" : "sh";
print(shell);                   // sh
```

## exec

`exec(cmd[, args[, opts]])` runs `cmd` with the array of string `args` and
waits; it returns `ExecResult { code, stdout, stderr }`. A non-zero exit is
not an error; a command not found is an `IOError` with code `notFound`.

`opts` is an `ExecOptions { input, dir, env }`, or an object converted to
one: `input` is stdin, `dir` the working directory, `env` an object added
to the current environment; all are optional.

```rune
let r = os.exec("echo", ["hello", "world"]);
print(r);   // ExecResult {code: 0, stdout: "hello world\n", stderr: ""}

let up = os.exec("tr", ["a-z", "A-Z"], { input: "shout\n" });
print(up.stdout.trim());   // SHOUT

let greet = os.exec("sh", ["-c", "echo $GREETING"], { env: { GREETING: "hi" } });
print(greet.stdout.trim()); // hi

let ls = os.exec("ls", ["/no/such/dir"]);
print(ls.code);             // 1

try {
    os.exec("no-such-command");
} catch e is IOError {
    print(e.code);          // notFound
}
```

## exit

`exit([code])` exits with `code`, 0 to 255, default 0, at once; `try` does
not catch it. To exit from the end of the program, return the code from
[`main`](../programs/main.md).

```rune
// status.rn
let r = os.exec("git", ["status", "--short"]);
if r.code != 0 {
    io.eprint(r.stderr.trim());
    os.exit(r.code);
}
io.write(r.stdout);
```

```sh
$ ./rune run status.rn; echo "exit $?"    # outside a git repository
fatal: not a git repository (or any of the parent directories): .git
exit 128
```
