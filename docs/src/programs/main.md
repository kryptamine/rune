# main

A script can simply run its statements from top to bottom. A command-line
tool usually wants more: its arguments and an exit code. For that, declare
`fun main` in the file you pass to `rune run`.

## Arguments and Exit Code

`main` runs after the top level of the file. Its parameter, if it has one,
is the program's arguments, an array of strings. Its value is the exit
code: `nil` exits with 0, an integer from 0 to 255 exits with that code.

```rune
// greet.rn
fun main(args) {
    if args.len() == 0 {
        io.eprint("usage: greet <name>...");
        return 2;
    }

    for name in args {
        print("hi " + name);
    }
}
```

```sh
$ rune run greet.rn ann bob
hi ann
hi bob
$ rune run greet.rn
usage: greet <name>...
$ echo $?
2
```

Arguments are always strings; convert them yourself, with `int`,
`number` and the like.

## Destructuring Arguments

The parameter can be a [pattern](../language/destructuring.md), which
names the arguments and gives defaults to the optional ones:

```rune
// repeat.rn
fun main([word, times = "1"]) {
    print(word.repeat(int(times)));
}
```

```sh
$ rune run repeat.rn ab 3
ababab
$ rune run repeat.rn ab
ab
```

A pattern without a rest takes exactly as many arguments as it lists, so
`rune run repeat.rn ab 3 x` fails with an `IndexError`. Collect the rest
to accept flags:

```rune
// show.rn
fun main([file, ...flags]) {
    let verbose = "-v" in flags;
    print(file, verbose);
}
```

```sh
$ rune run show.rn notes.txt -v
notes.txt true
```

## Top Level First

The top level of the file runs before `main`, so `main` sees everything
declared there. Without `main`, the top level is the whole program.

```rune
let greeting = "hello";

fun main() {
    print(greeting); // hello
}
```

## Rules

`main` takes at most one parameter. Its value must be `nil` or an
integer from 0 to 255. Since a body gives its last expression, a `main`
ending in `5` without a `;` exits with 5.

```rune
fun main() { return 256; } // ValueError: main must return nil or an exit code from 0 to 255, got 256.
```

```rune
fun main() { return "ok"; } // TypeError: main must return nil or an exit code from 0 to 255, got "ok".
```

To exit from deeper in the program, call `os.exit(code)`. An uncaught
error exits with 70; [Errors and Exit Codes](../errors-and-exit-codes.md)
lists every code.
