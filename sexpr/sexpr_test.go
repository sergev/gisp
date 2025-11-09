package sexpr

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sergev/gisp/lang"
)

func TestReadStringSuccessCases(t *testing.T) {
	listTail := lang.PairValue(
		lang.SymbolValue("a"),
		lang.PairValue(lang.SymbolValue("b"), lang.SymbolValue("c")),
	)

	cases := []struct {
		name  string
		input string
		want  []lang.Value
	}{
		{
			name:  "SingleInteger",
			input: "42",
			want:  []lang.Value{lang.IntValue(42)},
		},
		{
			name:  "RealNumbers",
			input: "3.14 -2e3",
			want: []lang.Value{
				lang.RealValue(3.14),
				lang.RealValue(-2000),
			},
		},
		{
			name:  "Booleans",
			input: "#t #f",
			want: []lang.Value{
				lang.BoolValue(true),
				lang.BoolValue(false),
			},
		},
		{
			name:  "StringsWithEscapes",
			input: `"hello\nworld" "tab\tquote\" backslash\\"`,
			want: []lang.Value{
				lang.StringValue("hello\nworld"),
				lang.StringValue("tab\tquote\" backslash\\"),
			},
		},
		{
			name:  "SymbolsWithPunctuation",
			input: "foo-bar? +symbol*",
			want: []lang.Value{
				lang.SymbolValue("foo-bar?"),
				lang.SymbolValue("+symbol*"),
			},
		},
		{
			name:  "ProperList",
			input: "(1 2 (3 4))",
			want: []lang.Value{
				lang.List(
					lang.IntValue(1),
					lang.IntValue(2),
					lang.List(lang.IntValue(3), lang.IntValue(4)),
				),
			},
		},
		{
			name:  "DottedList",
			input: "(a b . c)",
			want:  []lang.Value{listTail},
		},
		{
			name:  "QuotingForms",
			input: "'foo `(1 ,x ,@xs) ,bar",
			want: []lang.Value{
				lang.List(lang.SymbolValue("quote"), lang.SymbolValue("foo")),
				lang.List(
					lang.SymbolValue("quasiquote"),
					lang.List(
						lang.IntValue(1),
						lang.List(lang.SymbolValue("unquote"), lang.SymbolValue("x")),
						lang.List(lang.SymbolValue("unquote-splicing"), lang.SymbolValue("xs")),
					),
				),
				lang.List(lang.SymbolValue("unquote"), lang.SymbolValue("bar")),
			},
		},
		{
			name:  "WhitespaceAndComments",
			input: "  ; leading comment\n\n\t42 ; comment after value\n#t",
			want: []lang.Value{
				lang.IntValue(42),
				lang.BoolValue(true),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ReadString(tc.input)
			if err != nil {
				t.Fatalf("ReadString returned error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("ReadString produced %d values, want %d", len(got), len(tc.want))
			}
			for i := range tc.want {
				if !valuesEqual(got[i], tc.want[i]) {
					t.Fatalf("value %d mismatch:\n got: %s\nwant: %s", i, valueString(got[i]), valueString(tc.want[i]))
				}
			}
		})
	}
}

func TestParseLiteralSuccessCases(t *testing.T) {
	cases := []struct {
		name  string
		src   string
		start int
		want  lang.Value
		next  int
	}{
		{
			name:  "LeadingWhitespace",
			src:   "   #f tail",
			start: 0,
			want:  lang.BoolValue(false),
			next:  5,
		},
		{
			name:  "Symbol",
			src:   "foo(bar)",
			start: 0,
			want:  lang.SymbolValue("foo"),
			next:  4,
		},
		{
			name:  "EmbeddedList",
			src:   "-- `(list 1 2) ++",
			start: strings.Index("-- `(list 1 2) ++", "`"),
			want: lang.List(
				lang.SymbolValue("quasiquote"),
				lang.List(
					lang.SymbolValue("list"),
					lang.IntValue(1),
					lang.IntValue(2),
				),
			),
			next: strings.Index("-- `(list 1 2) ++", "`") + len("`(list 1 2)"),
		},
		{
			name:  "StringLiteral",
			src:   `prefix "hi"suffix`,
			start: strings.Index(`prefix "hi"suffix`, `"`),
			want:  lang.StringValue("hi"),
			next:  strings.Index(`prefix "hi"suffix`, `"`) + len("\"hi\""),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, next, err := ParseLiteral(tc.src, tc.start)
			if err != nil {
				t.Fatalf("ParseLiteral returned error: %v", err)
			}
			if next != tc.next {
				t.Fatalf("ParseLiteral next index %d, want %d", next, tc.next)
			}
			if !valuesEqual(got, tc.want) {
				t.Fatalf("ParseLiteral value mismatch:\n got: %s\nwant: %s", valueString(got), valueString(tc.want))
			}
		})
	}
}

func TestReadStringErrorCases(t *testing.T) {
	cases := []struct {
		name  string
		input string
		sub   string
	}{
		{name: "UnexpectedClose", input: ")", sub: "unexpected )"},
		{name: "UnterminatedList", input: "(1 2", sub: "EOF"},
		{name: "UnknownDispatch", input: "#x", sub: "unknown dispatch sequence"},
		{name: "DottedListMisuse", input: "(a . b c)", sub: "expected )"},
		{name: "UnterminatedString", input: `"unterminated`, sub: "unterminated string"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ReadString(tc.input); err == nil {
				t.Fatalf("expected error for %q", tc.input)
			} else if !strings.Contains(err.Error(), tc.sub) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.sub)
			}
		})
	}
}

func TestParseLiteralErrorCases(t *testing.T) {
	cases := []struct {
		name  string
		src   string
		start int
		sub   string
	}{
		{name: "UnexpectedClose", src: ")", start: 0, sub: "unexpected )"},
		{name: "UnknownDispatch", src: "#x", start: 0, sub: "unknown dispatch sequence"},
		{name: "UnterminatedString", src: "\"unterm", start: 0, sub: "unterminated string"},
		{name: "DottedMissingTail", src: "(a . )", start: 0, sub: "unexpected"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := ParseLiteral(tc.src, tc.start); err == nil {
				t.Fatalf("expected error for %q", tc.src)
			} else if !strings.Contains(err.Error(), tc.sub) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.sub)
			}
		})
	}
}

func valuesEqual(a, b lang.Value) bool {
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case lang.TypeEmpty:
		return true
	case lang.TypeBool:
		return a.Bool == b.Bool
	case lang.TypeInt:
		return a.Int == b.Int
	case lang.TypeReal:
		return a.Real == b.Real
	case lang.TypeString:
		return a.Str == b.Str
	case lang.TypeSymbol:
		return a.Sym == b.Sym
	case lang.TypePair:
		if a.Pair == nil || b.Pair == nil {
			return a.Pair == nil && b.Pair == nil
		}
		return valuesEqual(a.Pair.Car, b.Pair.Car) && valuesEqual(a.Pair.Cdr, b.Pair.Cdr)
	default:
		return false
	}
}

func valueString(v lang.Value) string {
	switch v.Type {
	case lang.TypePair:
		return fmt.Sprintf("%s", v.String())
	default:
		return v.String()
	}
}
