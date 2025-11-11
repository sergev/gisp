# Regex Pattern Demo Documentation

## Engine Overview

The regex demo implements a Unix-style pattern matcher written entirely in Gisp. It understands literal characters, the dot wildcard, character classes (including ranges and negation), escaped metacharacters, start and end anchors, capture groups, alternation, and the usual quantifiers such as star, plus, question mark, and bounded repetition. The entire engine is self-contained, so every capability is expressed using core language constructs rather than delegating to a host library.

## Architecture

- **Parsing pipeline**: The `parseRegex` routine converts the pattern string into an abstract syntax tree. It advances through the pattern character by character, recognizing expressions separated by the alternation bar, breaking each expression into a sequence of factors, and assigning indices for capture groups as it proceeds. Any leftover characters after the parse trigger an error, ensuring the AST represents the complete pattern.
- **Atoms and quantifiers**: `parseFactor` reads an atom (literal, group, character class, dot, anchor, or escape) and immediately checks whether the atom is followed by a quantifier. The helper translates postfix operators into repeat nodes with explicit minimum and maximum counts. Numeric bounds are read by `parseQuantifier`, which validates the range and allows an open upper limit through the use of `-1` as “unbounded”.
- **Character classes**: `parseCharClass` walks the content inside square brackets. It supports an initial caret for negation, single characters, escaped characters, and hyphen-delimited ranges. Each element becomes either a single-character entry or a range entry in the AST so the matcher can evaluate membership efficiently.
- **Matching dispatcher**: `matchNode` is the central interpreter for the AST. It delegates to specialized functions for literals, wildcards, anchors, character classes, groups, repeats, sequences, and alternations. Every helper returns a list of possible match states (positions plus capture metadata), which enables the engine to handle branching constructs without backtracking stacks managed by the host language.
- **Repetition handling**: `repeatExtend` is the workhorse for quantifiers. Starting from the current position, it recursively grows the match by applying the body node, keeping track of how many repetitions have succeeded and whether additional growth is allowed. It adds results to the output both when the minimum count has been met and when additional expansions remain possible.
- **Capture bookkeeping**: Whenever a group succeeds, `matchGroupNode` records its span using `setCapture`. After the engine identifies a winning state, `buildCaptureValues` assembles the final list of substrings for group zero (the full match) and every numbered capture, returning them alongside the match boundaries.
- **Execution helpers**: `compileRegex` combines parsing with the creation of a compiled data structure that stores the original pattern, the AST, and the number of capture groups. `regexExecFrom` drives the matcher: it tries to match the AST at each position from a given start index and stops once it finds a successful state. `regexSearch`, `matchRegex`, and `regexFindAll` provide search, full-match, and global-find behaviors on top of that primitive.

## Demonstration Patterns

`runDemo` iterates through six illustrative patterns. For each one it compiles the pattern, performs a search against the sample subject string, prints the first match with capture details, then lists every match discovered across the input. The output showcases both how the engine interprets the pattern and how it reports results.

### `te.t` — literal with wildcard

- **Meaning**: Matches any four-character substring that begins with the letters “te”, allows any single character in the third position, and ends with “t”.
- **How the engine handles it**: The parser produces a sequence of literal nodes for “te”, followed by a dot node, and finally another literal node for “t”. The matcher checks each component in order, with the dot node accepting any single character as long as the subject has not ended.

### `^hello$` — anchored literal

- **Meaning**: Requires the subject to be exactly the word “hello” with nothing before or after it.
- **How the engine handles it**: The first atom becomes a start-anchor node, ensuring the match must begin at position zero. The final atom becomes an end-anchor node, forcing the match to stop at the end of the subject. Between them sits a literal node for “hello”, so only an exact, complete match succeeds.

### `[a-z]+` — character class with repetition

- **Meaning**: Consumes one or more lowercase ASCII letters in a row.
- **How the engine handles it**: Parsing yields a character-class node containing a range from “a” to “z”, wrapped in a repeat node whose minimum is one and whose maximum is unbounded. During matching, the engine repeatedly checks whether the current character falls within the class and keeps extending the span until the next character fails the test.

### `colou?r|colour` — alternation with optional character

- **Meaning**: Accepts either the spelling “color” (without the “u”) or “colour” (with the “u”).
- **How the engine handles it**: The parser creates an alternation with two branches. The first branch is a sequence where the letters “colo” are followed by an optional “u” (implemented as a repeat allowing zero or one occurrence) and then the letter “r”. The second branch is the straightforward literal sequence “colour”. The matcher evaluates both branches from the same starting position and combines the successful states.

### `(ha){2,4}` — bounded repetition of a group

- **Meaning**: Matches the syllable “ha” repeated at least twice and at most four times.
- **How the engine handles it**: The parentheses create a capture-group node whose body is the literal “ha”. That group sits inside a repeat node with a minimum of two and a maximum of four. The matcher records the span of each successful group iteration and recursively attempts to add more copies until the upper limit is reached or the subject no longer matches.

### `[^aeiou]+` — negated character class

- **Meaning**: Matches one or more consecutive characters that are not vowels.
- **How the engine handles it**: The parser detects the caret immediately after the opening bracket and marks the class as negated. The items inside the class are the literal vowels “a”, “e”, “i”, “o”, and “u”. When matching, the engine tests each candidate character against the class; because it is negated, the match proceeds only when the character is not one of the listed vowels. The surrounding plus quantifier keeps consuming characters until a vowel appears or the subject ends.

## Running the Demo

Run the script with the Gisp interpreter (for example, by executing the file directly if your shell can locate the `gisp` binary). The program prints a heading for each example, shows the first match with capture groups, and lists every match found in the subject string so you can see how the engine behaves in practice.
