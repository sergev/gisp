package runtime

import (
	"testing"

	"github.com/sergev/gisp/lang"
)

func expectInt(t *testing.T, val lang.Value, want int64) {
	t.Helper()
	if val.Type != lang.TypeInt {
		t.Fatalf("expected integer %d, got %v", want, val)
	}
	if val.Int() != want {
		t.Fatalf("expected %d, got %d", want, val.Int())
	}
}

func expectReal(t *testing.T, val lang.Value, want float64) {
	t.Helper()
	if val.Type != lang.TypeReal {
		t.Fatalf("expected real %f, got %v", want, val)
	}
	if val.Real() != want {
		t.Fatalf("expected %f, got %f", want, val.Real())
	}
}

func expectBool(t *testing.T, val lang.Value, want bool) {
	t.Helper()
	if val.Type != lang.TypeBool {
		t.Fatalf("expected boolean %v, got %v", want, val)
	}
	if val.Bool() != want {
		t.Fatalf("expected %v, got %v", want, val.Bool())
	}
}

func TestBinaryOperatorsEndToEnd(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		assert func(*testing.T, lang.Value)
	}{
		{"addition", "1 + 2;", func(t *testing.T, v lang.Value) { expectInt(t, v, 3) }},
		{"subtraction", "5 - 8;", func(t *testing.T, v lang.Value) { expectInt(t, v, -3) }},
		{"multiplication", "6 * 7;", func(t *testing.T, v lang.Value) { expectInt(t, v, 42) }},
		{"division", "21 / 2;", func(t *testing.T, v lang.Value) { expectReal(t, v, 10.5) }},
		{"modulo", "123 % 45;", func(t *testing.T, v lang.Value) { expectInt(t, v, 33) }},
		{"bitwise-and", "12 & 10;", func(t *testing.T, v lang.Value) { expectInt(t, v, 8) }},
		{"bitwise-or", "12 | 10;", func(t *testing.T, v lang.Value) { expectInt(t, v, 14) }},
		{"bitwise-xor", "12 ^ 10;", func(t *testing.T, v lang.Value) { expectInt(t, v, 6) }},
		{"bitwise-clear", "15 &^ 10;", func(t *testing.T, v lang.Value) { expectInt(t, v, 5) }},
		{"shift-left", "3 << 4;", func(t *testing.T, v lang.Value) { expectInt(t, v, 48) }},
		{"shift-right", "48 >> 3;", func(t *testing.T, v lang.Value) { expectInt(t, v, 6) }},
		{"equals", "1 == 1;", func(t *testing.T, v lang.Value) { expectBool(t, v, true) }},
		{"not-equals", "1 != 2;", func(t *testing.T, v lang.Value) { expectBool(t, v, true) }},
		{"less-than", "3 < 4;", func(t *testing.T, v lang.Value) { expectBool(t, v, true) }},
		{"less-or-equal", "4 <= 3;", func(t *testing.T, v lang.Value) { expectBool(t, v, false) }},
		{"greater-than", "4 > 3;", func(t *testing.T, v lang.Value) { expectBool(t, v, true) }},
		{"greater-or-equal", "3 >= 4;", func(t *testing.T, v lang.Value) { expectBool(t, v, false) }},
		{"logical-and", "true && false;", func(t *testing.T, v lang.Value) { expectBool(t, v, false) }},
		{"logical-or", "false || true;", func(t *testing.T, v lang.Value) { expectBool(t, v, true) }},
		{"precedence", "1 + 2 * 3;", func(t *testing.T, v lang.Value) { expectInt(t, v, 7) }},
		{"shift-vs-add", "1 << 2 + 1;", func(t *testing.T, v lang.Value) { expectInt(t, v, 5) }},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ev := NewEvaluator()
			val, err := EvaluateGispString(ev, tc.src)
			if err != nil {
				t.Fatalf("EvaluateGispString(%q) error: %v", tc.src, err)
			}
			tc.assert(t, val)
		})
	}
}

func TestModuloDisplayOutput(t *testing.T) {
	ev := NewEvaluator()
	output := captureOutput(func() {
		if _, err := EvaluateGispString(ev, "display(123 % 45);"); err != nil {
			t.Fatalf("EvaluateGispString display modulo error: %v", err)
		}
	})
	if output != "33" {
		t.Fatalf("expected display output 33, got %q", output)
	}
}
