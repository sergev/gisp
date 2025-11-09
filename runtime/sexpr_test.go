package runtime

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sergev/gisp/lang"
	"github.com/sergev/gisp/sexpr"
)

func evalString(t *testing.T, ev *lang.Evaluator, src string) lang.Value {
	t.Helper()
	forms, err := sexpr.ReadString(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	val, err := ev.EvalAll(forms, nil)
	if err != nil {
		t.Fatalf("evaluation error: %v", err)
	}
	return val
}

func TestArithmetic(t *testing.T) {
	ev := NewEvaluator()
	val := evalString(t, ev, "(+ 1 2 3 4)")
	if val.Type != lang.TypeInt || val.Int != 10 {
		t.Fatalf("expected 10, got %s", val.String())
	}
	val = evalString(t, ev, "(* 2 3 4)")
	if val.Type != lang.TypeInt || val.Int != 24 {
		t.Fatalf("expected 24, got %s", val.String())
	}
}

func TestMacroExpansion(t *testing.T) {
	ev := NewEvaluator()
	evalString(t, ev, `
(define-macro (when condition . body)
  (list 'if condition
        (cons 'begin body)
        '#f))
`)
	val := evalString(t, ev, `
(begin
  (define flag #f)
  (when #t (set! flag 42))
  flag)
`)
	if val.Type != lang.TypeInt || val.Int != 42 {
		t.Fatalf("expected 42, got %s", val.String())
	}
}

func TestContinuation(t *testing.T) {
	ev := NewEvaluator()
	val := evalString(t, ev, `
(begin
  (define saved #f)
  (define result
    (call/cc (lambda (k)
               (set! saved k)
               'initial)))
  (if (eq? result 'initial)
      (saved 'second)
      result))
`)
	if val.Type != lang.TypeSymbol || val.Sym != "second" {
		t.Fatalf("expected symbol second, got %s", val.String())
	}
}

func TestTailRecursion(t *testing.T) {
	ev := NewEvaluator()
	val := evalString(t, ev, `
(begin
  (define (sum n acc)
    (if (= n 0)
        acc
        (sum (- n 1) (+ acc n))))
  (sum 10000 0))
`)
	if val.Type != lang.TypeInt || val.Int != 50005000 {
		t.Fatalf("unexpected result: %s", val.String())
	}
}

func TestExamples(t *testing.T) {
	baseExamples := filepath.Join("..", "examples")
	examples := []struct {
		name     string
		path     string
		validate func(t *testing.T, v lang.Value)
	}{
		{
			name: "hello",
			path: filepath.Join(baseExamples, "hello.gs"),
			validate: func(t *testing.T, v lang.Value) {
				t.Helper()
				if v.Type != lang.TypeEmpty {
					t.Fatalf("expected empty list from hello example, got %s", v.String())
				}
			},
		},
		{
			name: "continuation",
			path: filepath.Join(baseExamples, "continuation.gs"),
			validate: func(t *testing.T, v lang.Value) {
				t.Helper()
				if v.Type != lang.TypeInt || v.Int != 42 {
					t.Fatalf("expected 42 from continuation example, got %s", v.String())
				}
			},
		},
	}

	for _, ex := range examples {
		ex := ex
		t.Run(ex.name, func(t *testing.T) {
			ev := NewEvaluator()
			SetArgv(ev.Global, []string{})
			var val lang.Value
			var err error

			_ = captureOutput(func() {
				val, err = EvaluateFile(ev, ex.path)
			})

			if err != nil {
				t.Fatalf("example %s failed: %v", ex.name, err)
			}
			ex.validate(t, val)
		})
	}
}

func captureOutput(fn func()) string {
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}
