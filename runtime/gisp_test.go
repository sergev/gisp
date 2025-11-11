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
