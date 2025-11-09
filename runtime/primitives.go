package runtime

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/sergev/gisp/lang"
)

func installPrimitives(ev *lang.Evaluator) {
	env := ev.Global
	define := func(name string, fn lang.Primitive) {
		env.Define(name, lang.PrimitiveValue(fn))
	}

	define("+", primAdd)
	define("-", primSub)
	define("*", primMul)
	define("/", primDiv)

	define("=", primNumEq)
	define("<", primLess)
	define("<=", primLessEq)
	define(">", primGreater)
	define(">=", primGreaterEq)

	define("not", primNot)

	define("numberp", primIsNumber)
	define("integerp", primIsInteger)
	define("realp", primIsReal)
	define("booleanp", primIsBoolean)
	define("stringp", primIsString)
	define("symbolp", primIsSymbol)
	define("pairp", primIsPair)
	define("nullp", primIsNull)
	define("listp", primIsList)
	define("procedurep", primIsProcedure)

	define("cons", primCons)
	define("car", primCar)
	define("cdr", primCdr)
	define("list", primList)
	define("append", primAppend)
	define("length", primLength)

	define("eq", primEq)
	define("equal", primEqual)

	define("display", primDisplay)
	define("newline", primNewline)
	define("exit", primExit)

	define("apply", primApply)
	define("gensym", primGensym)
	define("stringAppend", primStringAppend)
	define("symbolToString", primSymbolToString)
	define("stringToSymbol", primStringToSymbol)
	define("numberToString", primNumberToString)
	define("stringToNumber", primStringToNumber)
}

func primAdd(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	sumInt := int64(0)
	sumFloat := 0.0
	useFloat := false
	for _, arg := range args {
		switch arg.Type {
		case lang.TypeInt:
			if useFloat {
				sumFloat += float64(arg.Int)
			} else {
				sumInt += arg.Int
			}
		case lang.TypeReal:
			if !useFloat {
				useFloat = true
				sumFloat = float64(sumInt)
			}
			sumFloat += arg.Real
		default:
			return lang.Value{}, typeError("+", "number", arg)
		}
	}
	if useFloat {
		return lang.RealValue(sumFloat), nil
	}
	return lang.IntValue(sumInt), nil
}

func primMul(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	prodInt := int64(1)
	prodFloat := 1.0
	useFloat := false
	if len(args) == 0 {
		return lang.IntValue(1), nil
	}
	for _, arg := range args {
		switch arg.Type {
		case lang.TypeInt:
			if useFloat {
				prodFloat *= float64(arg.Int)
			} else {
				prodInt *= arg.Int
			}
		case lang.TypeReal:
			if !useFloat {
				useFloat = true
				prodFloat = float64(prodInt)
			}
			prodFloat *= arg.Real
		default:
			return lang.Value{}, typeError("*", "number", arg)
		}
	}
	if useFloat {
		return lang.RealValue(prodFloat), nil
	}
	return lang.IntValue(prodInt), nil
}

func primSub(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) == 0 {
		return lang.Value{}, errors.New("- expects at least one argument")
	}
	first := args[0]
	useFloat := first.Type == lang.TypeReal
	accInt := int64(0)
	accFloat := 0.0
	switch first.Type {
	case lang.TypeInt:
		accInt = first.Int
	case lang.TypeReal:
		accFloat = first.Real
	default:
		return lang.Value{}, typeError("-", "number", first)
	}
	if len(args) == 1 {
		if useFloat {
			return lang.RealValue(-accFloat), nil
		}
		return lang.IntValue(-accInt), nil
	}
	for _, arg := range args[1:] {
		switch arg.Type {
		case lang.TypeInt:
			if useFloat {
				accFloat -= float64(arg.Int)
			} else {
				accInt -= arg.Int
			}
		case lang.TypeReal:
			if !useFloat {
				useFloat = true
				accFloat = float64(accInt)
			}
			accFloat -= arg.Real
		default:
			return lang.Value{}, typeError("-", "number", arg)
		}
	}
	if useFloat {
		return lang.RealValue(accFloat), nil
	}
	return lang.IntValue(accInt), nil
}

func primDiv(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) == 0 {
		return lang.Value{}, errors.New("/ expects at least one argument")
	}
	initial, err := toFloat(args[0])
	if err != nil {
		return lang.Value{}, typeError("/", "number", args[0])
	}
	if initial == 0 {
		return lang.Value{}, errors.New("division by zero")
	}
	acc := initial
	if len(args) == 1 {
		return lang.RealValue(1 / acc), nil
	}
	for _, arg := range args[1:] {
		val, err := toFloat(arg)
		if err != nil {
			return lang.Value{}, typeError("/", "number", arg)
		}
		if val == 0 {
			return lang.Value{}, errors.New("division by zero")
		}
		acc /= val
	}
	return lang.RealValue(acc), nil
}

func primNumEq(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) < 2 {
		return lang.BoolValue(true), nil
	}
	first, err := toFloat(args[0])
	if err != nil {
		return lang.Value{}, typeError("=", "number", args[0])
	}
	for _, arg := range args[1:] {
		val, err := toFloat(arg)
		if err != nil {
			return lang.Value{}, typeError("=", "number", arg)
		}
		if val != first {
			return lang.BoolValue(false), nil
		}
	}
	return lang.BoolValue(true), nil
}

func primLess(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return compareChain("<", func(a, b float64) bool { return a < b }, args)
}

func primLessEq(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return compareChain("<=", func(a, b float64) bool { return a <= b }, args)
}

func primGreater(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return compareChain(">", func(a, b float64) bool { return a > b }, args)
}

func primGreaterEq(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return compareChain(">=", func(a, b float64) bool { return a >= b }, args)
}

func compareChain(name string, cmp func(float64, float64) bool, args []lang.Value) (lang.Value, error) {
	if len(args) < 2 {
		return lang.BoolValue(true), nil
	}
	prev, err := toFloat(args[0])
	if err != nil {
		return lang.Value{}, typeError(name, "number", args[0])
	}
	for _, arg := range args[1:] {
		cur, err := toFloat(arg)
		if err != nil {
			return lang.Value{}, typeError(name, "number", arg)
		}
		if !cmp(prev, cur) {
			return lang.BoolValue(false), nil
		}
		prev = cur
	}
	return lang.BoolValue(true), nil
}

func primNot(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("not expects 1 argument, got %d", len(args))
	}
	return lang.BoolValue(!lang.IsTruthy(args[0])), nil
}

func primIsNumber(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return unaryTypePredicate("numberp", args, func(v lang.Value) bool {
		return v.Type == lang.TypeInt || v.Type == lang.TypeReal
	})
}

func primIsInteger(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return unaryTypePredicate("integerp", args, func(v lang.Value) bool {
		return v.Type == lang.TypeInt
	})
}

func primIsReal(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return unaryTypePredicate("realp", args, func(v lang.Value) bool {
		return v.Type == lang.TypeReal || v.Type == lang.TypeInt
	})
}

func primIsBoolean(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return unaryTypePredicate("booleanp", args, func(v lang.Value) bool {
		return v.Type == lang.TypeBool
	})
}

func primIsString(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return unaryTypePredicate("stringp", args, func(v lang.Value) bool {
		return v.Type == lang.TypeString
	})
}

func primIsSymbol(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return unaryTypePredicate("symbolp", args, func(v lang.Value) bool {
		return v.Type == lang.TypeSymbol
	})
}

func primIsPair(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return unaryTypePredicate("pairp", args, func(v lang.Value) bool {
		return v.Type == lang.TypePair
	})
}

func primIsNull(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return unaryTypePredicate("nullp", args, func(v lang.Value) bool {
		return v.Type == lang.TypeEmpty
	})
}

func primIsList(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return unaryTypePredicate("listp", args, func(v lang.Value) bool {
		_, err := lang.ToSlice(v)
		return err == nil
	})
}

func primIsProcedure(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return unaryTypePredicate("procedurep", args, func(v lang.Value) bool {
		return v.Type == lang.TypePrimitive || v.Type == lang.TypeClosure || v.Type == lang.TypeContinuation
	})
}

func primCons(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 2 {
		return lang.Value{}, fmt.Errorf("cons expects 2 arguments, got %d", len(args))
	}
	return lang.PairValue(args[0], args[1]), nil
}

func primCar(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("car expects 1 argument, got %d", len(args))
	}
	v := args[0]
	if v.Type != lang.TypePair || v.Pair == nil {
		return lang.Value{}, fmt.Errorf("car expects a pair")
	}
	return v.Pair.Car, nil
}

func primCdr(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("cdr expects 1 argument, got %d", len(args))
	}
	v := args[0]
	if v.Type != lang.TypePair || v.Pair == nil {
		return lang.Value{}, fmt.Errorf("cdr expects a pair")
	}
	return v.Pair.Cdr, nil
}

func primList(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return lang.List(args...), nil
}

func primAppend(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) == 0 {
		return lang.EmptyList, nil
	}
	result := args[len(args)-1]
	for i := len(args) - 2; i >= 0; i-- {
		items, err := lang.ToSlice(args[i])
		if err != nil {
			return lang.Value{}, fmt.Errorf("append expects lists: %w", err)
		}
		for j := len(items) - 1; j >= 0; j-- {
			result = lang.PairValue(items[j], result)
		}
	}
	return result, nil
}

func primLength(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("length expects 1 argument, got %d", len(args))
	}
	items, err := lang.ToSlice(args[0])
	if err != nil {
		return lang.Value{}, err
	}
	return lang.IntValue(int64(len(items))), nil
}

func primEq(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 2 {
		return lang.Value{}, fmt.Errorf("eq expects 2 arguments, got %d", len(args))
	}
	return lang.BoolValue(eqValues(args[0], args[1])), nil
}

func primEqual(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 2 {
		return lang.Value{}, fmt.Errorf("equal expects 2 arguments, got %d", len(args))
	}
	return lang.BoolValue(equalValues(args[0], args[1])), nil
}

func primDisplay(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("display expects 1 argument, got %d", len(args))
	}
	v := args[0]
	switch v.Type {
	case lang.TypeString:
		fmt.Fprint(os.Stdout, v.Str)
	default:
		fmt.Fprint(os.Stdout, v.String())
	}
	return lang.EmptyList, nil
}

func primNewline(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 0 {
		return lang.Value{}, fmt.Errorf("newline expects no arguments")
	}
	fmt.Fprintln(os.Stdout)
	return lang.EmptyList, nil
}

func primExit(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	code := 0
	if len(args) > 0 {
		if len(args) != 1 {
			return lang.Value{}, fmt.Errorf("exit expects at most 1 argument")
		}
		switch args[0].Type {
		case lang.TypeInt:
			code = int(args[0].Int)
		case lang.TypeBool:
			if args[0].Bool {
				code = 0
			} else {
				code = 1
			}
		default:
			return lang.Value{}, typeError("exit", "integer or boolean", args[0])
		}
	}
	os.Exit(code)
	return lang.EmptyList, nil
}

func primApply(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) < 2 {
		return lang.Value{}, fmt.Errorf("apply expects at least 2 arguments")
	}
	proc := args[0]
	var callArgs []lang.Value
	if len(args) > 2 {
		callArgs = append(callArgs, args[1:len(args)-1]...)
	}
	last := args[len(args)-1]
	lastArgs, err := lang.ToSlice(last)
	if err != nil {
		return lang.Value{}, fmt.Errorf("apply expects final argument to be a list")
	}
	callArgs = append(callArgs, lastArgs...)
	return ev.Apply(proc, callArgs)
}

var gensymCounter int64

func primGensym(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 0 {
		return lang.Value{}, fmt.Errorf("gensym expects no arguments")
	}
	name := fmt.Sprintf("g%d", gensymCounter)
	gensymCounter++
	return lang.SymbolValue(name), nil
}

func primStringAppend(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	var builder strings.Builder
	for _, arg := range args {
		if arg.Type != lang.TypeString {
			return lang.Value{}, typeError("stringAppend", "string", arg)
		}
		builder.WriteString(arg.Str)
	}
	return lang.StringValue(builder.String()), nil
}

func primSymbolToString(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("symbolToString expects 1 argument, got %d", len(args))
	}
	if args[0].Type != lang.TypeSymbol {
		return lang.Value{}, typeError("symbolToString", "symbol", args[0])
	}
	return lang.StringValue(args[0].Sym), nil
}

func primStringToSymbol(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("stringToSymbol expects 1 argument, got %d", len(args))
	}
	if args[0].Type != lang.TypeString {
		return lang.Value{}, typeError("stringToSymbol", "string", args[0])
	}
	return lang.SymbolValue(args[0].Str), nil
}

func primNumberToString(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("numberToString expects 1 argument, got %d", len(args))
	}
	switch args[0].Type {
	case lang.TypeInt:
		return lang.StringValue(strconv.FormatInt(args[0].Int, 10)), nil
	case lang.TypeReal:
		return lang.StringValue(strconv.FormatFloat(args[0].Real, 'g', -1, 64)), nil
	default:
		return lang.Value{}, typeError("numberToString", "number", args[0])
	}
}

func primStringToNumber(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("stringToNumber expects 1 argument, got %d", len(args))
	}
	if args[0].Type != lang.TypeString {
		return lang.Value{}, typeError("stringToNumber", "string", args[0])
	}
	str := strings.TrimSpace(args[0].Str)
	if str == "" {
		return lang.BoolValue(false), nil
	}
	if i, err := strconv.ParseInt(str, 10, 64); err == nil {
		return lang.IntValue(i), nil
	}
	if f, err := strconv.ParseFloat(str, 64); err == nil {
		return lang.RealValue(f), nil
	}
	return lang.BoolValue(false), nil
}

func unaryTypePredicate(name string, args []lang.Value, pred func(lang.Value) bool) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("%s expects 1 argument, got %d", name, len(args))
	}
	return lang.BoolValue(pred(args[0])), nil
}

func typeError(name, expected string, got lang.Value) error {
	return fmt.Errorf("%s expects %s, got %s", name, expected, typeName(got))
}

func typeName(v lang.Value) string {
	switch v.Type {
	case lang.TypeEmpty:
		return "empty-list"
	case lang.TypeBool:
		return "boolean"
	case lang.TypeInt:
		return "integer"
	case lang.TypeReal:
		return "real"
	case lang.TypeString:
		return "string"
	case lang.TypeSymbol:
		return "symbol"
	case lang.TypePair:
		return "pair"
	case lang.TypePrimitive:
		return "primitive"
	case lang.TypeClosure:
		return "closure"
	case lang.TypeContinuation:
		return "continuation"
	case lang.TypeMacro:
		return "macro"
	default:
		return "unknown"
	}
}

func toFloat(v lang.Value) (float64, error) {
	switch v.Type {
	case lang.TypeInt:
		return float64(v.Int), nil
	case lang.TypeReal:
		return v.Real, nil
	default:
		return 0, fmt.Errorf("expected number")
	}
}

func eqValues(a, b lang.Value) bool {
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case lang.TypeEmpty:
		return true
	case lang.TypeBool:
		return a.Bool == b.Bool
	case lang.TypeInt:
		return a.Int == b.Int
	case lang.TypeReal:
		return a.Real == b.Real
	case lang.TypeString:
		return a.Str == b.Str
	case lang.TypeSymbol:
		return a.Sym == b.Sym
	case lang.TypePair:
		return a.Pair == b.Pair
	case lang.TypePrimitive:
		return primitivePointer(a.Primitive) == primitivePointer(b.Primitive)
	case lang.TypeClosure:
		return a.Closure == b.Closure
	case lang.TypeContinuation:
		return a.Continuation == b.Continuation
	case lang.TypeMacro:
		return a.Macro == b.Macro
	default:
		return false
	}
}

func equalValues(a, b lang.Value) bool {
	if a.Type == lang.TypeInt && b.Type == lang.TypeReal {
		return float64(a.Int) == b.Real
	}
	if a.Type == lang.TypeReal && b.Type == lang.TypeInt {
		return a.Real == float64(b.Int)
	}
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case lang.TypeEmpty:
		return true
	case lang.TypeBool:
		return a.Bool == b.Bool
	case lang.TypeInt:
		return a.Int == b.Int
	case lang.TypeReal:
		return a.Real == b.Real
	case lang.TypeString:
		return a.Str == b.Str
	case lang.TypeSymbol:
		return a.Sym == b.Sym
	case lang.TypePair:
		if a.Pair == nil || b.Pair == nil {
			return a.Pair == b.Pair
		}
		return equalValues(a.Pair.Car, b.Pair.Car) && equalValues(a.Pair.Cdr, b.Pair.Cdr)
	case lang.TypePrimitive:
		return primitivePointer(a.Primitive) == primitivePointer(b.Primitive)
	case lang.TypeClosure:
		return a.Closure == b.Closure
	case lang.TypeContinuation:
		return a.Continuation == b.Continuation
	case lang.TypeMacro:
		return a.Macro == b.Macro
	default:
		return false
	}
}

func primitivePointer(p lang.Primitive) uintptr {
	if p == nil {
		return 0
	}
	return reflect.ValueOf(p).Pointer()
}
