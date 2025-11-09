package parser

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sergev/gisp/lang"
)

func lexAllTokens(t *testing.T, src string) []Token {
	t.Helper()
	lx := newLexer(src)
	var tokens []Token
	for {
		tok, err := lx.nextToken()
		if err != nil {
			t.Fatalf("unexpected lexer error after %d tokens: %v", len(tokens), err)
		}
		tokens = append(tokens, tok)
		if tok.Type == tokenEOF {
			break
		}
	}
	return tokens
}

func mustNextToken(t *testing.T, lx *lexer) Token {
	t.Helper()
	tok, err := lx.nextToken()
	if err != nil {
		t.Fatalf("unexpected lexer error: %v", err)
	}
	return tok
}

func TestLexerIdentifiersAndKeywords(t *testing.T) {
	src := "func var const if else while return true false foo _bar baz123"
	tokens := lexAllTokens(t, src)
	tokens = tokens[:len(tokens)-1] // drop EOF

	want := []struct {
		typ    TokenType
		lexeme string
	}{
		{tokenFunc, ""},
		{tokenVar, ""},
		{tokenConst, ""},
		{tokenIf, ""},
		{tokenElse, ""},
		{tokenWhile, ""},
		{tokenReturn, ""},
		{tokenTrue, ""},
		{tokenFalse, ""},
		{tokenIdentifier, "foo"},
		{tokenIdentifier, "_bar"},
		{tokenIdentifier, "baz123"},
	}

	if len(tokens) != len(want) {
		t.Fatalf("expected %d tokens, got %d", len(want), len(tokens))
	}

	for i, tt := range want {
		tok := tokens[i]
		if tok.Type != tt.typ {
			t.Errorf("token %d: expected type %v, got %v", i, tt.typ, tok.Type)
		}
		if tok.Lexeme != tt.lexeme {
			t.Errorf("token %d: expected lexeme %q, got %q", i, tt.lexeme, tok.Lexeme)
		}
	}
}

func TestLexerNumberLiterals(t *testing.T) {
	src := "0 123 3.14 6.022e23 1e-9 42e+7 10."
	tokens := lexAllTokens(t, src)
	tokens = tokens[:len(tokens)-1]

	wantLexemes := []string{
		"0",
		"123",
		"3.14",
		"6.022e23",
		"1e-9",
		"42e+7",
		"10.",
	}

	if len(tokens) != len(wantLexemes) {
		t.Fatalf("expected %d tokens, got %d", len(wantLexemes), len(tokens))
	}

	for i, lexeme := range wantLexemes {
		tok := tokens[i]
		if tok.Type != tokenNumber {
			t.Errorf("token %d: expected number type, got %v", i, tok.Type)
		}
		if tok.Lexeme != lexeme {
			t.Errorf("token %d: expected lexeme %q, got %q", i, lexeme, tok.Lexeme)
		}
	}
}

func TestLexerNumberErrors(t *testing.T) {
	lx := newLexer("1e")
	if _, err := lx.nextToken(); err == nil || !strings.Contains(err.Error(), "unterminated exponent") {
		t.Fatalf("expected unterminated exponent error, got %v", err)
	}
}

func TestLexerStringLiterals(t *testing.T) {
	src := "\"hello\\nworld\" \"tab\\tquote\\\" backslash\\\\\""
	tokens := lexAllTokens(t, src)
	tokens = tokens[:len(tokens)-1]

	want := []string{
		"hello\nworld",
		"tab\tquote\" backslash\\",
	}

	if len(tokens) != len(want) {
		t.Fatalf("expected %d tokens, got %d", len(want), len(tokens))
	}

	for i, expected := range want {
		tok := tokens[i]
		if tok.Type != tokenString {
			t.Errorf("token %d: expected string type, got %v", i, tok.Type)
		}
		value, ok := tok.Value.(string)
		if !ok {
			t.Fatalf("token %d: expected string value type, got %T", i, tok.Value)
		}
		if value != expected {
			t.Errorf("token %d: expected value %q, got %q", i, expected, value)
		}
	}
}

func TestLexerStringErrors(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name:    "newline",
			src:     "\"line1\nline2\"",
			wantErr: "newline in string literal",
		},
		{
			name:    "unterminated",
			src:     "\"unterminated",
			wantErr: "unterminated string literal",
		},
		{
			name:    "unterminated escape",
			src:     "\"unterminated escape " + string('\\'),
			wantErr: "unterminated escape sequence",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lx := newLexer(tc.src)
			if _, err := lx.nextToken(); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestLexerSExprLiteral(t *testing.T) {
	src := "`(list 1 2 3)"
	lx := newLexer(src)

	tok := mustNextToken(t, lx)
	if tok.Type != tokenSExpr {
		t.Fatalf("expected tokenSExpr, got %v", tok.Type)
	}

	value, ok := tok.Value.(lang.Value)
	if !ok {
		t.Fatalf("expected lang.Value, got %T", tok.Value)
	}

	want := lang.List(
		lang.SymbolValue("list"),
		lang.IntValue(1),
		lang.IntValue(2),
		lang.IntValue(3),
	)

	if !reflect.DeepEqual(value, want) {
		t.Fatalf("expected S-expr value %s, got %s", want.String(), value.String())
	}

	if tok.Pos.Offset != 0 || tok.Pos.Line != 1 || tok.Pos.Column != 1 {
		t.Fatalf("unexpected position: %+v", tok.Pos)
	}

	if tok, err := lx.nextToken(); err != nil || tok.Type != tokenEOF {
		t.Fatalf("expected EOF after S-expr literal, got token %v err %v", tok, err)
	}
}

func TestLexerSExprStopsBeforeComma(t *testing.T) {
	lx := newLexer("`'+,")

	tok := mustNextToken(t, lx)
	if tok.Type != tokenSExpr {
		t.Fatalf("expected tokenSExpr, got %v", tok.Type)
	}
	value, ok := tok.Value.(lang.Value)
	if !ok {
		t.Fatalf("expected lang.Value, got %T", tok.Value)
	}
	want := lang.List(
		lang.SymbolValue("quote"),
		lang.SymbolValue("+"),
	)
	if !reflect.DeepEqual(value, want) {
		t.Fatalf("expected value %s, got %s", want.String(), value.String())
	}

	commaTok := mustNextToken(t, lx)
	if commaTok.Type != tokenComma {
		t.Fatalf("expected comma token after S-expr, got %v", commaTok.Type)
	}

	if tok, err := lx.nextToken(); err != nil || tok.Type != tokenEOF {
		t.Fatalf("expected EOF after comma, got token %v err %v", tok, err)
	}
}

func TestLexerSkipWhitespaceAndComments(t *testing.T) {
	src := " \t\n// comment\n/* block\ncomment */\nfoo"
	lx := newLexer(src)

	tok := mustNextToken(t, lx)
	if tok.Type != tokenIdentifier {
		t.Fatalf("expected identifier token, got %v", tok.Type)
	}
	if tok.Lexeme != "foo" {
		t.Fatalf("expected lexeme foo, got %q", tok.Lexeme)
	}
	if tok.Pos.Line != 5 || tok.Pos.Column != 1 {
		t.Fatalf("expected position line 5 column 1, got %+v", tok.Pos)
	}
}

func TestLexerBlockCommentUnterminated(t *testing.T) {
	lx := newLexer("/* unterminated")
	if _, err := lx.nextToken(); err == nil || !strings.Contains(err.Error(), "unterminated block comment") {
		t.Fatalf("expected unterminated block comment error, got %v", err)
	}
}

func TestLexerOperatorAndPunctuationTokens(t *testing.T) {
	src := "+ - * / = == ! != < <= > >= && || , ; ( ) { } [ ]"
	lx := newLexer(src)

	want := []TokenType{
		tokenPlus,
		tokenMinus,
		tokenStar,
		tokenSlash,
		tokenAssign,
		tokenEqualEqual,
		tokenBang,
		tokenBangEqual,
		tokenLess,
		tokenLessEqual,
		tokenGreater,
		tokenGreaterEqual,
		tokenAndAnd,
		tokenOrOr,
		tokenComma,
		tokenSemicolon,
		tokenLParen,
		tokenRParen,
		tokenLBrace,
		tokenRBrace,
		tokenLBracket,
		tokenRBracket,
	}

	for i, typ := range want {
		tok := mustNextToken(t, lx)
		if tok.Type != typ {
			t.Fatalf("token %d: expected %v, got %v", i, typ, tok.Type)
		}
	}

	tok := mustNextToken(t, lx)
	if tok.Type != tokenEOF {
		t.Fatalf("expected EOF, got %v", tok.Type)
	}
}

func TestLexerUnsupportedPercentOperator(t *testing.T) {
	lx := newLexer("%")
	tok, err := lx.nextToken()
	if err == nil {
		t.Fatalf("expected error for unsupported operator")
	}
	if tok.Type != tokenIllegal {
		t.Fatalf("expected illegal token, got %v", tok.Type)
	}
	if !strings.Contains(err.Error(), "unsupported operator %") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestLexerSingleAmpersandAndPipe(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantErr string
	}{
		{"ampersand", "&", "unexpected '&'"},
		{"pipe", "|", "unexpected '|'"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lx := newLexer(tc.src)
			tok, err := lx.nextToken()
			if err == nil {
				t.Fatalf("expected error, got none")
			}
			if tok.Type != tokenIllegal {
				t.Fatalf("expected illegal token, got %v", tok.Type)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("unexpected error message: %v", err)
			}
		})
	}
}

func TestLexerUnexpectedCharacter(t *testing.T) {
	lx := newLexer("@")
	tok, err := lx.nextToken()
	if err == nil {
		t.Fatalf("expected error for unexpected character")
	}
	if tok.Type != tokenIllegal {
		t.Fatalf("expected illegal token, got %v", tok.Type)
	}
	if !strings.Contains(err.Error(), "unexpected character '@'") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestLexerInvalidUTF8(t *testing.T) {
	lx := newLexer(string([]byte{0xff}))
	if _, err := lx.nextToken(); err == nil || !strings.Contains(err.Error(), "invalid UTF-8 encoding") {
		t.Fatalf("expected invalid UTF-8 error, got %v", err)
	}
}
