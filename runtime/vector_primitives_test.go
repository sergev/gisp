package runtime

import (
	"strings"
	"testing"

	"github.com/sergev/gisp/lang"
)

func TestVectorPrimitives(t *testing.T) {
	ev := NewEvaluator()

	t.Run("construct access and mutate", func(t *testing.T) {
		vecVal, err := primVector(ev, []lang.Value{lang.IntValue(1), lang.BoolValue(true)})
		if err != nil {
			t.Fatalf("primVector error: %v", err)
		}
		if vecVal.Type != lang.TypeVector {
			t.Fatalf("expected vector type, got %v", vecVal.Type)
		}

		lengthVal, err := primVectorLength(ev, []lang.Value{vecVal})
		if err != nil {
			t.Fatalf("vectorLength error: %v", err)
		}
		if lengthVal.Int() != 2 {
			t.Fatalf("expected length 2, got %d", lengthVal.Int())
		}

		elem, err := primVectorRef(ev, []lang.Value{vecVal, lang.IntValue(1)})
		if err != nil {
			t.Fatalf("vectorRef error: %v", err)
		}
		if elem.Type != lang.TypeBool || !elem.Bool() {
			t.Fatalf("expected #t at index 1, got %v", elem)
		}

		if _, err := primVectorSet(ev, []lang.Value{vecVal, lang.IntValue(0), lang.StringValue("hi")}); err != nil {
			t.Fatalf("vectorSet error: %v", err)
		}
		updated, err := primVectorRef(ev, []lang.Value{vecVal, lang.IntValue(0)})
		if err != nil {
			t.Fatalf("vectorRef after set error: %v", err)
		}
		if updated.Type != lang.TypeString || updated.Str() != "hi" {
			t.Fatalf("expected updated string element, got %v", updated)
		}

		if _, err := primVectorFill(ev, []lang.Value{vecVal, lang.IntValue(42)}); err != nil {
			t.Fatalf("vectorFill error: %v", err)
		}
		for i := int64(0); i < 2; i++ {
			val, err := primVectorRef(ev, []lang.Value{vecVal, lang.IntValue(i)})
			if err != nil {
				t.Fatalf("vectorRef post-fill error: %v", err)
			}
			if val.Type != lang.TypeInt || val.Int() != 42 {
				t.Fatalf("expected filled element 42, got %v", val)
			}
		}
	})

	t.Run("makeVector optional fill and conversions", func(t *testing.T) {
		defaultVec, err := primMakeVector(ev, []lang.Value{lang.IntValue(2)})
		if err != nil {
			t.Fatalf("makeVector error: %v", err)
		}
		for i := int64(0); i < 2; i++ {
			val, err := primVectorRef(ev, []lang.Value{defaultVec, lang.IntValue(i)})
			if err != nil {
				t.Fatalf("vectorRef default fill error: %v", err)
			}
			if val.Type != lang.TypeEmpty {
				t.Fatalf("expected empty list default fill, got %v", val)
			}
		}

		filledVec, err := primMakeVector(ev, []lang.Value{lang.IntValue(3), lang.StringValue("x")})
		if err != nil {
			t.Fatalf("makeVector with fill error: %v", err)
		}
		for i := int64(0); i < 3; i++ {
			val, err := primVectorRef(ev, []lang.Value{filledVec, lang.IntValue(i)})
			if err != nil {
				t.Fatalf("vectorRef filled error: %v", err)
			}
			if val.Type != lang.TypeString || val.Str() != "x" {
				t.Fatalf("expected filled string element, got %v", val)
			}
		}

		listVal, err := primVectorToList(ev, []lang.Value{filledVec})
		if err != nil {
			t.Fatalf("vectorToList error: %v", err)
		}
		items, err := lang.ToSlice(listVal)
		if err != nil {
			t.Fatalf("vectorToList produced non-list: %v", err)
		}
		if len(items) != 3 || items[1].Str() != "x" {
			t.Fatalf("unexpected list conversion result: %v", items)
		}

		vecAgain, err := primListToVector(ev, []lang.Value{listVal})
		if err != nil {
			t.Fatalf("listToVector error: %v", err)
		}
		check, err := primVectorRef(ev, []lang.Value{vecAgain, lang.IntValue(2)})
		if err != nil {
			t.Fatalf("vectorRef after listToVector error: %v", err)
		}
		if check.Type != lang.TypeString || check.Str() != "x" {
			t.Fatalf("expected vector element 'x', got %v", check)
		}
	})

	t.Run("predicates and error handling", func(t *testing.T) {
		vecVal, err := primVector(ev, []lang.Value{})
		if err != nil {
			t.Fatalf("primVector empty error: %v", err)
		}
		isVec, err := primIsVector(ev, []lang.Value{vecVal})
		if err != nil {
			t.Fatalf("vectorp error: %v", err)
		}
		if !isVec.Bool() {
			t.Fatalf("expected vectorp to be true")
		}
		isVec, err = primIsVector(ev, []lang.Value{lang.IntValue(1)})
		if err != nil {
			t.Fatalf("vectorp type error: %v", err)
		}
		if isVec.Bool() {
			t.Fatalf("expected vectorp to be false on integer")
		}

		if _, err := primVectorRef(ev, []lang.Value{vecVal, lang.IntValue(0)}); err == nil || !strings.Contains(err.Error(), "out of range") {
			t.Fatalf("expected bounds error from vectorRef, got %v", err)
		}
		if _, err := primVectorSet(ev, []lang.Value{lang.IntValue(1), lang.IntValue(0), lang.BoolValue(true)}); err == nil || !strings.Contains(err.Error(), "vector") {
			t.Fatalf("expected type error from vectorSet, got %v", err)
		}
		if _, err := primListToVector(ev, []lang.Value{lang.IntValue(1)}); err == nil || !strings.Contains(err.Error(), "proper list") {
			t.Fatalf("expected listToVector type error, got %v", err)
		}
	})
}
