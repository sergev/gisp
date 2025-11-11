package parser

import (
	"errors"
	"strings"
	"testing"

	"github.com/sergev/gisp/lang"
)

func TestParseStringProducesCompiledForms(t *testing.T) {
	src := `
var answer = 41;
answer + 1;
`
	forms, err := ParseString(src)
	if err != nil {
		t.Fatalf("ParseString returned error: %v", err)
	}
	if len(forms) != 2 {
		t.Fatalf("expected two forms (define and expression), got %d", len(forms))
	}

	defineForm, err := lang.ToSlice(forms[0])
	if err != nil {
		t.Fatalf("expected define form to be a proper list: %v", err)
	}
	if defineForm[0].Type != lang.TypeSymbol || defineForm[0].Sym() != "define" {
		t.Fatalf("expected first form head to be define, got %v", defineForm[0])
	}
	if defineForm[1].Sym() != "answer" {
		t.Fatalf("expected define target answer, got %v", defineForm[1])
	}
	if defineForm[2].Int() != 41 {
		t.Fatalf("expected answer initializer 41, got %v", defineForm[2])
	}
}

func TestParseStringPropagatesSyntaxErrors(t *testing.T) {
	if _, err := ParseString("var = 1;"); err == nil || !strings.Contains(err.Error(), "expected identifier") {
		t.Fatalf("expected syntax error for malformed var declaration, got %v", err)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("boom")
}

func TestParseReaderHandlesIOReturns(t *testing.T) {
	if _, err := ParseReader(failingReader{}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected underlying IO error, got %v", err)
	}

	reader := strings.NewReader("const value = 5; value;")
	forms, err := ParseReader(reader)
	if err != nil {
		t.Fatalf("ParseReader returned error: %v", err)
	}
	if len(forms) != 2 {
		t.Fatalf("expected two forms from reader, got %d", len(forms))
	}
}

func TestParseStringUnaryCaret(t *testing.T) {
	src := "display(^123);\n"
	forms, err := ParseString(src)
	if err != nil {
		t.Fatalf("ParseString returned error: %v", err)
	}
	if len(forms) != 1 {
		t.Fatalf("expected single form, got %d", len(forms))
	}

	callForm, err := lang.ToSlice(forms[0])
	if err != nil {
		t.Fatalf("expected proper list for call form: %v", err)
	}
	if len(callForm) != 2 {
		t.Fatalf("expected display call with one argument, got %d elements", len(callForm))
	}
	if callForm[0].Type != lang.TypeSymbol || callForm[0].Sym() != "display" {
		t.Fatalf("expected display symbol, got %v", callForm[0])
	}

	argList, err := lang.ToSlice(callForm[1])
	if err != nil {
		t.Fatalf("expected argument list to be proper list: %v", err)
	}
	if len(argList) != 2 {
		t.Fatalf("expected unary ^ form to have head and value, got %d elements", len(argList))
	}
	if argList[0].Type != lang.TypeSymbol || argList[0].Sym() != "^" {
		t.Fatalf("expected ^ symbol as unary head, got %v", argList[0])
	}
	if argList[1].Type != lang.TypeInt || argList[1].Int() != 123 {
		t.Fatalf("expected integer literal 123, got %v", argList[1])
	}
}

func TestParseStringVectorLiteral(t *testing.T) {
	src := "var vec = #[1, 2, 3]; vec;\n"
	forms, err := ParseString(src)
	if err != nil {
		t.Fatalf("ParseString returned error: %v", err)
	}
	if len(forms) != 2 {
		t.Fatalf("expected define and expression forms, got %d", len(forms))
	}

	defineForm, err := lang.ToSlice(forms[0])
	if err != nil {
		t.Fatalf("expected define form to be a proper list: %v", err)
	}
	if len(defineForm) != 3 {
		t.Fatalf("expected define with three elements, got %d", len(defineForm))
	}
	if defineForm[2].Type != lang.TypePair {
		t.Fatalf("expected initializer to be list, got %v", defineForm[2])
	}
	initList, err := lang.ToSlice(defineForm[2])
	if err != nil {
		t.Fatalf("expected initializer to be proper list: %v", err)
	}
	if len(initList) != 4 || initList[0].Sym() != "vector" {
		t.Fatalf("expected (vector 1 2 3) initializer, got %v", initList)
	}
	for i := 1; i <= 3; i++ {
		if initList[i].Type != lang.TypeInt || initList[i].Int() != int64(i) {
			t.Fatalf("expected vector element %d, got %v", i, initList[i])
		}
	}
}
