## Overview

This example implements a Refal-style pattern matcher in Gisp. It recreates the familiar categories of Refal variables—single-symbol, generic term, word, and expression—and demonstrates how they interact with bracketed substructures and ordered clause evaluation. The file builds the matcher from first principles: list utilities, pattern descriptors, an environment for bindings, and a backtracking matcher that enumerates all consistent assignments.

## Pattern Building Blocks

- **`S` pattern (single-symbol variable)**: Represents Refal’s single-letter variable that may bind to exactly one atomic symbol. The matcher checks the next subject element; if it is a symbol, it records the binding and proceeds. If the symbol has already been bound, the matcher demands the same value before continuing. Implementation corresponds to `matchVariable` when the variable descriptor carries the single-symbol tag.
- **`T` pattern (generic term variable)**: Matches a single term that can be either a symbol or an entire bracketed expression. The matcher pulls one element, binds it (with consistency checks for repeat uses), and advances. This occurs in the `matchVariable` branch for the term tag, allowing nested lists to bind wholesale without inspecting their internal structure.
- **`W` pattern (word variable)**: Captures a non-empty sequence of contiguous symbols. The matcher counts how many symbols occur before any bracketed subexpression and tries every positive-length prefix, binding each candidate and recursively matching the remainder. This backtracking allows multiple matches when the pattern is ambiguous within the word prefix.
- **`E` pattern (expression variable)**: Covers any list fragment, including empty prefixes and combinations containing nested brackets. The matcher computes the length of the remaining subject and attempts every possible suffix split, binding the chosen prefix and exploring the rest recursively. Because it tries the longest fragment first, it mirrors Refal’s default greedy search before backtracking to shorter options.
- **Bracket patterns**: Refal treats brackets as structural delimiters. The helper `Br` constructs a pattern that expects a bracketed subexpression. When the matcher encounters a bracket descriptor, it requires the subject term to be a list, recursively matches its contents, and then continues with the outer sequence. The recursion ensures that nested structures are handled with the same variable semantics as the top level.

## Binding Environment

- **Recording assignments**: The environment is a list of triples storing the variable type, variable name, and bound value. New bindings are added by `bindValue`, which first looks for an existing entry. If none exists, it prepends the new binding; if the name already has a value, equality is required, otherwise the match fails.
- **Consulting bindings**: Functions such as `lookupBinding` and `displayEnv` retrieve the stored values. They support repeated variables and the demo routines, ensuring the same symbol or subexpression is enforced when a variable name reappears.

## Matching Workflow

- **Sequencing**: `matchSequence` iterates through the pattern list, invoking `matchVariable` for variable descriptors, recursing for bracketed segments, or expecting literal symbols otherwise. Successful completion requires both the pattern and the subject to be exhausted simultaneously.
- **Enumerating solutions**: `refalMatchAll` returns every consistent environment, while `refalMatchFirst` selects the first match (or reports failure). Backtracking emerges from the variable handlers, which explore alternative splits for `W` and `E` variables.

## Demonstrations

- **Word split**: Shows an `S` variable capturing the leading symbol, a `W` variable absorbing the rest of the word prefix, and a literal suffix. It highlights that word variables stop before bracketed structures and that single-symbol variables consume only one symbol.
- **Bracket match**: Demonstrates combining bracket patterns with `S` and `E` variables. The matcher dives into a nested expression, binds its parts, and leaves the trailing list tail for the expression variable.
- **Repeated variable**: Confirms that reusing the same `S` name requires identical subject values. The example also shows the negative case where mismatched symbols cause the match to fail without any bindings.
- **Backtracking**: Uses an `E` variable on both sides of a literal to illustrate that the matcher first binds the longest possible prefix and then shortens it to produce additional solutions.
- **Clause evaluation**: Emulates Refal’s ordered rules. Each clause pairs a pattern with a handler; the matcher tries them in sequence, invoking the handler of the first successful pattern. The factorial outline shows the base case and the recursive step returning a rewritten expression.
