# Performance

Before running, the resolved AST is compiled to bytecode for a register machine, in the style of Lua. In particular:

- every local, top-level variables included, lives in a register of its function's frame; a variable some nested function captures is kept in a heap cell that the closures share;
- a call pushes a frame onto one shared register stack, so it allocates nothing;
- comparisons in `if` and loop conditions jump directly; constant operands and literal expressions are folded; loops test their condition at the bottom;
- a value is two words: numbers, booleans and strings are stored inline and never allocate;
- a struct is one allocation of its fields; the compiler interns field names, so `p.x` finds the field by pointer, with no hashing or string comparison;
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
$ ./rune disasm countdown.rn
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

## Benchmarks

```sh
make bench
```
