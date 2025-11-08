# gisp

`gisp` is a minimal Scheme-inspired interpreter written in Go. It embraces a simple type system,
lexical scoping, macros, and first-class continuations while keeping the surface language concise
and approachable.

The project intentionally favors clarity over strict Scheme compatibility: standard procedures use
explicit names (`list`, `append`, `call/cc`, etc.), but the runtime behaviour follows familiar Lisp
semantics.

## Features

- Atoms, lists, integers (`int64`), reals (`float64`), booleans, and strings
- Proper lexical scope with closures and unified namespace (functions are values)
- Tail-call optimization to support deeply recursive programs
- First-class continuations via `call/cc`
- Non-hygienic macros (`define-macro`) for syntactic extensions
- Distinct empty list and `#f` values
- Basic standard library including arithmetic, comparison, list utilities, strings, and I/O
- Reader for s-expressions (numbers, strings with escapes, quoting, quasiquote, comments)
- Command-line interface offering a REPL and script execution (with shebang support)

## Documentation

- [Runtime primitives](docs/Primitives.md)
- [S-expression grammar](docs/S-Expressions.md)

Garbage collection relies entirely on the Go runtime—no additional memory management is required.

## Getting Started

### Prerequisites

- Go 1.25 or newer (the project targets Go 1.25.4)

### Build

```bash
go build
```

This produces the `gisp` executable in the repository root.

### Install

```bash
go install
```

or, equivalently:

```bash
./scripts/install.sh
```

The binary will be installed to `$GOBIN` (default: `$GOPATH/bin`).

### Run the REPL

```bash
./gisp
```

The REPL prints prompts (`gisp>`), evaluates expressions, and displays their results.

### Execute a Scheme Script

```bash
./gisp path/to/script.gs
```

Scripts may start with a Unix shebang (`#!/usr/bin/env gisp`) so they can be run directly when the
interpreter is on your `PATH`.

Passing `-` runs code from standard input.

## Examples

Example programs live in the `examples/` directory:

- `examples/hello.gs` – hello-world via `display` and `newline`
- `examples/continuation.gs` – demonstrates capturing and invoking continuations with `call/cc`

Run them with:

```bash
./gisp examples/hello.gs
./gisp examples/continuation.gs
```

Unit tests also execute these examples to ensure they stay in sync with the runtime.

## Testing

```bash
go test ./...
```

This runs unit tests for the interpreter core and confirms the example scripts evaluate successfully.

## Project Layout

```
.
├── examples/             # Sample Scheme programs
├── internal/
│   ├── lang/             # Runtime values, environments, evaluator
│   ├── reader/           # Tokenizer + s-expression reader
│   └── runtime/          # Primitives, library bootstrap, helpers
├── main.go               # CLI entry point / REPL
├── runtime_test.go       # End-to-end evaluator and example tests
├── go.mod                # Go module definition
└── LICENSE               # MIT License
```

## License

Distributed under the MIT License. See [`LICENSE`](LICENSE) for details.
