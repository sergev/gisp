package parser

import (
	"strings"
	"testing"

	"github.com/sergev/gisp/lang"
)

func TestParseFunction(t *testing.T) {
	src := `
func fact(n) {
	if n == 0 {
		return 1;
	}
	return n * fact(n - 1);
}
`
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(prog.Decls) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(prog.Decls))
	}
	fn, ok := prog.Decls[0].(*FuncDecl)
	if !ok {
		t.Fatalf("expected FuncDecl, got %T", prog.Decls[0])
	}
	if fn.Name != "fact" {
		t.Fatalf("expected function name fact, got %s", fn.Name)
	}
	if len(fn.Params) != 1 || fn.Params[0] != "n" {
		t.Fatalf("expected single parameter n, got %v", fn.Params)
	}
	if len(fn.Body.Stmts) != 2 {
		t.Fatalf("expected 2 statements in body, got %d", len(fn.Body.Stmts))
	}
	ifStmt, ok := fn.Body.Stmts[0].(*IfStmt)
	if !ok {
		t.Fatalf("expected first statement to be IfStmt, got %T", fn.Body.Stmts[0])
	}
	if _, ok := ifStmt.Then.Stmts[0].(*ReturnStmt); !ok {
		t.Fatalf("expected then-branch to contain ReturnStmt, got %T", ifStmt.Then.Stmts[0])
	}
	if _, ok := fn.Body.Stmts[1].(*ReturnStmt); !ok {
		t.Fatalf("expected second statement to be ReturnStmt, got %T", fn.Body.Stmts[1])
	}
}

func TestCompileFunctionProducesDefineLambda(t *testing.T) {
	src := `
func identity(x) {
	return x;
}
`
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	forms, err := CompileProgram(prog)
	if err != nil {
		t.Fatalf("CompileProgram: %v", err)
	}
	if len(forms) != 1 {
		t.Fatalf("expected 1 form, got %d", len(forms))
	}
	top := forms[0]
	items, err := lang.ToSlice(top)
	if err != nil {
		t.Fatalf("expected proper list: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected define form of length 3, got %d", len(items))
	}
	if items[0].Type != lang.TypeSymbol || items[0].Sym != "define" {
		t.Fatalf("expected define symbol, got %v", items[0])
	}
	if items[1].Type != lang.TypeSymbol || items[1].Sym != "identity" {
		t.Fatalf("expected identity symbol, got %v", items[1])
	}
	lambdaForm := items[2]
	lambdaSlice, err := lang.ToSlice(lambdaForm)
	if err != nil {
		t.Fatalf("expected lambda list: %v", err)
	}
	if len(lambdaSlice) < 3 || lambdaSlice[0].Sym != "lambda" {
		t.Fatalf("expected lambda form, got %v", lambdaForm)
	}
	bodyStr := lambdaSlice[2].String()
	if !strings.Contains(bodyStr, "call/cc") {
		t.Fatalf("expected call/cc in compiled body, got %s", bodyStr)
	}
}

func TestInlineSExprLiteral(t *testing.T) {
	src := "var expr = `(list 1 2 3);\n"
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	forms, err := CompileProgram(prog)
	if err != nil {
		t.Fatalf("CompileProgram: %v", err)
	}
	if len(forms) != 1 {
		t.Fatalf("expected single form, got %d", len(forms))
	}
	formSlice, err := lang.ToSlice(forms[0])
	if err != nil {
		t.Fatalf("expected proper list: %v", err)
	}
	if formSlice[0].Sym != "define" {
		t.Fatalf("expected define, got %v", formSlice[0])
	}
	if formSlice[1].Sym != "expr" {
		t.Fatalf("expected binding name expr, got %v", formSlice[1])
	}
	expected := "(list 1 2 3)"
	if formSlice[2].String() != expected {
		t.Fatalf("expected %s, got %s", expected, formSlice[2].String())
	}
}

func TestWhileCompilesToLetLoop(t *testing.T) {
	src := `
func countdown(n) {
	while n > 0 {
		n = n - 1;
	}
}
`
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	forms, err := CompileProgram(prog)
	if err != nil {
		t.Fatalf("CompileProgram: %v", err)
	}
	body := forms[0].String()
	if !strings.Contains(body, "(let") {
		t.Fatalf("expected while translation to contain let, got %s", body)
	}
	if !strings.Contains(body, "__gisp_loop_") {
		t.Fatalf("expected while translation to introduce loop binding, got %s", body)
	}
}

func TestLambdaExpression(t *testing.T) {
	src := `
var inc = func(x) {
	return x + 1;
};
`
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	forms, err := CompileProgram(prog)
	if err != nil {
		t.Fatalf("CompileProgram: %v", err)
	}
	form := forms[0].String()
	if !strings.Contains(form, "(lambda (x)") {
		t.Fatalf("expected lambda in compiled form, got %s", form)
	}
	if !strings.Contains(form, "call/cc") {
		t.Fatalf("expected lambda body to use call/cc for return, got %s", form)
	}
}
