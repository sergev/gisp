# gisp

`gisp` is a minimal Scheme-inspired interpreter written in Go. It embraces a simple type system,
lexical scoping, macros, and first-class continuations while keeping the language concise
and approachable. On top of the original S-expression syntax, gisp now includes **Gisp**, a Go-like
language that compiles into the same Scheme semantics (tail calls, macros, continuations).

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
- [Gisp tutorial](docs/Gisp-Tutorial.md)

## Getting Started

### Prerequisites

- Go 1.25 or newer (the project targets Go 1.25.4)

### Build

```bash
make
```

This produces the `gisp` executable in the repository root using the project `Makefile`.

### Install

```bash
make install
```

This installs the binary into `$GOBIN` (default: `$GOPATH/bin`). You can still run `go install`
directly if you prefer, but `make install` keeps the workflow consistent with the other targets.

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

Example programs live in the `examples/` directory:

- `examples/hello.gs` – hello-world via `display` and `newline`
- `examples/continuation.gs` – demonstrates capturing and invoking continuations with `call/cc`
- `examples/fact.gisp` – factorial and tail-recursive factorial using the Go-like syntax

Run them with:

```bash
./gisp examples/hello.gs
./gisp examples/continuation.gs
./gisp examples/fact.gisp
```

Unit tests also execute these examples to ensure they stay in sync with the runtime.

## Testing

```bash
make test
```

This wraps `go test ./...`, running unit tests for the interpreter core, parser, and examples.

## Project Layout

```
.
├── docs/                # Language docs (Gisp overview, grammars, primitives)
├── examples/            # Sample Scheme (.gs) and Gisp (.gisp) programs
├── lang/                # Runtime values, environments, evaluator
├── parser/              # Gisp lexer/parser and S-expression translator
├── sexpr/               # Shared s-expression parsing utilities
├── runtime/             # Primitives, library bootstrap, helpers
├── main.go              # CLI entry point / REPL
├── runtime_test.go      # End-to-end evaluator and example tests
├── Makefile             # make, make install, make test targets
├── go.mod               # Go module definition
└── LICENSE              # MIT License
```

## License

Distributed under the MIT License. See [`LICENSE`](LICENSE) for details.
