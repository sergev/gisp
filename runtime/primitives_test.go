package runtime

import (
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/sergev/gisp/lang"
)

func TestPrimSubAndDivEdgeCases(t *testing.T) {
	ev := NewEvaluator()

	t.Run("unary negation uses sign flip", func(t *testing.T) {
		val, err := primSub(ev, []lang.Value{lang.IntValue(5)})
		if err != nil {
			t.Fatalf("primSub error: %v", err)
		}
		if val.Type != lang.TypeInt || val.Int() != -5 {
			t.Fatalf("expected -5, got %v", val)
		}
	})

	t.Run("float promotion and division by zero", func(t *testing.T) {
		val, err := primSub(ev, []lang.Value{lang.RealValue(10), lang.IntValue(2)})
		if err != nil {
			t.Fatalf("primSub mixed types error: %v", err)
		}
		if val.Type != lang.TypeReal || val.Real() != 8 {
			t.Fatalf("expected 8.0, got %v", val)
		}

		if _, err := primDiv(ev, []lang.Value{lang.IntValue(4)}); err != nil {
			t.Fatalf("primDiv unary reciprocal error: %v", err)
		}

		if _, err := primDiv(ev, []lang.Value{lang.IntValue(4), lang.IntValue(0)}); err == nil || !strings.Contains(err.Error(), "division by zero") {
			t.Fatalf("expected division by zero error, got %v", err)
		}
	})
}

func TestPrimMod(t *testing.T) {
	ev := NewEvaluator()

	t.Run("supports chained modulo", func(t *testing.T) {
		val, err := primMod(ev, []lang.Value{lang.IntValue(123), lang.IntValue(45), lang.IntValue(7)})
		if err != nil {
			t.Fatalf("primMod failed: %v", err)
		}
		if val.Type != lang.TypeInt || val.Int() != ((123%45)%7) {
			t.Fatalf("expected modulo result %d, got %v", (123%45)%7, val)
		}
	})

	t.Run("validates arguments", func(t *testing.T) {
		if _, err := primMod(ev, []lang.Value{lang.IntValue(1)}); err == nil || !strings.Contains(err.Error(), "at least 2 arguments") {
			t.Fatalf("expected arity error, got %v", err)
		}
		if _, err := primMod(ev, []lang.Value{lang.RealValue(1.0), lang.IntValue(2)}); err == nil || !strings.Contains(err.Error(), "expects integer") {
			t.Fatalf("expected type error, got %v", err)
		}
		if _, err := primMod(ev, []lang.Value{lang.IntValue(10), lang.IntValue(0)}); err == nil || !strings.Contains(err.Error(), "modulo by zero") {
			t.Fatalf("expected modulo by zero error, got %v", err)
		}
	})
}

func TestPrimBitwiseOperations(t *testing.T) {
	ev := NewEvaluator()

	t.Run("xor handles unary complement and variadic xor", func(t *testing.T) {
		inputs := []int64{0, 1, 42, -17, math.MaxInt64}
		for _, in := range inputs {
			val, err := primBitXor(ev, []lang.Value{lang.IntValue(in)})
			if err != nil {
				t.Fatalf("primBitXor unary on %d failed: %v", in, err)
			}
			if val.Type != lang.TypeInt {
				t.Fatalf("expected integer result for %d, got %v", in, val.Type)
			}
			expect := ^in
			if val.Int() != expect {
				t.Fatalf("expected %d complement, got %d", expect, val.Int())
			}
		}

		val, err := primBitXor(ev, []lang.Value{
			lang.IntValue(0b1100),
			lang.IntValue(0b1010),
			lang.IntValue(0b0110),
		})
		if err != nil {
			t.Fatalf("primBitXor variadic failed: %v", err)
		}
		if val.Int() != 0b0000 {
			t.Fatalf("expected xor result 0, got %b", val.Int())
		}

		if _, err := primBitXor(ev, nil); err == nil || !strings.Contains(err.Error(), "at least 1 argument") {
			t.Fatalf("expected arity error, got %v", err)
		}
		if _, err := primBitXor(ev, []lang.Value{lang.RealValue(1.5)}); err == nil || !strings.Contains(err.Error(), "expects integer") {
			t.Fatalf("expected type error, got %v", err)
		}
	})

	t.Run("and/or/clear combine multiple integers", func(t *testing.T) {
		andVal, err := primBitAnd(ev, []lang.Value{
			lang.IntValue(0b1111),
			lang.IntValue(0b1010),
			lang.IntValue(0b0011),
		})
		if err != nil {
			t.Fatalf("primBitAnd failed: %v", err)
		}
		if andVal.Int() != 0b0010 {
			t.Fatalf("expected bitwise and 0b0010, got %b", andVal.Int())
		}

		orVal, err := primBitOr(ev, []lang.Value{
			lang.IntValue(0b0101),
			lang.IntValue(0b0011),
		})
		if err != nil {
			t.Fatalf("primBitOr failed: %v", err)
		}
		if orVal.Int() != 0b0111 {
			t.Fatalf("expected bitwise or 0b0111, got %b", orVal.Int())
		}

		clearVal, err := primBitClear(ev, []lang.Value{
			lang.IntValue(0b1111),
			lang.IntValue(0b1100),
			lang.IntValue(0b0011),
		})
		if err != nil {
			t.Fatalf("primBitClear failed: %v", err)
		}
		if clearVal.Int() != 0b0000 {
			t.Fatalf("expected bit clear result 0, got %b", clearVal.Int())
		}

		if _, err := primBitAnd(ev, []lang.Value{lang.IntValue(1)}); err == nil || !strings.Contains(err.Error(), "at least 2 arguments") {
			t.Fatalf("expected bit and arity error, got %v", err)
		}
		if _, err := primBitOr(ev, []lang.Value{lang.IntValue(1), lang.RealValue(2)}); err == nil || !strings.Contains(err.Error(), "expects integer") {
			t.Fatalf("expected bit or type error, got %v", err)
		}
		if _, err := primBitClear(ev, []lang.Value{lang.IntValue(1)}); err == nil || !strings.Contains(err.Error(), "at least 2 arguments") {
			t.Fatalf("expected bit clear arity error, got %v", err)
		}
	})

	t.Run("shift validates arguments", func(t *testing.T) {
		left, err := primShiftLeft(ev, []lang.Value{lang.IntValue(3), lang.IntValue(2)})
		if err != nil {
			t.Fatalf("primShiftLeft failed: %v", err)
		}
		if left.Int() != 12 {
			t.Fatalf("expected 12, got %d", left.Int())
		}

		right, err := primShiftRight(ev, []lang.Value{lang.IntValue(-16), lang.IntValue(2)})
		if err != nil {
			t.Fatalf("primShiftRight failed: %v", err)
		}
		if right.Int() != -4 {
			t.Fatalf("expected arithmetic shift to -4, got %d", right.Int())
		}

		if _, err := primShiftLeft(ev, []lang.Value{lang.IntValue(1), lang.IntValue(-1)}); err == nil || !strings.Contains(err.Error(), "non-negative shift") {
			t.Fatalf("expected negative shift error, got %v", err)
		}
		if _, err := primShiftRight(ev, []lang.Value{lang.RealValue(1.5), lang.IntValue(1)}); err == nil || !strings.Contains(err.Error(), "expects integer") {
			t.Fatalf("expected shift type error, got %v", err)
		}
	})
}

func TestPrimRandomIntegerAndSeed(t *testing.T) {
	ev := NewEvaluator()

	t.Run("arity and type validation", func(t *testing.T) {
		if _, err := primRandomInteger(ev, nil); err == nil || !strings.Contains(err.Error(), "expects 1 argument") {
			t.Fatalf("expected arity error from randomInteger, got %v", err)
		}
		if _, err := primRandomInteger(ev, []lang.Value{lang.RealValue(3.14)}); err == nil || !strings.Contains(err.Error(), "integer") {
			t.Fatalf("expected type error from randomInteger, got %v", err)
		}
		if _, err := primRandomInteger(ev, []lang.Value{lang.IntValue(0)}); err == nil || !strings.Contains(err.Error(), "positive") {
			t.Fatalf("expected positive limit error, got %v", err)
		}
		if _, err := primRandomSeed(ev, nil); err == nil || !strings.Contains(err.Error(), "expects 1 argument") {
			t.Fatalf("expected arity error from randomSeed, got %v", err)
		}
		if _, err := primRandomSeed(ev, []lang.Value{lang.RealValue(2)}); err == nil || !strings.Contains(err.Error(), "integer") {
			t.Fatalf("expected type error from randomSeed, got %v", err)
		}
	})

	t.Run("deterministic sequence with seeding", func(t *testing.T) {
		const limit = int64(16)
		if _, err := primRandomSeed(ev, []lang.Value{lang.IntValue(123)}); err != nil {
			t.Fatalf("randomSeed failed: %v", err)
		}
		expectGen := rand.New(rand.NewSource(123))
		expect := expectGen.Int63n(limit)

		val, err := primRandomInteger(ev, []lang.Value{lang.IntValue(limit)})
		if err != nil {
			t.Fatalf("randomInteger failed: %v", err)
		}
		if val.Type != lang.TypeInt {
			t.Fatalf("expected integer result, got %v", val)
		}
		if val.Int() != expect {
			t.Fatalf("expected %d, got %d", expect, val.Int())
		}

		if _, err := primRandomSeed(ev, []lang.Value{lang.IntValue(123)}); err != nil {
			t.Fatalf("randomSeed reseed failed: %v", err)
		}
		expectGen = rand.New(rand.NewSource(123))
		expect = expectGen.Int63n(limit)
		val, err = primRandomInteger(ev, []lang.Value{lang.IntValue(limit)})
		if err != nil {
			t.Fatalf("randomInteger after reseed failed: %v", err)
		}
		if val.Int() != expect {
			t.Fatalf("expected %d after reseed, got %d", expect, val.Int())
		}
	})
}

func TestCompoundAssignPrimitives(t *testing.T) {
	ev := NewEvaluator()
	env := lang.NewEnv(ev.Global)

	env.Define("count", lang.IntValue(10))
	val, err := ev.Eval(compoundAssignExpr("+=", "count", lang.IntValue(5)), env)
	if err != nil {
		t.Fatalf("+= evaluation failed: %v", err)
	}
	if val.Type != lang.TypeInt || val.Int() != 15 {
		t.Fatalf("+= returned unexpected value %v", val)
	}
	if got, err := env.Get("count"); err != nil || got.Int() != 15 {
		t.Fatalf("count not updated, got %v err %v", got, err)
	}

	env.Define("ratio", lang.RealValue(9))
	val, err = ev.Eval(compoundAssignExpr("/=", "ratio", lang.IntValue(3)), env)
	if err != nil {
		t.Fatalf("/= evaluation failed: %v", err)
	}
	if val.Type != lang.TypeReal || val.Real() != 3 {
		t.Fatalf("/= returned unexpected value %v", val)
	}
	if got, err := env.Get("ratio"); err != nil || got.Real() != 3 {
		t.Fatalf("ratio not updated, got %v err %v", got, err)
	}

	env.Define("flags", lang.IntValue(0b1111))
	val, err = ev.Eval(compoundAssignExpr("&^=", "flags", lang.IntValue(0b1010)), env)
	if err != nil {
		t.Fatalf("&^= evaluation failed: %v", err)
	}
	if val.Type != lang.TypeInt || val.Int() != 0b0101 {
		t.Fatalf("&^= returned unexpected value %v", val)
	}
	if got, err := env.Get("flags"); err != nil || got.Int() != 0b0101 {
		t.Fatalf("flags not updated, got %v err %v", got, err)
	}

	_, err = ev.Eval(lang.List(
		lang.SymbolValue("+="),
		lang.IntValue(1),
		lang.IntValue(2),
	), env)
	if err == nil || !strings.Contains(err.Error(), "symbol") {
		t.Fatalf("expected symbol type error, got %v", err)
	}
}

func TestPostIncDecPrimitives(t *testing.T) {
	ev := NewEvaluator()
	env := lang.NewEnv(ev.Global)

	env.Define("counter", lang.IntValue(10))
	val, err := ev.Eval(lang.List(
		lang.SymbolValue("++"),
		lang.List(
			lang.SymbolValue("quote"),
			lang.SymbolValue("counter"),
		),
	), env)
	if err != nil {
		t.Fatalf("++ evaluation failed: %v", err)
	}
	if val.Type != lang.TypeInt || val.Int() != 11 {
		t.Fatalf("++ returned unexpected value %v", val)
	}
	if got, err := env.Get("counter"); err != nil || got.Int() != 11 {
		t.Fatalf("counter not incremented, got %v err %v", got, err)
	}

	env.Define("ratio", lang.RealValue(3.5))
	val, err = ev.Eval(lang.List(
		lang.SymbolValue("--"),
		lang.List(
			lang.SymbolValue("quote"),
			lang.SymbolValue("ratio"),
		),
	), env)
	if err != nil {
		t.Fatalf("-- evaluation failed: %v", err)
	}
	if val.Type != lang.TypeReal || val.Real() != 2.5 {
		t.Fatalf("-- returned unexpected value %v", val)
	}
	if got, err := env.Get("ratio"); err != nil || got.Real() != 2.5 {
		t.Fatalf("ratio not decremented, got %v err %v", got, err)
	}

	env.Define("text", lang.StringValue("hello"))
	_, err = ev.Eval(lang.List(
		lang.SymbolValue("++"),
		lang.List(
			lang.SymbolValue("quote"),
			lang.SymbolValue("text"),
		),
	), env)
	if err == nil || !strings.Contains(err.Error(), "number") {
		t.Fatalf("expected type error for ++ on string, got %v", err)
	}
}

func compoundAssignExpr(op, name string, value lang.Value) lang.Value {
	return lang.List(
		lang.SymbolValue(op),
		lang.List(
			lang.SymbolValue("quote"),
			lang.SymbolValue(name),
		),
		value,
	)
}

func TestPrimRead(t *testing.T) {
	ev := NewEvaluator()

	t.Run("arity validation", func(t *testing.T) {
		if _, err := primRead(ev, []lang.Value{lang.IntValue(1)}); err == nil || !strings.Contains(err.Error(), "no arguments") {
			t.Fatalf("expected arity error from read, got %v", err)
		}
	})

	t.Run("reads successive datums and EOF", func(t *testing.T) {
		setReadInput(strings.NewReader("(+ 1 2) 42 #t"))
		t.Cleanup(func() { setReadInput(nil) })

		expr, err := primRead(ev, nil)
		if err != nil {
			t.Fatalf("primRead failed: %v", err)
		}
		items, err := lang.ToSlice(expr)
		if err != nil {
			t.Fatalf("expected list from first datum, got error: %v", err)
		}
		if len(items) != 3 || items[0].Type != lang.TypeSymbol || items[0].Sym() != "+" {
			t.Fatalf("unexpected first datum: %v", expr)
		}

		val, err := primRead(ev, nil)
		if err != nil {
			t.Fatalf("primRead second datum failed: %v", err)
		}
		if val.Type != lang.TypeInt || val.Int() != 42 {
			t.Fatalf("expected 42, got %v", val)
		}

		boolVal, err := primRead(ev, nil)
		if err != nil {
			t.Fatalf("primRead third datum failed: %v", err)
		}
		if boolVal.Type != lang.TypeBool || !boolVal.Bool() {
			t.Fatalf("expected #t, got %v", boolVal)
		}

		eofVal, err := primRead(ev, nil)
		if err != nil {
			t.Fatalf("primRead EOF failed: %v", err)
		}
		if eofVal.Type != lang.TypeEOF {
			t.Fatalf("expected EOF object, got %v", eofVal)
		}

		again, err := primRead(ev, nil)
		if err != nil {
			t.Fatalf("primRead repeated EOF failed: %v", err)
		}
		if again.Type != lang.TypeEOF {
			t.Fatalf("expected EOF object on subsequent read, got %v", again)
		}
	})
}

func TestPrimComparisonAndNot(t *testing.T) {
	ev := NewEvaluator()

	val, err := primNot(ev, []lang.Value{lang.BoolValue(true)})
	if err != nil {
		t.Fatalf("primNot error: %v", err)
	}
	if val.Type != lang.TypeBool || val.Bool() {
		t.Fatalf("expected #f, got %v", val)
	}

	if _, err := primLess(ev, []lang.Value{lang.IntValue(1), lang.StringValue("nope")}); err == nil || !strings.Contains(err.Error(), "number") {
		t.Fatalf("expected type error from primLess, got %v", err)
	}
}

func TestPrimListAndPairMutation(t *testing.T) {
	ev := NewEvaluator()

	t.Run("append requires lists", func(t *testing.T) {
		if _, err := primAppend(ev, []lang.Value{lang.IntValue(1), lang.EmptyList}); err == nil || !strings.Contains(err.Error(), "append expects lists") {
			t.Fatalf("expected append error, got %v", err)
		}
	})

	t.Run("append concatenates lists", func(t *testing.T) {
		val, err := primAppend(ev, []lang.Value{
			lang.List(lang.IntValue(1)),
			lang.List(lang.IntValue(2), lang.IntValue(3)),
		})
		if err != nil {
			t.Fatalf("primAppend error: %v", err)
		}
		items, err := lang.ToSlice(val)
		if err != nil {
			t.Fatalf("ToSlice append result: %v", err)
		}
		if len(items) != 3 || items[0].Int() != 1 || items[1].Int() != 2 || items[2].Int() != 3 {
			t.Fatalf("unexpected append result: %v", items)
		}
	})

	t.Run("set-first!/set-rest! mutate pair", func(t *testing.T) {
		pair := lang.PairValue(lang.IntValue(1), lang.IntValue(2))
		if _, err := primSetFirst(ev, []lang.Value{pair, lang.IntValue(10)}); err != nil {
			t.Fatalf("primSetFirst error: %v", err)
		}
		if pair.Pair().First.Int() != 10 {
			t.Fatalf("expected updated first=10, got %v", pair.Pair().First)
		}
		if _, err := primSetRest(ev, []lang.Value{pair, lang.IntValue(99)}); err != nil {
			t.Fatalf("primSetRest error: %v", err)
		}
		if pair.Pair().Rest.Int() != 99 {
			t.Fatalf("expected updated rest=99, got %v", pair.Pair().Rest)
		}
	})

	t.Run("length arity validation", func(t *testing.T) {
		if _, err := primLength(ev, nil); err == nil || !strings.Contains(err.Error(), "length expects 1 argument") {
			t.Fatalf("expected length arity error, got %v", err)
		}
	})
}

func TestPrimEqualityVariants(t *testing.T) {
	ev := NewEvaluator()

	pair := lang.PairValue(lang.IntValue(1), lang.IntValue(2))
	pairCopy := lang.PairValue(lang.IntValue(1), lang.IntValue(2))

	eqVal, err := primEq(ev, []lang.Value{pair, pair})
	if err != nil {
		t.Fatalf("primEq error: %v", err)
	}
	if !eqVal.Bool() {
		t.Fatalf("expected eq? to be true for identical pair pointer")
	}

	eqVal, err = primEq(ev, []lang.Value{pair, pairCopy})
	if err != nil {
		t.Fatalf("primEq error: %v", err)
	}
	if eqVal.Bool() {
		t.Fatalf("expected eq? to be false for structurally equal but distinct pairs")
	}

	equalVal, err := primEqual(ev, []lang.Value{lang.IntValue(3), lang.RealValue(3)})
	if err != nil {
		t.Fatalf("primEqual error: %v", err)
	}
	if !equalVal.Bool() {
		t.Fatalf("expected equal? to treat int/real same when numerically equal")
	}
}

func TestPrimStringAndNumberHelpers(t *testing.T) {
	ev := NewEvaluator()

	appendVal, err := primStringAppend(ev, []lang.Value{
		lang.StringValue("foo"), lang.StringValue("bar"),
	})
	if err != nil {
		t.Fatalf("primStringAppend error: %v", err)
	}
	if appendVal.Str() != "foobar" {
		t.Fatalf("expected foobar, got %q", appendVal.Str())
	}

	if _, err := primStringAppend(ev, []lang.Value{lang.StringValue("ok"), lang.IntValue(1)}); err == nil || !strings.Contains(err.Error(), "stringAppend expects string") {
		t.Fatalf("expected stringAppend type error, got %v", err)
	}

	numVal, err := primStringToNumber(ev, []lang.Value{lang.StringValue("   42 ")})
	if err != nil {
		t.Fatalf("primStringToNumber error: %v", err)
	}
	if numVal.Type != lang.TypeInt || numVal.Int() != 42 {
		t.Fatalf("expected integer 42, got %v", numVal)
	}

	invalid, err := primStringToNumber(ev, []lang.Value{lang.StringValue("not-a-number")})
	if err != nil {
		t.Fatalf("primStringToNumber error on invalid input: %v", err)
	}
	if invalid.Type != lang.TypeBool || invalid.Bool() {
		t.Fatalf("expected #f for invalid conversion, got %v", invalid)
	}
}

func TestPrimApplyAndDisplay(t *testing.T) {
	ev := NewEvaluator()
	plus, err := ev.Global.Get("+")
	if err != nil {
		t.Fatalf("failed to get + primitive: %v", err)
	}

	result, err := primApply(ev, []lang.Value{
		plus,
		lang.IntValue(1),
		lang.IntValue(2),
		lang.List(lang.IntValue(3), lang.IntValue(4)),
	})
	if err != nil {
		t.Fatalf("primApply error: %v", err)
	}
	if result.Type != lang.TypeInt || result.Int() != 10 {
		t.Fatalf("expected 10 from primApply, got %v", result)
	}

	if _, err := primApply(ev, []lang.Value{plus, lang.IntValue(1), lang.IntValue(2), lang.IntValue(3)}); err == nil || !strings.Contains(err.Error(), "apply expects final argument to be a list") {
		t.Fatalf("expected primApply final argument error, got %v", err)
	}

	output := captureOutput(func() {
		val, err := primDisplay(ev, []lang.Value{lang.StringValue("hi")})
		if err != nil {
			t.Fatalf("primDisplay error: %v", err)
		}
		if val.Type != lang.TypeEmpty {
			t.Fatalf("expected empty list from display, got %v", val)
		}
	})
	if output != "hi" {
		t.Fatalf("expected display to write hi, got %q", output)
	}

	output = captureOutput(func() {
		if _, err := primNewline(ev, nil); err != nil {
			t.Fatalf("primNewline error: %v", err)
		}
	})
	if output != "\n" {
		t.Fatalf("expected newline output, got %q", output)
	}
}

func TestPrimMap(t *testing.T) {
	ev := NewEvaluator()

	mapVal, err := ev.Global.Get("map")
	if err != nil {
		t.Fatalf("failed to get map primitive: %v", err)
	}

	doubleProc := lang.ClosureValue(
		[]string{"x"},
		"",
		[]lang.Value{
			lang.List(
				lang.SymbolValue("*"),
				lang.SymbolValue("x"),
				lang.IntValue(2),
			),
		},
		ev.Global,
	)

	t.Run("maps over list", func(t *testing.T) {
		input := lang.List(lang.IntValue(1), lang.IntValue(2), lang.IntValue(3))

		result, err := ev.Apply(mapVal, []lang.Value{doubleProc, input})
		if err != nil {
			t.Fatalf("map application failed: %v", err)
		}

		items, err := lang.ToSlice(result)
		if err != nil {
			t.Fatalf("map result not a list: %v", err)
		}
		expected := []int64{2, 4, 6}
		if len(items) != len(expected) {
			t.Fatalf("expected %d items, got %d", len(expected), len(items))
		}
		for i, exp := range expected {
			if items[i].Type != lang.TypeInt || items[i].Int() != exp {
				t.Fatalf("item %d: expected %d, got %v", i, exp, items[i])
			}
		}
	})

	t.Run("empty list returns empty list", func(t *testing.T) {
		result, err := ev.Apply(mapVal, []lang.Value{doubleProc, lang.EmptyList})
		if err != nil {
			t.Fatalf("map on empty list failed: %v", err)
		}
		if result.Type != lang.TypeEmpty {
			t.Fatalf("expected empty list, got %v", result)
		}
	})
}

func TestPrimFilter(t *testing.T) {
	ev := NewEvaluator()

	filterVal, err := ev.Global.Get("filter")
	if err != nil {
		t.Fatalf("failed to get filter primitive: %v", err)
	}

	buildList := func(vals ...int64) lang.Value {
		items := make([]lang.Value, len(vals))
		for i, v := range vals {
			items[i] = lang.IntValue(v)
		}
		return lang.List(items...)
	}

	positiveProc := lang.ClosureValue(
		[]string{"x"},
		"",
		[]lang.Value{
			lang.List(
				lang.SymbolValue("if"),
				lang.List(
					lang.SymbolValue(">"),
					lang.SymbolValue("x"),
					lang.IntValue(0),
				),
				lang.BoolValue(true),
				lang.BoolValue(false),
			),
		},
		ev.Global,
	)

	alwaysTrueProc := lang.ClosureValue(
		[]string{"_"},
		"",
		[]lang.Value{
			lang.BoolValue(true),
		},
		ev.Global,
	)

	tests := []struct {
		name      string
		predicate lang.Value
		input     lang.Value
		expected  []int64
	}{
		{
			name:      "filters positive integers",
			predicate: positiveProc,
			input:     buildList(-2, 0, 3, 5, -1),
			expected:  []int64{3, 5},
		},
		{
			name:      "returns empty when no elements match",
			predicate: positiveProc,
			input:     buildList(-5, -1),
			expected:  []int64{},
		},
		{
			name:      "handles empty list input",
			predicate: positiveProc,
			input:     lang.EmptyList,
			expected:  []int64{},
		},
		{
			name:      "keeps all elements when predicate always true",
			predicate: alwaysTrueProc,
			input:     buildList(1, 2, 3),
			expected:  []int64{1, 2, 3},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result, err := ev.Apply(filterVal, []lang.Value{tc.predicate, tc.input})
			if err != nil {
				t.Fatalf("filter application failed: %v", err)
			}
			items, err := lang.ToSlice(result)
			if err != nil {
				t.Fatalf("filter result not a list: %v", err)
			}
			if len(items) != len(tc.expected) {
				t.Fatalf("expected %d items, got %d", len(tc.expected), len(items))
			}
			for i, exp := range tc.expected {
				if items[i].Type != lang.TypeInt || items[i].Int() != exp {
					t.Fatalf("item %d: expected %d, got %v", i, exp, items[i])
				}
			}
			if len(tc.expected) == 0 && result.Type != lang.TypeEmpty {
				t.Fatalf("expected empty list result, got %v", result)
			}
		})
	}
}
