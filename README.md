# gisp

`gisp` is a minimal Scheme-inspired interpreter written in Go. It offers a simple type system,
lexical scoping, macros, and first-class continuations while keeping the language approachable.
Alongside the traditional S-expression syntax, the project ships with **Gisp**, a Go-flavoured
surface language that compiles down to the same Scheme semantics (tail calls, macros, continuations).

The project intentionally favors clarity over strict Scheme compatibility: standard procedures use
explicit names (`list`, `append`, `call/cc`, etc.), but the runtime behaviour follows familiar Lisp
semantics.

## Features

- Go-like Gisp language with inline s-expression escapes
- Atoms, lists, integers (`int64`), reals (`float64`), booleans, and strings
- Proper lexical scope with closures and unified namespace (functions are values)
- Tail-call optimization to support deeply recursive programs
- First-class continuations via `call/cc`
- Non-hygienic macros (`define-macro`) for syntactic extensions
- Distinct empty list and `#f` values
- Basic standard library including arithmetic, comparison, list utilities, strings, and I/O
- Reader for s-expressions (numbers, strings with escapes, quoting, quasiquote, comments)
- Command-line interface offering a REPL and script execution (with shebang support)

Garbage collection relies entirely on the Go runtime—no additional memory management is required.

## Documentation

- [Gisp language guide](docs/Language.md)
- [Runtime primitives](docs/Primitives.md)
- [S-expression grammar](docs/S-Expressions.md)
- [Step-by-step tutorial](docs/Gisp-Tutorial.md)

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

Sample programs live in `examples/`:

- `hello.gs` – hello world via `display` and `newline`
- `continuation.gs` – demonstrates capturing and invoking continuations with `call/cc`
- `fact.gisp` – factorials using the Go-like syntax
- `tutorial_*.gisp` – end-to-end walkthrough files referenced by the tutorial
- `puzzle15.gisp` – interactive 15 puzzle; enter moves as symbols (`up`, `down`, `left`, `right`)
- `snobol_patterns.gisp` – Snobol-inspired backtracking pattern matcher written in pure Gisp

Run any of them with the interpreter:

```bash
./gisp examples/hello.gs
./gisp examples/fact.gisp
```

Use the `read` primitive when you need to accept raw Scheme data at runtime:

```gisp
display("Enter a datum: ")
var datum = read()
display("You typed: ")
display(datum)
newline()
```

The regression test suite keeps these examples in sync with the runtime.

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
