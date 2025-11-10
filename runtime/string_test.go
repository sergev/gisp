package runtime

import (
	"strings"
	"testing"

	"github.com/sergev/gisp/lang"
)

func TestPrimStringLength(t *testing.T) {
	ev := NewEvaluator()

	val, err := primStringLength(ev, []lang.Value{lang.StringValue("hello")})
	if err != nil {
		t.Fatalf("primStringLength returned error: %v", err)
	}
	if val.Type != lang.TypeInt {
		t.Fatalf("expected integer result, got %v", val)
	}
	if val.Int() != 5 {
		t.Fatalf("expected length 5, got %d", val.Int())
	}
}

func TestPrimStringLengthTypeError(t *testing.T) {
	ev := NewEvaluator()

	_, err := primStringLength(ev, []lang.Value{lang.IntValue(3)})
	if err == nil {
		t.Fatalf("expected type error for non-string argument")
	}
	if !strings.Contains(err.Error(), "stringLength expects string") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestPrimMakeStringDefaultsToSpace(t *testing.T) {
	ev := NewEvaluator()

	val, err := primMakeString(ev, []lang.Value{lang.IntValue(3)})
	if err != nil {
		t.Fatalf("primMakeString returned error: %v", err)
	}
	if val.Type != lang.TypeString {
		t.Fatalf("expected string result, got %v", val)
	}
	if val.Str() != "   " {
		t.Fatalf("expected three spaces, got %q", val.Str())
	}
}

func TestPrimMakeStringCustomFill(t *testing.T) {
	ev := NewEvaluator()

	val, err := primMakeString(ev, []lang.Value{
		lang.IntValue(4),
		lang.StringValue("*"),
	})
	if err != nil {
		t.Fatalf("primMakeString returned error: %v", err)
	}
	if val.Str() != "****" {
		t.Fatalf("expected \"****\", got %q", val.Str())
	}
}

func TestPrimMakeStringErrors(t *testing.T) {
	ev := NewEvaluator()

	_, err := primMakeString(ev, []lang.Value{lang.IntValue(-1)})
	if err == nil {
		t.Fatalf("expected error for negative length")
	}

	_, err = primMakeString(ev, []lang.Value{lang.StringValue("oops")})
	if err == nil {
		t.Fatalf("expected type error for non-integer length")
	}

	_, err = primMakeString(ev, []lang.Value{
		lang.IntValue(2),
		lang.StringValue("xy"),
	})
	if err == nil {
		t.Fatalf("expected error for multi-character fill string")
	}

	_, err = primMakeString(ev, []lang.Value{
		lang.IntValue(2),
		lang.IntValue(1),
	})
	if err == nil {
		t.Fatalf("expected type error for non-string fill argument")
	}
}

func TestPrimStringSliceBasic(t *testing.T) {
	ev := NewEvaluator()

	val, err := primStringSlice(ev, []lang.Value{
		lang.StringValue("pattern"),
		lang.IntValue(1),
		lang.IntValue(4),
	})
	if err != nil {
		t.Fatalf("primStringSlice returned error: %v", err)
	}
	if val.Type != lang.TypeString {
		t.Fatalf("expected string result, got %v", val)
	}
	if val.Str() != "att" {
		t.Fatalf("expected \"att\", got %q", val.Str())
	}
}

func TestPrimStringSliceOptionalEnd(t *testing.T) {
	ev := NewEvaluator()

	val, err := primStringSlice(ev, []lang.Value{
		lang.StringValue("pattern"),
		lang.IntValue(4),
	})
	if err != nil {
		t.Fatalf("primStringSlice returned error: %v", err)
	}
	if val.Str() != "ern" {
		t.Fatalf("expected \"ern\", got %q", val.Str())
	}
}

func TestPrimStringSliceErrors(t *testing.T) {
	ev := NewEvaluator()

	_, err := primStringSlice(ev, []lang.Value{
		lang.IntValue(1),
		lang.IntValue(0),
	})
	if err == nil {
		t.Fatalf("expected type error for non-string source")
	}

	_, err = primStringSlice(ev, []lang.Value{
		lang.StringValue("data"),
		lang.StringValue("oops"),
	})
	if err == nil {
		t.Fatalf("expected type error for non-integer start")
	}

	_, err = primStringSlice(ev, []lang.Value{
		lang.StringValue("data"),
		lang.IntValue(5),
	})
	if err == nil {
		t.Fatalf("expected error for start out of range")
	}

	_, err = primStringSlice(ev, []lang.Value{
		lang.StringValue("data"),
		lang.IntValue(1),
		lang.IntValue(0),
	})
	if err == nil {
		t.Fatalf("expected error for end before start")
	}

	_, err = primStringSlice(ev, []lang.Value{
		lang.StringValue("data"),
		lang.IntValue(1),
		lang.IntValue(10),
	})
	if err == nil {
		t.Fatalf("expected error for end out of range")
	}
}
