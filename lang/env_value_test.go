package lang

import (
	"strings"
	"testing"
)

func TestEnvParentLookupAndErrors(t *testing.T) {
	parent := NewEnv(nil)
	parent.Define("x", IntValue(1))
	child := NewEnv(parent)

	if err := child.Set("x", IntValue(2)); err != nil {
		t.Fatalf("Set should update parent binding: %v", err)
	}
	val, err := parent.Get("x")
	if err != nil || val.Int() != 2 {
		t.Fatalf("expected parent value updated to 2, got %v err=%v", val, err)
	}

	if err := child.Set("missing", IntValue(0)); err == nil || !strings.Contains(err.Error(), "unbound variable") {
		t.Fatalf("expected error updating missing binding, got %v", err)
	}

	if _, err := child.Get("missing"); err == nil || !strings.Contains(err.Error(), "unbound variable") {
		t.Fatalf("expected error fetching missing binding, got %v", err)
	}

	if child.Parent() != parent {
		t.Fatalf("expected Parent to expose enclosing environment")
	}
}

func TestPairToStringAndTypeHelpers(t *testing.T) {
	pair := PairValue(IntValue(1), IntValue(2))
	if got := pairToString(pair); got != "(1. 2)" {
		t.Fatalf("expected dotted pair string, got %q", got)
	}

	vector := VectorValue([]Value{IntValue(1), BoolValue(true)})
	if got := vector.String(); got != "#(1 #t)" {
		t.Fatalf("expected vector string, got %q", got)
	}

	formatted := List(IntValue(1), IntValue(2), IntValue(3)).String()
	if formatted != "(1 2 3)" {
		t.Fatalf("expected proper list string, got %q", formatted)
	}

	if contStr := ContinuationValue(nil, nil, nil).String(); contStr != "<continuation>" {
		t.Fatalf("expected continuation string, got %q", contStr)
	}
	if macroStr := MacroValue(nil, "", nil, nil).String(); macroStr != "<macro>" {
		t.Fatalf("expected macro string, got %q", macroStr)
	}
	if unknown := (Value{Type: ValueType(99)}).String(); unknown != "<unknown>" {
		t.Fatalf("expected unknown string fallback, got %q", unknown)
	}
}
