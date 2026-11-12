---
title: Hello, Rune
date: 2026-09-01
tags: [rune, intro]
---
# Hello, Rune

Rune is a small **dynamically typed** language with a *register VM*.
This post was rendered by a Rune program, see [the repo](https://github.com/kryptamine/rune).

## Why another language

> Because writing one is the best way to understand them.

## A taste

```rune
fun fib(n) {
    return n < 2 ? n : fib(n - 1) + fib(n - 2);
}
```

Things worth noting:

- immutable by default, `let mut` when you need it
- structs with `impl` blocks
- `match` with guards
