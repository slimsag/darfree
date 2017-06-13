# Package darfree uses black magic to release memory on Darwin.

darfree is a package that uses ⚫️ black magic (assembly and runtime monkey
patching) to lower the perceived memory consumption of Go programs on
Darwin / amd64 (Mac OS) systems. Since it is based on page size, the actual
'gains' are dependent upon the program itself. They can be as much as your
largest heap allocation. See cmd/darfree for an example which uses ~500MB
without darfree, and only ~17MB with.

## Background

Go allocates memory in pages, and later advises the operating system that it
may free unneeded memory using madvise(..., MADV_FREE) calls. On Linux, this
system call immediately frees the memory page. However, on Darwin machines
the kernel will decide to retain that memory as part of your program's RSS
(residential set size) until memory pressure by your application or another
running on the system causes the kernel to _truly_ free that memory.
In reality, Darwin does the right thing. Keeping the memory as part of your
program's RSS causes no harm, and actually benefits your program by making
subsequent allocations faster. However, from a user's perspective looking at
their "Activity Monitor" it will look like your application is leaking or
using much more memory than it actually is. That is what darfree solves.

## Beware

I advise you really do not use this package at all, unless you really
understand what is going on and what the trade-offs are. At any point in
time this package may become unmaintained, and may break with future Go
versions.

## Usage

```Go
package main

import _ "github.com/slimsag/darfree"

func main() {
  // ...
}
```
