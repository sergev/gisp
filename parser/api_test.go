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
