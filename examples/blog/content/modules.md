---
title: Splitting a program into modules
date: 2026-10-15
tags: [rune, modules]
---
Each file has its own top level, and `export` marks what others may see.

## Importing

1. `import md.block;` binds the module as `block`
2. `import md.block as md;` binds it as `md`
3. members are checked at compile time

## Private names

Anything without `export` stays inside the file, so two modules can both have a `helper` & never clash.

---

That's it.
