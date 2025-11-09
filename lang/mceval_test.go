package lang_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sergev/gisp/lang"
	"github.com/sergev/gisp/runtime"
	"github.com/sergev/gisp/sexpr"
)

func TestMetacircularEvaluatorScheme(t *testing.T) {
	ev := runtime.NewEvaluator()
	scriptPath := filepath.Join("..", "examples", "mceval.gs")
	if _, err := runtime.EvaluateFile(ev, scriptPath); err != nil {
		t.Fatalf("failed loading mceval.gs: %v", err)
	}
	runMetacircularAssertions(t, ev)
}

func TestMetacircularEvaluatorGisp(t *testing.T) {
	ev := runtime.NewEvaluator()
	const condMacro = `
(define-macro (cond . clauses)
  (define (expand remaining)
    (if (null? remaining)
        false
        (let ((first (car remaining))
              (rest (cdr remaining)))
          (if (eq? (car first) 'else)
              (cons 'begin (cdr first))
              (list 'if (car first)
                    (cons 'begin (cdr first))
                    (expand rest))))))
  (expand clauses))
`
	if _, err := runtime.EvaluateReader(ev, strings.NewReader(condMacro)); err != nil {
		t.Fatalf("failed to install cond macro: %v", err)
	}

	scriptPath := filepath.Join("..", "examples", "mceval.gisp")
	code, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read mceval.gisp: %v", err)
	}
	if _, err := runtime.EvaluateGispReader(ev, bytes.NewReader(code)); err != nil {
		t.Fatalf("failed loading mceval.gisp: %v", err)
	}
	runMetacircularAssertions(t, ev)
}

func runMetacircularAssertions(t *testing.T, ev *lang.Evaluator) {
	t.Helper()

	setupProc, err := ev.Global.Get("setup-environment")
	if err != nil {
		t.Fatalf("setup-environment not found: %v", err)
	}
	envVal, err := ev.Apply(setupProc, nil)
	if err != nil {
		t.Fatalf("setup-environment execution failed: %v", err)
	}

	evalProc, err := ev.Global.Get("eval")
	if err != nil {
		t.Fatalf("eval procedure not found: %v", err)
	}

	evalSICP := func(code string) lang.Value {
		forms, err := sexpr.ReadString(code)
		if err != nil {
			t.Fatalf("failed to parse %q: %v", code, err)
		}
		if len(forms) != 1 {
			t.Fatalf("expected single expression in %q", code)
		}
		form := forms[0]
		result, err := ev.Apply(evalProc, []lang.Value{form, envVal})
		if err != nil {
			t.Fatalf("evaluation failed for %q: %v", code, err)
		}
		return result
	}

	if got := evalSICP("(+ 1 2)"); got.Type != lang.TypeInt || got.Int() != 3 {
		t.Fatalf("expected (+ 1 2) => 3, got %s", got.String())
	}

	if got := evalSICP("((lambda (x) (* x x)) 5)"); got.Type != lang.TypeInt || got.Int() != 25 {
		t.Fatalf("expected lambda square => 25, got %s", got.String())
	}

	defineRes := evalSICP("(define (fact n) (if (= n 0) 1 (* n (fact (- n 1)))))")
	if defineRes.Type != lang.TypeSymbol || defineRes.Sym() != "ok" {
		t.Fatalf("expected define to return symbol ok, got %s", defineRes.String())
	}

	if got := evalSICP("(fact 5)"); got.Type != lang.TypeInt || got.Int() != 120 {
		t.Fatalf("expected (fact 5) => 120, got %s", got.String())
	}
}
