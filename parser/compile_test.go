package parser

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/sergev/gisp/lang"
)

type datumSymbol string

func valueToDatum(t testing.TB, v lang.Value) interface{} {
	t.Helper()
	switch v.Type {
	case lang.TypeSymbol:
		return datumSymbol(v.Sym())
	case lang.TypeInt:
		return v.Int()
	case lang.TypeReal:
		return v.Real()
	case lang.TypeString:
		return v.Str()
	case lang.TypeBool:
		return v.Bool()
	case lang.TypeEmpty:
		return []interface{}{}
	case lang.TypePair:
		items, err := lang.ToSlice(v)
		if err != nil {
			t.Fatalf("expected proper list, got error: %v", err)
		}
		out := make([]interface{}, len(items))
		for i, item := range items {
			out[i] = valueToDatum(t, item)
		}
		return out
	default:
		t.Fatalf("unsupported lang.Value type %v", v.Type)
		return nil
	}
}

func valueToSymbolSlice(t testing.TB, v lang.Value) []string {
	t.Helper()
	items := valueToDatum(t, v).([]interface{})
	out := make([]string, len(items))
	for i, item := range items {
		sym, ok := item.(datumSymbol)
		if !ok {
			t.Fatalf("expected symbol at index %d, got %#v", i, item)
		}
		out[i] = string(sym)
	}
	return out
}

func requireListHead(t testing.TB, v lang.Value, head string) []interface{} {
	t.Helper()
	datum := valueToDatum(t, v)
	list, ok := datum.([]interface{})
	if !ok {
		t.Fatalf("expected list, got %#v", datum)
	}
	if len(list) == 0 {
		t.Fatalf("expected non-empty list")
	}
	sym, ok := list[0].(datumSymbol)
	if !ok {
		t.Fatalf("expected symbol head, got %#v", list[0])
	}
	if string(sym) != head {
		t.Fatalf("expected head %q, got %q", head, sym)
	}
	return list
}

func containsSymbolWithPrefix(node interface{}, prefix string) bool {
	switch n := node.(type) {
	case datumSymbol:
		return strings.HasPrefix(string(n), prefix)
	case []interface{}:
		for _, child := range n {
			if containsSymbolWithPrefix(child, prefix) {
				return true
			}
		}
	}
	return false
}

func TestCompileProgramNil(t *testing.T) {
	forms, err := CompileProgram(nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if forms != nil {
		t.Fatalf("expected nil forms, got %#v", forms)
	}
}

func TestCompileProgramMultipleDecls(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&VarDecl{
				Name: "x",
				Init: &NumberExpr{Value: "1"},
			},
			&ExprDecl{
				Expr: &CallExpr{
					Callee: &IdentifierExpr{Name: "print"},
					Args:   []Expr{&IdentifierExpr{Name: "x"}},
				},
			},
		},
	}
	forms, err := CompileProgram(prog)
	if err != nil {
		t.Fatalf("CompileProgram error: %v", err)
	}
	if len(forms) != 2 {
		t.Fatalf("expected 2 forms, got %d", len(forms))
	}
	defineDatum := valueToDatum(t, forms[0]).([]interface{})
	if head := defineDatum[0].(datumSymbol); string(head) != "define" {
		t.Fatalf("expected define head, got %q", head)
	}
	callDatum := valueToDatum(t, forms[1]).([]interface{})
	if head := callDatum[0].(datumSymbol); string(head) != "print" {
		t.Fatalf("expected print call, got %q", head)
	}
	if arg := callDatum[1].(datumSymbol); string(arg) != "x" {
		t.Fatalf("expected argument x, got %q", arg)
	}
}

func TestCompileDeclFunction(t *testing.T) {
	b := &builder{}
	decl := &FuncDecl{
		Name:   "identity",
		Params: []string{"x"},
		Body: &BlockStmt{
			Stmts: []Stmt{
				&ReturnStmt{Result: &IdentifierExpr{Name: "x"}},
			},
		},
	}
	form, err := compileFuncDecl(b, decl, compileContext{})
	if err != nil {
		t.Fatalf("compileFuncDecl: %v", err)
	}
	define := requireListHead(t, form, "define")
	if len(define) != 3 {
		t.Fatalf("expected define form of length 3, got %d", len(define))
	}
	if sym := define[1].(datumSymbol); string(sym) != "identity" {
		t.Fatalf("expected identity, got %q", sym)
	}
	lambdaList, ok := define[2].([]interface{})
	if !ok {
		t.Fatalf("expected lambda list, got %#v", define[2])
	}
	if sym := lambdaList[0].(datumSymbol); string(sym) != "lambda" {
		t.Fatalf("expected lambda head, got %q", sym)
	}
	params := lambdaList[1].([]interface{})
	if len(params) != 1 || string(params[0].(datumSymbol)) != "x" {
		t.Fatalf("unexpected params %#v", params)
	}
	body := lambdaList[2]
	if !containsSymbolWithPrefix(body, "__gisp_return_") {
		t.Fatalf("expected return gensym in body: %#v", body)
	}
}

func TestCompileFuncDeclReturnError(t *testing.T) {
	b := &builder{}
	decl := &FuncDecl{Name: "noop", Body: &BlockStmt{}}
	if _, err := compileFuncDecl(b, decl, compileContext{}.withReturn("r")); err != nil {
		t.Fatalf("unexpected error with existing return context: %v", err)
	}
}

func TestCompileTopLevelBinding(t *testing.T) {
	b := &builder{}
	decl := &VarDecl{Name: "answer", Init: &NumberExpr{Value: "42"}}
	val, err := compileTopLevelBinding(b, decl, compileContext{})
	if err != nil {
		t.Fatalf("compileTopLevelBinding: %v", err)
	}
	define := requireListHead(t, val, "define")
	if len(define) != 3 {
		t.Fatalf("expected define list length 3, got %d", len(define))
	}
	if sym := define[1].(datumSymbol); string(sym) != "answer" {
		t.Fatalf("expected answer symbol, got %q", sym)
	}
	if num := define[2].(int64); num != 42 {
		t.Fatalf("expected value 42, got %d", num)
	}
}

func TestCompileTopLevelBindingEmpty(t *testing.T) {
	b := &builder{}
	decl := &VarDecl{Name: "empty"}
	val, err := compileTopLevelBinding(b, decl, compileContext{})
	if err != nil {
		t.Fatalf("compileTopLevelBinding: %v", err)
	}
	define := requireListHead(t, val, "define")
	if _, ok := define[2].([]interface{}); !ok {
		t.Fatalf("expected empty list value, got %#v", define[2])
	}
}

func TestCompileExprDecl(t *testing.T) {
	b := &builder{}
	ctx := compileContext{}
	expr, err := compileDecl(b, &ExprDecl{Expr: &IdentifierExpr{Name: "foo"}}, ctx)
	if err != nil {
		t.Fatalf("compileDecl: %v", err)
	}
	if len(expr) != 1 {
		t.Fatalf("expected single form, got %d", len(expr))
	}
	if sym := expr[0].Sym(); sym != "foo" {
		t.Fatalf("expected symbol foo, got %s", sym)
	}
}

func TestCompileDeclUnsupported(t *testing.T) {
	b := &builder{}
	_, err := compileDecl(b, unsupportedDecl{}, compileContext{})
	if err == nil || !strings.Contains(err.Error(), "unsupported top-level declaration") {
		t.Fatalf("expected unsupported decl error, got %v", err)
	}
}

func TestCompileBlockNil(t *testing.T) {
	val, err := compileBlock(&builder{}, nil, compileContext{})
	if err != nil {
		t.Fatalf("compileBlock: %v", err)
	}
	if datum := valueToDatum(t, val); fmt.Sprintf("%#v", datum) != "[]interface {}{}" {
		t.Fatalf("expected empty list, got %#v", datum)
	}
}

func TestCompileStmtsVarDecl(t *testing.T) {
	b := &builder{}
	stmt := &VarDecl{Name: "temp", Init: &NumberExpr{Value: "5"}}
	result, err := compileStmtWithRest(b, stmt, lang.StringValue("done"), compileContext{})
	if err != nil {
		t.Fatalf("compileStmtWithRest: %v", err)
	}
	let := requireListHead(t, result, "let")
	if !containsSymbolWithPrefix(let, "temp") {
		t.Fatalf("expected binding for temp, got %#v", let)
	}
}

func TestCompileStmtAssign(t *testing.T) {
	b := &builder{}
	stmt := &AssignStmt{
		Name:   "x",
		Target: &IdentifierExpr{Name: "x"},
		Expr:   &NumberExpr{Value: "10"},
		Op:     tokenAssign,
	}
	result, err := compileStmtWithRest(b, stmt, lang.SymbolValue("rest"), compileContext{})
	if err != nil {
		t.Fatalf("compileStmtWithRest: %v", err)
	}
	begin := requireListHead(t, result, "begin")
	setExpr := begin[1].([]interface{})
	if string(setExpr[0].(datumSymbol)) != "set!" {
		t.Fatalf("expected set!, got %#v", setExpr[0])
	}
	if string(setExpr[1].(datumSymbol)) != "x" {
		t.Fatalf("expected set! target x, got %q", setExpr[1])
	}
	if val := setExpr[2].(int64); val != 10 {
		t.Fatalf("expected value 10, got %d", val)
	}
}

func TestCompileStmtCompoundAssign(t *testing.T) {
	b := &builder{}
	stmt := &AssignStmt{
		Name:   "x",
		Target: &IdentifierExpr{Name: "x"},
		Expr:   &NumberExpr{Value: "5"},
		Op:     tokenPlusAssign,
	}
	result, err := compileStmtWithRest(b, stmt, lang.SymbolValue("rest"), compileContext{})
	if err != nil {
		t.Fatalf("compileStmtWithRest: %v", err)
	}
	begin := requireListHead(t, result, "begin")
	call := begin[1].([]interface{})
	if string(call[0].(datumSymbol)) != "+=" {
		t.Fatalf("expected compound primitive +=, got %#v", call[0])
	}
	quote := call[1].([]interface{})
	if len(quote) != 2 || string(quote[0].(datumSymbol)) != "quote" || string(quote[1].(datumSymbol)) != "x" {
		t.Fatalf("expected quoted target, got %#v", quote)
	}
	if val := call[2].(int64); val != 5 {
		t.Fatalf("expected value 5, got %d", val)
	}
}

func TestCompileStmtIndexAssign(t *testing.T) {
	b := &builder{}
	stmt := &AssignStmt{
		Target: &IndexExpr{
			Target: &IdentifierExpr{Name: "flags"},
			Index:  &IdentifierExpr{Name: "candidate"},
		},
		Expr: &BoolExpr{Value: false},
		Op:   tokenAssign,
	}
	result, err := compileStmtWithRest(b, stmt, lang.SymbolValue("rest"), compileContext{})
	if err != nil {
		t.Fatalf("compileStmtWithRest index assign: %v", err)
	}
	begin := requireListHead(t, result, "begin")
	call, ok := begin[1].([]interface{})
	if !ok {
		t.Fatalf("expected vectorSet call, got %#v", begin[1])
	}
	if head, ok := call[0].(datumSymbol); !ok || string(head) != "vectorSet" {
		t.Fatalf("expected vectorSet head, got %#v", call[0])
	}
	if sym, ok := call[1].(datumSymbol); !ok || string(sym) != "flags" {
		t.Fatalf("expected flags as first argument, got %#v", call[1])
	}
	if sym, ok := call[2].(datumSymbol); !ok || string(sym) != "candidate" {
		t.Fatalf("expected candidate as second argument, got %#v", call[2])
	}
	if val, ok := call[3].(bool); !ok || val {
		t.Fatalf("expected false boolean as third argument, got %#v", call[3])
	}
}

func TestCompileAssignDecl(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&AssignStmt{
				Name:   "count",
				Target: &IdentifierExpr{Name: "count"},
				Expr:   &NumberExpr{Value: "42"},
				Op:     tokenAssign,
			},
		},
	}
	forms, err := CompileProgram(prog)
	if err != nil {
		t.Fatalf("CompileProgram assign decl: %v", err)
	}
	if len(forms) != 1 {
		t.Fatalf("expected single form, got %d", len(forms))
	}
	setExpr := requireListHead(t, forms[0], "set!")
	if len(setExpr) != 3 {
		t.Fatalf("expected set! form length 3, got %d", len(setExpr))
	}
	if sym := setExpr[1].(datumSymbol); string(sym) != "count" {
		t.Fatalf("expected target count, got %q", sym)
	}
	if val := setExpr[2].(int64); val != 42 {
		t.Fatalf("expected value 42, got %d", val)
	}
}

func TestCompileAssignDeclVector(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&AssignStmt{
				Target: &IndexExpr{
					Target: &IdentifierExpr{Name: "flags"},
					Index:  &NumberExpr{Value: "1"},
				},
				Expr: &BoolExpr{Value: false},
				Op:   tokenAssign,
			},
		},
	}
	forms, err := CompileProgram(prog)
	if err != nil {
		t.Fatalf("CompileProgram vector assign decl: %v", err)
	}
	if len(forms) != 1 {
		t.Fatalf("expected single form, got %d", len(forms))
	}
	call := requireListHead(t, forms[0], "vectorSet")
	if len(call) != 4 {
		t.Fatalf("expected vectorSet form length 4, got %d", len(call))
	}
	if sym := call[1].(datumSymbol); string(sym) != "flags" {
		t.Fatalf("expected flags as first argument, got %#v", call[1])
	}
	if idx := call[2].(int64); idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}
	if val, ok := call[3].(bool); !ok || val {
		t.Fatalf("expected false value, got %#v", call[3])
	}
}

func TestCompileStmtIncDec(t *testing.T) {
	b := &builder{}
	stmt := &IncDecStmt{Name: "count", Op: tokenPlusPlus}
	result, err := compileStmtWithRest(b, stmt, lang.SymbolValue("rest"), compileContext{})
	if err != nil {
		t.Fatalf("compileStmtWithRest: %v", err)
	}
	begin := requireListHead(t, result, "begin")
	call, ok := begin[1].([]interface{})
	if !ok {
		t.Fatalf("expected call form, got %#v", begin[1])
	}
	if string(call[0].(datumSymbol)) != "++" {
		t.Fatalf("expected ++ primitive, got %#v", call[0])
	}
	quote, ok := call[1].([]interface{})
	if !ok || len(quote) != 2 {
		t.Fatalf("expected quoted symbol, got %#v", call[1])
	}
	if string(quote[0].(datumSymbol)) != "quote" || string(quote[1].(datumSymbol)) != "count" {
		t.Fatalf("expected quote count, got %#v", quote)
	}
}

func TestCompileStmtExpr(t *testing.T) {
	b := &builder{}
	stmt := &ExprStmt{Expr: &IdentifierExpr{Name: "print"}}
	result, err := compileStmtWithRest(b, stmt, lang.SymbolValue("rest"), compileContext{})
	if err != nil {
		t.Fatalf("compileStmtWithRest: %v", err)
	}
	begin := requireListHead(t, result, "begin")
	if sym := begin[1].(datumSymbol); string(sym) != "print" {
		t.Fatalf("expected print expression, got %q", sym)
	}
}

func TestCompileStmtBlock(t *testing.T) {
	b := &builder{}
	stmt := &BlockStmt{
		Stmts: []Stmt{
			&ExprStmt{Expr: &IdentifierExpr{Name: "inner"}},
		},
	}
	result, err := compileStmtWithRest(b, stmt, lang.SymbolValue("rest"), compileContext{})
	if err != nil {
		t.Fatalf("compileStmtWithRest: %v", err)
	}
	begin := requireListHead(t, result, "begin")
	if !containsSymbolWithPrefix(begin, "inner") {
		t.Fatalf("expected inner symbol, got %#v", begin)
	}
}

func TestCompileStmtIfWithElse(t *testing.T) {
	b := &builder{}
	stmt := &IfStmt{
		Cond: &BoolExpr{Value: true},
		Then: &BlockStmt{
			Stmts: []Stmt{&ExprStmt{Expr: &IdentifierExpr{Name: "then-branch"}}},
		},
		Else: &BlockStmt{
			Stmts: []Stmt{&ExprStmt{Expr: &IdentifierExpr{Name: "else-branch"}}},
		},
	}
	result, err := compileStmtWithRest(b, stmt, lang.SymbolValue("rest"), compileContext{})
	if err != nil {
		t.Fatalf("compileStmtWithRest: %v", err)
	}
	begin := requireListHead(t, result, "begin")
	ifExpr := begin[1].([]interface{})
	if string(ifExpr[0].(datumSymbol)) != "if" {
		t.Fatalf("expected if form, got %#v", ifExpr[0])
	}
	thenDatum := ifExpr[2].([]interface{})
	if !containsSymbolWithPrefix(thenDatum, "then-branch") {
		t.Fatalf("missing then branch, got %#v", thenDatum)
	}
	elseDatum := ifExpr[3].([]interface{})
	if !containsSymbolWithPrefix(elseDatum, "else-branch") {
		t.Fatalf("missing else branch, got %#v", elseDatum)
	}
}

func TestCompileStmtIfWithoutElse(t *testing.T) {
	b := &builder{}
	stmt := &IfStmt{
		Cond: &BoolExpr{Value: false},
		Then: &BlockStmt{Stmts: []Stmt{}},
	}
	result, err := compileStmtWithRest(b, stmt, lang.SymbolValue("rest"), compileContext{})
	if err != nil {
		t.Fatalf("compileStmtWithRest: %v", err)
	}
	begin := requireListHead(t, result, "begin")
	ifExpr := begin[1].([]interface{})
	if _, ok := ifExpr[3].([]interface{}); !ok {
		t.Fatalf("expected empty else list, got %#v", ifExpr[3])
	}
}

func TestCompileStmtWhile(t *testing.T) {
	b := &builder{}
	stmt := &WhileStmt{
		Cond: &BoolExpr{Value: true},
		Body: &BlockStmt{
			Stmts: []Stmt{
				&ExprStmt{Expr: &IdentifierExpr{Name: "loop-body"}},
			},
		},
	}
	result, err := compileStmtWithRest(b, stmt, lang.SymbolValue("rest"), compileContext{})
	if err != nil {
		t.Fatalf("compileStmtWithRest: %v", err)
	}
	begin := requireListHead(t, result, "begin")
	if len(begin) != 3 {
		t.Fatalf("expected begin form with call/cc and rest, got %d elements", len(begin))
	}
	callCCForm, ok := begin[1].([]interface{})
	if !ok {
		t.Fatalf("expected list form for call/cc, got %#v", begin[1])
	}
	if string(callCCForm[0].(datumSymbol)) != "call/cc" {
		t.Fatalf("expected call/cc form, got %#v", callCCForm[0])
	}
	lambdaForm, ok := callCCForm[1].([]interface{})
	if !ok || string(lambdaForm[0].(datumSymbol)) != "lambda" {
		t.Fatalf("expected lambda continuation, got %#v", callCCForm[1])
	}
	params := lambdaForm[1].([]interface{})
	if len(params) != 1 {
		t.Fatalf("expected single parameter to lambda, got %d", len(params))
	}
	breakSym, ok := params[0].(datumSymbol)
	if !ok || !strings.HasPrefix(string(breakSym), "__gisp_break_") {
		t.Fatalf("expected break gensym parameter, got %#v", params[0])
	}
	letForm, ok := lambdaForm[2].([]interface{})
	if !ok || string(letForm[0].(datumSymbol)) != "let" {
		t.Fatalf("expected let form, got %#v", lambdaForm[2])
	}
	bindings := letForm[1].([]interface{})
	if len(bindings) != 1 {
		t.Fatalf("expected single binding, got %d", len(bindings))
	}
	binding := bindings[0].([]interface{})
	if len(binding) != 2 {
		t.Fatalf("expected binding pair, got %#v", binding)
	}
	loopName, ok := binding[0].(datumSymbol)
	if !ok || !strings.HasPrefix(string(loopName), "__gisp_loop_") {
		t.Fatalf("expected loop gensym binding, got %#v", binding[0])
	}
	if _, ok := binding[1].([]interface{}); !ok {
		t.Fatalf("expected empty list initializer, got %#v", binding[1])
	}
	letBody := letForm[2].([]interface{})
	if string(letBody[0].(datumSymbol)) != "begin" {
		t.Fatalf("expected begin in let body, got %#v", letBody[0])
	}
	setForm := letBody[1].([]interface{})
	if string(setForm[0].(datumSymbol)) != "set!" {
		t.Fatalf("expected set! form, got %#v", setForm[0])
	}
	if setTarget := setForm[1].(datumSymbol); setTarget != loopName {
		t.Fatalf("expected set! target %q, got %q", loopName, setTarget)
	}
	loopLambda := setForm[2].([]interface{})
	if string(loopLambda[0].(datumSymbol)) != "lambda" {
		t.Fatalf("expected lambda form, got %#v", loopLambda[0])
	}
	lambdaBody := loopLambda[2]
	if !containsSymbolWithPrefix(lambdaBody, string(loopName)) {
		t.Fatalf("expected recursive loop call in lambda body, got %#v", lambdaBody)
	}
	callForm := letBody[2].([]interface{})
	if len(callForm) != 1 {
		t.Fatalf("expected single-element call form, got %#v", callForm)
	}
	callSym, ok := callForm[0].(datumSymbol)
	if !ok || callSym != loopName {
		t.Fatalf("expected tail call to loop function %q, got %#v", loopName, callForm[0])
	}
	if restSym, ok := begin[2].(datumSymbol); !ok || restSym != "rest" {
		t.Fatalf("expected rest continuation as final begin expr, got %#v", begin[2])
	}
}

func TestCompileStmtBreakRequiresLoop(t *testing.T) {
	b := &builder{}
	_, err := compileStmtWithRest(b, &BreakStmt{}, lang.SymbolValue("rest"), compileContext{})
	if err == nil || !strings.Contains(err.Error(), "break not allowed") {
		t.Fatalf("expected break context error, got %v", err)
	}
}

func TestCompileStmtContinueRequiresLoop(t *testing.T) {
	b := &builder{}
	_, err := compileStmtWithRest(b, &ContinueStmt{}, lang.SymbolValue("rest"), compileContext{})
	if err == nil || !strings.Contains(err.Error(), "continue not allowed") {
		t.Fatalf("expected continue context error, got %v", err)
	}
}

func TestCompileStmtBreakAndContinueForms(t *testing.T) {
	b := &builder{}
	ctx := compileContext{}.withLoop("break-handler", "loop-handler")

	breakVal, err := compileStmtWithRest(b, &BreakStmt{}, lang.SymbolValue("rest"), ctx)
	if err != nil {
		t.Fatalf("compileStmtWithRest (break): %v", err)
	}
	breakDatum := valueToDatum(t, breakVal).([]interface{})
	if len(breakDatum) != 2 {
		t.Fatalf("expected break invocation length 2, got %d", len(breakDatum))
	}
	if sym := breakDatum[0].(datumSymbol); string(sym) != "break-handler" {
		t.Fatalf("expected break handler symbol, got %q", sym)
	}
	if _, ok := breakDatum[1].([]interface{}); !ok {
		t.Fatalf("expected empty list argument to break handler, got %#v", breakDatum[1])
	}

	continueVal, err := compileStmtWithRest(b, &ContinueStmt{}, lang.SymbolValue("rest"), ctx)
	if err != nil {
		t.Fatalf("compileStmtWithRest (continue): %v", err)
	}
	continueDatum := valueToDatum(t, continueVal).([]interface{})
	if len(continueDatum) != 1 {
		t.Fatalf("expected continue call with no args, got %d elements", len(continueDatum))
	}
	if sym := continueDatum[0].(datumSymbol); string(sym) != "loop-handler" {
		t.Fatalf("expected continue to call loop handler, got %q", sym)
	}
}

func TestCompileStmtReturnWithValue(t *testing.T) {
	b := &builder{}
	ctx := compileContext{}.withReturn("ret")
	stmt := &ReturnStmt{Result: &NumberExpr{Value: "7"}}
	result, err := compileStmtWithRest(b, stmt, lang.SymbolValue("rest"), ctx)
	if err != nil {
		t.Fatalf("compileStmtWithRest: %v", err)
	}
	call := valueToDatum(t, result).([]interface{})
	if len(call) != 2 {
		t.Fatalf("expected return invocation length 2, got %d", len(call))
	}
	if sym := call[0].(datumSymbol); string(sym) != "ret" {
		t.Fatalf("expected ret symbol, got %q", sym)
	}
	if val := call[1].(int64); val != 7 {
		t.Fatalf("expected return value 7, got %d", val)
	}
}

func TestCompileStmtReturnWithoutValue(t *testing.T) {
	b := &builder{}
	ctx := compileContext{}.withReturn("ret")
	stmt := &ReturnStmt{}
	result, err := compileStmtWithRest(b, stmt, lang.SymbolValue("rest"), ctx)
	if err != nil {
		t.Fatalf("compileStmtWithRest: %v", err)
	}
	call := valueToDatum(t, result).([]interface{})
	if _, ok := call[1].([]interface{}); !ok {
		t.Fatalf("expected empty list return value, got %#v", call[1])
	}
}

func TestCompileStmtReturnWithoutContext(t *testing.T) {
	b := &builder{}
	stmt := &ReturnStmt{}
	_, err := compileStmtWithRest(b, stmt, lang.SymbolValue("rest"), compileContext{})
	if err == nil || !strings.Contains(err.Error(), "return not allowed") {
		t.Fatalf("expected return context error, got %v", err)
	}
}

func TestCompileStmtUnsupported(t *testing.T) {
	b := &builder{}
	_, err := compileStmtWithRest(b, unsupportedStmt{}, lang.SymbolValue("rest"), compileContext{})
	if err == nil || !strings.Contains(err.Error(), "unsupported statement") {
		t.Fatalf("expected unsupported statement error, got %v", err)
	}
}

func TestCompileExprIdentifier(t *testing.T) {
	val, err := compileExpr(&builder{}, &IdentifierExpr{Name: "x"}, compileContext{})
	if err != nil {
		t.Fatalf("compileExpr: %v", err)
	}
	if sym := val.Sym(); sym != "x" {
		t.Fatalf("expected symbol x, got %s", sym)
	}
}

func TestCompileExprNumber(t *testing.T) {
	val, err := compileExpr(&builder{}, &NumberExpr{Value: "123"}, compileContext{})
	if err != nil {
		t.Fatalf("compileExpr: %v", err)
	}
	if val.Type != lang.TypeInt || val.Int() != 123 {
		t.Fatalf("expected int 123, got %#v", val)
	}
	floatVal, err := compileExpr(&builder{}, &NumberExpr{Value: "3.5"}, compileContext{})
	if err != nil {
		t.Fatalf("compileExpr float: %v", err)
	}
	if floatVal.Type != lang.TypeReal || math.Abs(floatVal.Real()-3.5) > 1e-9 {
		t.Fatalf("expected real 3.5, got %#v", floatVal)
	}
}

func TestCompileExprStringBoolList(t *testing.T) {
	b := &builder{}
	strVal, err := compileExpr(b, &StringExpr{Value: "hello"}, compileContext{})
	if err != nil || strVal.Type != lang.TypeString || strVal.Str() != "hello" {
		t.Fatalf("unexpected string result %#v, err %v", strVal, err)
	}
	boolVal, err := compileExpr(b, &BoolExpr{Value: true}, compileContext{})
	if err != nil || boolVal.Type != lang.TypeBool || !boolVal.Bool() {
		t.Fatalf("unexpected bool result %#v, err %v", boolVal, err)
	}
	listVal, err := compileExpr(b, &ListExpr{
		Elements: []Expr{
			&NumberExpr{Value: "1"},
			&NumberExpr{Value: "2"},
		},
	}, compileContext{})
	if err != nil {
		t.Fatalf("compileExpr list: %v", err)
	}
	list := requireListHead(t, listVal, "list")
	if len(list) != 3 {
		t.Fatalf("expected list form of length 3, got %d", len(list))
	}
	if list[1].(int64) != 1 || list[2].(int64) != 2 {
		t.Fatalf("unexpected list contents %#v", list[1:])
	}
}

func TestCompileExprLambda(t *testing.T) {
	b := &builder{}
	expr := &LambdaExpr{
		Params: []string{"x", "y"},
		Body: &BlockStmt{
			Stmts: []Stmt{&ReturnStmt{Result: &IdentifierExpr{Name: "x"}}},
		},
	}
	val, err := compileExpr(b, expr, compileContext{})
	if err != nil {
		t.Fatalf("compileExpr lambda: %v", err)
	}
	lambda := requireListHead(t, val, "lambda")
	params := lambda[1].([]interface{})
	if len(params) != 2 || string(params[0].(datumSymbol)) != "x" || string(params[1].(datumSymbol)) != "y" {
		t.Fatalf("unexpected parameters %#v", params)
	}
	body := lambda[2]
	if !containsSymbolWithPrefix(body, "__gisp_return_") {
		t.Fatalf("expected gensym return in lambda body, got %#v", body)
	}
}

func TestCompileExprCall(t *testing.T) {
	b := &builder{}
	expr := &CallExpr{
		Callee: &IdentifierExpr{Name: "sum"},
		Args: []Expr{
			&NumberExpr{Value: "1"},
			&NumberExpr{Value: "2"},
		},
	}
	val, err := compileExpr(b, expr, compileContext{})
	if err != nil {
		t.Fatalf("compileExpr call: %v", err)
	}
	call := valueToDatum(t, val).([]interface{})
	if len(call) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(call))
	}
	if string(call[0].(datumSymbol)) != "sum" || call[1].(int64) != 1 || call[2].(int64) != 2 {
		t.Fatalf("unexpected call %#v", call)
	}
}

func TestCompileExprIndex(t *testing.T) {
	b := &builder{}
	expr := &IndexExpr{
		Target: &IdentifierExpr{Name: "flags"},
		Index:  &IdentifierExpr{Name: "candidate"},
	}
	val, err := compileExpr(b, expr, compileContext{})
	if err != nil {
		t.Fatalf("compileExpr index: %v", err)
	}
	call := requireListHead(t, val, "vectorRef")
	if len(call) != 3 {
		t.Fatalf("expected vectorRef form with 3 elements, got %d", len(call))
	}
	if sym, ok := call[1].(datumSymbol); !ok || string(sym) != "flags" {
		t.Fatalf("expected flags symbol as target, got %#v", call[1])
	}
	if sym, ok := call[2].(datumSymbol); !ok || string(sym) != "candidate" {
		t.Fatalf("expected candidate symbol as index, got %#v", call[2])
	}
}

func TestCompileExprUnary(t *testing.T) {
	b := &builder{}
	tests := []struct {
		name string
		op   TokenType
		head string
	}{
		{"negate", tokenMinus, "-"},
		{"not", tokenBang, "not"},
		{"bitwise complement", tokenCaret, "^"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, err := compileExpr(b, &UnaryExpr{Op: tc.op, Expr: &NumberExpr{Value: "1"}}, compileContext{})
			if err != nil {
				t.Fatalf("compileExpr unary: %v", err)
			}
			list := requireListHead(t, val, tc.head)
			if len(list) != 2 || list[1].(int64) != 1 {
				t.Fatalf("unexpected unary list %#v", list)
			}
		})
	}
}

func TestCompileExprUnaryUnsupported(t *testing.T) {
	_, err := compileExpr(&builder{}, &UnaryExpr{Op: tokenPlus, Expr: &NumberExpr{Value: "1"}}, compileContext{})
	if err == nil || !strings.Contains(err.Error(), "unsupported unary operator") {
		t.Fatalf("expected unsupported unary operator error, got %v", err)
	}
}

func TestCompileExprNil(t *testing.T) {
	val, err := compileExpr(&builder{}, &NilExpr{}, compileContext{})
	if err != nil {
		t.Fatalf("compileExpr nil: %v", err)
	}
	if val.Type != lang.TypeEmpty {
		t.Fatalf("expected lang.EmptyList, got %v", val.Type)
	}
	empty := valueToDatum(t, val).([]interface{})
	if len(empty) != 0 {
		t.Fatalf("expected empty list datum, got %#v", empty)
	}
}

func TestCompileExprBinary(t *testing.T) {
	b := &builder{}
	tests := []struct {
		name string
		op   TokenType
		head string
	}{
		{"add", tokenPlus, "+"},
		{"sub", tokenMinus, "-"},
		{"mul", tokenStar, "*"},
		{"div", tokenSlash, "/"},
		{"mod", tokenPercent, "%"},
		{"eq", tokenEqualEqual, "="},
		{"lt", tokenLess, "<"},
		{"le", tokenLessEqual, "<="},
		{"gt", tokenGreater, ">"},
		{"ge", tokenGreaterEqual, ">="},
		{"band", tokenAmpersand, "&"},
		{"bor", tokenPipe, "|"},
		{"bxor", tokenCaret, "^"},
		{"bclear", tokenAmpersandCaret, "&^"},
		{"shl", tokenShiftLeft, "<<"},
		{"shr", tokenShiftRight, ">>"},
		{"and", tokenAndAnd, "and"},
		{"or", tokenOrOr, "or"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, err := compileExpr(b, &BinaryExpr{
				Op:    tc.op,
				Left:  &NumberExpr{Value: "1"},
				Right: &NumberExpr{Value: "2"},
			}, compileContext{})
			if err != nil {
				t.Fatalf("compileExpr binary: %v", err)
			}
			list := requireListHead(t, val, tc.head)
			if len(list) != 3 || list[1].(int64) != 1 || list[2].(int64) != 2 {
				t.Fatalf("unexpected binary list %#v", list)
			}
		})
	}
}

func TestCompileExprLogicalAndWithEquality(t *testing.T) {
	b := &builder{}
	expr := &BinaryExpr{
		Op: tokenAndAnd,
		Left: &CallExpr{
			Callee: &IdentifierExpr{Name: "isNumberValue"},
			Args:   []Expr{&IdentifierExpr{Name: "a"}},
		},
		Right: &BinaryExpr{
			Op:    tokenEqualEqual,
			Left:  &IdentifierExpr{Name: "a"},
			Right: &NumberExpr{Value: "0"},
		},
	}
	val, err := compileExpr(b, expr, compileContext{})
	if err != nil {
		t.Fatalf("compileExpr and with equality: %v", err)
	}
	andList := requireListHead(t, val, "and")
	if len(andList) != 3 {
		t.Fatalf("expected and form of length 3, got %d", len(andList))
	}
	callForm, ok := andList[1].([]interface{})
	if !ok {
		t.Fatalf("expected call list, got %#v", andList[1])
	}
	if head := callForm[0].(datumSymbol); string(head) != "isNumberValue" {
		t.Fatalf("expected call head isNumberValue, got %q", head)
	}
	if len(callForm) != 2 || string(callForm[1].(datumSymbol)) != "a" {
		t.Fatalf("unexpected call args %#v", callForm)
	}
	eqForm, ok := andList[2].([]interface{})
	if !ok {
		t.Fatalf("expected equality list, got %#v", andList[2])
	}
	if head := eqForm[0].(datumSymbol); string(head) != "=" {
		t.Fatalf("expected equality head, got %q", head)
	}
	if len(eqForm) != 3 {
		t.Fatalf("expected equality form of length 3, got %d", len(eqForm))
	}
	if sym := eqForm[1].(datumSymbol); string(sym) != "a" {
		t.Fatalf("expected left operand a, got %q", sym)
	}
	if num := eqForm[2].(int64); num != 0 {
		t.Fatalf("expected numeric literal 0, got %d", num)
	}
}

func TestCompileExprBinaryNotEqual(t *testing.T) {
	val, err := compileExpr(&builder{}, &BinaryExpr{
		Op:    tokenBangEqual,
		Left:  &NumberExpr{Value: "1"},
		Right: &NumberExpr{Value: "2"},
	}, compileContext{})
	if err != nil {
		t.Fatalf("compileExpr binary !=: %v", err)
	}
	not := requireListHead(t, val, "not")
	inner := not[1].([]interface{})
	if string(inner[0].(datumSymbol)) != "=" {
		t.Fatalf("expected equals inside not, got %#v", inner[0])
	}
}

func TestCompileExprBinaryUnsupported(t *testing.T) {
	_, err := compileExpr(&builder{}, &BinaryExpr{
		Op:    tokenIllegal,
		Left:  &NumberExpr{Value: "1"},
		Right: &NumberExpr{Value: "2"},
	}, compileContext{})
	if err == nil || !strings.Contains(err.Error(), "unsupported binary operator") {
		t.Fatalf("expected unsupported binary operator error, got %v", err)
	}
}

func TestCompileExprSExprLiteral(t *testing.T) {
	raw := lang.List(lang.SymbolValue("list"), lang.IntValue(1))
	val, err := compileExpr(&builder{}, &SExprLiteral{Value: raw}, compileContext{})
	if err != nil {
		t.Fatalf("compileExpr sexpr: %v", err)
	}
	if val.String() != raw.String() {
		t.Fatalf("expected literal to pass through, got %s", val.String())
	}
}

func TestCompileExprListLiteral(t *testing.T) {
	b := &builder{}
	expr := &ListExpr{
		Elements: []Expr{
			&NumberExpr{Value: "1"},
			&NumberExpr{Value: "2"},
		},
	}
	val, err := compileExpr(b, expr, compileContext{})
	if err != nil {
		t.Fatalf("compileExpr list literal: %v", err)
	}
	list := requireListHead(t, val, "list")
	if len(list) != 3 {
		t.Fatalf("expected list form of length 3, got %d", len(list))
	}
	if _, ok := list[1].(int64); !ok {
		t.Fatalf("expected first element to be int literal, got %#v", list[1])
	}
	if _, ok := list[2].(int64); !ok {
		t.Fatalf("expected second element to be int literal, got %#v", list[2])
	}
}

func TestCompileExprListLiteralEmpty(t *testing.T) {
	b := &builder{}
	expr := &ListExpr{}
	val, err := compileExpr(b, expr, compileContext{})
	if err != nil {
		t.Fatalf("compileExpr empty list literal: %v", err)
	}
	list := requireListHead(t, val, "list")
	if len(list) != 1 {
		t.Fatalf("expected (list) with length 1, got %d elements", len(list))
	}
}

func TestCompileExprSwitch(t *testing.T) {
	expr := &SwitchExpr{
		Clauses: []*SwitchClause{
			{
				Cond: &IdentifierExpr{Name: "isPositive"},
				Body: &NumberExpr{Value: "1"},
			},
			{
				Cond: &IdentifierExpr{Name: "isNegative"},
				Body: &UnaryExpr{
					Op:   tokenMinus,
					Expr: &NumberExpr{Value: "1"},
				},
			},
		},
		Default: &NumberExpr{Value: "0"},
	}
	val, err := compileExpr(&builder{}, expr, compileContext{})
	if err != nil {
		t.Fatalf("compileExpr switch: %v", err)
	}
	condList := requireListHead(t, val, "cond")
	if len(condList) != 4 {
		t.Fatalf("expected cond with three clauses, got %d elements", len(condList))
	}
	firstClause, ok := condList[1].([]interface{})
	if !ok || len(firstClause) != 2 {
		t.Fatalf("unexpected first clause %#v", condList[1])
	}
	if sym, ok := firstClause[0].(datumSymbol); !ok || string(sym) != "isPositive" {
		t.Fatalf("expected predicate symbol isPositive, got %#v", firstClause[0])
	}
	if num, ok := firstClause[1].(int64); !ok || num != 1 {
		t.Fatalf("expected body literal 1, got %#v", firstClause[1])
	}
	secondClause, ok := condList[2].([]interface{})
	if !ok || len(secondClause) != 2 {
		t.Fatalf("unexpected second clause %#v", condList[2])
	}
	if sym, ok := secondClause[0].(datumSymbol); !ok || string(sym) != "isNegative" {
		t.Fatalf("expected predicate symbol isNegative, got %#v", secondClause[0])
	}
	if exprList, ok := secondClause[1].([]interface{}); !ok || len(exprList) != 2 {
		t.Fatalf("expected unary expression list, got %#v", secondClause[1])
	}
	elseClause, ok := condList[3].([]interface{})
	if !ok || len(elseClause) != 2 {
		t.Fatalf("unexpected else clause %#v", condList[3])
	}
	if sym, ok := elseClause[0].(datumSymbol); !ok || string(sym) != "else" {
		t.Fatalf("expected else symbol, got %#v", elseClause[0])
	}
	if num, ok := elseClause[1].(int64); !ok || num != 0 {
		t.Fatalf("expected default literal 0, got %#v", elseClause[1])
	}
}

func TestCompileExprIf(t *testing.T) {
	expr := &IfExpr{
		Cond: &IdentifierExpr{Name: "ready"},
		Then: &NumberExpr{Value: "1"},
		Else: &NumberExpr{Value: "2"},
	}
	val, err := compileExpr(&builder{}, expr, compileContext{})
	if err != nil {
		t.Fatalf("compileExpr if: %v", err)
	}
	ifForm := requireListHead(t, val, "if")
	if len(ifForm) != 4 {
		t.Fatalf("expected if form length 4, got %d", len(ifForm))
	}
	if cond, ok := ifForm[1].(datumSymbol); !ok || string(cond) != "ready" {
		t.Fatalf("expected condition symbol ready, got %#v", ifForm[1])
	}
	if thenVal, ok := ifForm[2].(int64); !ok || thenVal != 1 {
		t.Fatalf("expected then literal 1, got %#v", ifForm[2])
	}
	if elseVal, ok := ifForm[3].(int64); !ok || elseVal != 2 {
		t.Fatalf("expected else literal 2, got %#v", ifForm[3])
	}
}

func TestCompileExprUnsupported(t *testing.T) {
	_, err := compileExpr(&builder{}, badExpr{}, compileContext{})
	if err == nil || !strings.Contains(err.Error(), "unsupported expression") {
		t.Fatalf("expected unsupported expression error, got %v", err)
	}
}

func TestParseNumberIntegerAndFloat(t *testing.T) {
	val, err := parseNumber("42")
	if err != nil {
		t.Fatalf("parseNumber: %v", err)
	}
	if val.Type != lang.TypeInt || val.Int() != 42 {
		t.Fatalf("expected int 42, got %#v", val)
	}
	val, err = parseNumber("6.022")
	if err != nil {
		t.Fatalf("parseNumber float: %v", err)
	}
	if val.Type != lang.TypeReal || math.Abs(val.Real()-6.022) > 1e-9 {
		t.Fatalf("expected real 6.022, got %#v", val)
	}
	val, err = parseNumber("1e2")
	if err != nil {
		t.Fatalf("parseNumber exp: %v", err)
	}
	if val.Type != lang.TypeReal || math.Abs(val.Real()-100) > 1e-9 {
		t.Fatalf("expected real 100, got %#v", val)
	}
}

func TestParseNumberInvalid(t *testing.T) {
	if _, err := parseNumber("123abc"); err == nil || !strings.Contains(err.Error(), "invalid integer literal") {
		t.Fatalf("expected integer error, got %v", err)
	}
	if _, err := parseNumber("not-a-number"); err == nil || !strings.Contains(err.Error(), "invalid float literal") {
		t.Fatalf("expected float error, got %v", err)
	}
	if _, err := parseNumber("1.2.3"); err == nil || !strings.Contains(err.Error(), "invalid float literal") {
		t.Fatalf("expected float error, got %v", err)
	}
}

type unsupportedDecl struct{}

func (unsupportedDecl) Pos() Position { return Position{} }
func (unsupportedDecl) declNode()     {}

type unsupportedStmt struct{}

func (unsupportedStmt) Pos() Position { return Position{} }
func (unsupportedStmt) stmtNode()     {}

type badExpr struct{}

func (badExpr) Pos() Position { return Position{} }
func (badExpr) exprNode()     {}
