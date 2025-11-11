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

func dropTrailingSemicolons(tokens []Token) []Token {
	for len(tokens) > 0 && tokens[len(tokens)-1].Type == tokenSemicolon {
		tokens = tokens[:len(tokens)-1]
	}
	return tokens
}

func TestLexerIdentifiersAndKeywords(t *testing.T) {
	src := "func var const if else while return true false nil foo _bar baz123"
	tokens := lexAllTokens(t, src)
	tokens = tokens[:len(tokens)-1] // drop EOF
	tokens = dropTrailingSemicolons(tokens)

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
		{tokenNil, ""},
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
	tokens = dropTrailingSemicolons(tokens)

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
	tokens = dropTrailingSemicolons(tokens)

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

	tok = mustNextToken(t, lx)
	if tok.Type == tokenSemicolon {
		tok = mustNextToken(t, lx)
	}
	if tok.Type != tokenEOF {
		t.Fatalf("expected EOF after S-expr literal, got token %v", tok)
	}
}

func TestLexerCompoundAssignmentTokens(t *testing.T) {
	src := "x += y -= z *= w /= q %= r <<= s >>= t &= u |= v ^= w &^= z"
	tokens := lexAllTokens(t, src)
	tokens = tokens[:len(tokens)-1]
	tokens = dropTrailingSemicolons(tokens)

	want := []TokenType{
		tokenIdentifier, tokenPlusAssign, tokenIdentifier,
		tokenMinusAssign, tokenIdentifier, tokenStarAssign,
		tokenIdentifier, tokenSlashAssign, tokenIdentifier,
		tokenPercentAssign, tokenIdentifier, tokenShiftLeftAssign,
		tokenIdentifier, tokenShiftRightAssign, tokenIdentifier,
		tokenAmpersandAssign, tokenIdentifier, tokenPipeAssign,
		tokenIdentifier, tokenCaretAssign, tokenIdentifier,
		tokenAmpersandCaretAssign, tokenIdentifier,
	}

	if len(tokens) != len(want) {
		t.Fatalf("expected %d tokens, got %d", len(want), len(tokens))
	}

	for i, typ := range want {
		if tokens[i].Type != typ {
			t.Fatalf("token %d: expected %v, got %v", i, typ, tokens[i].Type)
		}
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
	src := "+ - * / % ^ & | << >> &^ = == ! != < <= > >= && || , ; ( ) { } [ ]"
	lx := newLexer(src)

	want := []TokenType{
		tokenPlus,
		tokenMinus,
		tokenStar,
		tokenSlash,
		tokenPercent,
		tokenCaret,
		tokenAmpersand,
		tokenPipe,
		tokenShiftLeft,
		tokenShiftRight,
		tokenAmpersandCaret,
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
	if tok.Type == tokenSemicolon {
		tok = mustNextToken(t, lx)
	}
	if tok.Type != tokenEOF {
		t.Fatalf("expected EOF, got %v", tok.Type)
	}
}

func TestLexerPostIncDec(t *testing.T) {
	src := "x++\ny--"
	tokens := lexAllTokens(t, src)
	var types []TokenType
	for _, tok := range tokens {
		if tok.Type == tokenEOF || tok.Type == tokenSemicolon {
			continue
		}
		types = append(types, tok.Type)
	}
	want := []TokenType{
		tokenIdentifier,
		tokenPlusPlus,
		tokenIdentifier,
		tokenMinusMinus,
	}
	if !reflect.DeepEqual(types, want) {
		t.Fatalf("unexpected token sequence: got %v want %v", types, want)
	}
}

func TestLexerBitwiseOperators(t *testing.T) {
	src := "& | &^ << >>"
	lx := newLexer(src)

	want := []TokenType{
		tokenAmpersand,
		tokenPipe,
		tokenAmpersandCaret,
		tokenShiftLeft,
		tokenShiftRight,
	}

	for i, typ := range want {
		tok := mustNextToken(t, lx)
		if tok.Type != typ {
			t.Fatalf("token %d: expected %v, got %v", i, typ, tok.Type)
		}
	}

	if tok := mustNextToken(t, lx); tok.Type != tokenEOF {
		t.Fatalf("expected EOF, got %v", tok.Type)
	}
}

func TestLexerAutomaticSemicolonsAtNewline(t *testing.T) {
	src := "var x = 1\nvar y = 2\n"
	tokens := lexAllTokens(t, src)

	var got []TokenType
	for _, tok := range tokens {
		got = append(got, tok.Type)
	}

	want := []TokenType{
		tokenVar, tokenIdentifier, tokenAssign, tokenNumber, tokenSemicolon,
		tokenVar, tokenIdentifier, tokenAssign, tokenNumber, tokenSemicolon,
		tokenEOF,
	}

	if len(got) != len(want) {
		t.Fatalf("token count mismatch\ngot:  %v\nwant: %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("token %d: expected %v, got %v", i, want[i], got[i])
		}
	}
}

func TestLexerSemicolonBeforeClosingBrace(t *testing.T) {
	src := "func f() { return }"
	tokens := lexAllTokens(t, src)

	var sequence []TokenType
	for _, tok := range tokens {
		sequence = append(sequence, tok.Type)
	}

	if len(sequence) > 1 && sequence[len(sequence)-2] == tokenSemicolon && sequence[len(sequence)-1] == tokenEOF {
		sequence = append([]TokenType{}, sequence[:len(sequence)-2]...)
		sequence = append(sequence, tokenEOF)
	}

	want := []TokenType{
		tokenFunc, tokenIdentifier, tokenLParen, tokenRParen, tokenLBrace,
		tokenReturn, tokenSemicolon, tokenRBrace, tokenEOF,
	}

	if !reflect.DeepEqual(sequence, want) {
		t.Fatalf("unexpected token sequence\n got: %v\nwant: %v", sequence, want)
	}
}

func TestLexerAutomaticSemicolonInsideCallArgBlock(t *testing.T) {
	src := `
func demo() {
	var saved = callcc(func(k) {
		return "initial return"
	})
}
`
	tokens := lexAllTokens(t, src)

	var types []TokenType
	for _, tok := range tokens {
		types = append(types, tok.Type)
	}

	// Ensure the return inside the nested function literal is terminated automatically.
	found := false
	for i := 0; i < len(types)-2; i++ {
		if types[i] == tokenReturn && types[i+1] == tokenString && types[i+2] == tokenSemicolon {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected automatic semicolon after return string, got sequence %v", types)
	}

	// Ensure the variable declaration statement also ends with an automatic semicolon.
	varDeclSeq := []TokenType{
		tokenVar, tokenIdentifier, tokenAssign, tokenIdentifier, tokenLParen,
	}
	start := indexOfSubslice(types, varDeclSeq)
	if start == -1 {
		t.Fatalf("expected to find var declaration tokens in %v", types)
	}
	if idx := indexOfToken(types[start+len(varDeclSeq):], tokenSemicolon); idx == -1 {
		t.Fatalf("expected automatic semicolon after call expression, got sequence %v", types)
	}
}

func indexOfSubslice(haystack, needle []TokenType) int {
outer:
	for i := 0; i <= len(haystack)-len(needle); i++ {
		for j := range needle {
			if haystack[i+j] != needle[j] {
				continue outer
			}
		}
		return i
	}
	return -1
}

func indexOfToken(types []TokenType, target TokenType) int {
	for i, typ := range types {
		if typ == target {
			return i
		}
	}
	return -1
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
