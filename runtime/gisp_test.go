package runtime

import (
	"path/filepath"
	"sort"
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

func TestTutorialExamples(t *testing.T) {
	baseExamples := filepath.Join("..", "examples")
	matches, err := filepath.Glob(filepath.Join(baseExamples, "tutorial_*.gisp"))
	if err != nil {
		t.Fatalf("failed to glob tutorial examples: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no tutorial examples found")
	}

	sort.Strings(matches)

	for _, script := range matches {
		script := script
		t.Run(filepath.Base(script), func(t *testing.T) {
			ev := NewEvaluator()
			SetArgv(ev.Global, []string{})

			var val lang.Value
			captureOutput(func() {
				var evalErr error
				val, evalErr = EvaluateFile(ev, script)
				if evalErr != nil {
					t.Fatalf("EvaluateFile(%s) error: %v", script, evalErr)
				}
			})

			if val.Type == lang.TypeContinuation || val.Type == lang.TypeMacro {
				t.Fatalf("unexpected result type %v from %s", val.Type, script)
			}
		})
	}
}
