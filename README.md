Gisp is a programming language that uses Go-like syntax but compiles to a dynamic interpreted runtime featuring Scheme-inspired semantics — including lists, vectors, tail calls, and continuations.

## Features

- Go-like Gisp language with inline s-expression escapes
- Mutable vectors with native `vec[index]` reads and `vec[index] = value` writes
- Atoms, lists, integers (`int64`), reals (`float64`), booleans, and strings
- Proper lexical scope with closures and unified namespace (functions are values)
- Tail-call optimization to support deeply recursive programs
- First-class continuations via `call/cc`
- Non-hygienic macros (`define-macro`) for syntactic extensions
- Distinct empty list and `false` values
- Basic standard library including arithmetic, comparison, list utilities, strings, and I/O
- Unary primitives cover Go-style numeric negation, logical `not`, and bitwise complement via `^`
- Reader for s-expressions (numbers, strings with escapes, quoting, quasiquote, comments)
- Command-line interface offering a REPL and script execution (with shebang support)

Garbage collection relies entirely on the Go runtime—no additional memory management is required.

## Documentation

- [Step-by-step tutorial](docs/Gisp-Tutorial.md)
- [Gisp language guide](docs/Language.md)
- [Runtime primitives](docs/Primitives.md)
- [S-expression grammar](docs/S-Expressions.md)

## Getting Started

### Prerequisites

- Go 1.25 or newer (the module targets Go 1.25.4)

### Build

```bash
make
```

This builds the `gisp` executable in the repository root.

### Install

```bash
DESTDIR=$HOME/.local make install
```

`make install` drops the binary under `$DESTDIR/usr/bin` (default: `$HOME/.local/usr/bin`).
Adjust `DESTDIR` to suit your environment, or run `go install` if you prefer the standard Go flow.

### Run the REPL

```bash
./gisp
```

The REPL prints prompts (`gisp>`), evaluates expressions, and displays their results.

### Execute a Script (.gs or .gisp)

```bash
./gisp path/to/script.gs
./gisp path/to/program.gisp
```

Scripts may start with a Unix shebang (`#!/usr/bin/env gisp`) so they can be run directly when the
interpreter is on your `PATH`.

Passing `-` runs code from standard input.

## Examples

Browse the full catalog in [`examples/README.md`](examples/README.md). A few quick starts:

- [`tutorial_01_hello.gisp`](examples/tutorial_01_hello.gisp) — first steps with `display` and the Go-like surface syntax.
- [`fact.gisp`](examples/fact.gisp) — recursive and tail-recursive factorial implementations.
- [`continuation.gisp`](examples/continuation.gisp) — capture and resume computations with `callcc`.
- [`maze.gisp`](examples/maze.gisp) — generate and render random mazes with ASCII/Unicode art.
- [`regex_patterns.gisp`](examples/regex_patterns.gisp) — explore the regex matcher DSL with real-world patterns.

Run any example with the interpreter:

```bash
./gisp examples/hello.gs
./gisp examples/fact.gisp
```

## Testing

```bash
make test
```

The `test` target drives `gotestsum` over `go test ./...`. If `gotestsum` is missing it will be installed
automatically (`go install gotest.tools/gotestsum@latest`).

Generate coverage with:

```bash
make cover
```

## Project Layout

```
.
├── docs/                # Language docs (syntax, primitives, tutorial)
├── examples/            # Sample Scheme (.gs) and Gisp (.gisp) programs
├── lang/                # Runtime values, environments, and evaluator
├── parser/              # Gisp lexer/parser and compiler
├── runtime/             # Primitives, library bootstrap, helpers, tests
├── sexpr/               # Shared s-expression parsing utilities
├── main.go              # CLI entry point / REPL
├── Makefile             # build, test, install targets
├── go.mod               # Go module definition
└── LICENSE              # MIT License
```

## License

Distributed under the MIT License. See [`LICENSE`](LICENSE) for details.
