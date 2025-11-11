package main

import (
	"testing"

	"github.com/sergev/gisp/runtime"
)

func TestParseGispGoSyntax(t *testing.T) {
	forms, err := parseGisp(`
var x = 40;
x + 2;
`)
	if err != nil {
		t.Fatalf("parseGisp returned error: %v", err)
	}
	if len(forms) == 0 {
		t.Fatalf("expected compiled forms, got none")
	}

	ev := runtime.NewEvaluator()
	val, err := ev.EvalAll(forms, nil)
	if err != nil {
		t.Fatalf("EvalAll returned error: %v", err)
	}
	if got, want := val.String(), "42"; got != want {
		t.Fatalf("EvalAll => %s, want %s", got, want)
	}

	if _, err := parseGisp("if true {"); err == nil || !isIncomplete(err) {
		t.Fatalf("expected incomplete error for open block, got %v", err)
	}
}
