# Gisp Runtime Primitives

This document summarizes the built-in primitives that are installed into the global environment. The semantics below reflect the current implementation in `runtime/primitives.go`.

## Arithmetic

- `+` — Adds its numeric arguments. Accepts any number of integers and reals; result type is integer if all inputs are integers, otherwise real.
- `-` — Subtracts subsequent numbers from the first. Unary form negates the single numeric argument. Mixed integer/real inputs promote to real.
- `*` — Multiplies numeric arguments. With no arguments the result is `1`. Mixed integer/real inputs promote to real.
- `/` — Divides the first numeric argument by each subsequent one. Unary form returns the reciprocal. Always returns a real. Division by zero raises an error.
- `%` — Calculates the remainder of integer division. Requires at least two integer arguments and applies left-to-right. Division by zero raises an error.
- `++`, `--` — Post-increment and post-decrement statements. Expect a single quoted symbol naming an existing numeric binding. They add or subtract 1 from either integers or reals (promoting integers when needed), store the updated value back into the same binding, and return the new value.
- `+=`, `-=`, `*=`, `/=`, `%=` — Compound numeric assignments. Expect two arguments: a quoted symbol naming an existing binding and a numeric delta. They read the current binding, apply the corresponding arithmetic primitive, store the result back into the same binding, and return the updated value.

## Bitwise and Shift Operators

- `&` — Bitwise AND across two or more integer arguments.
- `|` — Bitwise OR across two or more integer arguments.
- `^` — Bitwise XOR across one or more integer arguments. With a single argument it returns the bitwise complement; with multiple arguments it XORs left-to-right.
- `&^` — Bit clear. Takes two or more integers and applies Go-style bit clearing (`a &^ b`).
- `<<` — Left shift. Exactly two integer arguments: the value and a non-negative shift amount.
- `>>` — Right shift. Exactly two integer arguments: the value and a non-negative shift amount. Uses arithmetic shifting for signed integers.
- `<<=`, `>>=`, `&=`, `|=`, `^=`, `&^=` — Compound bitwise assignments. Like the arithmetic forms, they accept a quoted symbol naming the target binding and a single integer operand. They apply the corresponding bitwise or shifting primitive, mutate the binding in place, and return the updated value.

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
- `first` — Returns the first element of a pair. Errors if the argument is not a pair.
- `rest` — Returns the tail of a pair. Errors if the argument is not a pair.
- `setFirst` / `set-first!` — Mutates the first element of a pair. Takes the target pair and the new value, returning the updated pair. Errors if the first argument is not a pair.
- `setRest` / `set-rest!` — Mutates the tail of a pair. Takes the target pair and the new value, returning the updated pair. Errors if the first argument is not a pair.
- `list` — Builds a proper list from any number of arguments.
- `append` — Appends zero or more lists, with the last argument allowed to be any value. The final argument is returned as-is when earlier lists are exhausted. Non-list arguments before the final one raise an error.
- `length` — Returns the integer length of a proper list; errors on non-lists.

## Vector Operations

Vectors are fixed-length, mutable, zero-based indexed sequences. They complement lists when constant-time random access or in-place updates are required.

- `vector` — Allocates a fresh vector whose elements are the evaluated arguments. `(vector 1 2 3)` yields a three-element vector.
- `vectorp` — Predicate that returns `#t` when its argument is a vector.
- `makeVector` — `(makeVector n [fill])` creates a vector of length `n`. The optional fill value is evaluated once and written into each slot; it defaults to the empty list `()`. Length must be a non-negative integer that fits the host platform.
- `vectorLength` — Returns the integer length of a vector. Errors on non-vector input.
- `vectorRef` — `(vectorRef vec index)` returns the element at the given zero-based integer `index`. Out-of-range indices raise an error.
- `vectorSet` — `(vectorSet vec index value)` mutates the element at `index` to `value`, returning the same vector. Out-of-range indices raise an error.
- `vectorFill` — `(vectorFill vec value)` overwrites every element of `vec` with `value`, mutating in place and returning the vector.
- `vectorToList` — Converts a vector into a freshly allocated proper list containing the same elements.
- `listToVector` — Converts a proper list into a fresh vector. Non-lists raise an error.

Literal vectors use the reader notation `#(elem ...)`, which is sugar for calling `vector`.

## Control Flow

- `cond` — Evaluates each clause in order and returns the body from the first clause whose predicate is truthy. Clauses are pairs of predicate/body expressions. An optional final clause starting with the symbol `else` serves as a default. When no predicates succeed and no `else` clause is present, the result is the empty list.

## Equality Predicates

- `eq` — Identity comparison. For primitives, compares the underlying function pointer; for pairs and other compound types, checks pointer equality. Use this when you need reference equality from inline s-expressions.
- `equal` — Structural equality. Numbers of different exactness compare by value; pairs are traversed recursively. Reachable from Gisp via backticks when deep comparison is required.

## I/O and Process Control

- `display` — Prints the argument to standard output. Strings are printed raw; other values use their external representation. Returns the empty list.
- `newline` — Outputs a newline to standard output. Takes no arguments.
- `read` — Reads the next datum from standard input, returning parsed numbers, lists, symbols, etc. When the stream is exhausted it returns the EOF object.
- `exit` — Terminates the process. Optional single argument may be an integer exit code or boolean (`#t` → `0`, `#f` → `1`). More than one argument raises an error.

## Higher-Order Utilities

- `apply` — Applies a procedure to arguments. Takes the procedure, followed by zero or more direct arguments, ending with a list whose elements are appended to the call.
- `map` — Applies a procedure to each element of a list, returning a newly allocated list of results. Accepts two arguments: a procedure and a list. When the list is empty, the result is the empty list.
- `filter` — Retains the elements of a list for which the predicate returns a truthy value. Accepts a predicate procedure and a list, recursing through the list like `map` and returning a newly allocated list of matches. Empty inputs or all-false predicates yield the empty list.
- `gensym` — Generates a fresh symbol of the form `gN`. Takes no arguments.
- `randomInteger` — Returns a uniformly distributed integer in the half-open range `[0, limit)`. Requires a single positive integer argument.
- `randomSeed` — Resets the generator used by `randomInteger`. Takes a single integer seed and returns the empty list.

## String and Symbol Operations

- `stringLength` — Returns the length of a string. Errors on non-string input.
- `makeString` — Builds a new string of a given non-negative length. An optional single-character string supplies the fill character (defaults to a space). Errors on non-integer lengths, negative lengths, non-string fills, or fill strings longer than one character.
- `stringAppend` — Concatenates string arguments. Non-string arguments raise a type error.
- `stringSlice` — Extracts a substring using zero-based indices. Takes a string, a start index, and an optional end index (defaulting to the string length). Indices must be integers within bounds; the end must not precede the start.
- `symbolToString` — Converts a symbol to a string. Requires exactly one symbol argument.
- `stringToSymbol` — Interns a string as a symbol. Requires exactly one string argument.
- `numberToString` — Converts an integer or real to its textual representation.
- `stringToNumber` — Parses a string into an integer or real. Returns `#f` if parsing fails or string is empty after trimming.
