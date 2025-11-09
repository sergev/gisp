# Gisp Tutorial

This tutorial walks from the first `display` call through macro metaprogramming and numerical experiments, showing how Gisp's Go-flavored surface maps to the Scheme core that powers the interpreter. Along the way we port several classic Scheme programs, most of them popularised by *Structure and Interpretation of Computer Programs* (SICP), into idiomatic Gisp so you can see how the language handles both floating-point computation and symbolic manipulation.

Use this document side by side with `docs/Language.md`, `docs/Primitives.md`, and the examples in `examples/` if you want more precise reference material.

---

## 1. Getting Started Quickly

### Installing and Building

Gisp targets Go 1.25.4 or newer. Clone the repository and build the interpreter:

```bash
git clone https://github.com/whatever/gisp.git
cd gisp
make            # builds ./gisp
make install    # optional: installs to $GOBIN
```

### Running the REPL

Start the read-eval-print loop with:

```bash
./gisp
```

The prompt is `gisp>`. Enter expressions or definitions, press Enter, and the interpreter prints the result. Use `Ctrl+D` (Unix) or `Ctrl+Z` (Windows) to exit, or run `(exit)`/`exit()` from a script.

### Running Scripts

Gisp understands both `.gs` (s-expression syntax) and `.gisp` files:

```bash
./gisp examples/hello.gs
./gisp examples/fact.gisp
```

Scripts may start with `#!/usr/bin/env gisp`, letting you run them directly once the binary is on your `PATH`.

---

## 2. First Steps in Gisp

### Hello, Gisp

Save the following as `hello.gisp` and execute it with `./gisp hello.gisp`:

```gisp
func main() {
	display("Hello from Gisp!");
	newline();
}

main();
```

- `func` declares a function.
- `display` and `newline` come from the runtime primitives.
- Bare expressions at top level (like `main();`) are allowed, so there is no implicit `main` entry point.

### Statements and Semicolons

Every non-block statement ends with a semicolon. Blocks use braces:

```gisp
func main() {
	var total = 0;
	while total < 5 {
		total = total + 1;
	}
	display(total);
	newline();
}

main();
```

Whitespace is mostly insignificant, and single-line (`//`) plus block (`/* ... */`) comments work the same way as in Go.

### Numeric Types and Floating-Point Arithmetic

Gisp numbers are either 64-bit integers or IEEE 754 doubles (`float64`). Arithmetic primitives promote automatically when a real number is involved, and `/` always returns a real. Try the following in the REPL:

```gisp
var radius = 2.5;
const pi = 3.141592653589793;
var area = pi * radius * radius;

display("Circle area: ");
display(area);
newline();

var ratio = 7 / 2;
display("Ratio 7/2 = ");
display(ratio);
newline();
```

Use scientific notation (`1.2e-3`) for very small or large values. The parser tests cover integers, decimals, and exponent forms, so you get the same behaviour in scripts and the REPL.

### Strings and Booleans

Strings use double quotes and support standard escape sequences (`\n`, `\t`, `\\`, `\"`). Booleans are `true` and `false`, and only `false` is considered falsy at runtime.

```gisp
func main() {
	var greeting = "Line one\nLine two";
	var excited = true;

	if excited {
		display(greeting);
		newline();
	}
}

main();
```

### Variables and Lexical Scope

`var` introduces a mutable binding, `const` an immutable one. Functions capture values lexically (closures):

```gisp
func counter(start) {
	var value = start;
	return func(step) {
		value = value + step;
		return value;
	};
}

var c = counter(10);
display(c(1));  // 11
display(c(5));  // 16
newline();
```

---

## 3. Control Flow and Functions

### Conditionals

`if` statements require a block. Chained conditions use `else`:

```gisp
func classify(n) {
	if n < 0 {
		return "negative";
	} else {
		if n == 0 {
			return "zero";
		}
	}
	return "positive";
}

func main() {
	display(classify(-3));
	display(classify(0));
	display(classify(8));
	newline();
}

main();
```

### Loops

`while` compiles to tail-recursive Scheme loops, so recursion depth is not a concern:

```gisp
func accumulateUntil(limit) {
	var sum = 0.0;
	var term = 1.0;

	while sum < limit {
		sum = sum + term;
		term = term / 2.0;
	}
	return sum;
}

display(accumulateUntil(2.0));
newline();
```

### Higher-Order Functions

Functions are first-class values. The following implements `iterate`, a helper that repeatedly applies a function:

```gisp
func iterate(fn, value, count) {
	var result = value;
	var remaining = count;

	while remaining > 0 {
		result = fn(result);
		remaining = remaining - 1;
	}
	return result;
}

var double = func(x) { return x * 2; };
display(iterate(double, 1, 5)); // 32
newline();
```

---

## 4. Lists, Data, and Runtime Primitives

List literals (`[a, b, c]`) are syntactic sugar for `list` in Scheme. You can interoperate with list primitives directly:

```gisp
var numbers = [1, 2, 3, 4];
display(car(numbers)); // 1
display(cdr(numbers)); // (2 3 4)
newline();
```

Predicate names now follow the trailing `p` convention (`nullp`, `numberp`, etc.), so you can call them directly from Gisp:

```gisp
func isEmpty(xs) {
	return nullp(xs);
}

func head(xs) {
	return car(xs);
}

func tail(xs) {
	return cdr(xs);
}
```

With these helpers you can re-create familiar library procedures:

```gisp
func map(fn, xs) {
	if isEmpty(xs) {
		return [];
	}
	return cons(fn(head(xs)), map(fn, tail(xs)));
}

func filter(pred, xs) {
	if isEmpty(xs) {
		return [];
	}
	var first = head(xs);
	var rest = tail(xs);

	if pred(first) {
		return cons(first, filter(pred, rest));
	}
	return filter(pred, rest);
}

display(map(func(n) { return n * n; }, numbers));
newline();
```

---

## 5. Inline S-Expressions and Macros

Anything wrapped in backticks is copied straight into the Scheme expansion, which means you can reach every primitive or macro facility. Define higher-order utilities or macros once and reuse them from Gisp:

```gisp
`(define-macro (unless condition . body)
	(list 'if condition '#f (cons 'begin body)));

func demo(value) {
	`(unless value
		(begin (display "value was false")
		       (newline)));
}

func main() {
	demo(false);
	demo(true);
}

main();
```

You can also splice in existing Scheme libraries:

```gisp
var compose = `(lambda (f g)
	(lambda (x) (f (g x))));

var increment = func(n) { return n + 1; };
var double = func(n) { return n * 2; };

display(compose(increment, double)(10)); // 21
newline();
```

---

## 6. Numeric Techniques with Floating Point

This section focuses on careful floating-point work. The helper below mirrors Scheme's `(abs)` for reals:

```gisp
func abs(x) {
	if x < 0 {
		return -x;
	}
	return x;
}

display(abs(-3.5));
display(abs(2));
newline();
```

### Running Averages

```gisp
func isEmpty(xs) {
	return nullp(xs);
}

func head(xs) {
	return car(xs);
}

func tail(xs) {
	return cdr(xs);
}

func runningAverage(samples) {
	var sum = 0.0;
	var count = 0;
	var totals = [];
	var cursor = samples;

	while !isEmpty(cursor) {
		sum = sum + head(cursor);
		count = count + 1;
		totals = append(totals, [sum / count]);
		cursor = tail(cursor);
	}
	return totals;
}

display(runningAverage([10.0, 12.0, 18.0, 20.0]));
newline();
```

Because division always returns a real, the averages stay in floating-point even though `count` is an integer.

---

## 7. Scheme Classics Rewritten in Gisp

Each program below starts life as a Scheme example and is rewritten in Gisp. All of them run unmodified in the interpreter. Comments highlight where the Gisp version diverges from the original.

The `examples/sierpinski.gisp` port shows how the new `stringLength` and `makeString` primitives let us carry over string-heavy Scheme programs (like the classic Sierpiński triangle) without rethinking the original recursion.

### 7.1 Newton's Method for Square Roots (SICP Section 1.1)

The SICP procedure iteratively improves a guess `g` using `(average g (/ x g))` until the square is close enough. In Gisp we translate the structure while keeping the floating-point behaviour explicit:

```gisp
func average(a, b) {
	return (a + b) / 2.0;
}

func improve(guess, x) {
	return average(guess, x / guess);
}

func goodEnough(guess, x, tolerance) {
	return abs(guess * guess - x) < tolerance;
}

func sqrtNewton(x) {
	if x == 0 {
		return 0;
	}

	var guess = x / 2.0;
	const tolerance = 1e-9;

	while true {
		var next = improve(guess, x);
		if goodEnough(next, x, tolerance) {
			return next;
		}
		guess = next;
	}
}

display(sqrtNewton(2));
newline(); // ~1.414213562373095
```

Promotion rules let us mix integer literals (like `2`) with floating-point ones; all intermediate calculations happen in double precision.

### 7.2 Adaptive Trapezoidal Integration (inspired by SICP Section 1.3)

SICP introduces numerical integration using Simpson's rule. Without a modulus operator in Gisp we adapt the scheme to the trapezoidal rule and add adaptive refinement based on error estimates.

```gisp
func trapEstimate(f, a, b) {
	return (f(a) + f(b)) * (b - a) / 2.0;
}

func adaptiveTrap(f, a, b, tolerance) {
	var mid = (a + b) / 2.0;
	var left = trapEstimate(f, a, mid);
	var right = trapEstimate(f, mid, b);
	var together = trapEstimate(f, a, b);

	if abs((left + right) - together) < 3 * tolerance {
		return left + right;
	}
	return adaptiveTrap(f, a, mid, tolerance / 2.0) +
		adaptiveTrap(f, mid, b, tolerance / 2.0);
}

func integrate(f, a, b, tolerance) {
	return adaptiveTrap(f, a, b, tolerance);
}

func unitCircle(x) {
	return 4.0 / (1 + x * x);
}

display(integrate(unitCircle, 0.0, 1.0, 1e-6));
newline(); // ~3.141592653589793 (pi via quarter-circle area)
```

The recursive subdivision mirrors the Scheme version, and the tolerance handling reinforces that Gisp comfortably handles real-valued recursion. Integrating the quarter-circle density `4/(1 + x^2)` over `[0, 1]` reproduces pi with the requested accuracy, making this an approachable floating-point benchmark.

### 7.3 Symbolic Differentiation (SICP Section 2.3)

SICP's differentiator manipulates algebraic expressions represented as lists. The Gisp port stays close to the original, leaning on the new predicate names so we can stay in surface Gisp while still reaching into list structures.

```gisp
func isNumberValue(expr) {
	return numberp(expr);
}

func isVariable(expr) {
	return symbolp(expr);
}

func sameVariable(v1, v2) {
	return eq(v1, v2);
}

func isPair(expr) {
	return pairp(expr);
}

func isTagged(expr, tag) {
	return isPair(expr) && eq(car(expr), tag);
}

func makeSum(a, b) {
	if isNumberValue(a) && a == 0 {
		return b;
	}
	if isNumberValue(b) && b == 0 {
		return a;
	}
	return cons(`'+ , [a, b]);
}

func makeProduct(a, b) {
	if (isNumberValue(a) && a == 0) || (isNumberValue(b) && b == 0) {
		return 0;
	}
	if isNumberValue(a) && a == 1 {
		return b;
	}
	if isNumberValue(b) && b == 1 {
		return a;
	}
	return cons(`'* , [a, b]);
}

func makeExponent(base, exponent) {
	return cons(`'pow , [base, exponent]);
}

func sumAddend(expr) {
	return `(car (cdr expr));
}

func sumAugend(expr) {
	return `(car (cdr (cdr expr)));
}

func productMultiplier(expr) {
	return `(car (cdr expr));
}

func productMultiplicand(expr) {
	return `(car (cdr (cdr expr)));
}

func exponentBase(expr) {
	return `(car (cdr expr));
}

func exponentPower(expr) {
	return `(car (cdr (cdr expr)));
}

func deriv(expr, variable) {
	if isNumberValue(expr) {
		return 0;
	}
	if isVariable(expr) {
		if sameVariable(expr, variable) {
			return 1;
		}
		return 0;
	}
	if isTagged(expr, `'+ ) {
		return makeSum(
			deriv(sumAddend(expr), variable),
			deriv(sumAugend(expr), variable)
		);
	}
	if isTagged(expr, `'* ) {
		return makeSum(
			makeProduct(
				deriv(productMultiplier(expr), variable),
				productMultiplicand(expr)
			),
			makeProduct(
				productMultiplier(expr),
				deriv(productMultiplicand(expr), variable)
			)
		);
	}
	if isTagged(expr, `'pow ) {
		var base = exponentBase(expr);
		var power = exponentPower(expr);
		return makeProduct(
			makeProduct(power, deriv(base, variable)),
			makeExponent(base, makeSum(power, -1))
		);
	}
	return cons(`'unknown-derivative , [expr]);
}

var expression = cons(`'* , [
	makeSum(`'x , 1),
	makeExponent(`'x , 3),
]);

display(deriv(expression, `'x ));
newline();
```

The emphasis is on reusing Scheme's list selectors (`cadr`, `caddr`) via inline calls while the control flow (conditionals, helper functions, and returns) is entirely Gisp. The end result prints the list representation of the derivative; unknown patterns fall back to tagging the original expression with the symbol `unknown-derivative`, which you can later hook into your own error-reporting pipeline.

### 7.4 Escape Continuations for Tree Search (inspired by Friedman & Felleisen)

One of Scheme's signature features is `call/cc`. We can reproduce the classic "search for a solution and exit early" example in Gisp:

```gisp
var findFirst = `(lambda (pred xs)
	(call/cc (lambda (exit)
		(let loop ((items xs))
			(if (nullp items)
				'#f
				(if (pred (car items))
					(exit (car items))
					(loop (cdr items))))))));

func main() {
	var firstLarge = findFirst(func(n) { return n > 10; }, [1, 3, 5, 8, 13, 21]);
	display(firstLarge);
	newline();
}

main();
```

Here we embed a short Scheme loop that uses `call/cc` to escape. The predicate itself is a Gisp closure, showing how values move freely between the two syntaxes. This pattern is particularly useful when porting Scheme code that already relies on continuations.

---

## 8. Putting It All Together: A Mini Data Pipeline

The following full script pulls together control flow, floating-point math, list processing, and a macro. It produces a moving z-score, filtering the stream whenever the absolute deviation crosses a threshold.

```gisp
`(define-macro (when condition . body)
	(list 'if condition (cons 'begin body) '#f));

func abs(x) {
	if x < 0 {
		return -x;
	}
	return x;
}

func average(a, b) {
	return (a + b) / 2.0;
}

func improve(guess, x) {
	return average(guess, x / guess);
}

func goodEnough(guess, x, tolerance) {
	return abs(guess * guess - x) < tolerance;
}

func sqrtNewton(x) {
	if x == 0 {
		return 0;
	}

	var guess = x / 2.0;
	const tolerance = 1e-9;

	while true {
		var next = improve(guess, x);
		if goodEnough(next, x, tolerance) {
			return next;
		}
		guess = next;
	}
}

func isEmpty(xs) {
	return nullp(xs);
}

func head(xs) {
	return car(xs);
}

func tail(xs) {
	return cdr(xs);
}

func map(fn, xs) {
	if isEmpty(xs) {
		return [];
	}
	return cons(fn(head(xs)), map(fn, tail(xs)));
}

func mean(xs) {
	var total = 0.0;
	var count = 0;
	var cursor = xs;

	while !isEmpty(cursor) {
		total = total + head(cursor);
		count = count + 1;
		cursor = tail(cursor);
	}
	return total / count;
}

func variance(xs, avg) {
	var total = 0.0;
	var cursor = xs;

	while !isEmpty(cursor) {
		var diff = head(cursor) - avg;
		total = total + diff * diff;
		cursor = tail(cursor);
	}
	return total / length(xs);
}

func length(xs) {
	var count = 0;
	var cursor = xs;

	while !isEmpty(cursor) {
		count = count + 1;
		cursor = tail(cursor);
	}
	return count;
}

func zScores(xs) {
	var avg = mean(xs);
	var std = sqrtNewton(variance(xs, avg));

	return map(func(x) {
		return (x - avg) / std;
	}, xs);
}

func alertOnOutlier(xs, threshold) {
	var scores = zScores(xs);
	var cursor = scores;
	var index = 0;

	while !isEmpty(cursor) {
		var score = head(cursor);
		if abs(score) > threshold {
			display("Outlier at index ");
			display(index);
			display(": ");
			display(score);
			newline();
		}
		index = index + 1;
		cursor = tail(cursor);
	}
}

alertOnOutlier([10.0, 10.5, 10.2, 11.0, 26.0, 10.1], 2.5);
```

Because every helper is written in Gisp, you can copy the file into a project and extend it, for example by adding file I/O via Scheme's `call-with-input-file` using another inline expression.

---

## 9. Exercises and Next Steps

- **Warm up:** Expand the `runningAverage` function to return both the averages and the variance at each step. Make sure you keep results in floating-point space.
- **Macro practice:** Extend the `unless` macro so it accepts an optional "else" branch. Test it by rewriting the `alertOnOutlier` procedure.
- **Performance experiment:** Compare `sqrtNewton` with Go's `math.Sqrt` by embedding the standard library through inline Scheme FFI bindings.
- **Symbolic manipulation:** Augment the differentiator to handle sums and products of more than two terms by normalising the list representation.

For deeper reading, explore:

- `docs/Language.md` for the full grammar.
- `docs/Primitives.md` to see every primitive exposed by the runtime.
- The `parser/` package for examples of translating Gisp into Scheme forms.

Happy hacking! The more Scheme material you port, the more comfortable you'll get switching between the two syntactic worlds that Gisp unifies.


