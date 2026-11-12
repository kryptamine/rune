# main

If the entry file declares `fun main`, it runs after the top level. If it
takes a parameter, that is the program's arguments, an array of strings.
It returns `nil` to exit with 0, or an exit code from 0 to 255.

```rune
// greet.rn
fun main(args) {
    if len(args) == 0 {
        io.eprint("usage: greet <name>...");
        return 2;
    }

    for name in args {
        print("hi " + name);
    }
}
```

```sh
$ ./rune run greet.rn ann bob
hi ann
hi bob
$ ./rune run greet.rn
usage: greet <name>...
$ echo $?
2
```

Arguments are always strings; convert them yourself:

```rune
// repeat.rn
fun main([word, times = "1"]) {
    print(word.repeat(int(times)));
}
```

```sh
$ ./rune run repeat.rn ab 3
ababab
```

The top level runs first, so `main` sees everything declared there.
Without `main` the top level is the whole program.

```rune
let greeting = "hello";

fun main() {
    print(greeting); // hello
}
```

`main` takes at most one parameter. Returning an integer out of range is a
`ValueError`, and anything else but `nil` or an integer a `TypeError`:

```rune
fun main() { return 256; } // ValueError: main must return nil or an exit code from 0 to 255, got 256.
```

```rune
fun main() { return "ok"; } // TypeError: main must return nil or an exit code from 0 to 255, got "ok".
```

To exit from deeper in the program, use `os.exit(code)`.
