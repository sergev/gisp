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
	if val.Int != 120 {
		t.Fatalf("expected 120, got %d", val.Int)
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
	runTutorialExample(t, "tutorial_05_counter_closure.gisp", "1116")
}

func TestTutorial06Iterate(t *testing.T) {
	runTutorialExample(t, "tutorial_06_iterate.gisp", "32")
}

func TestTutorial07ListHelpers(t *testing.T) {
	runTutorialExample(t, "tutorial_07_list_helpers.gisp", "1(2 3 4)\n(1 4 9 16)")
}

func TestTutorial08UnlessMacro(t *testing.T) {
	runTutorialExample(t, "tutorial_08_unless_macro.gisp", "value was false")
}

func TestTutorial09Compose(t *testing.T) {
	runTutorialExample(t, "tutorial_09_compose.gisp", "21")
}

func TestTutorial10Abs(t *testing.T) {
	runTutorialExample(t, "tutorial_10_abs.gisp", "3.52")
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
	runTutorialExample(t, "tutorial_16_zscore_pipeline.gisp", "")
}

func TestTutorial17Classify(t *testing.T) {
	runTutorialExample(t, "tutorial_17_classify.gisp", "negativezeropositive")
}

func TestTutorial18AccumulateUntil(t *testing.T) {
	runTutorialExample(t, "tutorial_18_accumulate_until.gisp", "2")
}
