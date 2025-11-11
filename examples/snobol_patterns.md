Snobol Pattern Toolkit in Gisp
==============================

This document explains how `examples/snobol_patterns.gisp` recreates classical Snobol4 pattern matching using the Gisp language. The program builds a library of pattern constructors, a backtracking matcher, and several demonstrations. Every pattern works by consuming characters from a subject string and returning a list of potential match states, each state describing how far the match progressed and what substrings were captured along the way.

Match States and Utilities
--------------------------

Before describing the patterns, the file sets up helpers that behave like Snobol’s internal machinery. A match state is a pair containing the current cursor position within the subject string and a list of captured name–value pairs. List helpers provide deterministic ordering, and string helpers expose slicing and character set membership. These foundations let each pattern focus purely on how it advances the cursor.

Elementary Snobol Patterns
--------------------------

The core of the file mirrors Snobol’s built-in patterns. Each constructor returns a function that receives the subject string, the current position, and the capture list, then produces zero or more new states.

* `lit` (literal): Succeeds only when the next characters in the subject exactly match the given text. Internally it slices the subject at the current position, compares the slice to the literal, and advances the cursor by the literal’s length on success.
* `any`: Accepts a single character belonging to a supplied set. It reads the current character, checks whether the character appears in the set, and advances by one when the check passes.
* `notAny`: The complement of `any`. It succeeds when the next character is absent from the provided set, again moving the cursor forward by one. Failure occurs either at end-of-string or when a character is inside the set.
* `span`: Consumes one or more consecutive characters that all belong to the given set. It repeatedly advances until it reaches a character outside the set, failing if nothing was consumed, and returns the state positioned after the span.
* `breakSet`: The Snobol `BREAK` primitive. It advances until it encounters a character that *is* inside the supplied set, and it fails if the first character already belongs to the set. Effectively it captures the longest prefix composed of characters outside the terminator set.
* `len`: Models Snobol’s `LEN(n)`. It simply advances the cursor by an exact count when enough characters remain; otherwise it fails. Negative lengths are rejected immediately.
* `arb`: Replicates Snobol’s nondeterministic `ARB`. Instead of choosing a single length greedily, it enumerates every possible end position from the current cursor through the end of the subject, returning a state for each option. This allows later combinators to backtrack over all lengths.
* `rem`: Implements `REM`, meaning “the remainder of the string.” It always succeeds and jumps the cursor directly to the subject’s length.
* `pos`: Matches only when the current cursor equals a required absolute index. It neither consumes characters nor alters captures; it functions as a positional guard.
* `rpos`: The symmetrical “reverse position” check. It verifies that the number of characters remaining equals the expected value, enforcing an anchor measured from the end.

Pattern Combinators
-------------------

Beyond the elementary primitives, the file defines machinery that assembles them into larger patterns:

* `opt`: A direct translation of Snobol optionals. It yields both the untouched state and every state produced by the optional pattern, effectively making the component optional during backtracking.
* `alt`: Disjunction over a list of alternatives. For each candidate pattern it gathers all resulting states, concatenates them, and hands the combined list back for further exploration.
* `seq`: Sequential composition. It threads the set of current states through each pattern in turn, much like Snobol’s concatenation operator. Failure of any component prunes the path.
* `arbno`: Snobol’s `ARBNO`. The implementation repeatedly applies the given pattern as long as it continues to consume characters. It uses a queue so that each accepted repetition can lead to yet another attempt, emitting every stopping point including the zero-length case.
* `capture`: Adds semantically named captures. Whenever the wrapped pattern succeeds, it slices the subject from the starting position to the new cursor and stores the fragment under the provided symbol before returning the updated state.
* `guard`: Filters matches with an extra predicate, similar to Snobol pattern predicates. After the inner pattern succeeds, the guard inspects the consumed substring and keeps only the states for which the custom predicate returns true.

Matching Engine
---------------

The runtime collects states into user-friendly match records. `matchFirst` walks the subject string from left to right, feeds the pattern into each starting position, and picks the best result (longest advance, then richest capture set). `matchAll` gathers every possible match. These routines reconstruct Snobol’s backtracking search, because each pattern returns multiple states and the engine explores them in order.

Demonstrations
--------------

Three demo functions show how the pieces come together:

* Syllable splitting combines `span`, `any`, and `rem` with `capture` to divide a word into onset, nucleus, and coda captures.
* Configuration parsing uses `breakSet`, `span`, `opt`, and `arbno` to read repeated key–value pairs separated by semicolons, later assembling capture lists into structured results.
* Log parsing applies `pos`, `span`, `capture`, `guard`, `len`, and `rem` to validate an ISO-like date inside a log entry and expose the captured fields.

Together, these examples confirm that the elementary Snobol patterns—and the combinators that compose them—faithfully reproduce Snobol4-style pattern matching behavior within pure Gisp.
