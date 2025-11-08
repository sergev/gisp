# Gisp S-Expression Grammar

This document describes the concrete syntax accepted by the reader implementation in `reader/reader.go`. The language is an S-expression dialect inspired by Scheme.

## Lexical Structure

- **Whitespace** — Any Unicode space characters separate tokens. Newlines are whitespace.
- **Comments** — A semicolon `;` starts a line comment that runs to the end of the line. Comments may appear between forms.
- **Delimiters** — Parentheses `(` `)` delimit lists. A dot `.` inside a list introduces a dotted pair.
- **Dispatch Prefix** — A leading `#` introduces special tokens (`#t`, `#f`).
- **Quote Prefixes** — The single quote `'`, backtick `` ` ``, and comma `,` (optionally followed by `@`) expand into list forms (see below).

## Grammar Overview

The reader parses a sequence of expressions. In EBNF-like notation:

```
program    ::= { expression }
expression ::= list
             | quoted
             | boolean
             | string
             | number
             | symbol
```

### Lists and Pairs

```
list       ::= "(" list-contents ")"
list-contents
            ::= /* empty */                      ; the empty list
             | expression { expression }         ; proper list
             | expression { expression } "." expression
```

- Proper lists yield nested pairs ending in the empty list.
- Dotted lists allow the final cdr to be any expression. Only a single dot is permitted, and it must be followed by exactly one expression before the closing `)`.

### Quoting Forms

```
quoted     ::= "'" expression            ; expands to (quote expression)
             | "`" expression            ; expands to (quasiquote expression)
             | "," expression            ; expands to (unquote expression)
             | ",@" expression           ; expands to (unquote-splicing expression)
```

The reader rewrites each prefixed form into the corresponding list with a leading symbol (`quote`, `quasiquote`, `unquote`, or `unquote-splicing`).

### Booleans

```
boolean    ::= "#t" | "#f"
```

No other dispatch sequences are recognized; encountering `#` followed by another rune is an error.

### Strings

```
string     ::= '"' { character | escape } '"'
escape     ::= "\" ("n" | "t" | "\" | '"' | any-rune)
```

- Strings may contain standard escapes for newline (`\n`), tab (`\t`), backslash (`\\`), and double quote (`\"`). Any other escaped rune is included verbatim.
- Unterminated strings or escape sequences raise errors.

### Numbers

```
number     ::= integer | real
integer    ::= [+-]? digits
real       ::= [+-]? digits "." digits [ exponent ]
             | [+-]? digits exponent
exponent   ::= ("e" | "E") [+-]? digits
```

The reader delegates to Go's `strconv` routines:

- Tokens parsed by `strconv.ParseInt` become integers.
- Tokens parsed by `strconv.ParseFloat` become reals.
- Non-numeric tokens fall through to symbols.

### Symbols

```
symbol     ::= token
token      ::= character { character }
character  ::= any-rune except whitespace, '(', ')', '"', ';'
```

Symbols are case-sensitive and may include punctuation other than the reserved characters above.

## Error Conditions

The reader reports errors for:

- Unexpected `)` outside a list.
- Unterminated lists, strings, or dotted lists.
- Misplaced dots (e.g., leading dot, multiple dots, or dot not followed by an expression).
- Unknown dispatch sequences after `#`.
- Empty tokens (e.g., due to adjacent delimiters with no content).

These errors surface as Go `error` values from the reader functions.
