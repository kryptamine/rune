# Performance

Rune compiles a program before it runs it. The parser builds an AST, the
resolver checks names and scopes, and the compiler turns the result into
bytecode for a register machine, in the style of Lua. A VM then runs the
bytecode. This page describes what makes that fast.

## Registers and calls

- Every local variable, top-level ones included, lives in a register of
  its function's frame. A variable captured by a nested function lives in
  a heap cell instead, shared by the closures.
- A call pushes a frame onto one shared register stack, so calling a
  function allocates nothing.
- A value is two machine words. Numbers, booleans and strings are stored
  inline and never allocate.

## Compile-time work

- Constant expressions are folded: `60 * 1000` becomes `60000`.
- A `const` has no variable at all: each read of it is the literal, in
  every file that imports it.
- Members of a namespace, such as `math.floor`, are bound at compile time.
- A comparison in an `if` or a loop condition jumps directly, with no
  boolean made in between. Loops test their condition at the bottom.
- A struct is one allocation holding its fields. Field names are interned
  when the program is compiled, so `p.x` finds its field by pointer, with
  no hashing and no string comparison.

## Value semantics without copying

Arrays, objects, maps, sets, buffers and structs behave as values: every
assignment or argument is a copy. Copies are lazy, so this costs little:

- A copy only marks the value as shared. The next write to a shared value
  clones it first, shallowly, and a write along a path such as
  `a.b[0] = x` clones only the levels that are shared.
- A variable read for the last time is moved, not copied: nothing is
  marked shared. `return r` is such a read. A loop, `catch` or `match`
  variable, a `mut` parameter, and a captured or exported variable are
  never moved from.
- An argument that is fresh, or a variable read for the last time, is
  given up to the call, which moves it on. So `a = add(a, x)` copies
  nothing.
- A `mut` parameter, `mut self`, and a `for mut` variable work on their
  place's value directly and write it back at the end, so changing them in
  place copies nothing.
- `for x in xs` walks `xs` shared, without copying it.

## Containers and output

- A map or a set is an open-addressing hash table over the values
  themselves, in insertion order.
- `print` output is buffered and written out when the program ends, on a
  runtime error too, or before reading stdin or writing stderr.

## Reading the bytecode

`rune disasm file.rn` prints the bytecode of each function: one
instruction per line, an opcode and its operands, which are registers or
indices of constants.

```rune
// sum.rn
fun sumTo(n) {
    let mut total = 0;
    let mut i = 1;
    while i <= n {
        total += i;
        i += 1;
    }
    total * (60 * 1000)
}
```

```sh
$ rune disasm sum.rn
<script>: arity 1, 2 registers, 0 upvalues
    0  LOADK         1 0 0
    1  CLOSURE       1 0 0
    2  RETURNNIL     0 0 0

sumTo: arity 1, 4 registers, 0 upvalues
    0  LOADK         1 0 0
    1  LOADK         2 0 1
    2  JMP           0 0 5
    3  ADD           1 1 2
    4  ADDK          2 2 1
    5  JMPCMP        2 0 3  x=1 y=true
    6  MULK          3 1 2
    7  RETURN        3 0 0
    8  RETURNNIL     0 0 0
```

`n` is register 0, `total` register 1 and `i` register 2. Reading it:

- 0–1: `total` and `i` get the constants `0` and `1`.
- 2: jump to the loop condition, which sits at the bottom (5).
- 3–4: the body. `total += i` is one `ADD`; `i += 1` is one `ADDK`, with
  the constant `1` as an operand.
- 5: `JMPCMP` compares `i` with `n` and jumps back to the body (3) while
  `i <= n` holds.
- 6–7: `60 * 1000` was folded to the constant `60000` at compile time, so
  the result is a single `MULK`.

## Garbage collection

`rune` lets the heap grow to five times the live data before it collects
garbage (`GOGC=400`). That trades memory for speed, and matters most on a
single CPU. Set `GOGC` in the environment to override it.

## Benchmarks

```sh
make bench                              # all benchmarks
make bench filter=Containers count=6    # arrays, maps, sets, nested writes
```
