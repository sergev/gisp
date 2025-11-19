package runtime

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sergev/gisp/lang"
)

func TestEvaluateGispString(t *testing.T) {
	ev := NewEvaluator()
	src := `
func fact(n) {
	if n == 0 {
		return 1;
	}
	return n * fact(n - 1);
}

fact(5);
`
	val, err := EvaluateGispString(ev, src)
	if err != nil {
		t.Fatalf("EvaluateGispString returned error: %v", err)
	}
	if val.Type != lang.TypeInt {
		t.Fatalf("expected integer result, got %v", val)
	}
	if val.Int() != 120 {
		t.Fatalf("expected 120, got %d", val.Int())
	}
}

func TestEvaluateGispSwitch(t *testing.T) {
	ev := NewEvaluator()
	src := `
var x = -3;
var sign = switch {
case x > 0: 1;
case x < 0: -1;
default: 0;
};
sign;
`
	val, err := EvaluateGispString(ev, src)
	if err != nil {
		t.Fatalf("EvaluateGispString switch returned error: %v", err)
	}
	if val.Type != lang.TypeInt || val.Int() != -1 {
		t.Fatalf("expected -1, got %v", val)
	}
}

func TestEvaluateGispWhileBreakContinue(t *testing.T) {
	ev := NewEvaluator()
	src := `
func compute() {
	var n = 0
	var sum = 0
	while n < 6 {
		n = n + 1
		if n == 3 {
			continue
		}
		if n > 4 {
			break
		}
		sum = sum + n
	}
	return sum
}
compute()
`
	val, err := EvaluateGispString(ev, src)
	if err != nil {
		t.Fatalf("EvaluateGispString while/break/continue returned error: %v", err)
	}
	if val.Type != lang.TypeInt || val.Int() != 7 {
		t.Fatalf("expected result 7, got %v", val)
	}
}

func runTutorialExample(t *testing.T, scriptName, expected string) {
	t.Helper()

	scriptPath := filepath.Join("..", "examples", scriptName)

	ev := NewEvaluator()
	SetArgv(ev.Global, []string{})

	var (
		val     lang.Value
		evalErr error
	)

	output := captureOutput(func() {
		val, evalErr = EvaluateFile(ev, scriptPath)
	})

	if evalErr != nil {
		t.Fatalf("EvaluateFile(%s) error: %v", scriptName, evalErr)
	}

	if val.Type == lang.TypeContinuation || val.Type == lang.TypeMacro {
		t.Fatalf("unexpected result type %v from %s", val.Type, scriptName)
	}

	actual := strings.TrimSpace(output)
	expectedTrimmed := strings.TrimSpace(expected)
	if actual != expectedTrimmed {
		t.Fatalf("unexpected output for %s\nexpected: %q\ngot:      %q", scriptName, expectedTrimmed, actual)
	}
}

func TestTutorial01Hello(t *testing.T) {
	runTutorialExample(t, "tutorial_01_hello.gisp", "Hello from Gisp!")
}

func TestTutorial02WhileLoop(t *testing.T) {
	runTutorialExample(t, "tutorial_02_while_loop.gisp", "5")
}

func TestTutorial03CircleArea(t *testing.T) {
	runTutorialExample(t, "tutorial_03_circle_area.gisp", "Circle area: 19.634954084936208\nRatio 7/2 = 3.5")
}

func TestTutorial04StringsBooleans(t *testing.T) {
	runTutorialExample(t, "tutorial_04_strings_booleans.gisp", "Line one\nLine two")
}

func TestTutorial05CounterClosure(t *testing.T) {
	runTutorialExample(t, "tutorial_05_counter_closure.gisp", "11\n16")
}

func TestTutorial06Iterate(t *testing.T) {
	runTutorialExample(t, "tutorial_06_iterate.gisp", "32")
}

func TestTutorial07ListHelpers(t *testing.T) {
	runTutorialExample(t, "tutorial_07_list_helpers.gisp", "1\n(2 3 4)\n(1 4 9 16)")
}

func TestTutorial07Vectors(t *testing.T) {
	runTutorialExample(t, "tutorial_07_vectors.gisp", "(2 3 5 7 11 13 17 19 23 29 31 37 41 43 47)")
}

func TestTutorial08UnlessMacro(t *testing.T) {
	runTutorialExample(t, "tutorial_08_unless_macro.gisp", "value was false")
}

func TestTutorial09Compose(t *testing.T) {
	runTutorialExample(t, "tutorial_09_compose.gisp", "21")
}

func TestTutorial10Abs(t *testing.T) {
	runTutorialExample(t, "tutorial_10_abs.gisp", "3.5\n2")
}

func TestTutorial11RunningAverage(t *testing.T) {
	runTutorialExample(t, "tutorial_11_running_average.gisp", "(10 11 13.333333333333334 15)")
}

func TestTutorial12SqrtNewton(t *testing.T) {
	runTutorialExample(t, "tutorial_12_sqrt_newton.gisp", "1.4142135623746899")
}

func TestTutorial13AdaptiveTrapezoid(t *testing.T) {
	runTutorialExample(t, "tutorial_13_adaptive_trapezoid.gisp", "3.1415925809016922")
}

func TestTutorial14SymbolicDiff(t *testing.T) {
	runTutorialExample(t, "tutorial_14_symbolic_diff.gisp", "(+ (pow x 3) (* (+ x 1) (* 3 (pow x (+ 3 -1)))))")
}

func TestTutorial15CallccFindFirst(t *testing.T) {
	runTutorialExample(t, "tutorial_15_callcc_find_first.gisp", "13")
}

func TestTutorial16ZscorePipeline(t *testing.T) {
	runTutorialExample(t, "tutorial_16_zscore_pipeline.gisp", `Analyzing values: (10 10.5 10.2 11 26 10.1)
Mean: 12.966666666666667
Variance: 34.08222222222222
Standard deviation: 5.837998134825175
Z-scores: (-0.5081650590070815 -0.4225192625452138 -0.47390674042233455 -0.3368734660833462 2.232500427772684 -0.49103589971470807)`)
}

func TestTutorial17Classify(t *testing.T) {
	runTutorialExample(t, "tutorial_17_classify.gisp", "negative\nzero\npositive")
}

func TestTutorial18AccumulateUntil(t *testing.T) {
	runTutorialExample(t, "tutorial_18_accumulate_until.gisp", "2")
}

func TestContinuationExample(t *testing.T) {
	runTutorialExample(
		t,
		"continuation.gisp",
		"Demonstrating call/cc\nFirst result: initial return\nInvoking continuation with 42\nFirst result: 42\nContinuation produced: 42",
	)
}

func TestSnobolPatternMatcherExample(t *testing.T) {
	runTutorialExample(
		t,
		"snobol_patterns.gisp",
		`== Snobol-style syllable split ==
syllable:
  matched: strand
  captures:
    onset => str
    nucleus => a
    coda => nd

== Configuration pairs with ARBNO/BREAK ==
pairs:
  matched: name = Alice; age=34; city=Rlyeh;
  captures:
    key => name
    value => Alice
    key => age
    value => 34
    key => city
    value => Rlyeh
  pairs:
    name => Alice
    age => 34
    city => Rlyeh

== Log line with LEN/POS/RPOS ==
log:
  matched: ERROR 2025-11-10 parser: unexpected token ';'
  captures:
    level => ERROR
    year => 2025
    month => 11
    day => 10
    date => 2025-11-10
    module => parser
    message => unexpected token ';'
  decoded date: 2025-11-10
  module: parser
  message: unexpected token ';'`,
	)
}

func TestRegexPatternMatcherExample(t *testing.T) {
	runTutorialExample(
		t,
		"regex_patterns.gisp",
		`== Regex Engine Demo ==
== Literal and dot ==
Pattern /te.t/ matched "test" within "testing" from 0 to 4.
Captures: ["test"]
All matches for /te.t/ in "testing":
  #0: "test" @ [0, 4) captures=["test"]

== Anchors ==
Pattern /^hello$/ did not match "well hello there".
All matches for /^hello$/ in "well hello there":
  (none)

== Character classes ==
Pattern /[a-z]+/ matched "def" within "ABC def 123" from 4 to 7.
Captures: ["def"]
All matches for /[a-z]+/ in "ABC def 123":
  #0: "def" @ [4, 7) captures=["def"]

== Grouping and alternation ==
Pattern /colou?r|colour/ matched "colour" within "The colour palette" from 4 to 10.
Captures: ["colour"]
All matches for /colou?r|colour/ in "The colour palette":
  #0: "colour" @ [4, 10) captures=["colour"]

== Quantifiers with bounds ==
Pattern /(ha){2,4}/ matched "hahaha" within "hahaha wow hahaha" from 0 to 6.
Captures: ["hahaha", "ha"]
All matches for /(ha){2,4}/ in "hahaha wow hahaha":
  #0: "hahaha" @ [0, 6) captures=["hahaha", "ha"]
  #1: "hahaha" @ [11, 17) captures=["hahaha", "ha"]

== Negated class ==
Pattern /[^aeiou]+/ matched "q" within "queue rhythm" from 0 to 1.
Captures: ["q"]
All matches for /[^aeiou]+/ in "queue rhythm":
  #0: "q" @ [0, 1) captures=["q"]
  #1: " rhythm" @ [5, 12) captures=[" rhythm"]`,
	)
}

// loadUnify loads the unify.gisp file and returns an evaluator with it loaded
func loadUnify(t *testing.T) *lang.Evaluator {
	t.Helper()
	ev := NewEvaluator()
	SetArgv(ev.Global, []string{})
	scriptPath := filepath.Join("..", "examples", "unify.gisp")
	_, err := EvaluateFile(ev, scriptPath)
	if err != nil {
		t.Fatalf("failed to load unify.gisp: %v", err)
	}
	return ev
}

// testUnify calls the unify function with two arguments and returns the result
// u and v should be Gisp expressions (can use backticks for s-expressions)
func testUnify(t *testing.T, ev *lang.Evaluator, u, v string) lang.Value {
	t.Helper()
	src := "unify(" + u + ", " + v + ")"
	val, err := EvaluateGispString(ev, src)
	if err != nil {
		t.Fatalf("unify(%s, %s) evaluation error: %v", u, v, err)
	}
	return val
}

func TestUnifyBasic(t *testing.T) {
	ev := loadUnify(t)

	// Variable to variable - should succeed
	val := testUnify(t, ev, "`'x", "`'y")
	// Result should be a term (not a string error)
	if val.Type == lang.TypeString && (val.Str() == "cycle" || val.Str() == "clash") {
		t.Fatalf("unify('x, 'y) should succeed, got error: %s", val.String())
	}

	// Variable to term - should succeed
	val = testUnify(t, ev, "`'x", "`'(f a)")
	if val.Type == lang.TypeString && (val.Str() == "cycle" || val.Str() == "clash") {
		t.Fatalf("unify('x, (f a)) should succeed, got error: %s", val.String())
	}

	// Term to variable - should succeed
	val = testUnify(t, ev, "`'(f a)", "`'x")
	if val.Type == lang.TypeString && (val.Str() == "cycle" || val.Str() == "clash") {
		t.Fatalf("unify((f a), 'x) should succeed, got error: %s", val.String())
	}

	// Same variable - should succeed
	val = testUnify(t, ev, "`'x", "`'x")
	if val.Type == lang.TypeString && (val.Str() == "cycle" || val.Str() == "clash") {
		t.Fatalf("unify('x, 'x) should succeed, got error: %s", val.String())
	}
}

func TestUnifyTermsSuccess(t *testing.T) {
	ev := loadUnify(t)

	// Same structure - should succeed
	val := testUnify(t, ev, "`'(f a)", "`'(f a)")
	if val.Type == lang.TypeString && (val.Str() == "cycle" || val.Str() == "clash") {
		t.Fatalf("unify((f a), (f a)) should succeed, got error: %s", val.String())
	}

	// Different variables - should succeed
	val = testUnify(t, ev, "`'(f x)", "`'(f y)")
	if val.Type == lang.TypeString && (val.Str() == "cycle" || val.Str() == "clash") {
		t.Fatalf("unify((f x), (f y)) should succeed, got error: %s", val.String())
	}

	// Nested structures - should succeed
	val = testUnify(t, ev, "`'(f (g x))", "`'(f (g a))")
	if val.Type == lang.TypeString && (val.Str() == "cycle" || val.Str() == "clash") {
		t.Fatalf("unify((f (g x)), (f (g a))) should succeed, got error: %s", val.String())
	}

	// Multiple arguments - should succeed
	val = testUnify(t, ev, "`'(f x y)", "`'(f a b)")
	if val.Type == lang.TypeString && (val.Str() == "cycle" || val.Str() == "clash") {
		t.Fatalf("unify((f x y), (f a b)) should succeed, got error: %s", val.String())
	}
}

func TestUnifyTermsClash(t *testing.T) {
	ev := loadUnify(t)

	// Different heads - should return "clash"
	val := testUnify(t, ev, "`'(f a)", "`'(g a)")
	if val.Type != lang.TypeString || val.Str() != "clash" {
		t.Fatalf("unify((f a), (g a)) should return \"clash\", got: %s", val.String())
	}

	// Different lengths - should return "clash"
	val = testUnify(t, ev, "`'(f a)", "`'(f a b)")
	if val.Type != lang.TypeString || val.Str() != "clash" {
		t.Fatalf("unify((f a), (f a b)) should return \"clash\", got: %s", val.String())
	}

	// Different structure - should return "clash"
	val = testUnify(t, ev, "`'(f a b)", "`'(f a)")
	if val.Type != lang.TypeString || val.Str() != "clash" {
		t.Fatalf("unify((f a b), (f a)) should return \"clash\", got: %s", val.String())
	}
}

func TestUnifyCycle(t *testing.T) {
	ev := loadUnify(t)

	// Direct cycle - should return "cycle"
	val := testUnify(t, ev, "`'x", "`'(f x)")
	if val.Type != lang.TypeString || val.Str() != "cycle" {
		t.Fatalf("unify('x, (f x)) should return \"cycle\", got: %s", val.String())
	}

	// Indirect cycle - should return "cycle"
	val = testUnify(t, ev, "`'x", "`'(f (g x))")
	if val.Type != lang.TypeString || val.Str() != "cycle" {
		t.Fatalf("unify('x, (f (g x))) should return \"cycle\", got: %s", val.String())
	}

	// More complex cycle
	val = testUnify(t, ev, "`'x", "`'(f y x)")
	if val.Type != lang.TypeString || val.Str() != "cycle" {
		t.Fatalf("unify('x, (f y x)) should return \"cycle\", got: %s", val.String())
	}
}

func TestUnifyComplex(t *testing.T) {
	ev := loadUnify(t)

	// Complex nested structure with multiple variables
	val := testUnify(t, ev, "`'(f x (g y))", "`'(f a (g b))")
	if val.Type == lang.TypeString && (val.Str() == "cycle" || val.Str() == "clash") {
		t.Fatalf("unify((f x (g y)), (f a (g b))) should succeed, got error: %s", val.String())
	}

	// Deeply nested structure
	val = testUnify(t, ev, "`'(f (g (h x)))", "`'(f (g (h a)))")
	if val.Type == lang.TypeString && (val.Str() == "cycle" || val.Str() == "clash") {
		t.Fatalf("unify((f (g (h x))), (f (g (h a)))) should succeed, got error: %s", val.String())
	}

	// Multiple variables in different positions
	val = testUnify(t, ev, "`'(f x y z)", "`'(f a b c)")
	if val.Type == lang.TypeString && (val.Str() == "cycle" || val.Str() == "clash") {
		t.Fatalf("unify((f x y z), (f a b c)) should succeed, got error: %s", val.String())
	}

	// Variable appears multiple times - should succeed if consistent
	val = testUnify(t, ev, "`'(f x x)", "`'(f a a)")
	if val.Type == lang.TypeString && (val.Str() == "cycle" || val.Str() == "clash") {
		t.Fatalf("unify((f x x), (f a a)) should succeed, got error: %s", val.String())
	}
}
