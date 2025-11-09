# Gisp Language

Gisp is a Go-flavoured programming language that targets the existing Scheme
runtime. It keeps the runtime semantics of the underlying evaluator—lexical
scope, first-class procedures, proper tail calls, and access to all existing
primitives—while providing a statement-oriented syntax that feels familiar to
Go developers.

## Key Features

- **Go-like syntax:** `func`, `if`, `while`, `return`, and braces for blocks.
- **Compiled to Scheme:** Source is translated into s-expressions before
  evaluation, so all existing primitives, macros, and libraries remain usable.
- **Tail-call optimisation:** Tail-recursive functions translate into the same
  Scheme constructs, inheriting TCO guarantees from the evaluator.
- **Inline s-expressions:** Use backtick literals to splice raw Scheme forms
  anywhere an expression is expected.

## Syntax Summary

- **Declarations:** `func`, `var`, and `const` at the top level.
- **Statements:** variable declarations, assignment, expression statements,
  `if`/`else`, `while`, and `return`. Every non-block statement must end with a
  semicolon.
- **Expressions:** infix operators `+`, `-`, `*`, `/`, `==`, `!=`, `<`, `<=`,
  `>`, `>=`, logical `&&`/`||`, unary `!` and unary negation. `==` compiles to
  the runtime numeric primitive `=` (and `!=` expands to `(not (= ...))`), so it
  expects numbers; use the `eq` and `equal` primitives via backticks when you
  need identity or structural comparison of non-numeric values.
- **Special forms:** `switch` expressions select the first truthy case and
  compile down to the runtime `cond`.
- **Literals:** numbers, strings, booleans (`true`/`false`), and list literals
  using `[a, b, ...]`.
- **Anonymous functions:** `func(params) { ... }` produces a closure with the
  same semantics as Scheme lambdas (including lexical scope and recursion).
- **Inline Scheme:** `` var quoted = `(list 1 2 3); `` inserts the exact
  s-expression `(list 1 2 3)` into the compiled output.

### Symbol Literals in Backticks

Inline s-expression literals are handed to the Scheme-style reader in `sexpr`, so all of Scheme's prefix sugar is available. A bare token like `` `+ `` reads as the symbol `+`, and `` `'+ `` expands to `(quote +)`. Prefer those forms over spelling out `(quote ...)` manually—for example, `cons(`'+, args)` is identical to `cons(`(quote +), args)` but shorter. We intentionally do **not** rewrite string literals such as `"+"` into symbols: strings are plain data, and automatic coercion would make it impossible to represent an actual string containing a plus sign. If you do need to turn a string into a symbol at runtime, use the existing `stringToSymbol` primitive instead of overloading the reader.

## Formal Grammar

The language borrows Go-style statements, blocks, and infix expressions while
preserving Scheme semantics. Source files may freely embed raw s-expressions
using backtick literals, which are delegated to the existing Scheme reader.

* Whitespace is insignificant except to separate tokens.
* Line (`// ...`) and block (`/* ... */`) comments are skipped.
* All statements that are not blocks must end with a semicolon (`;`).

```
Program        = { TopLevelDecl } ;

TopLevelDecl   = FuncDecl | VarDecl | ConstDecl | ExprStmt ;

FuncDecl       = "func" Identifier "(" [ ParamList ] ")" Block ;
ParamList      = Parameter { "," Parameter } ;
Parameter      = Identifier ;

VarDecl        = "var" Identifier [ "=" Expression ] ";" ;
ConstDecl      = "const" Identifier "=" Expression ";" ;

Block          = "{" { Statement } "}" ;

Statement      =
      VarDecl
    | ConstDecl
    | AssignStmt
    | ExprStmt
    | IfStmt
    | WhileStmt
    | ReturnStmt
    | Block
    ;

AssignStmt     = Identifier "=" Expression ";" ;
ExprStmt       = Expression ";" ;

IfStmt         = "if" Expression Block [ "else" Block ] ;
WhileStmt      = "while" Expression Block ;

ReturnStmt     = "return" [ Expression ] ";" ;

Expression     = OrExpr ;

OrExpr         = AndExpr { "||" AndExpr } ;
AndExpr        = EqualityExpr { "&&" EqualityExpr } ;
EqualityExpr   = RelationalExpr { EqualityOp RelationalExpr } ;
RelationalExpr = AddExpr { RelOp AddExpr } ;
AddExpr        = MulExpr { AddOp MulExpr } ;
MulExpr        = PrefixExpr { MulOp PrefixExpr } ;
PrefixExpr     = { PrefixOp } PostfixExpr ;
PostfixExpr    = PrimaryExpr { CallSuffix } ;

CallSuffix     = "(" [ ArgList ] ")" ;
ArgList        = Expression { "," Expression } ;

PrimaryExpr    = Identifier
               | Number
               | String
               | Boolean
               | ListLiteral
               | LambdaExpr
               | SwitchExpr
               | SExprLiteral
               | "(" Expression ")"
               ;
SwitchExpr     = "switch" "{" { SwitchClause } [ DefaultClause ] "}"
SwitchClause   = "case" Expression ":" Expression [ ";" ]
DefaultClause  = "default" ":" Expression [ ";" ]


LambdaExpr     = "func" "(" [ ParamList ] ")" Block ;
ListLiteral    = "[" [ ArgList ] "]" ;
SExprLiteral   = "`" SExpression ;

EqualityOp     = "==" | "!=" ;
RelOp          = "<" | "<=" | ">" | ">=" ;
AddOp          = "+" | "-" ;
MulOp          = "*" | "/" ;
PrefixOp       = "-" | "!" ;

Identifier     = letter { letter | digit | "_" } ;
Number         = digit { digit } [ "." digit { digit } ] ;
String         = "\"" { any_char_except_quote } "\"" ;
Boolean        = "true" | "false" ;
SExpression    = parsed by the Scheme reader (See `reader`).
```

## Examples

```gisp
func fact(n) {
	if n == 0 {
		return 1;
	}
	return n * fact(n - 1);
}

func fact_tr(n, acc) {
	if n == 0 {
		return acc;
	}
	return fact_tr(n - 1, acc * n);
}

var expr = `(map (lambda (x) (+ x 1)) '(1 2 3));
```

## Runtime Integration

- Files ending in `.gisp` are parsed with the Gisp syntax when loaded through
  `runtime.EvaluateFile`.
- `runtime.EvaluateGispString` and `runtime.EvaluateGispReader` provide direct
  helpers for evaluating Gisp snippets.
- The produced forms run through the same evaluator as raw s-expressions; new
  forms can seamlessly call existing primitives, macros, and libraries.

## Notes on Control Flow

`return` statements are implemented using continuations so they exit the nearest
containing function, matching the behaviour of Scheme's `call/cc`. `while` is
compiled into a recursive loop that preserves tail recursion, so `continue` and
`break` are not required; use conditionals and function exits as needed.

