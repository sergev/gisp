package runtime

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sergev/gisp/lang"
	"github.com/sergev/gisp/sexpr"
)

var (
	randomMu   sync.Mutex
	randomRand = rand.New(rand.NewSource(time.Now().UnixNano()))

	readMu     sync.Mutex
	readStream = sexpr.NewReader(os.Stdin)
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
	define("%", primMod)
	define("++", primPostInc)
	define("--", primPostDec)
	define("+=", primAddAssign)
	define("-=", primSubAssign)
	define("*=", primMulAssign)
	define("/=", primDivAssign)
	define("%=", primModAssign)
	define("&", primBitAnd)
	define("|", primBitOr)
	define("^", primBitXor)
	define("&^", primBitClear)
	define("<<", primShiftLeft)
	define(">>", primShiftRight)
	define("<<=", primShiftLeftAssign)
	define(">>=", primShiftRightAssign)
	define("&=", primBitAndAssign)
	define("|=", primBitOrAssign)
	define("^=", primBitXorAssign)
	define("&^=", primBitClearAssign)

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
	define("first", primFirst)
	define("rest", primRest)
	define("setFirst", primSetFirst)
	define("setRest", primSetRest)
	define("list", primList)
	define("append", primAppend)
	define("length", primLength)
	define("vector", primVector)
	define("vectorp", primIsVector)
	define("makeVector", primMakeVector)
	define("vectorLength", primVectorLength)
	define("vectorRef", primVectorRef)
	define("vectorSet", primVectorSet)
	define("vectorFill", primVectorFill)
	define("vectorToList", primVectorToList)
	define("listToVector", primListToVector)

	define("eq", primEq)
	define("equal", primEqual)

	define("display", primDisplay)
	define("newline", primNewline)
	define("read", primRead)
	define("exit", primExit)
	define("error", primError)

	define("apply", primApply)
	define("gensym", primGensym)
	define("randomInteger", primRandomInteger)
	define("randomSeed", primRandomSeed)
	define("stringLength", primStringLength)
	define("makeString", primMakeString)
	define("stringAppend", primStringAppend)
	define("stringSlice", primStringSlice)
	define("symbolToString", primSymbolToString)
	define("stringToSymbol", primStringToSymbol)
	define("numberToString", primNumberToString)
	define("stringToNumber", primStringToNumber)

	env.Define("callcc", lang.ClosureValue(
		[]string{"f"},
		"",
		[]lang.Value{
			lang.List(
				lang.SymbolValue("call/cc"),
				lang.SymbolValue("f"),
			),
		},
		env,
	))

	env.Define("map", lang.ClosureValue(
		[]string{"proc", "lst"},
		"",
		[]lang.Value{
			lang.List(
				lang.SymbolValue("if"),
				lang.List(
					lang.SymbolValue("nullp"),
					lang.SymbolValue("lst"),
				),
				lang.List(
					lang.SymbolValue("quote"),
					lang.EmptyList,
				),
				lang.List(
					lang.SymbolValue("cons"),
					lang.List(
						lang.SymbolValue("proc"),
						lang.List(
							lang.SymbolValue("first"),
							lang.SymbolValue("lst"),
						),
					),
					lang.List(
						lang.SymbolValue("map"),
						lang.SymbolValue("proc"),
						lang.List(
							lang.SymbolValue("rest"),
							lang.SymbolValue("lst"),
						),
					),
				),
			),
		},
		env,
	))

	env.Define("filter", lang.ClosureValue(
		[]string{"pred", "lst"},
		"",
		[]lang.Value{
			lang.List(
				lang.SymbolValue("cond"),
				lang.List(
					lang.List(
						lang.SymbolValue("nullp"),
						lang.SymbolValue("lst"),
					),
					lang.List(
						lang.SymbolValue("quote"),
						lang.EmptyList,
					),
				),
				lang.List(
					lang.List(
						lang.SymbolValue("pred"),
						lang.List(
							lang.SymbolValue("first"),
							lang.SymbolValue("lst"),
						),
					),
					lang.List(
						lang.SymbolValue("cons"),
						lang.List(
							lang.SymbolValue("first"),
							lang.SymbolValue("lst"),
						),
						lang.List(
							lang.SymbolValue("filter"),
							lang.SymbolValue("pred"),
							lang.List(
								lang.SymbolValue("rest"),
								lang.SymbolValue("lst"),
							),
						),
					),
				),
				lang.List(
					lang.SymbolValue("else"),
					lang.List(
						lang.SymbolValue("filter"),
						lang.SymbolValue("pred"),
						lang.List(
							lang.SymbolValue("rest"),
							lang.SymbolValue("lst"),
						),
					),
				),
			),
		},
		env,
	))
}

func primAdd(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	sumInt := int64(0)
	sumFloat := 0.0
	useFloat := false
	for _, arg := range args {
		switch arg.Type {
		case lang.TypeInt:
			if useFloat {
				sumFloat += float64(arg.Int())
			} else {
				sumInt += arg.Int()
			}
		case lang.TypeReal:
			if !useFloat {
				useFloat = true
				sumFloat = float64(sumInt)
			}
			sumFloat += arg.Real()
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
				prodFloat *= float64(arg.Int())
			} else {
				prodInt *= arg.Int()
			}
		case lang.TypeReal:
			if !useFloat {
				useFloat = true
				prodFloat = float64(prodInt)
			}
			prodFloat *= arg.Real()
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
		accInt = first.Int()
	case lang.TypeReal:
		accFloat = first.Real()
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
				accFloat -= float64(arg.Int())
			} else {
				accInt -= arg.Int()
			}
		case lang.TypeReal:
			if !useFloat {
				useFloat = true
				accFloat = float64(accInt)
			}
			accFloat -= arg.Real()
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

func primMod(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) < 2 {
		return lang.Value{}, errors.New("% expects at least 2 arguments")
	}
	if args[0].Type != lang.TypeInt {
		return lang.Value{}, typeError("%", "integer", args[0])
	}
	result := args[0].Int()
	for _, arg := range args[1:] {
		if arg.Type != lang.TypeInt {
			return lang.Value{}, typeError("%", "integer", arg)
		}
		divisor := arg.Int()
		if divisor == 0 {
			return lang.Value{}, errors.New("modulo by zero")
		}
		result %= divisor
	}
	return lang.IntValue(result), nil
}

func primAddAssign(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return compoundAssign(ev, "+=", args, func(current, delta lang.Value) (lang.Value, error) {
		return primAdd(ev, []lang.Value{current, delta})
	})
}

func primSubAssign(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return compoundAssign(ev, "-=", args, func(current, delta lang.Value) (lang.Value, error) {
		return primSub(ev, []lang.Value{current, delta})
	})
}

func primMulAssign(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return compoundAssign(ev, "*=", args, func(current, delta lang.Value) (lang.Value, error) {
		return primMul(ev, []lang.Value{current, delta})
	})
}

func primDivAssign(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return compoundAssign(ev, "/=", args, func(current, delta lang.Value) (lang.Value, error) {
		return primDiv(ev, []lang.Value{current, delta})
	})
}

func primModAssign(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return compoundAssign(ev, "%=", args, func(current, delta lang.Value) (lang.Value, error) {
		return primMod(ev, []lang.Value{current, delta})
	})
}

func primPostInc(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return incDecPrimitive(ev, "++", args, 1)
}

func primPostDec(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return incDecPrimitive(ev, "--", args, -1)
}

func primShiftLeftAssign(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return compoundAssign(ev, "<<=", args, func(current, delta lang.Value) (lang.Value, error) {
		return primShiftLeft(ev, []lang.Value{current, delta})
	})
}

func primShiftRightAssign(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return compoundAssign(ev, ">>=", args, func(current, delta lang.Value) (lang.Value, error) {
		return primShiftRight(ev, []lang.Value{current, delta})
	})
}

func primBitAndAssign(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return compoundAssign(ev, "&=", args, func(current, delta lang.Value) (lang.Value, error) {
		return primBitAnd(ev, []lang.Value{current, delta})
	})
}

func primBitOrAssign(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return compoundAssign(ev, "|=", args, func(current, delta lang.Value) (lang.Value, error) {
		return primBitOr(ev, []lang.Value{current, delta})
	})
}

func primBitXorAssign(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return compoundAssign(ev, "^=", args, func(current, delta lang.Value) (lang.Value, error) {
		return primBitXor(ev, []lang.Value{current, delta})
	})
}

func primBitClearAssign(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return compoundAssign(ev, "&^=", args, func(current, delta lang.Value) (lang.Value, error) {
		return primBitClear(ev, []lang.Value{current, delta})
	})
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

func primBitAnd(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) < 2 {
		return lang.Value{}, fmt.Errorf("& expects at least 2 arguments, got %d", len(args))
	}
	result, err := requireIntArg("&", args[0])
	if err != nil {
		return lang.Value{}, err
	}
	for _, arg := range args[1:] {
		value, err := requireIntArg("&", arg)
		if err != nil {
			return lang.Value{}, err
		}
		result &= value
	}
	return lang.IntValue(result), nil
}

func primBitOr(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) < 2 {
		return lang.Value{}, fmt.Errorf("| expects at least 2 arguments, got %d", len(args))
	}
	result, err := requireIntArg("|", args[0])
	if err != nil {
		return lang.Value{}, err
	}
	for _, arg := range args[1:] {
		value, err := requireIntArg("|", arg)
		if err != nil {
			return lang.Value{}, err
		}
		result |= value
	}
	return lang.IntValue(result), nil
}

func primBitXor(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) == 0 {
		return lang.Value{}, fmt.Errorf("^ expects at least 1 argument, got %d", len(args))
	}
	if len(args) == 1 {
		value, err := requireIntArg("^", args[0])
		if err != nil {
			return lang.Value{}, err
		}
		return lang.IntValue(^value), nil
	}
	result, err := requireIntArg("^", args[0])
	if err != nil {
		return lang.Value{}, err
	}
	for _, arg := range args[1:] {
		value, err := requireIntArg("^", arg)
		if err != nil {
			return lang.Value{}, err
		}
		result ^= value
	}
	return lang.IntValue(result), nil
}

func primBitClear(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) < 2 {
		return lang.Value{}, fmt.Errorf("&^ expects at least 2 arguments, got %d", len(args))
	}
	result, err := requireIntArg("&^", args[0])
	if err != nil {
		return lang.Value{}, err
	}
	for _, arg := range args[1:] {
		value, err := requireIntArg("&^", arg)
		if err != nil {
			return lang.Value{}, err
		}
		result &^= value
	}
	return lang.IntValue(result), nil
}

func primShiftLeft(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 2 {
		return lang.Value{}, fmt.Errorf("<< expects 2 arguments, got %d", len(args))
	}
	value, err := requireIntArg("<<", args[0])
	if err != nil {
		return lang.Value{}, err
	}
	shift, err := requireIntArg("<<", args[1])
	if err != nil {
		return lang.Value{}, err
	}
	if shift < 0 {
		return lang.Value{}, fmt.Errorf("<< expects non-negative shift, got %d", shift)
	}
	return lang.IntValue(value << uint(shift)), nil
}

func primShiftRight(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 2 {
		return lang.Value{}, fmt.Errorf(">> expects 2 arguments, got %d", len(args))
	}
	value, err := requireIntArg(">>", args[0])
	if err != nil {
		return lang.Value{}, err
	}
	shift, err := requireIntArg(">>", args[1])
	if err != nil {
		return lang.Value{}, err
	}
	if shift < 0 {
		return lang.Value{}, fmt.Errorf(">> expects non-negative shift, got %d", shift)
	}
	return lang.IntValue(value >> uint(shift)), nil
}

func primRandomInteger(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("randomInteger expects 1 argument, got %d", len(args))
	}
	limitVal := args[0]
	if limitVal.Type != lang.TypeInt {
		return lang.Value{}, typeError("randomInteger", "integer", limitVal)
	}
	limit := limitVal.Int()
	if limit <= 0 {
		return lang.Value{}, fmt.Errorf("randomInteger limit must be positive, got %d", limit)
	}
	randomMu.Lock()
	result := randomRand.Int63n(limit)
	randomMu.Unlock()
	return lang.IntValue(result), nil
}

func primRandomSeed(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("randomSeed expects 1 argument, got %d", len(args))
	}
	seedVal := args[0]
	if seedVal.Type != lang.TypeInt {
		return lang.Value{}, typeError("randomSeed", "integer", seedVal)
	}
	randomMu.Lock()
	randomRand.Seed(seedVal.Int())
	randomMu.Unlock()
	return lang.EmptyList, nil
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

func primFirst(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("first expects 1 argument, got %d", len(args))
	}
	v := args[0]
	p := v.Pair()
	if v.Type != lang.TypePair || p == nil {
		return lang.Value{}, fmt.Errorf("first expects a pair")
	}
	return p.First, nil
}

func primRest(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("rest expects 1 argument, got %d", len(args))
	}
	v := args[0]
	p := v.Pair()
	if v.Type != lang.TypePair || p == nil {
		return lang.Value{}, fmt.Errorf("rest expects a pair")
	}
	return p.Rest, nil
}

func primSetFirst(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 2 {
		return lang.Value{}, fmt.Errorf("set-first! expects 2 arguments, got %d", len(args))
	}
	pair := args[0]
	p := pair.Pair()
	if pair.Type != lang.TypePair || p == nil {
		return lang.Value{}, fmt.Errorf("set-first! expects a pair")
	}
	p.First = args[1]
	return pair, nil
}

func primSetRest(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 2 {
		return lang.Value{}, fmt.Errorf("set-rest! expects 2 arguments, got %d", len(args))
	}
	pair := args[0]
	p := pair.Pair()
	if pair.Type != lang.TypePair || p == nil {
		return lang.Value{}, fmt.Errorf("set-rest! expects a pair")
	}
	p.Rest = args[1]
	return pair, nil
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

func primVector(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return lang.VectorValue(args), nil
}

func primIsVector(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	return unaryTypePredicate("vectorp", args, func(v lang.Value) bool {
		return v.Type == lang.TypeVector
	})
}

func primMakeVector(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return lang.Value{}, fmt.Errorf("makeVector expects 1 or 2 arguments, got %d", len(args))
	}
	sizeArg := args[0]
	if sizeArg.Type != lang.TypeInt {
		return lang.Value{}, typeError("makeVector", "integer", sizeArg)
	}
	length64 := sizeArg.Int()
	if length64 < 0 {
		return lang.Value{}, fmt.Errorf("makeVector length must be non-negative, got %d", length64)
	}
	length := int(length64)
	if int64(length) != length64 {
		return lang.Value{}, fmt.Errorf("makeVector length %d exceeds platform limit", length64)
	}
	fill := lang.EmptyList
	if len(args) == 2 {
		fill = args[1]
	}
	return lang.NewVector(length, fill), nil
}

func primVectorLength(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("vectorLength expects 1 argument, got %d", len(args))
	}
	vec, err := requireVectorArg("vectorLength", args[0])
	if err != nil {
		return lang.Value{}, err
	}
	return lang.IntValue(int64(len(vec.Elements))), nil
}

func primVectorRef(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 2 {
		return lang.Value{}, fmt.Errorf("vectorRef expects 2 arguments, got %d", len(args))
	}
	vec, err := requireVectorArg("vectorRef", args[0])
	if err != nil {
		return lang.Value{}, err
	}
	indexArg := args[1]
	if indexArg.Type != lang.TypeInt {
		return lang.Value{}, typeError("vectorRef", "integer", indexArg)
	}
	idx64 := indexArg.Int()
	length := len(vec.Elements)
	if idx64 < 0 || idx64 >= int64(length) {
		return lang.Value{}, fmt.Errorf("vectorRef index %d out of range for length %d", idx64, length)
	}
	idx := int(idx64)
	return vec.Elements[idx], nil
}

func primVectorSet(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 3 {
		return lang.Value{}, fmt.Errorf("vectorSet expects 3 arguments, got %d", len(args))
	}
	vecVal := args[0]
	vec, err := requireVectorArg("vectorSet", vecVal)
	if err != nil {
		return lang.Value{}, err
	}
	indexArg := args[1]
	if indexArg.Type != lang.TypeInt {
		return lang.Value{}, typeError("vectorSet", "integer", indexArg)
	}
	idx64 := indexArg.Int()
	length := len(vec.Elements)
	if idx64 < 0 || idx64 >= int64(length) {
		return lang.Value{}, fmt.Errorf("vectorSet index %d out of range for length %d", idx64, length)
	}
	idx := int(idx64)
	vec.Elements[idx] = args[2]
	return vecVal, nil
}

func primVectorFill(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 2 {
		return lang.Value{}, fmt.Errorf("vectorFill expects 2 arguments, got %d", len(args))
	}
	vecVal := args[0]
	vec, err := requireVectorArg("vectorFill", vecVal)
	if err != nil {
		return lang.Value{}, err
	}
	fill := args[1]
	for i := range vec.Elements {
		vec.Elements[i] = fill
	}
	return vecVal, nil
}

func primVectorToList(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("vectorToList expects 1 argument, got %d", len(args))
	}
	vec, err := requireVectorArg("vectorToList", args[0])
	if err != nil {
		return lang.Value{}, err
	}
	return lang.List(vec.Elements...), nil
}

func primListToVector(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("listToVector expects 1 argument, got %d", len(args))
	}
	items, err := lang.ToSlice(args[0])
	if err != nil {
		return lang.Value{}, fmt.Errorf("listToVector expects a proper list: %w", err)
	}
	return lang.VectorValue(items), nil
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
		fmt.Fprint(os.Stdout, v.Str())
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

func primRead(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 0 {
		return lang.Value{}, fmt.Errorf("read expects no arguments")
	}
	readMu.Lock()
	defer readMu.Unlock()
	if readStream == nil {
		readStream = sexpr.NewReader(os.Stdin)
	}
	val, err := readStream.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return lang.EOFObject, nil
		}
		return lang.Value{}, err
	}
	return val, nil
}

func primExit(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	code := 0
	if len(args) > 0 {
		if len(args) != 1 {
			return lang.Value{}, fmt.Errorf("exit expects at most 1 argument")
		}
		switch args[0].Type {
		case lang.TypeInt:
			code = int(args[0].Int())
		case lang.TypeBool:
			if args[0].Bool() {
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

func primError(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) == 0 {
		return lang.Value{}, fmt.Errorf("error")
	}
	parts := make([]string, len(args))
	for i, arg := range args {
		if arg.Type == lang.TypeString {
			parts[i] = arg.Str()
		} else {
			parts[i] = arg.String()
		}
	}
	return lang.Value{}, fmt.Errorf("%s", strings.Join(parts, " "))
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
		builder.WriteString(arg.Str())
	}
	return lang.StringValue(builder.String()), nil
}

func primStringSlice(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return lang.Value{}, fmt.Errorf("stringSlice expects 2 or 3 arguments, got %d", len(args))
	}
	source := args[0]
	if source.Type != lang.TypeString {
		return lang.Value{}, typeError("stringSlice", "string", source)
	}
	startVal := args[1]
	if startVal.Type != lang.TypeInt {
		return lang.Value{}, typeError("stringSlice", "integer", startVal)
	}
	start := startVal.Int()
	str := source.Str()
	length := int64(len(str))
	if start < 0 || start > length {
		return lang.Value{}, fmt.Errorf("stringSlice start index %d out of range 0..%d", start, length)
	}
	end := length
	if len(args) == 3 {
		endVal := args[2]
		if endVal.Type != lang.TypeInt {
			return lang.Value{}, typeError("stringSlice", "integer", endVal)
		}
		end = endVal.Int()
		if end < 0 || end > length {
			return lang.Value{}, fmt.Errorf("stringSlice end index %d out of range 0..%d", end, length)
		}
	}
	if end < start {
		return lang.Value{}, fmt.Errorf("stringSlice end index %d precedes start %d", end, start)
	}
	return lang.StringValue(str[start:end]), nil
}

func primStringLength(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("stringLength expects 1 argument, got %d", len(args))
	}
	if args[0].Type != lang.TypeString {
		return lang.Value{}, typeError("stringLength", "string", args[0])
	}
	return lang.IntValue(int64(len(args[0].Str()))), nil
}

func primMakeString(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return lang.Value{}, fmt.Errorf("makeString expects 1 or 2 arguments, got %d", len(args))
	}
	lengthArg := args[0]
	if lengthArg.Type != lang.TypeInt {
		return lang.Value{}, typeError("makeString", "integer", lengthArg)
	}
	length := lengthArg.Int()
	if length < 0 {
		return lang.Value{}, fmt.Errorf("makeString length must be non-negative, got %d", length)
	}
	fill := " "
	if len(args) == 2 {
		if args[1].Type != lang.TypeString {
			return lang.Value{}, typeError("makeString", "string", args[1])
		}
		str := args[1].Str()
		if len(str) != 1 {
			return lang.Value{}, fmt.Errorf("makeString expects single-character fill string, got length %d", len(str))
		}
		fill = str
	}
	if length == 0 {
		return lang.StringValue(""), nil
	}
	var builder strings.Builder
	builder.Grow(int(length))
	for i := int64(0); i < length; i++ {
		builder.WriteString(fill)
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
	return lang.StringValue(args[0].Sym()), nil
}

func primStringToSymbol(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("stringToSymbol expects 1 argument, got %d", len(args))
	}
	if args[0].Type != lang.TypeString {
		return lang.Value{}, typeError("stringToSymbol", "string", args[0])
	}
	return lang.SymbolValue(args[0].Str()), nil
}

func primNumberToString(ev *lang.Evaluator, args []lang.Value) (lang.Value, error) {
	if len(args) != 1 {
		return lang.Value{}, fmt.Errorf("numberToString expects 1 argument, got %d", len(args))
	}
	switch args[0].Type {
	case lang.TypeInt:
		return lang.StringValue(strconv.FormatInt(args[0].Int(), 10)), nil
	case lang.TypeReal:
		return lang.StringValue(strconv.FormatFloat(args[0].Real(), 'g', -1, 64)), nil
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
	str := strings.TrimSpace(args[0].Str())
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

func requireIntArg(name string, v lang.Value) (int64, error) {
	if v.Type != lang.TypeInt {
		return 0, typeError(name, "integer", v)
	}
	return v.Int(), nil
}

func requireVectorArg(name string, v lang.Value) (*lang.Vector, error) {
	if v.Type != lang.TypeVector {
		return nil, typeError(name, "vector", v)
	}
	vec := v.Vector()
	if vec == nil {
		return nil, fmt.Errorf("%s received malformed vector", name)
	}
	return vec, nil
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
	case lang.TypeVector:
		return "vector"
	case lang.TypePrimitive:
		return "primitive"
	case lang.TypeClosure:
		return "closure"
	case lang.TypeContinuation:
		return "continuation"
	case lang.TypeMacro:
		return "macro"
	case lang.TypeEOF:
		return "eof-object"
	default:
		return "unknown"
	}
}

func toFloat(v lang.Value) (float64, error) {
	switch v.Type {
	case lang.TypeInt:
		return float64(v.Int()), nil
	case lang.TypeReal:
		return v.Real(), nil
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
		return a.Bool() == b.Bool()
	case lang.TypeInt:
		return a.Int() == b.Int()
	case lang.TypeReal:
		return a.Real() == b.Real()
	case lang.TypeString:
		return a.Str() == b.Str()
	case lang.TypeSymbol:
		return a.Sym() == b.Sym()
	case lang.TypePair:
		return a.Pair() == b.Pair()
	case lang.TypeVector:
		return a.Vector() == b.Vector()
	case lang.TypePrimitive:
		return primitivePointer(a.Primitive()) == primitivePointer(b.Primitive())
	case lang.TypeClosure:
		return a.Closure() == b.Closure()
	case lang.TypeContinuation:
		return a.Continuation() == b.Continuation()
	case lang.TypeMacro:
		return a.Macro() == b.Macro()
	case lang.TypeEOF:
		return true
	default:
		return false
	}
}

func equalValues(a, b lang.Value) bool {
	if a.Type == lang.TypeInt && b.Type == lang.TypeReal {
		return float64(a.Int()) == b.Real()
	}
	if a.Type == lang.TypeReal && b.Type == lang.TypeInt {
		return a.Real() == float64(b.Int())
	}
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case lang.TypeEmpty:
		return true
	case lang.TypeBool:
		return a.Bool() == b.Bool()
	case lang.TypeInt:
		return a.Int() == b.Int()
	case lang.TypeReal:
		return a.Real() == b.Real()
	case lang.TypeString:
		return a.Str() == b.Str()
	case lang.TypeSymbol:
		return a.Sym() == b.Sym()
	case lang.TypePair:
		ap := a.Pair()
		bp := b.Pair()
		if ap == nil || bp == nil {
			return ap == bp
		}
		return equalValues(ap.First, bp.First) && equalValues(ap.Rest, bp.Rest)
	case lang.TypeVector:
		av := a.Vector()
		bv := b.Vector()
		if av == nil || bv == nil {
			return av == bv
		}
		if len(av.Elements) != len(bv.Elements) {
			return false
		}
		for i := range av.Elements {
			if !equalValues(av.Elements[i], bv.Elements[i]) {
				return false
			}
		}
		return true
	case lang.TypePrimitive:
		return primitivePointer(a.Primitive()) == primitivePointer(b.Primitive())
	case lang.TypeClosure:
		return a.Closure() == b.Closure()
	case lang.TypeContinuation:
		return a.Continuation() == b.Continuation()
	case lang.TypeMacro:
		return a.Macro() == b.Macro()
	case lang.TypeEOF:
		return true
	default:
		return false
	}
}

func setReadInput(r io.Reader) {
	readMu.Lock()
	defer readMu.Unlock()
	if r == nil {
		readStream = sexpr.NewReader(os.Stdin)
		return
	}
	readStream = sexpr.NewReader(r)
}

func primitivePointer(p lang.Primitive) uintptr {
	if p == nil {
		return 0
	}
	return reflect.ValueOf(p).Pointer()
}

func compoundAssign(ev *lang.Evaluator, name string, args []lang.Value, updater func(lang.Value, lang.Value) (lang.Value, error)) (lang.Value, error) {
	targetName, delta, err := compoundAssignArgs(name, args)
	if err != nil {
		return lang.Value{}, err
	}
	env := ev.CurrentEnv()
	if env == nil {
		env = ev.Global
	}
	return env.Update(targetName, func(current lang.Value) (lang.Value, error) {
		return updater(current, delta)
	})
}

func compoundAssignArgs(name string, args []lang.Value) (string, lang.Value, error) {
	if len(args) != 2 {
		return "", lang.Value{}, fmt.Errorf("%s expects 2 arguments, got %d", name, len(args))
	}
	target := args[0]
	if target.Type != lang.TypeSymbol {
		return "", lang.Value{}, typeError(name, "symbol", target)
	}
	return target.Sym(), args[1], nil
}

func incDecPrimitive(ev *lang.Evaluator, name string, args []lang.Value, delta int64) (lang.Value, error) {
	targetName, err := incDecArgs(name, args)
	if err != nil {
		return lang.Value{}, err
	}
	env := ev.CurrentEnv()
	if env == nil {
		env = ev.Global
	}
	return env.Update(targetName, func(current lang.Value) (lang.Value, error) {
		switch current.Type {
		case lang.TypeInt:
			return lang.IntValue(current.Int() + delta), nil
		case lang.TypeReal:
			return lang.RealValue(current.Real() + float64(delta)), nil
		default:
			return lang.Value{}, typeError(name, "number", current)
		}
	})
}

func incDecArgs(name string, args []lang.Value) (string, error) {
	if len(args) == 0 || len(args) > 2 {
		return "", fmt.Errorf("%s expects 1 or 2 arguments, got %d", name, len(args))
	}
	target := args[0]
	if target.Type != lang.TypeSymbol {
		return "", typeError(name, "symbol", target)
	}
	return target.Sym(), nil
}
