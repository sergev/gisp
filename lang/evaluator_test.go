package lang

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

func newTestEvaluator() *Evaluator {
	ev := NewEvaluator()

	ev.Global.Define("+", PrimitiveValue(func(_ *Evaluator, args []Value) (Value, error) {
		var sum int64
		for _, arg := range args {
			switch arg.Type {
			case TypeInt:
				sum += arg.Int()
			default:
				return Value{}, errors.New("+: expected integers")
			}
		}
		return IntValue(sum), nil
	}))

	ev.Global.Define("*", PrimitiveValue(func(_ *Evaluator, args []Value) (Value, error) {
		prod := int64(1)
		for _, arg := range args {
			switch arg.Type {
			case TypeInt:
				prod *= arg.Int()
			default:
				return Value{}, errors.New("*: expected integers")
			}
		}
		return IntValue(prod), nil
	}))

	ev.Global.Define("cons", PrimitiveValue(func(_ *Evaluator, args []Value) (Value, error) {
		if len(args) != 2 {
			return Value{}, errors.New("cons: expected 2 arguments")
		}
		return PairValue(args[0], args[1]), nil
	}))

	ev.Global.Define("append", PrimitiveValue(func(_ *Evaluator, args []Value) (Value, error) {
		var collected []Value
		for _, arg := range args {
			items, err := ToSlice(arg)
			if err != nil {
				return Value{}, errors.New("append: expected proper list")
			}
			collected = append(collected, items...)
		}
		return List(collected...), nil
	}))

	ev.Global.Define("list", PrimitiveValue(func(_ *Evaluator, args []Value) (Value, error) {
		return List(args...), nil
	}))

	ev.Global.Define("identity", PrimitiveValue(func(_ *Evaluator, args []Value) (Value, error) {
		if len(args) != 1 {
			return Value{}, errors.New("identity: expected 1 argument")
		}
		return args[0], nil
	}))

	return ev
}

func mustEval(t *testing.T, ev *Evaluator, expr Value) Value {
	t.Helper()
	val, err := ev.Eval(expr, nil)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	return val
}

func mustEvalAll(t *testing.T, ev *Evaluator, exprs ...Value) Value {
	t.Helper()
	val, err := ev.EvalAll(exprs, nil)
	if err != nil {
		t.Fatalf("EvalAll error: %v", err)
	}
	return val
}

func TestEvaluatorEvalLiteralValues(t *testing.T) {
	ev := newTestEvaluator()

	tests := []struct {
		name string
		val  Value
	}{
		{"empty", EmptyList},
		{"bool-true", BoolValue(true)},
		{"bool-false", BoolValue(false)},
		{"int", IntValue(42)},
		{"real", RealValue(math.Pi)},
		{"string", StringValue("hello")},
		{"primitive", PrimitiveValue(func(*Evaluator, []Value) (Value, error) { return EmptyList, nil })},
		{"closure", ClosureValue([]string{"x"}, "", []Value{SymbolValue("x")}, ev.Global)},
		{"macro", MacroValue([]string{"x"}, "", []Value{SymbolValue("x")}, ev.Global)},
		{"continuation", ContinuationValue(nil, ev.Global, ev)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mustEval(t, ev, tt.val)
			if !valuesEqual(got, tt.val) {
				t.Fatalf("expected %v, got %v", tt.val, got)
			}
		})
	}
}

func TestEvaluatorEvalSymbol(t *testing.T) {
	ev := newTestEvaluator()
	ev.Global.Define("answer", IntValue(42))

	got := mustEval(t, ev, SymbolValue("answer"))
	if got.Type != TypeInt || got.Int() != 42 {
		t.Fatalf("expected 42, got %v", got)
	}
}

func TestEvaluatorEvalSymbolUnbound(t *testing.T) {
	ev := newTestEvaluator()
	_, err := ev.Eval(SymbolValue("missing"), nil)
	if err == nil {
		t.Fatal("expected error for unbound symbol")
	}
}

func TestEvaluatorEvalAll(t *testing.T) {
	ev := newTestEvaluator()
	result := mustEvalAll(t, ev, IntValue(1), IntValue(2), IntValue(3))
	if result.Type != TypeInt || result.Int() != 3 {
		t.Fatalf("expected last value 3, got %v", result)
	}
}

func TestEvaluatorApplyPrimitive(t *testing.T) {
	ev := newTestEvaluator()
	proc := SymbolValue("+")
	call := List(proc, IntValue(1), IntValue(2), IntValue(3))
	res := mustEval(t, ev, call)
	if res.Type != TypeInt || res.Int() != 6 {
		t.Fatalf("expected 6, got %v", res)
	}
}

func TestEvaluatorApplyClosure(t *testing.T) {
	ev := newTestEvaluator()
	fn := List(SymbolValue("lambda"), List(SymbolValue("x"), SymbolValue("y")), List(SymbolValue("+"), SymbolValue("x"), SymbolValue("y")))
	call := List(fn, IntValue(5), IntValue(7))
	res := mustEval(t, ev, call)
	if res.Type != TypeInt || res.Int() != 12 {
		t.Fatalf("expected 12, got %v", res)
	}
}

func TestEvaluatorApplyContinuation(t *testing.T) {
	ev := newTestEvaluator()
	k := ContinuationValue(nil, ev.Global, ev)
	val, err := ev.Apply(k, []Value{IntValue(99)})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if val.Type != TypeInt || val.Int() != 99 {
		t.Fatalf("expected 99, got %v", val)
	}
}

func TestEvaluatorApplyNonCallable(t *testing.T) {
	ev := newTestEvaluator()
	_, err := ev.Apply(IntValue(1), nil)
	if err == nil {
		t.Fatal("expected error applying non-callable")
	}
}

func TestEvaluatorQuote(t *testing.T) {
	ev := newTestEvaluator()
	expr := List(SymbolValue("quote"), List(IntValue(1), IntValue(2)))
	result := mustEval(t, ev, expr)
	if result.Type != TypePair {
		t.Fatalf("expected pair, got %v", result)
	}

	_, err := ev.Eval(List(SymbolValue("quote"), IntValue(1), IntValue(2)), nil)
	if err == nil {
		t.Fatal("expected error for extra args to quote")
	}
}

func TestEvaluatorIf(t *testing.T) {
	ev := newTestEvaluator()

	thenExpr := List(SymbolValue("if"), BoolValue(true), IntValue(1), IntValue(2))
	ifVal := mustEval(t, ev, thenExpr)
	if ifVal.Type != TypeInt || ifVal.Int() != 1 {
		t.Fatalf("expected 1, got %v", ifVal)
	}

	elseExpr := List(SymbolValue("if"), BoolValue(false), IntValue(1), IntValue(2))
	elseVal := mustEval(t, ev, elseExpr)
	if elseVal.Type != TypeInt || elseVal.Int() != 2 {
		t.Fatalf("expected 2, got %v", elseVal)
	}

	noAlt := mustEval(t, ev, List(SymbolValue("if"), BoolValue(false), IntValue(1)))
	if noAlt.Type != TypeEmpty {
		t.Fatalf("expected empty list, got %v", noAlt)
	}

	_, err := ev.Eval(List(SymbolValue("if"), BoolValue(true)), nil)
	if err == nil {
		t.Fatal("expected error for too few args to if")
	}
}

func TestEvaluatorCondSelectsClause(t *testing.T) {
	ev := newTestEvaluator()
	ev.Global.Define("truthy", BoolValue(true))
	expr := List(
		SymbolValue("cond"),
		List(BoolValue(false), IntValue(1)),
		List(SymbolValue("truthy"), IntValue(2)),
		List(SymbolValue("else"), IntValue(3)),
	)
	val := mustEval(t, ev, expr)
	if val.Type != TypeInt || val.Int() != 2 {
		t.Fatalf("expected 2, got %v", val)
	}
}

func TestEvaluatorCondElseFallback(t *testing.T) {
	ev := newTestEvaluator()
	expr := List(
		SymbolValue("cond"),
		List(BoolValue(false), IntValue(1)),
		List(SymbolValue("else"), IntValue(5)),
	)
	val := mustEval(t, ev, expr)
	if val.Type != TypeInt || val.Int() != 5 {
		t.Fatalf("expected 5, got %v", val)
	}

	noMatch := mustEval(t, ev, List(
		SymbolValue("cond"),
		List(BoolValue(false), IntValue(1)),
	))
	if noMatch.Type != TypeEmpty {
		t.Fatalf("expected empty list result, got %v", noMatch)
	}
}

func TestEvaluatorCondElseMustBeLast(t *testing.T) {
	ev := newTestEvaluator()
	_, err := ev.Eval(List(
		SymbolValue("cond"),
		List(SymbolValue("else"), IntValue(1)),
		List(BoolValue(true), IntValue(2)),
	), nil)
	if err == nil || !strings.Contains(err.Error(), "else clause must be last") {
		t.Fatalf("expected else clause error, got %v", err)
	}
}

func TestEvaluatorCondClauseValidation(t *testing.T) {
	ev := newTestEvaluator()
	_, err := ev.Eval(List(SymbolValue("cond"), BoolValue(true)), nil)
	if err == nil || !strings.Contains(err.Error(), "cond clause must be a list") {
		t.Fatalf("expected clause list error, got %v", err)
	}

	_, err = ev.Eval(List(SymbolValue("cond"), List(BoolValue(true))), nil)
	if err == nil || !strings.Contains(err.Error(), "predicate and result expression") {
		t.Fatalf("expected clause arity error, got %v", err)
	}
}

func TestEvaluatorBegin(t *testing.T) {
	ev := newTestEvaluator()

	empty := mustEval(t, ev, List(SymbolValue("begin")))
	if empty.Type != TypeEmpty {
		t.Fatalf("expected empty list, got %v", empty)
	}

	val := mustEval(t, ev, List(SymbolValue("begin"), IntValue(1), IntValue(2), IntValue(3)))
	if val.Type != TypeInt || val.Int() != 3 {
		t.Fatalf("expected 3, got %v", val)
	}
}

func TestEvaluatorLambda(t *testing.T) {
	ev := newTestEvaluator()

	lambdaExpr := List(SymbolValue("lambda"), List(SymbolValue("x")), SymbolValue("x"))
	val := mustEval(t, ev, lambdaExpr)
	if val.Type != TypeClosure {
		t.Fatalf("expected closure, got %v", val)
	}

	variadic := mustEval(t, ev, List(SymbolValue("lambda"), SymbolValue("args"), List(SymbolValue("list"), SymbolValue("args"))))
	if variadic.Type != TypeClosure {
		t.Fatalf("expected variadic closure, got %v", variadic)
	}

	_, err := ev.Eval(List(SymbolValue("lambda"), List(SymbolValue("x"))), nil)
	if err == nil {
		t.Fatal("expected error for lambda without body")
	}
}

func TestEvaluatorDefine(t *testing.T) {
	ev := newTestEvaluator()

	defineVal := List(SymbolValue("define"), SymbolValue("x"), IntValue(5))
	res := mustEval(t, ev, defineVal)
	if res.Type != TypeInt || res.Int() != 5 {
		t.Fatalf("expected 5, got %v", res)
	}
	got, err := ev.Global.Get("x")
	if err != nil || got.Int() != 5 {
		t.Fatalf("expected global x = 5, got %v err=%v", got, err)
	}

	defineFn := List(
		SymbolValue("define"),
		List(SymbolValue("add1"), SymbolValue("n")),
		List(SymbolValue("+"), SymbolValue("n"), IntValue(1)),
	)
	resFn := mustEval(t, ev, defineFn)
	if resFn.Type != TypeClosure {
		t.Fatalf("expected closure, got %v", resFn)
	}
	call := List(SymbolValue("add1"), IntValue(10))
	callRes := mustEval(t, ev, call)
	if callRes.Int() != 11 {
		t.Fatalf("expected 11, got %v", callRes)
	}

	_, err = ev.Eval(List(SymbolValue("define"), IntValue(1), IntValue(2)), nil)
	if err == nil {
		t.Fatal("expected error for invalid define target")
	}
}

func TestEvaluatorDefineMacro(t *testing.T) {
	ev := newTestEvaluator()

	defineMacro := List(
		SymbolValue("define-macro"),
		List(SymbolValue("when"), SymbolValue("cond"), SymbolValue("body")),
		List(SymbolValue("if"), SymbolValue("cond"), SymbolValue("body"), SymbolValue("#f")),
	)

	macroVal := mustEval(t, ev, defineMacro)
	if macroVal.Type != TypeMacro {
		t.Fatalf("expected macro, got %v", macroVal)
	}

	ev.Global.Define("#f", BoolValue(false))
	whenTrue := mustEval(t, ev, List(SymbolValue("when"), BoolValue(true), IntValue(9)))
	if whenTrue.Type != TypeInt || whenTrue.Int() != 9 {
		t.Fatalf("expected 9, got %v", whenTrue)
	}

	whenFalse := mustEval(t, ev, List(SymbolValue("when"), BoolValue(false), IntValue(9)))
	if whenFalse.Type != TypeBool || whenFalse.Bool() {
		t.Fatalf("expected #f, got %v", whenFalse)
	}

	_, err := ev.Eval(List(SymbolValue("define-macro"), SymbolValue("bad")), nil)
	if err == nil {
		t.Fatal("expected error for malformed macro definition")
	}
}

func TestEvaluatorSet(t *testing.T) {
	ev := newTestEvaluator()
	ev.Global.Define("x", IntValue(1))

	setExpr := List(SymbolValue("set!"), SymbolValue("x"), IntValue(10))
	val := mustEval(t, ev, setExpr)
	if val.Type != TypeInt || val.Int() != 10 {
		t.Fatalf("expected 10, got %v", val)
	}
	bound, err := ev.Global.Get("x")
	if err != nil || bound.Int() != 10 {
		t.Fatalf("expected x updated to 10, got %v err=%v", bound, err)
	}

	_, err = ev.Eval(List(SymbolValue("set!"), IntValue(1), IntValue(2)), nil)
	if err == nil {
		t.Fatal("expected error for non-symbol set! target")
	}
}

func TestEvaluatorLet(t *testing.T) {
	ev := newTestEvaluator()
	letExpr := List(
		SymbolValue("let"),
		List(
			List(SymbolValue("x"), IntValue(2)),
			List(SymbolValue("y"), IntValue(3)),
		),
		List(SymbolValue("+"), SymbolValue("x"), SymbolValue("y")),
	)
	val := mustEval(t, ev, letExpr)
	if val.Type != TypeInt || val.Int() != 5 {
		t.Fatalf("expected 5, got %v", val)
	}

	_, err := ev.Eval(List(SymbolValue("let"), IntValue(1), IntValue(2)), nil)
	if err == nil {
		t.Fatal("expected error for malformed let")
	}
}

func TestEvaluatorQuasiQuote(t *testing.T) {
	ev := newTestEvaluator()
	ev.Global.Define("a", IntValue(4))
	ev.Global.Define("rest", List(IntValue(2), IntValue(3)))

	symExpr := List(SymbolValue("quasiquote"), SymbolValue("foo"))
	symVal := mustEval(t, ev, symExpr)
	if symVal.Type != TypeSymbol || symVal.Sym() != "foo" {
		t.Fatalf("expected foo, got %v", symVal)
	}

	expr := List(SymbolValue("quasiquote"), List(List(SymbolValue("unquote"), SymbolValue("a"))))
	val := mustEval(t, ev, expr)
	if val.Type != TypeInt || val.Int() != 4 {
		t.Fatalf("expected 4, got %v", val)
	}

	listExpr := List(
		SymbolValue("quasiquote"),
		List(
			IntValue(1),
			List(SymbolValue("unquote-splicing"), SymbolValue("rest")),
		),
	)
	valList := mustEval(t, ev, listExpr)
	items, err := ToSlice(valList)
	if err != nil {
		t.Fatalf("ToSlice error: %v", err)
	}
	if len(items) != 3 || items[0].Int() != 1 || items[1].Int() != 2 || items[2].Int() != 3 {
		t.Fatalf("unexpected quasiquote result: %v", valList)
	}

	_, err = ev.Eval(List(SymbolValue("quasiquote"), IntValue(1), IntValue(2)), nil)
	if err == nil {
		t.Fatal("expected error for quasiquote arity")
	}
}

func TestEvaluatorCallCC(t *testing.T) {
	ev := newTestEvaluator()

	escape := List(
		SymbolValue("call/cc"),
		List(
			SymbolValue("lambda"),
			List(SymbolValue("k")),
			List(
				SymbolValue("begin"),
				List(SymbolValue("k"), IntValue(42)),
				IntValue(100),
			),
		),
	)
	val := mustEval(t, ev, escape)
	if val.Type != TypeInt || val.Int() != 42 {
		t.Fatalf("expected 42, got %v", val)
	}

	capture := List(
		SymbolValue("call/cc"),
		List(
			SymbolValue("lambda"),
			List(SymbolValue("k")),
			SymbolValue("k"),
		),
	)
	cont := mustEval(t, ev, capture)
	if cont.Type != TypeContinuation {
		t.Fatalf("expected continuation, got %v", cont)
	}

	result, err := ev.Apply(cont, []Value{IntValue(7)})
	if err != nil {
		t.Fatalf("Apply continuation error: %v", err)
	}
	if result.Int() != 7 {
		t.Fatalf("expected 7 from continuation, got %v", result)
	}
}

func TestParseParams(t *testing.T) {
	params, rest, err := parseParams(List(SymbolValue("x"), SymbolValue("y")))
	if err != nil {
		t.Fatalf("parseParams error: %v", err)
	}
	if len(params) != 2 || params[0] != "x" || params[1] != "y" || rest != "" {
		t.Fatalf("unexpected params: %v rest=%q", params, rest)
	}

	params, rest, err = parseParams(SymbolValue("rest"))
	if err != nil {
		t.Fatalf("parseParams variadic error: %v", err)
	}
	if len(params) != 0 || rest != "rest" {
		t.Fatalf("expected rest param, got params=%v rest=%q", params, rest)
	}

	_, _, err = parseParams(IntValue(1))
	if err == nil {
		t.Fatal("expected error for invalid parameter list")
	}
}

func TestBindParameters(t *testing.T) {
	env := NewEnv(nil)
	err := bindParameters(env, []string{"x", "y"}, "", []Value{IntValue(1), IntValue(2)})
	if err != nil {
		t.Fatalf("bindParameters error: %v", err)
	}
	x, _ := env.Get("x")
	y, _ := env.Get("y")
	if x.Int() != 1 || y.Int() != 2 {
		t.Fatalf("unexpected bindings x=%v y=%v", x, y)
	}

	env2 := NewEnv(nil)
	err = bindParameters(env2, []string{"x"}, "rest", []Value{IntValue(1), IntValue(2), IntValue(3)})
	if err != nil {
		t.Fatalf("bindParameters variadic error: %v", err)
	}
	rest, _ := env2.Get("rest")
	list, err2 := ToSlice(rest)
	if err2 != nil || len(list) != 2 || list[0].Int() != 2 || list[1].Int() != 3 {
		t.Fatalf("unexpected rest binding: %v", rest)
	}

	err = bindParameters(NewEnv(nil), []string{"x", "y"}, "", []Value{IntValue(1)})
	if err == nil {
		t.Fatal("expected error for too few args")
	}

	err = bindParameters(NewEnv(nil), []string{"x"}, "", []Value{IntValue(1), IntValue(2)})
	if err == nil {
		t.Fatal("expected error for too many args without rest")
	}
}

func TestExpandQuasiQuote(t *testing.T) {
	expandedSym, err := expandQuasiQuote(SymbolValue("foo"), 1)
	if err != nil {
		t.Fatalf("expandQuasiQuote symbol error: %v", err)
	}
	expectedSym := List(SymbolValue("quote"), SymbolValue("foo"))
	if !valuesEqual(expandedSym, expectedSym) {
		t.Fatalf("expected %v, got %v", expectedSym, expandedSym)
	}

	expanded, err := expandQuasiQuote(List(List(SymbolValue("unquote"), SymbolValue("a"))), 1)
	if err != nil {
		t.Fatalf("expandQuasiQuote error: %v", err)
	}
	if expanded.Type != TypeSymbol || expanded.Sym() != "a" {
		t.Fatalf("unexpected expansion: %v", expanded)
	}

	_, err = expandQuasiQuote(List(List(SymbolValue("unquote"))), 1)
	if err == nil {
		t.Fatal("expected error for malformed unquote")
	}
}

func TestTaggedForm(t *testing.T) {
	value := List(SymbolValue("unquote"), IntValue(1))
	arg, ok, err := taggedForm(value, "unquote")
	if err != nil || !ok {
		t.Fatalf("expected tagged form, err=%v ok=%v", err, ok)
	}
	if arg.Type != TypeInt || arg.Int() != 1 {
		t.Fatalf("unexpected argument %v", arg)
	}

	_, ok, err = taggedForm(IntValue(1), "unquote")
	if err != nil || ok {
		t.Fatalf("expected no match, ok=%v err=%v", ok, err)
	}

	_, _, err = taggedForm(List(SymbolValue("unquote"), IntValue(1), IntValue(2)), "unquote")
	if err == nil {
		t.Fatal("expected error for arity")
	}
}

func TestListToSliceRaw(t *testing.T) {
	list := List(IntValue(1), IntValue(2), IntValue(3))
	slice, err := listToSliceRaw(list)
	if err != nil {
		t.Fatalf("listToSliceRaw error: %v", err)
	}
	if len(slice) != 3 || slice[0].Int() != 1 || slice[2].Int() != 3 {
		t.Fatalf("unexpected slice: %v", slice)
	}

	_, err = listToSliceRaw(IntValue(1))
	if err == nil {
		t.Fatal("expected error for improper list")
	}
}

type mockFrame struct {
	id      int
	cloned  bool
	applied bool
}

func (m *mockFrame) apply(_ *Evaluator, _ Value, _ *evalState) error {
	m.applied = true
	return nil
}

func (m *mockFrame) clone() frame {
	return &mockFrame{id: m.id, cloned: true}
}

func TestCloneFrames(t *testing.T) {
	frames := []frame{&mockFrame{id: 1}, &mockFrame{id: 2}}
	cloned := cloneFrames(frames)
	if len(cloned) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(cloned))
	}
	if cloned[0] == frames[0] || cloned[1] == frames[1] {
		t.Fatal("expected distinct cloned frames")
	}
	if mf, ok := cloned[0].(*mockFrame); !ok || !mf.cloned {
		t.Fatalf("expected cloned flag on first frame, got %#v", cloned[0])
	}
}

func TestIsTruthy(t *testing.T) {
	if !IsTruthy(IntValue(1)) {
		t.Fatal("int should be truthy")
	}
	if !IsTruthy(BoolValue(true)) {
		t.Fatal("true should be truthy")
	}
	if IsTruthy(BoolValue(false)) {
		t.Fatal("false should be falsy")
	}
}

func valuesEqual(a, b Value) bool {
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case TypeEmpty:
		return true
	case TypeBool:
		return a.Bool() == b.Bool()
	case TypeInt:
		return a.Int() == b.Int()
	case TypeReal:
		return a.Real() == b.Real()
	case TypeString:
		return a.Str() == b.Str()
	case TypeSymbol:
		return a.Sym() == b.Sym()
	case TypePair:
		ap := a.Pair()
		bp := b.Pair()
		if ap == nil || bp == nil {
			return ap == bp
		}
		return valuesEqual(ap.First, bp.First) && valuesEqual(ap.Rest, bp.Rest)
	case TypePrimitive:
		ap := a.Primitive()
		bp := b.Primitive()
		if ap == nil || bp == nil {
			return ap == nil && bp == nil
		}
		return reflect.ValueOf(ap).Pointer() == reflect.ValueOf(bp).Pointer()
	case TypeClosure:
		return a.Closure() == b.Closure()
	case TypeContinuation:
		return a.Continuation() == b.Continuation()
	case TypeMacro:
		return a.Macro() == b.Macro()
	default:
		return false
	}
}
