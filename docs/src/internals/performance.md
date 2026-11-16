# Performance

Before running, the resolved AST is compiled to bytecode for a register machine, in the style of Lua. In particular:

- every local, top-level variables included, lives in a register of its function's frame; a variable some nested function captures is kept in a heap cell that the closures share;
- a call pushes a frame onto one shared register stack, so it allocates nothing;
- comparisons in `if` and loop conditions jump directly; constant operands and literal expressions are folded; loops test their condition at the bottom;
- a value is two words: numbers, booleans and strings are stored inline and never allocate;
- a struct is one allocation of its fields; the compiler interns field names, so `p.x` finds the field by pointer, with no hashing or string comparison;
- an array, object, map, set, buffer or struct is copied lazily: a copy marks it shared, and the next write clones it, shallowly, first; a `mut self` and a `mut` parameter change their caller's value in place. A copy from a variable read for the last time, `return r` included, is a move and marks nothing; a parameter, a loop, catch or match variable and a captured or exported one are never moved from;
- a write along a path copies only the levels another holder shares; a `mut` parameter and a `for mut` variable take their place's value without a copy and write it back after, so changing them in place copies nothing. A call of a function with `mut` parameters takes the slow call path; a `for mut` whose body never names its collection walks it in place, another walks it as it was and copies on the first write-back;
- a `const` has no variable: reads of it fold like the literal, in every file;
- a map or a set is an open-addressing table over the values themselves, in insertion order;
- a `for`-`in` walks its collection shared instead of copying it, and a loop over an array that runs to its end hands its cursor on to the next;
- `print` output is buffered and flushed when the program ends (also on a runtime error).

## Reading the bytecode

`rune disasm file.rn` prints the bytecode of each function: an opcode and
its operands, registers or indices of constants.

```rune
// countdown.rn
fun countdown(n) {
    let mut left = n;
    while left > 0 {
        left -= 1;
    }
    return 60 * 1000;
}
```

```sh
$ rune disasm countdown.rn
<script>: arity 1, 2 registers, 0 upvalues
    0  LOADK         1 0 0
    1  CLOSURE       1 0 0
    2  RETURNNIL     0 0 0

countdown: arity 1, 3 registers, 0 upvalues
    0  MOVE          1 0 0
    1  JMP           0 0 3
    2  SUBK          1 1 0
    3  JMPCMPK       1 1 2  x=2 y=true
    4  LOADK         2 0 2
    5  RETURN        2 0 0
    6  RETURNNIL     0 0 0
```

`n` is register 0 and `left` register 1. The loop jumps to its condition
at the bottom (1), where `JMPCMPK` compares `left` with the constant `0`
and jumps back to the body (2) while it holds, with no boolean made in
between; `left -= 1` is a single `SUBK` with a constant operand, and
`60 * 1000` was folded to `60000` at compile time (4).

## Garbage collection

`rune` lets the heap grow to five times the live data before collecting
(`GOGC=400`), which matters most on a single CPU. Set `GOGC` to override it.

## Benchmarks

```sh
make bench
make bench filter=Containers count=6   # arrays, maps, sets, nested writes
```
