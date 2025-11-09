# Gisp Runtime Primitives

This document summarizes the built-in primitives that are installed into the global environment. The semantics below reflect the current implementation in `runtime/primitives.go`.

## Arithmetic

- `+` — Adds its numeric arguments. Accepts any number of integers and reals; result type is integer if all inputs are integers, otherwise real.
- `-` — Subtracts subsequent numbers from the first. Unary form negates the single numeric argument. Mixed integer/real inputs promote to real.
- `*` — Multiplies numeric arguments. With no arguments the result is `1`. Mixed integer/real inputs promote to real.
- `/` — Divides the first numeric argument by each subsequent one. Unary form returns the reciprocal. Always returns a real. Division by zero raises an error.

## Numeric Comparisons

- `=` — Numeric equality across integers and reals; accepts any number of arguments. Returns `#t` for zero or one argument. The Gisp surface operator `==` compiles directly to this primitive (and `!=` expands to `(not (= ...))`), so it inherits the requirement that all arguments are numeric.
- `<`, `<=`, `>`, `>=` — Chainable numeric comparisons. Non-numeric arguments raise a type error. Zero or one argument returns `#t`.

## Boolean Logic

- `not` — Unary logical negation. Treats values using the evaluator truthiness (`#f` only is false) and returns a boolean.

## Type Predicates

All of the following expect exactly one argument and return `#t` or `#f`. Predicate names now use the Common Lisp-style `p` suffix instead of Scheme punctuation (for example, `null?` became `nullp`).

- `numberp` — True for integers or reals.
- `integerp` — True for integers.
- `realp` — True for reals or integers.
- `booleanp` — True for booleans.
- `stringp` — True for strings.
- `symbolp` — True for symbols.
- `pairp` — True for pairs (cons cells).
- `nullp` — True for the empty list.
- `listp` — True if the argument can be viewed as a proper list (`lang.ToSlice` succeeds).
- `procedurep` — True for primitives, closures, or continuations.

## List Construction and Access

- `cons` — Constructs a pair from two arguments.
- `car` — Returns the first element of a pair. Errors if the argument is not a pair.
- `cdr` — Returns the tail of a pair. Errors if the argument is not a pair.
- `setCar` / `set-car!` — Mutates the first element of a pair. Takes the target pair and the new value, returning the updated pair. Errors if the first argument is not a pair.
- `setCdr` / `set-cdr!` — Mutates the tail of a pair. Takes the target pair and the new value, returning the updated pair. Errors if the first argument is not a pair.
- `list` — Builds a proper list from any number of arguments.
- `append` — Appends zero or more lists, with the last argument allowed to be any value. The final argument is returned as-is when earlier lists are exhausted. Non-list arguments before the final one raise an error.
- `length` — Returns the integer length of a proper list; errors on non-lists.

## Control Flow

- `cond` — Evaluates each clause in order and returns the body from the first clause whose predicate is truthy. Clauses are pairs of predicate/body expressions. An optional final clause starting with the symbol `else` serves as a default. When no predicates succeed and no `else` clause is present, the result is the empty list.

## Equality Predicates

- `eq` — Identity comparison. For primitives, compares the underlying function pointer; for pairs and other compound types, checks pointer equality. Use this when you need reference equality from inline s-expressions.
- `equal` — Structural equality. Numbers of different exactness compare by value; pairs are traversed recursively. Reachable from Gisp via backticks when deep comparison is required.

## I/O and Process Control

- `display` — Prints the argument to standard output. Strings are printed raw; other values use their external representation. Returns the empty list.
- `newline` — Outputs a newline to standard output. Takes no arguments.
- `exit` — Terminates the process. Optional single argument may be an integer exit code or boolean (`#t` → `0`, `#f` → `1`). More than one argument raises an error.

## Higher-Order Utilities

- `apply` — Applies a procedure to arguments. Takes the procedure, followed by zero or more direct arguments, ending with a list whose elements are appended to the call.
- `gensym` — Generates a fresh symbol of the form `gN`. Takes no arguments.

## String and Symbol Operations

- `stringAppend` — Concatenates string arguments. Non-string arguments raise a type error.
- `symbolToString` — Converts a symbol to a string. Requires exactly one symbol argument.
- `stringToSymbol` — Interns a string as a symbol. Requires exactly one string argument.
- `numberToString` — Converts an integer or real to its textual representation.
- `stringToNumber` — Parses a string into an integer or real. Returns `#f` if parsing fails or string is empty after trimming.
