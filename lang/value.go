package lang

import (
	"fmt"
	"math"
)

// ValueType enumerates the different runtime value categories.
type ValueType int

const (
	TypeEmpty ValueType = iota
	TypeBool
	TypeInt
	TypeReal
	TypeString
	TypeSymbol
	TypePair
	TypePrimitive
	TypeClosure
	TypeContinuation
	TypeMacro
)

// Value represents any runtime object in the interpreter.
type Value struct {
	Type ValueType

	Bool bool
	Int  int64
	Real float64
	Str  string
	Sym  string

	Pair         *Pair
	Primitive    Primitive
	Closure      *Closure
	Continuation *Continuation
	Macro        *Macro
}

// Pair represents a cons cell.
type Pair struct {
	Car Value
	Cdr Value
}

// Primitive represents a built-in Go function exposed to the interpreter.
type Primitive func(*Evaluator, []Value) (Value, error)

// Closure represents a user-defined function with lexical scope.
type Closure struct {
	Params []string
	Rest   string
	Body   []Value
	Env    *Env
}

// Macro represents a macro transformer.
type Macro struct {
	Params []string
	Rest   string
	Body   []Value
	Env    *Env
}

// Continuation represents a captured continuation.
type Continuation struct {
	Frames []frame
	Env    *Env
	Eval   *Evaluator
}

// EmptyList is the singleton empty list value.
var EmptyList = Value{Type: TypeEmpty}

// BoolValue returns the boolean Value equivalent.
func BoolValue(b bool) Value {
	return Value{Type: TypeBool, Bool: b}
}

// IntValue constructs an integer Value.
func IntValue(i int64) Value {
	return Value{Type: TypeInt, Int: i}
}

// RealValue constructs a floating-point Value.
func RealValue(f float64) Value {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return Value{Type: TypeReal, Real: f}
	}
	return Value{Type: TypeReal, Real: f}
}

// StringValue constructs a string Value.
func StringValue(s string) Value {
	return Value{Type: TypeString, Str: s}
}

// SymbolValue constructs a symbol Value.
func SymbolValue(s string) Value {
	return Value{Type: TypeSymbol, Sym: s}
}

// PairValue constructs a pair Value.
func PairValue(car, cdr Value) Value {
	return Value{
		Type: TypePair,
		Pair: &Pair{Car: car, Cdr: cdr},
	}
}

// List constructs a proper list from provided values.
func List(vals ...Value) Value {
	result := EmptyList
	for i := len(vals) - 1; i >= 0; i-- {
		result = PairValue(vals[i], result)
	}
	return result
}

// ToSlice converts a proper list into a Go slice.
func ToSlice(list Value) ([]Value, error) {
	var out []Value
	cur := list
	for cur.Type != TypeEmpty {
		if cur.Type != TypePair || cur.Pair == nil {
			return nil, fmt.Errorf("expected proper list")
		}
		out = append(out, cur.Pair.Car)
		cur = cur.Pair.Cdr
	}
	return out, nil
}

// PrimitiveValue wraps the primitive function.
func PrimitiveValue(fn Primitive) Value {
	return Value{
		Type:      TypePrimitive,
		Primitive: fn,
	}
}

// ClosureValue wraps a closure.
func ClosureValue(params []string, rest string, body []Value, env *Env) Value {
	return Value{
		Type:    TypeClosure,
		Closure: &Closure{Params: params, Rest: rest, Body: body, Env: env},
	}
}

// MacroValue wraps a macro transformer.
func MacroValue(params []string, rest string, body []Value, env *Env) Value {
	return Value{
		Type:  TypeMacro,
		Macro: &Macro{Params: params, Rest: rest, Body: body, Env: env},
	}
}

// ContinuationValue wraps a continuation.
func ContinuationValue(frames []frame, env *Env, ev *Evaluator) Value {
	return Value{
		Type: TypeContinuation,
		Continuation: &Continuation{
			Frames: frames,
			Env:    env,
			Eval:   ev,
		},
	}
}

func (v Value) String() string {
	switch v.Type {
	case TypeEmpty:
		return "()"
	case TypeBool:
		if v.Bool {
			return "#t"
		}
		return "#f"
	case TypeInt:
		return fmt.Sprintf("%d", v.Int)
	case TypeReal:
		return fmt.Sprintf("%g", v.Real)
	case TypeString:
		return fmt.Sprintf("%q", v.Str)
	case TypeSymbol:
		return v.Sym
	case TypePair:
		return pairToString(v)
	case TypePrimitive:
		return "<primitive>"
	case TypeClosure:
		return "<closure>"
	case TypeContinuation:
		return "<continuation>"
	case TypeMacro:
		return "<macro>"
	default:
		return "<unknown>"
	}
}

func pairToString(v Value) string {
	out := "("
	cur := v
	first := true
	for {
		if cur.Type != TypePair || cur.Pair == nil {
			out += fmt.Sprintf(". %s)", cur.String())
			break
		}
		if !first {
			out += " "
		}
		out += cur.Pair.Car.String()
		cdr := cur.Pair.Cdr
		if cdr.Type == TypeEmpty {
			out += ")"
			break
		}
		cur = cdr
		first = false
	}
	return out
}
