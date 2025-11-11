# Gisp Examples

This directory collects runnable examples, ports, and supporting notes that showcase different parts of the Gisp language and runtime.

## Tutorial Series

- `tutorial_01_hello.gisp` — print a greeting; first contact with `display`.
- `tutorial_02_while_loop.gisp` — introduce `while` loops and simple counters.
- `tutorial_03_circle_area.gisp` — work with numeric expressions and constants.
- `tutorial_04_strings_booleans.gisp` — demonstrate string literals and boolean flow.
- `tutorial_05_counter_closure.gisp` — build closures that capture mutable state.
- `tutorial_06_iterate.gisp` — write higher-order functions and pass callbacks.
- `tutorial_07_list_helpers.gisp` — implement recursive list utilities (`map`, `filter`).
- `tutorial_07_vectors.gisp` — use vectors and index-based loops via a sieve of Eratosthenes.
- `tutorial_08_unless_macro.gisp` — define and invoke a macro using backquoted forms.
- `tutorial_09_compose.gisp` — compose functions and invoke the result immediately.
- `tutorial_10_abs.gisp` — revisit conditional expressions in a reusable helper.
- `tutorial_11_running_average.gisp` — compute running averages over a list.
- `tutorial_12_sqrt_newton.gisp` — approximate square roots with Newton iteration.
- `tutorial_13_adaptive_trapezoid.gisp` — perform adaptive numerical integration.
- `tutorial_14_symbolic_diff.gisp` — differentiate symbolic expressions (SICP-inspired).
- `tutorial_15_callcc_find_first.gisp` — find the first matching item with `callcc`.
- `tutorial_16_zscore_pipeline.gisp` — calculate z-scores and flag outliers.
- `tutorial_17_classify.gisp` — nested `if`/`else` for multi-branch decisions.
- `tutorial_18_accumulate_until.gisp` — accumulate series terms until a threshold.

## Language Ports and Benchmarks

- `continuation.gisp` — continuation demo showing how to capture and resume with `callcc`.
- `continuation.gs` — same continuation example using s-expression syntax.
- `fact.gisp` — factorial calculation in both recursive and tail-recursive styles.
- `gc_stress.gisp` — allocation-heavy benchmark covering lists, closures, and symbols.
- `gc_stress.scm` — original Scheme benchmark source for comparison.
- `mceval.gisp` — SICP’s metacircular evaluator adapted to Gisp syntax.
- `mceval.gs` — the evaluator in its original Lisp-style notation.
- `sierpinski.gisp` — render a Sierpiński triangle with recursive string assembly.
- `sierpinski.scm` — Scheme counterpart of the Sierpiński example.
- `maze.gisp` — randomized depth-first maze generator that emits Unicode art.
- `puzzle15.gisp` — solver utilities for the classic sliding 15-puzzle.

## Pattern-Matching Examples

- `regex_patterns.gisp` — showcase Gisp’s regex library with composable matchers.
- `regex_patterns.md` — walkthrough of the regex example scenarios.
- `refal_patterns.gisp` — emulate REFAL-style pattern rewrites in Gisp.
- `refal_patterns.md` — notes explaining the REFAL translation.
- `snobol_patterns.gisp` — port of SNOBOL pattern matching idioms.
- `snobol_patterns.md` — commentary on the SNOBOL-inspired approach.

> All `.gisp` files can be executed directly (`./filename.gisp`) when marked executable, or via `gisp examples/<name>.gisp`. `.gs` variations keep the original Scheme-style surface syntax for comparison.

