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
	TypeEOF
)

// Value represents any runtime object in the interpreter.
type Value struct {
	Type    ValueType
	payload interface{}
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

// EOFObject represents the end-of-file marker returned by read operations.
var EOFObject = Value{Type: TypeEOF}

// BoolValue returns the boolean Value equivalent.
func BoolValue(b bool) Value {
	return Value{Type: TypeBool, payload: b}
}

// IntValue constructs an integer Value.
func IntValue(i int64) Value {
	return Value{Type: TypeInt, payload: i}
}

// RealValue constructs a floating-point Value.
func RealValue(f float64) Value {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return Value{Type: TypeReal, payload: f}
	}
	return Value{Type: TypeReal, payload: f}
}

// StringValue constructs a string Value.
func StringValue(s string) Value {
	return Value{Type: TypeString, payload: s}
}

// SymbolValue constructs a symbol Value.
func SymbolValue(s string) Value {
	return Value{Type: TypeSymbol, payload: s}
}

// PairValue constructs a pair Value.
func PairValue(car, cdr Value) Value {
	return Value{
		Type:    TypePair,
		payload: &Pair{Car: car, Cdr: cdr},
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
		p := cur.Pair()
		if cur.Type != TypePair || p == nil {
			return nil, fmt.Errorf("expected proper list")
		}
		out = append(out, p.Car)
		cur = p.Cdr
	}
	return out, nil
}

// PrimitiveValue wraps the primitive function.
func PrimitiveValue(fn Primitive) Value {
	return Value{
		Type:    TypePrimitive,
		payload: fn,
	}
}

// ClosureValue wraps a closure.
func ClosureValue(params []string, rest string, body []Value, env *Env) Value {
	return Value{
		Type:    TypeClosure,
		payload: &Closure{Params: params, Rest: rest, Body: body, Env: env},
	}
}

// MacroValue wraps a macro transformer.
func MacroValue(params []string, rest string, body []Value, env *Env) Value {
	return Value{
		Type:    TypeMacro,
		payload: &Macro{Params: params, Rest: rest, Body: body, Env: env},
	}
}

// ContinuationValue wraps a continuation.
func ContinuationValue(frames []frame, env *Env, ev *Evaluator) Value {
	return Value{
		Type: TypeContinuation,
		payload: &Continuation{
			Frames: frames,
			Env:    env,
			Eval:   ev,
		},
	}
}

func (v Value) Bool() bool {
	if b, ok := v.payload.(bool); ok {
		return b
	}
	return false
}

func (v Value) Int() int64 {
	if i, ok := v.payload.(int64); ok {
		return i
	}
	return 0
}

func (v Value) Real() float64 {
	if f, ok := v.payload.(float64); ok {
		return f
	}
	return 0
}

func (v Value) Str() string {
	if s, ok := v.payload.(string); ok {
		return s
	}
	return ""
}

func (v Value) Sym() string {
	if s, ok := v.payload.(string); ok {
		return s
	}
	return ""
}

func (v Value) Pair() *Pair {
	if p, ok := v.payload.(*Pair); ok {
		return p
	}
	return nil
}

func (v Value) Primitive() Primitive {
	if p, ok := v.payload.(Primitive); ok {
		return p
	}
	return nil
}

func (v Value) Closure() *Closure {
	if c, ok := v.payload.(*Closure); ok {
		return c
	}
	return nil
}

func (v Value) Continuation() *Continuation {
	if c, ok := v.payload.(*Continuation); ok {
		return c
	}
	return nil
}

func (v Value) Macro() *Macro {
	if m, ok := v.payload.(*Macro); ok {
		return m
	}
	return nil
}

func (v Value) String() string {
	switch v.Type {
	case TypeEmpty:
		return "()"
	case TypeBool:
		if v.Bool() {
			return "#t"
		}
		return "#f"
	case TypeInt:
		return fmt.Sprintf("%d", v.Int())
	case TypeReal:
		return fmt.Sprintf("%g", v.Real())
	case TypeString:
		return fmt.Sprintf("%q", v.Str())
	case TypeSymbol:
		return v.Sym()
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
	case TypeEOF:
		return "#<eof>"
	default:
		return "<unknown>"
	}
}

func pairToString(v Value) string {
	out := "("
	cur := v
	first := true
	for {
		p := cur.Pair()
		if cur.Type != TypePair || p == nil {
			out += fmt.Sprintf(". %s)", cur.String())
			break
		}
		if !first {
			out += " "
		}
		out += p.Car.String()
		cdr := p.Cdr
		if cdr.Type == TypeEmpty {
			out += ")"
			break
		}
		cur = cdr
		first = false
	}
	return out
}
