package runtime

import (
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
