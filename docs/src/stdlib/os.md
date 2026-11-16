# os

The environment, other programs, and the exit code.

| Member | Is |
|---|---|
| `os.env(name)` | the value of an environment variable, or `nil` |
| `os.setEnv(name, value)`, `os.unsetEnv(name)` | set or remove a variable |
| `os.environ()` | all variables, as an object |
| `os.platform` | `"darwin"`, `"linux"`, `"windows"`, ... |
| `os.exec(cmd[, args[, opts]])` | runs a command and waits for it |
| `os.exit([code])` | ends the program |

## Environment

`env(name)` is the variable's value, or `nil` if it is not set, so `??`
gives a default. `setEnv` and `unsetEnv` change the environment of the
program and of the commands it runs.

```rune
let port = number(os.env("PORT") ?? "8080");
print(port);                     // 8080

os.setEnv("APP_MODE", "test");
print(os.env("APP_MODE"));       // test
os.unsetEnv("APP_MODE");
print(os.env("APP_MODE"));       // nil

print("PATH" in os.environ());   // true
```

`platform` tells the systems apart:

```rune
let shell = if os.platform == "windows" { "cmd" } else { "sh" };
print(shell);   // sh
```

## exec

`exec(cmd[, args[, opts]])` runs `cmd` with `args`, an array of strings,
and waits for it to finish. It returns an
`ExecResult { code, stdout, stderr }`.

A non-zero exit code is not an error; check `code`. A command that does
not exist is an `IOError` with code `"notFound"`.

```rune
let r = os.exec("echo", ["hello", "world"]);
print(r);                   // ExecResult {code: 0, stdout: "hello world\n", stderr: ""}
print(r.stdout.trim());     // hello world

let failed = os.exec("sh", ["-c", "echo oops >&2; exit 3"]);
print(failed.code);         // 3
print(failed.stderr);       // oops

try {
    os.exec("no-such-command");
} catch e is IOError {
    print(e.code);          // notFound
}
```

`opts` is an `ExecOptions { input, dir, env }`, or an object with any of
those keys:

- `input` — a string fed to the command's stdin;
- `dir` — the directory to run it in;
- `env` — an object of variables to add to the current environment.

```rune
let up = os.exec("tr", ["a-z", "A-Z"], { input: "shout\n" });
print(up.stdout.trim());      // SHOUT

let here = os.exec("pwd", [], { dir: "/tmp" });
print(here.stdout.trim());    // /tmp

let greet = os.exec("sh", ["-c", "echo $GREETING, $USER_NAME"], {
    env: { GREETING: "hi", USER_NAME: "ann" },
});
print(greet.stdout.trim());   // hi, ann
```

## exit

`exit([code])` ends the program at once with `code`, 0 to 255, 0 by
default. `try` does not catch it, and `defer` does not run. To set the exit
code at the normal end of a program, return it from
[`main`](../programs/main.md) instead.

```rune
// backup.rn
let r = os.exec("cp", ["data.db", "/backups/"]);
if r.code != 0 {
    io.eprint("backup failed with code {r.code}");
    os.exit(r.code);
}
print("backup done");
```

```sh
$ rune run backup.rn; echo "exit $?"
backup failed with code 1
exit 1
```

A code out of range is a `ValueError`:

```rune
os.exit(300); // ValueError: os.exit() expects an exit code from 0 to 255, got 300.
```
