package parser

import (
	"math"
	"strings"
	"testing"

	"github.com/sergev/gisp/lang"
)

type sexprSymbol string

func parseProgramFromSource(t *testing.T, src string) *Program {
	t.Helper()
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	return prog
}

func compileSource(t *testing.T, src string) []lang.Value {
	t.Helper()
	prog := parseProgramFromSource(t, src)
	forms, err := CompileProgram(prog)
	if err != nil {
		t.Fatalf("CompileProgram error: %v", err)
	}
	return forms
}

func toDatum(t *testing.T, v lang.Value) interface{} {
	t.Helper()
	switch v.Type {
	case lang.TypeSymbol:
		return sexprSymbol(v.Sym())
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
			out[i] = toDatum(t, item)
		}
		return out
	default:
		t.Fatalf("unsupported lang.Value type %v", v.Type)
		return nil
	}
}

func containsHead(node interface{}, head string) bool {
	switch n := node.(type) {
	case []interface{}:
		if len(n) > 0 {
			if sym, ok := n[0].(sexprSymbol); ok && string(sym) == head {
				return true
			}
		}
		for _, child := range n {
			if containsHead(child, head) {
				return true
			}
		}
	}
	return false
}

func containsSymbolPrefix(node interface{}, prefix string) bool {
	switch n := node.(type) {
	case sexprSymbol:
		return strings.HasPrefix(string(n), prefix)
	case []interface{}:
		for _, child := range n {
			if containsSymbolPrefix(child, prefix) {
				return true
			}
		}
	}
	return false
}

func TestParseFunction(t *testing.T) {
	src := `
func fact(n) {
	if n == 0 {
		return 1
	}
	return n * fact(n - 1)
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
	if items[0].Type != lang.TypeSymbol || items[0].Sym() != "define" {
		t.Fatalf("expected define symbol, got %v", items[0])
	}
	if items[1].Type != lang.TypeSymbol || items[1].Sym() != "identity" {
		t.Fatalf("expected identity symbol, got %v", items[1])
	}
	lambdaForm := items[2]
	lambdaSlice, err := lang.ToSlice(lambdaForm)
	if err != nil {
		t.Fatalf("expected lambda list: %v", err)
	}
	if len(lambdaSlice) < 3 || lambdaSlice[0].Sym() != "lambda" {
		t.Fatalf("expected lambda form, got %v", lambdaForm)
	}
	bodyStr := lambdaSlice[2].String()
	if !strings.Contains(bodyStr, "call/cc") {
		t.Fatalf("expected call/cc in compiled body, got %s", bodyStr)
	}
}

func TestParseIncDecStatements(t *testing.T) {
	src := `
func demo() {
	x++
	y--;
}
`
	prog := parseProgramFromSource(t, src)
	if len(prog.Decls) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(prog.Decls))
	}
	fn, ok := prog.Decls[0].(*FuncDecl)
	if !ok {
		t.Fatalf("expected FuncDecl, got %T", prog.Decls[0])
	}
	if len(fn.Body.Stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(fn.Body.Stmts))
	}
	first, ok := fn.Body.Stmts[0].(*IncDecStmt)
	if !ok || first.Op != tokenPlusPlus || first.Name != "x" {
		t.Fatalf("expected first stmt to be x++, got %#v", fn.Body.Stmts[0])
	}
	second, ok := fn.Body.Stmts[1].(*IncDecStmt)
	if !ok || second.Op != tokenMinusMinus || second.Name != "y" {
		t.Fatalf("expected second stmt to be y--, got %#v", fn.Body.Stmts[1])
	}
}

func TestParseIncDecDisallowedInExpressions(t *testing.T) {
	src := `
var x = 1;
var y = x++;
`
	if _, err := Parse(src); err == nil || !strings.Contains(err.Error(), "not allowed in expression context") {
		t.Fatalf("expected expression context error, got %v", err)
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
	if formSlice[0].Sym() != "define" {
		t.Fatalf("expected define, got %v", formSlice[0])
	}
	if formSlice[1].Sym() != "expr" {
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

func TestParseVarConstAndExprDecls(t *testing.T) {
	src := `
var counter = 0;
const limit = 10;
var empty;
foo();
`
	prog := parseProgramFromSource(t, src)
	if len(prog.Decls) != 4 {
		t.Fatalf("expected 4 declarations, got %d", len(prog.Decls))
	}

	varDecl0, ok := prog.Decls[0].(*VarDecl)
	if !ok {
		t.Fatalf("expected first decl to be VarDecl, got %T", prog.Decls[0])
	}
	if varDecl0.Name != "counter" {
		t.Fatalf("expected name counter, got %s", varDecl0.Name)
	}
	numInit, ok := varDecl0.Init.(*NumberExpr)
	if !ok || numInit.Value != "0" {
		t.Fatalf("expected numeric initializer 0, got %#v", varDecl0.Init)
	}
	if varDecl0.Const {
		t.Fatalf("expected mutable var for counter")
	}

	varDecl1, ok := prog.Decls[1].(*VarDecl)
	if !ok {
		t.Fatalf("expected second decl to be VarDecl, got %T", prog.Decls[1])
	}
	if !varDecl1.Const {
		t.Fatalf("expected const flag for limit")
	}
	if varDecl1.Name != "limit" {
		t.Fatalf("expected name limit, got %s", varDecl1.Name)
	}
	limitInit, ok := varDecl1.Init.(*NumberExpr)
	if !ok || limitInit.Value != "10" {
		t.Fatalf("expected numeric initializer 10, got %#v", varDecl1.Init)
	}

	varDecl2, ok := prog.Decls[2].(*VarDecl)
	if !ok {
		t.Fatalf("expected third decl to be VarDecl, got %T", prog.Decls[2])
	}
	if varDecl2.Init != nil {
		t.Fatalf("expected nil initializer for empty var, got %#v", varDecl2.Init)
	}
	if varDecl2.Const {
		t.Fatalf("expected mutable var for empty")
	}

	exprDecl, ok := prog.Decls[3].(*ExprDecl)
	if !ok {
		t.Fatalf("expected fourth decl to be ExprDecl, got %T", prog.Decls[3])
	}
	call, ok := exprDecl.Expr.(*CallExpr)
	if !ok {
		t.Fatalf("expected call expression, got %T", exprDecl.Expr)
	}
	callee, ok := call.Callee.(*IdentifierExpr)
	if !ok || callee.Name != "foo" {
		t.Fatalf("expected call to foo, got %#v", call.Callee)
	}
	if len(call.Args) != 0 {
		t.Fatalf("expected zero arguments, got %d", len(call.Args))
	}
}

func TestParseFunctionBodyStatements(t *testing.T) {
	src := `
func demo(a, b) {
	var sum = a + b
	const limit = 10
	sum = sum + limit
	if sum > limit {
		return sum
	} else {
		return
	}
	while sum < 100 {
		sum = sum + 1
	}
	{
		var inner = 1
	}
	print(sum)
}
`
	prog := parseProgramFromSource(t, src)
	if len(prog.Decls) != 1 {
		t.Fatalf("expected single declaration, got %d", len(prog.Decls))
	}
	fn, ok := prog.Decls[0].(*FuncDecl)
	if !ok {
		t.Fatalf("expected FuncDecl, got %T", prog.Decls[0])
	}
	if len(fn.Params) != 2 || fn.Params[0] != "a" || fn.Params[1] != "b" {
		t.Fatalf("unexpected parameters: %v", fn.Params)
	}
	body := fn.Body
	if len(body.Stmts) != 7 {
		t.Fatalf("expected 7 statements in body, got %d", len(body.Stmts))
	}

	declSum, ok := body.Stmts[0].(*VarDecl)
	if !ok || declSum.Name != "sum" {
		t.Fatalf("expected first stmt var sum, got %#v", body.Stmts[0])
	}
	if _, ok := declSum.Init.(*BinaryExpr); !ok {
		t.Fatalf("expected binary initializer for sum, got %#v", declSum.Init)
	}

	declLimit, ok := body.Stmts[1].(*VarDecl)
	if !ok || declLimit.Name != "limit" || !declLimit.Const {
		t.Fatalf("expected const limit declaration, got %#v", body.Stmts[1])
	}

	assignSum, ok := body.Stmts[2].(*AssignStmt)
	if !ok || assignSum.Name != "sum" {
		t.Fatalf("expected assignment to sum, got %#v", body.Stmts[2])
	}
	if assignSum.Op != tokenAssign {
		t.Fatalf("expected simple assignment operator, got %v", assignSum.Op)
	}

	ifStmt, ok := body.Stmts[3].(*IfStmt)
	if !ok {
		t.Fatalf("expected IfStmt, got %#v", body.Stmts[3])
	}
	if _, ok := ifStmt.Cond.(*BinaryExpr); !ok {
		t.Fatalf("expected binary condition, got %#v", ifStmt.Cond)
	}
	if len(ifStmt.Then.Stmts) != 1 {
		t.Fatalf("expected single statement in then branch, got %d", len(ifStmt.Then.Stmts))
	}
	if len(ifStmt.Else.Stmts) != 1 {
		t.Fatalf("expected single statement in else branch, got %d", len(ifStmt.Else.Stmts))
	}
	returnThen, ok := ifStmt.Then.Stmts[0].(*ReturnStmt)
	if !ok || returnThen.Result == nil {
		t.Fatalf("expected return with value in then branch, got %#v", ifStmt.Then.Stmts[0])
	}
	returnElse, ok := ifStmt.Else.Stmts[0].(*ReturnStmt)
	if !ok || returnElse.Result != nil {
		t.Fatalf("expected bare return in else branch, got %#v", ifStmt.Else.Stmts[0])
	}

	whileStmt, ok := body.Stmts[4].(*WhileStmt)
	if !ok {
		t.Fatalf("expected WhileStmt, got %#v", body.Stmts[4])
	}
	if len(whileStmt.Body.Stmts) != 1 {
		t.Fatalf("expected single statement in while body, got %d", len(whileStmt.Body.Stmts))
	}
	if _, ok := whileStmt.Body.Stmts[0].(*AssignStmt); !ok {
		t.Fatalf("expected assignment inside while body, got %#v", whileStmt.Body.Stmts[0])
	}

	innerBlock, ok := body.Stmts[5].(*BlockStmt)
	if !ok {
		t.Fatalf("expected nested block, got %#v", body.Stmts[5])
	}
	if len(innerBlock.Stmts) != 1 {
		t.Fatalf("expected single statement in nested block, got %d", len(innerBlock.Stmts))
	}
	if _, ok := innerBlock.Stmts[0].(*VarDecl); !ok {
		t.Fatalf("expected var decl in nested block, got %#v", innerBlock.Stmts[0])
	}

	exprStmt, ok := body.Stmts[6].(*ExprStmt)
	if !ok {
		t.Fatalf("expected trailing expression statement, got %#v", body.Stmts[6])
	}
	if call, ok := exprStmt.Expr.(*CallExpr); !ok || call.Callee.(*IdentifierExpr).Name != "print" {
		t.Fatalf("expected call to print, got %#v", exprStmt.Expr)
	}
}

func TestParseCompoundAssignments(t *testing.T) {
	src := `
func demo() {
	count += 2;
	count <<= shift;
	flags &= mask
}
`
	prog := parseProgramFromSource(t, src)
	if len(prog.Decls) != 1 {
		t.Fatalf("expected single declaration, got %d", len(prog.Decls))
	}
	fn, ok := prog.Decls[0].(*FuncDecl)
	if !ok {
		t.Fatalf("expected FuncDecl, got %T", prog.Decls[0])
	}
	if len(fn.Body.Stmts) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(fn.Body.Stmts))
	}
	check := func(idx int, name string, op TokenType) {
		stmt, ok := fn.Body.Stmts[idx].(*AssignStmt)
		if !ok {
			t.Fatalf("statement %d: expected AssignStmt, got %#v", idx, fn.Body.Stmts[idx])
		}
		if stmt.Name != name {
			t.Fatalf("statement %d: expected target %q, got %q", idx, name, stmt.Name)
		}
		if stmt.Op != op {
			t.Fatalf("statement %d: expected op %v, got %v", idx, op, stmt.Op)
		}
	}

	check(0, "count", tokenPlusAssign)
	check(1, "count", tokenShiftLeftAssign)
	check(2, "flags", tokenAmpersandAssign)
}

func TestParseVarDeclWithOptionalSemicolon(t *testing.T) {
	src := `
func demo() {
	var withSemi = 1;
	var withoutSemi = 2
}
`
	prog := parseProgramFromSource(t, src)
	if len(prog.Decls) != 1 {
		t.Fatalf("expected single declaration, got %d", len(prog.Decls))
	}
	fn, ok := prog.Decls[0].(*FuncDecl)
	if !ok {
		t.Fatalf("expected FuncDecl, got %T", prog.Decls[0])
	}
	if len(fn.Body.Stmts) != 2 {
		t.Fatalf("expected 2 statements in body, got %d", len(fn.Body.Stmts))
	}

	withSemi, ok := fn.Body.Stmts[0].(*VarDecl)
	if !ok || withSemi.Name != "withSemi" {
		t.Fatalf("expected first stmt var withSemi, got %#v", fn.Body.Stmts[0])
	}
	numOne, ok := withSemi.Init.(*NumberExpr)
	if !ok || numOne.Value != "1" {
		t.Fatalf("expected numeric initializer 1, got %#v", withSemi.Init)
	}

	withoutSemi, ok := fn.Body.Stmts[1].(*VarDecl)
	if !ok || withoutSemi.Name != "withoutSemi" {
		t.Fatalf("expected second stmt var withoutSemi, got %#v", fn.Body.Stmts[1])
	}
	numTwo, ok := withoutSemi.Init.(*NumberExpr)
	if !ok || numTwo.Value != "2" {
		t.Fatalf("expected numeric initializer 2, got %#v", withoutSemi.Init)
	}
}

func TestElseMustFollowClosingBraceOnSameLine(t *testing.T) {
	src := `
func demo() {
	if true {
		return
	}
	else {
		return
	}
}
`
	if _, err := Parse(src); err == nil || !strings.Contains(err.Error(), "unexpected token") {
		t.Fatalf("expected parse error complaining about misplaced else, got %v", err)
	}
}

func TestParseExpressionPrecedence(t *testing.T) {
	prog := parseProgramFromSource(t, "var value = 1 + 2 * 3 == 7 && !false\n")
	if len(prog.Decls) != 1 {
		t.Fatalf("expected single declaration, got %d", len(prog.Decls))
	}
	varDecl, ok := prog.Decls[0].(*VarDecl)
	if !ok {
		t.Fatalf("expected VarDecl, got %T", prog.Decls[0])
	}
	root, ok := varDecl.Init.(*BinaryExpr)
	if !ok || root.Op != tokenAndAnd {
		t.Fatalf("expected && binary expression, got %#v", varDecl.Init)
	}
	leftEq, ok := root.Left.(*BinaryExpr)
	if !ok || leftEq.Op != tokenEqualEqual {
		t.Fatalf("expected == expression on left, got %#v", root.Left)
	}
	rightUnary, ok := root.Right.(*UnaryExpr)
	if !ok || rightUnary.Op != tokenBang {
		t.Fatalf("expected ! unary expression on right, got %#v", root.Right)
	}
	if boolLit, ok := rightUnary.Expr.(*BoolExpr); !ok || boolLit.Value {
		t.Fatalf("expected !false, got %#v", rightUnary.Expr)
	}

	plusExpr, ok := leftEq.Left.(*BinaryExpr)
	if !ok || plusExpr.Op != tokenPlus {
		t.Fatalf("expected + expression, got %#v", leftEq.Left)
	}
	starExpr, ok := plusExpr.Right.(*BinaryExpr)
	if !ok || starExpr.Op != tokenStar {
		t.Fatalf("expected * expression, got %#v", plusExpr.Right)
	}
	if leftNum, ok := plusExpr.Left.(*NumberExpr); !ok || leftNum.Value != "1" {
		t.Fatalf("expected literal 1, got %#v", plusExpr.Left)
	}
	if leftStar, ok := starExpr.Left.(*NumberExpr); !ok || leftStar.Value != "2" {
		t.Fatalf("expected literal 2, got %#v", starExpr.Left)
	}
	if rightStar, ok := starExpr.Right.(*NumberExpr); !ok || rightStar.Value != "3" {
		t.Fatalf("expected literal 3, got %#v", starExpr.Right)
	}
	if rightNum, ok := leftEq.Right.(*NumberExpr); !ok || rightNum.Value != "7" {
		t.Fatalf("expected literal 7, got %#v", leftEq.Right)
	}
}

func TestParseIfExpression(t *testing.T) {
	src := `
var result = if cond {
	valueTrue
} else {
	valueFalse
};
`
	prog := parseProgramFromSource(t, src)
	if len(prog.Decls) != 1 {
		t.Fatalf("expected single declaration, got %d", len(prog.Decls))
	}
	varDecl, ok := prog.Decls[0].(*VarDecl)
	if !ok {
		t.Fatalf("expected VarDecl, got %T", prog.Decls[0])
	}
	ifExpr, ok := varDecl.Init.(*IfExpr)
	if !ok {
		t.Fatalf("expected IfExpr initializer, got %#v", varDecl.Init)
	}
	condIdent, ok := ifExpr.Cond.(*IdentifierExpr)
	if !ok || condIdent.Name != "cond" {
		t.Fatalf("expected condition identifier cond, got %#v", ifExpr.Cond)
	}
	thenIdent, ok := ifExpr.Then.(*IdentifierExpr)
	if !ok || thenIdent.Name != "valueTrue" {
		t.Fatalf("expected then identifier valueTrue, got %#v", ifExpr.Then)
	}
	elseIdent, ok := ifExpr.Else.(*IdentifierExpr)
	if !ok || elseIdent.Name != "valueFalse" {
		t.Fatalf("expected else identifier valueFalse, got %#v", ifExpr.Else)
	}
}

func TestParseLambdaAndListLiteral(t *testing.T) {
	src := `
var fn = func(x, y) {
	return x + y;
};
var data = [1, "two", true, func() {
	return;
}];
var result = fn(1, 2);
`
	prog := parseProgramFromSource(t, src)
	if len(prog.Decls) != 3 {
		t.Fatalf("expected three declarations, got %d", len(prog.Decls))
	}

	fnDecl := prog.Decls[0].(*VarDecl)
	lambda, ok := fnDecl.Init.(*LambdaExpr)
	if !ok {
		t.Fatalf("expected lambda initializer, got %#v", fnDecl.Init)
	}
	if len(lambda.Params) != 2 || lambda.Params[0] != "x" || lambda.Params[1] != "y" {
		t.Fatalf("unexpected lambda params: %v", lambda.Params)
	}
	if len(lambda.Body.Stmts) != 1 {
		t.Fatalf("expected single statement in lambda body, got %d", len(lambda.Body.Stmts))
	}
	if ret, ok := lambda.Body.Stmts[0].(*ReturnStmt); !ok || ret.Result == nil {
		t.Fatalf("expected return with value, got %#v", lambda.Body.Stmts[0])
	}

	dataDecl := prog.Decls[1].(*VarDecl)
	list, ok := dataDecl.Init.(*ListExpr)
	if !ok {
		t.Fatalf("expected list literal, got %#v", dataDecl.Init)
	}
	if len(list.Elements) != 4 {
		t.Fatalf("expected 4 list elements, got %d", len(list.Elements))
	}
	if num, ok := list.Elements[0].(*NumberExpr); !ok || num.Value != "1" {
		t.Fatalf("expected first element numeric literal, got %#v", list.Elements[0])
	}
	if str, ok := list.Elements[1].(*StringExpr); !ok || str.Value != "two" {
		t.Fatalf("expected second element string literal, got %#v", list.Elements[1])
	}
	if boolean, ok := list.Elements[2].(*BoolExpr); !ok || !boolean.Value {
		t.Fatalf("expected third element true literal, got %#v", list.Elements[2])
	}
	nestedLambda, ok := list.Elements[3].(*LambdaExpr)
	if !ok {
		t.Fatalf("expected fourth element lambda, got %#v", list.Elements[3])
	}
	if len(nestedLambda.Params) != 0 {
		t.Fatalf("expected lambda with no params, got %v", nestedLambda.Params)
	}
	if len(nestedLambda.Body.Stmts) != 1 {
		t.Fatalf("expected single statement in nested lambda, got %d", len(nestedLambda.Body.Stmts))
	}
	if ret, ok := nestedLambda.Body.Stmts[0].(*ReturnStmt); !ok || ret.Result != nil {
		t.Fatalf("expected bare return in nested lambda, got %#v", nestedLambda.Body.Stmts[0])
	}

	resultDecl := prog.Decls[2].(*VarDecl)
	call, ok := resultDecl.Init.(*CallExpr)
	if !ok {
		t.Fatalf("expected call expression, got %#v", resultDecl.Init)
	}
	if callee, ok := call.Callee.(*IdentifierExpr); !ok || callee.Name != "fn" {
		t.Fatalf("expected call to fn, got %#v", call.Callee)
	}
	if len(call.Args) != 2 {
		t.Fatalf("expected two call arguments, got %d", len(call.Args))
	}
	if arg0, ok := call.Args[0].(*NumberExpr); !ok || arg0.Value != "1" {
		t.Fatalf("unexpected first argument %#v", call.Args[0])
	}
	if arg1, ok := call.Args[1].(*NumberExpr); !ok || arg1.Value != "2" {
		t.Fatalf("unexpected second argument %#v", call.Args[1])
	}
}

func TestParseVectorLiteral(t *testing.T) {
	src := `
var values = #[1, "two", true, func() {
	return;
}];
`
	prog := parseProgramFromSource(t, src)
	if len(prog.Decls) != 1 {
		t.Fatalf("expected single declaration, got %d", len(prog.Decls))
	}
	decl, ok := prog.Decls[0].(*VarDecl)
	if !ok {
		t.Fatalf("expected VarDecl, got %T", prog.Decls[0])
	}
	vec, ok := decl.Init.(*VectorExpr)
	if !ok {
		t.Fatalf("expected VectorExpr initializer, got %#v", decl.Init)
	}
	if len(vec.Elements) != 4 {
		t.Fatalf("expected 4 vector elements, got %d", len(vec.Elements))
	}
	if num, ok := vec.Elements[0].(*NumberExpr); !ok || num.Value != "1" {
		t.Fatalf("expected first element numeric literal, got %#v", vec.Elements[0])
	}
	if str, ok := vec.Elements[1].(*StringExpr); !ok || str.Value != "two" {
		t.Fatalf("expected second element string literal, got %#v", vec.Elements[1])
	}
	if boolean, ok := vec.Elements[2].(*BoolExpr); !ok || !boolean.Value {
		t.Fatalf("expected third element true literal, got %#v", vec.Elements[2])
	}
	if _, ok := vec.Elements[3].(*LambdaExpr); !ok {
		t.Fatalf("expected lambda as fourth element, got %#v", vec.Elements[3])
	}
}

func TestParseIndexExpression(t *testing.T) {
	prog := parseProgramFromSource(t, "var value = flags[candidate];\n")
	if len(prog.Decls) != 1 {
		t.Fatalf("expected single declaration, got %d", len(prog.Decls))
	}
	varDecl, ok := prog.Decls[0].(*VarDecl)
	if !ok {
		t.Fatalf("expected VarDecl, got %T", prog.Decls[0])
	}
	indexExpr, ok := varDecl.Init.(*IndexExpr)
	if !ok {
		t.Fatalf("expected index expression initializer, got %#v", varDecl.Init)
	}
	baseIdent, ok := indexExpr.Target.(*IdentifierExpr)
	if !ok || baseIdent.Name != "flags" {
		t.Fatalf("expected base identifier flags, got %#v", indexExpr.Target)
	}
	indexIdent, ok := indexExpr.Index.(*IdentifierExpr)
	if !ok || indexIdent.Name != "candidate" {
		t.Fatalf("expected index identifier candidate, got %#v", indexExpr.Index)
	}
}

func TestParseIndexAssignment(t *testing.T) {
	src := `
func disable(flags, candidate) {
	flags[candidate] = false;
}
`
	prog := parseProgramFromSource(t, src)
	if len(prog.Decls) != 1 {
		t.Fatalf("expected single declaration, got %d", len(prog.Decls))
	}
	fnDecl, ok := prog.Decls[0].(*FuncDecl)
	if !ok {
		t.Fatalf("expected FuncDecl, got %T", prog.Decls[0])
	}
	if len(fnDecl.Body.Stmts) != 1 {
		t.Fatalf("expected single statement in function body, got %d", len(fnDecl.Body.Stmts))
	}
	assign, ok := fnDecl.Body.Stmts[0].(*AssignStmt)
	if !ok {
		t.Fatalf("expected assignment statement, got %#v", fnDecl.Body.Stmts[0])
	}
	if assign.Name != "" {
		t.Fatalf("expected identifier name to be empty for index target, got %q", assign.Name)
	}
	indexExpr, ok := assign.Target.(*IndexExpr)
	if !ok {
		t.Fatalf("expected index target, got %#v", assign.Target)
	}
	baseIdent, ok := indexExpr.Target.(*IdentifierExpr)
	if !ok || baseIdent.Name != "flags" {
		t.Fatalf("expected base identifier flags, got %#v", indexExpr.Target)
	}
	indexIdent, ok := indexExpr.Index.(*IdentifierExpr)
	if !ok || indexIdent.Name != "candidate" {
		t.Fatalf("expected index identifier candidate, got %#v", indexExpr.Index)
	}
	boolExpr, ok := assign.Expr.(*BoolExpr)
	if !ok || boolExpr.Value {
		t.Fatalf("expected false boolean assignment, got %#v", assign.Expr)
	}
}

func TestParseTopLevelIndexAssignment(t *testing.T) {
	src := `
var flags = #[true, true, true]
flags[1] = false
`
	prog := parseProgramFromSource(t, src)
	if len(prog.Decls) != 2 {
		t.Fatalf("expected two declarations, got %d", len(prog.Decls))
	}
	assign, ok := prog.Decls[1].(*AssignStmt)
	if !ok {
		t.Fatalf("expected second declaration to be AssignStmt, got %T", prog.Decls[1])
	}
	indexExpr, ok := assign.Target.(*IndexExpr)
	if !ok {
		t.Fatalf("expected index target, got %#v", assign.Target)
	}
	if base, ok := indexExpr.Target.(*IdentifierExpr); !ok || base.Name != "flags" {
		t.Fatalf("expected base identifier flags, got %#v", indexExpr.Target)
	}
	if idx, ok := indexExpr.Index.(*NumberExpr); !ok || idx.Value != "1" {
		t.Fatalf("expected numeric index 1, got %#v", indexExpr.Index)
	}
	if _, ok := assign.Expr.(*BoolExpr); !ok {
		t.Fatalf("expected boolean assignment, got %#v", assign.Expr)
	}
}

func TestParseEmptyVectorLiteral(t *testing.T) {
	prog := parseProgramFromSource(t, "var empty = #[]\n")
	if len(prog.Decls) != 1 {
		t.Fatalf("expected single declaration, got %d", len(prog.Decls))
	}
	decl, ok := prog.Decls[0].(*VarDecl)
	if !ok {
		t.Fatalf("expected VarDecl, got %T", prog.Decls[0])
	}
	vec, ok := decl.Init.(*VectorExpr)
	if !ok {
		t.Fatalf("expected VectorExpr initializer, got %#v", decl.Init)
	}
	if len(vec.Elements) != 0 {
		t.Fatalf("expected empty vector literal, got %d elements", len(vec.Elements))
	}
}

func TestParseNilLiteral(t *testing.T) {
	prog := parseProgramFromSource(t, "var empty = nil\n")
	if len(prog.Decls) != 1 {
		t.Fatalf("expected single declaration, got %d", len(prog.Decls))
	}
	varDecl, ok := prog.Decls[0].(*VarDecl)
	if !ok {
		t.Fatalf("expected VarDecl, got %T", prog.Decls[0])
	}
	nilExpr, ok := varDecl.Init.(*NilExpr)
	if !ok {
		t.Fatalf("expected NilExpr initializer, got %#v", varDecl.Init)
	}
	if nilExpr.Posn.Line == 0 {
		t.Fatalf("expected position information on NilExpr, got %#v", nilExpr.Posn)
	}
}

func TestParseSwitchExpr(t *testing.T) {
	src := `
var sign = switch {
case x > 0: 1;
case x < 0: -1;
default: 0;
};
`
	prog := parseProgramFromSource(t, src)
	if len(prog.Decls) != 1 {
		t.Fatalf("expected single declaration, got %d", len(prog.Decls))
	}
	decl, ok := prog.Decls[0].(*VarDecl)
	if !ok {
		t.Fatalf("expected VarDecl, got %T", prog.Decls[0])
	}
	switchExpr, ok := decl.Init.(*SwitchExpr)
	if !ok {
		t.Fatalf("expected SwitchExpr initializer, got %#v", decl.Init)
	}
	if len(switchExpr.Clauses) != 2 {
		t.Fatalf("expected 2 case clauses, got %d", len(switchExpr.Clauses))
	}
	for i, clause := range switchExpr.Clauses {
		if clause.Cond == nil || clause.Body == nil {
			t.Fatalf("clause %d missing cond/body: %#v", i, clause)
		}
	}
	if switchExpr.Default == nil {
		t.Fatalf("expected default clause")
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name:    "unterminated block",
			src:     "func f() { var x = 1",
			wantErr: "expected } to close block",
		},
		{
			name:    "unexpected token",
			src:     "var x = );",
			wantErr: "unexpected token )",
		},
		{
			name: "switch case after default",
			src: `
var value = switch {
default: 0;
case true: 1;
};
`,
			wantErr: "case clause cannot follow default",
		},
		{
			name: "switch missing case",
			src: `
var value = switch {
};
`,
			wantErr: "switch requires at least one case",
		},
		{
			name:    "vector missing closing bracket",
			src:     "var bad = #[1, 2\n",
			wantErr: "expected ]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Parse(tc.src); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestParseIncompleteDetection(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		src        string
		incomplete bool
	}{
		{
			name:       "unterminated block",
			src:        "func f() {",
			incomplete: true,
		},
		{
			name:       "unterminated string literal",
			src:        `var s = "unfinished`,
			incomplete: true,
		},
		{
			name:       "syntax error",
			src:        "var x = );",
			incomplete: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Parse(tc.src); err == nil {
				t.Fatalf("expected error parsing %q", tc.src)
			} else if IsIncomplete(err) != tc.incomplete {
				t.Fatalf("IsIncomplete(%q) = %v, want %v (err=%v)", tc.src, IsIncomplete(err), tc.incomplete, err)
			}
		})
	}
}

func TestCompileTopLevelBindings(t *testing.T) {
	forms := compileSource(t, `
var counter = 2 + 3;
const empty;
`)
	if len(forms) != 2 {
		t.Fatalf("expected 2 forms, got %d", len(forms))
	}

	defineCounter, ok := toDatum(t, forms[0]).([]interface{})
	if !ok {
		t.Fatalf("expected list for define counter")
	}
	if head, ok := defineCounter[0].(sexprSymbol); !ok || head != "define" {
		t.Fatalf("expected define head, got %#v", defineCounter[0])
	}
	if sym, ok := defineCounter[1].(sexprSymbol); !ok || sym != "counter" {
		t.Fatalf("expected symbol counter, got %#v", defineCounter[1])
	}
	counterExpr, ok := defineCounter[2].([]interface{})
	if !ok {
		t.Fatalf("expected list expression for counter initializer")
	}
	if op, ok := counterExpr[0].(sexprSymbol); !ok || op != "+" {
		t.Fatalf("expected + operator, got %#v", counterExpr[0])
	}
	if left, ok := counterExpr[1].(int64); !ok || left != 2 {
		t.Fatalf("expected left operand 2, got %#v", counterExpr[1])
	}
	if right, ok := counterExpr[2].(int64); !ok || right != 3 {
		t.Fatalf("expected right operand 3, got %#v", counterExpr[2])
	}

	defineEmpty, ok := toDatum(t, forms[1]).([]interface{})
	if !ok {
		t.Fatalf("expected list for define empty")
	}
	if sym, ok := defineEmpty[1].(sexprSymbol); !ok || sym != "empty" {
		t.Fatalf("expected symbol empty, got %#v", defineEmpty[1])
	}
	if emptyVal, ok := defineEmpty[2].([]interface{}); !ok || len(emptyVal) != 0 {
		t.Fatalf("expected empty list initializer, got %#v", defineEmpty[2])
	}
}

func TestCompileExpressionForms(t *testing.T) {
	type checkFn func(t *testing.T, expr interface{})

	getHead := func(list []interface{}) string {
		if len(list) == 0 {
			return ""
		}
		if sym, ok := list[0].(sexprSymbol); ok {
			return string(sym)
		}
		return ""
	}

	cases := []struct {
		name string
		src  string
		want checkFn
	}{
		{
			name: "NotEqual",
			src:  "var expr = 1 != 2;\n",
			want: func(t *testing.T, expr interface{}) {
				list, ok := expr.([]interface{})
				if !ok {
					t.Fatalf("expected list expr, got %#v", expr)
				}
				if getHead(list) != "not" {
					t.Fatalf("expected not head, got %#v", list)
				}
				inner, ok := list[1].([]interface{})
				if !ok || getHead(inner) != "=" {
					t.Fatalf("expected inner = list, got %#v", list[1])
				}
				if left, ok := inner[1].(int64); !ok || left != 1 {
					t.Fatalf("expected left operand 1, got %#v", inner[1])
				}
				if right, ok := inner[2].(int64); !ok || right != 2 {
					t.Fatalf("expected right operand 2, got %#v", inner[2])
				}
			},
		},
		{
			name: "UnaryMinus",
			src:  "var expr = -5;\n",
			want: func(t *testing.T, expr interface{}) {
				list, ok := expr.([]interface{})
				if !ok || getHead(list) != "-" {
					t.Fatalf("expected unary - list, got %#v", expr)
				}
				if val, ok := list[1].(int64); !ok || val != 5 {
					t.Fatalf("expected operand 5, got %#v", list[1])
				}
			},
		},
		{
			name: "LogicalAnd",
			src:  "var expr = true && false;\n",
			want: func(t *testing.T, expr interface{}) {
				list, ok := expr.([]interface{})
				if !ok || getHead(list) != "and" {
					t.Fatalf("expected and list, got %#v", expr)
				}
				if val, ok := list[1].(bool); !ok || !val {
					t.Fatalf("expected true operand, got %#v", list[1])
				}
				if val, ok := list[2].(bool); !ok || val {
					t.Fatalf("expected false operand, got %#v", list[2])
				}
			},
		},
		{
			name: "LogicalOr",
			src:  "var expr = true || false;\n",
			want: func(t *testing.T, expr interface{}) {
				list, ok := expr.([]interface{})
				if !ok || getHead(list) != "or" {
					t.Fatalf("expected or list, got %#v", expr)
				}
				if val, ok := list[1].(bool); !ok || !val {
					t.Fatalf("expected true operand, got %#v", list[1])
				}
				if val, ok := list[2].(bool); !ok || val {
					t.Fatalf("expected false operand, got %#v", list[2])
				}
			},
		},
		{
			name: "ArithmeticPrecedence",
			src:  "var expr = 1 + 2 * 3;\n",
			want: func(t *testing.T, expr interface{}) {
				list, ok := expr.([]interface{})
				if !ok || getHead(list) != "+" {
					t.Fatalf("expected + list, got %#v", expr)
				}
				if left, ok := list[1].(int64); !ok || left != 1 {
					t.Fatalf("expected left operand 1, got %#v", list[1])
				}
				right, ok := list[2].([]interface{})
				if !ok || getHead(right) != "*" {
					t.Fatalf("expected * list on right, got %#v", list[2])
				}
				if a, ok := right[1].(int64); !ok || a != 2 {
					t.Fatalf("expected operand 2, got %#v", right[1])
				}
				if b, ok := right[2].(int64); !ok || b != 3 {
					t.Fatalf("expected operand 3, got %#v", right[2])
				}
			},
		},
		{
			name: "LogicalNot",
			src:  "var expr = !true;\n",
			want: func(t *testing.T, expr interface{}) {
				list, ok := expr.([]interface{})
				if !ok || getHead(list) != "not" {
					t.Fatalf("expected not list, got %#v", expr)
				}
				if val, ok := list[1].(bool); !ok || !val {
					t.Fatalf("expected operand true, got %#v", list[1])
				}
			},
		},
		{
			name: "VectorLiteral",
			src:  "var expr = #[1, 2, 3];\n",
			want: func(t *testing.T, expr interface{}) {
				list, ok := expr.([]interface{})
				if !ok || getHead(list) != "vector" {
					t.Fatalf("expected vector form, got %#v", expr)
				}
				if len(list) != 4 {
					t.Fatalf("expected vector with 3 elements, got %#v", list)
				}
				for i := 1; i <= 3; i++ {
					if val, ok := list[i].(int64); !ok || val != int64(i) {
						t.Fatalf("expected element %d to be %d, got %#v", i, i, list[i])
					}
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			forms := compileSource(t, tc.src)
			if len(forms) != 1 {
				t.Fatalf("expected single form, got %d", len(forms))
			}
			define, ok := toDatum(t, forms[0]).([]interface{})
			if !ok || len(define) < 3 {
				t.Fatalf("expected define list, got %#v", forms[0])
			}
			tc.want(t, define[2])
		})
	}
}

func TestCompileFunctionStatements(t *testing.T) {
	forms := compileSource(t, `
func demo(x) {
	var total = x
	while total < 10 {
		total = total + 1;
	}
	if total == 10 {
		return total;
	} else {
		return;
	}
	print(total);
}
`)
	if len(forms) != 1 {
		t.Fatalf("expected single form, got %d", len(forms))
	}
	define, ok := toDatum(t, forms[0]).([]interface{})
	if !ok || len(define) != 3 {
		t.Fatalf("expected define list, got %#v", forms[0])
	}
	if head, ok := define[0].(sexprSymbol); !ok || head != "define" {
		t.Fatalf("expected define head, got %#v", define[0])
	}
	lambdaForm, ok := define[2].([]interface{})
	if !ok || len(lambdaForm) != 3 {
		t.Fatalf("expected lambda form, got %#v", define[2])
	}
	if head, ok := lambdaForm[0].(sexprSymbol); !ok || head != "lambda" {
		t.Fatalf("expected lambda head, got %#v", lambdaForm[0])
	}
	paramList, ok := lambdaForm[1].([]interface{})
	if !ok || len(paramList) != 1 || paramList[0] != sexprSymbol("x") {
		t.Fatalf("unexpected parameter list %#v", lambdaForm[1])
	}
	callCC, ok := lambdaForm[2].([]interface{})
	if !ok || len(callCC) != 2 || callCC[0] != sexprSymbol("call/cc") {
		t.Fatalf("expected call/cc form, got %#v", lambdaForm[2])
	}
	innerLambda, ok := callCC[1].([]interface{})
	if !ok || len(innerLambda) != 3 || innerLambda[0] != sexprSymbol("lambda") {
		t.Fatalf("expected inner lambda, got %#v", callCC[1])
	}
	retParams, ok := innerLambda[1].([]interface{})
	if !ok || len(retParams) != 1 {
		t.Fatalf("expected single return parameter, got %#v", innerLambda[1])
	}
	retSym, ok := retParams[0].(sexprSymbol)
	if !ok {
		t.Fatalf("expected symbol return parameter, got %#v", retParams[0])
	}
	body := innerLambda[2]
	if !containsHead(body, "let") {
		t.Fatalf("expected let form in body, got %#v", body)
	}
	if !containsHead(body, "set!") {
		t.Fatalf("expected set! form in body, got %#v", body)
	}
	if !containsHead(body, "if") {
		t.Fatalf("expected if form in body, got %#v", body)
	}
	if !containsHead(body, "begin") {
		t.Fatalf("expected begin form in body, got %#v", body)
	}
	if !containsSymbolPrefix(body, "__gisp_loop_") {
		t.Fatalf("expected generated loop symbol in body, got %#v", body)
	}
	if !containsHead(body, string(retSym)) {
		t.Fatalf("expected return invocation with %s, got %#v", retSym, body)
	}
	if !containsSymbolPrefix(body, "__gisp_return_") {
		t.Fatalf("expected generated return symbol usage in body, got %#v", body)
	}
}

func TestCompileTopLevelExprDecl(t *testing.T) {
	forms := compileSource(t, "foo(1, 2);\n")
	if len(forms) != 1 {
		t.Fatalf("expected single form, got %d", len(forms))
	}
	expr, ok := toDatum(t, forms[0]).([]interface{})
	if !ok || len(expr) != 3 {
		t.Fatalf("expected function call list, got %#v", forms[0])
	}
	if head, ok := expr[0].(sexprSymbol); !ok || head != "foo" {
		t.Fatalf("expected call to foo, got %#v", expr[0])
	}
	if arg0, ok := expr[1].(int64); !ok || arg0 != 1 {
		t.Fatalf("expected first argument 1, got %#v", expr[1])
	}
	if arg1, ok := expr[2].(int64); !ok || arg1 != 2 {
		t.Fatalf("expected second argument 2, got %#v", expr[2])
	}
}

func TestParseNumber(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantTyp lang.ValueType
		wantInt int64
		wantF   float64
		wantErr bool
	}{
		{"Integer", "42", lang.TypeInt, 42, 0, false},
		{"NegativeInteger", "-7", lang.TypeInt, -7, 0, false},
		{"Float", "3.14", lang.TypeReal, 0, 3.14, false},
		{"Scientific", "1e3", lang.TypeReal, 0, 1000, false},
		{"Invalid", "12x", 0, 0, 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			val, err := parseNumber(tc.src)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseNumber error: %v", err)
			}
			if val.Type != tc.wantTyp {
				t.Fatalf("expected type %v, got %v", tc.wantTyp, val.Type)
			}
			switch tc.wantTyp {
			case lang.TypeInt:
				if val.Int() != tc.wantInt {
					t.Fatalf("expected int %d, got %d", tc.wantInt, val.Int())
				}
			case lang.TypeReal:
				if math.Abs(val.Real()-tc.wantF) > 1e-9 {
					t.Fatalf("expected float %f, got %f", tc.wantF, val.Real())
				}
			}
		})
	}
}
