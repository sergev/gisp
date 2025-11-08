package parser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sergev/gisp/lang"
)

// parseSExprLiteral parses a single Scheme expression starting at the provided byte offset.
// It returns the parsed value alongside the byte index immediately following the expression.
func parseSExprLiteral(src string, start int) (lang.Value, int, error) {
	sc := newSExprScanner(src, start)
	if err := sc.skipWhitespace(); err != nil {
		return lang.Value{}, 0, err
	}
	val, err := readSExpr(sc)
	if err != nil {
		return lang.Value{}, 0, err
	}
	return val, sc.pos, nil
}

type runeWidth struct {
	r rune
	w int
}

type sExprScanner struct {
	src  string
	pos  int
	undo []runeWidth
}

var errUnexpectedEOF = errors.New("unexpected EOF")

func newSExprScanner(src string, start int) *sExprScanner {
	return &sExprScanner{src: src, pos: start}
}

func (sc *sExprScanner) read() (rune, int, error) {
	if len(sc.undo) > 0 {
		last := sc.undo[len(sc.undo)-1]
		sc.undo = sc.undo[:len(sc.undo)-1]
		sc.pos += last.w
		return last.r, last.w, nil
	}
	if sc.pos >= len(sc.src) {
		return 0, 0, errUnexpectedEOF
	}
	r, w := utf8.DecodeRuneInString(sc.src[sc.pos:])
	if r == utf8.RuneError && w == 1 {
		return 0, 0, fmt.Errorf("invalid UTF-8 encoding at byte %d", sc.pos)
	}
	sc.pos += w
	return r, w, nil
}

func (sc *sExprScanner) unread(r rune, w int) {
	sc.pos -= w
	sc.undo = append(sc.undo, runeWidth{r: r, w: w})
}

func (sc *sExprScanner) peek() (rune, error) {
	r, w, err := sc.read()
	if err != nil {
		return 0, err
	}
	sc.unread(r, w)
	return r, nil
}

func (sc *sExprScanner) skipWhitespace() error {
	for {
		r, w, err := sc.read()
		if err != nil {
			return err
		}
		switch {
		case unicode.IsSpace(r):
			continue
		case r == ';':
			if err := sc.skipLine(); err != nil {
				return err
			}
			continue
		default:
			sc.unread(r, w)
			return nil
		}
	}
}

func (sc *sExprScanner) skipLine() error {
	for {
		r, _, err := sc.read()
		if err != nil {
			return err
		}
		if r == '\n' {
			return nil
		}
	}
}

func readSExpr(sc *sExprScanner) (lang.Value, error) {
	r, w, err := sc.read()
	if err != nil {
		return lang.Value{}, err
	}
	switch r {
	case '(':
		return readSExprList(sc)
	case '\'':
		expr, err := readSExpr(sc)
		if err != nil {
			return lang.Value{}, err
		}
		return lang.List(lang.SymbolValue("quote"), expr), nil
	case '`':
		expr, err := readSExpr(sc)
		if err != nil {
			return lang.Value{}, err
		}
		return lang.List(lang.SymbolValue("quasiquote"), expr), nil
	case ',':
		next, err := sc.peek()
		if err != nil {
			return lang.Value{}, err
		}
		if next == '@' {
			if _, _, err := sc.read(); err != nil {
				return lang.Value{}, err
			}
			expr, err := readSExpr(sc)
			if err != nil {
				return lang.Value{}, err
			}
			return lang.List(lang.SymbolValue("unquote-splicing"), expr), nil
		}
		expr, err := readSExpr(sc)
		if err != nil {
			return lang.Value{}, err
		}
		return lang.List(lang.SymbolValue("unquote"), expr), nil
	case '"':
		return readSExprString(sc)
	case '#':
		return readSExprDispatch(sc)
	default:
		if unicode.IsSpace(r) {
			return readSExpr(sc)
		}
		if r == ')' {
			return lang.Value{}, fmt.Errorf("unexpected )")
		}
		sc.unread(r, w)
		return readSExprAtom(sc)
	}
}

func readSExprDispatch(sc *sExprScanner) (lang.Value, error) {
	r, _, err := sc.read()
	if err != nil {
		return lang.Value{}, err
	}
	switch r {
	case 't':
		return lang.BoolValue(true), nil
	case 'f':
		return lang.BoolValue(false), nil
	default:
		return lang.Value{}, fmt.Errorf("unknown dispatch sequence: #%c", r)
	}
}

func readSExprList(sc *sExprScanner) (lang.Value, error) {
	if err := sc.skipWhitespace(); err != nil {
		if errors.Is(err, errUnexpectedEOF) {
			return lang.Value{}, errors.New("unterminated list")
		}
		return lang.Value{}, err
	}
	r, err := sc.peek()
	if err != nil {
		return lang.Value{}, err
	}
	if r == ')' {
		if _, _, err := sc.read(); err != nil {
			return lang.Value{}, err
		}
		return lang.EmptyList, nil
	}
	var elems []lang.Value
	for {
		if err := sc.skipWhitespace(); err != nil {
			return lang.Value{}, err
		}
		next, err := sc.peek()
		if err != nil {
			return lang.Value{}, err
		}
		if next == ')' {
			if _, _, err := sc.read(); err != nil {
				return lang.Value{}, err
			}
			break
		}
		if next == '.' {
			if _, _, err := sc.read(); err != nil {
				return lang.Value{}, err
			}
			if err := sc.skipWhitespace(); err != nil {
				return lang.Value{}, err
			}
			cdr, err := readSExpr(sc)
			if err != nil {
				return lang.Value{}, err
			}
			if err := sc.skipWhitespace(); err != nil {
				return lang.Value{}, err
			}
			r, _, err := sc.read()
			if err != nil || r != ')' {
				if err == nil {
					return lang.Value{}, fmt.Errorf("expected ) after dotted pair, got %q", r)
				}
				return lang.Value{}, err
			}
			return buildSExprDottedList(elems, cdr), nil
		}
		elem, err := readSExpr(sc)
		if err != nil {
			return lang.Value{}, err
		}
		elems = append(elems, elem)
	}
	return lang.List(elems...), nil
}

func buildSExprDottedList(elems []lang.Value, tail lang.Value) lang.Value {
	result := tail
	for i := len(elems) - 1; i >= 0; i-- {
		result = lang.PairValue(elems[i], result)
	}
	return result
}

func readSExprAtom(sc *sExprScanner) (lang.Value, error) {
	var builder strings.Builder
	for {
		r, w, err := sc.read()
		if err != nil {
			if errors.Is(err, errUnexpectedEOF) {
				break
			}
			return lang.Value{}, err
		}
		if unicode.IsSpace(r) || r == '(' || r == ')' || r == '"' || r == ';' {
			sc.unread(r, w)
			break
		}
		builder.WriteRune(r)
	}
	token := builder.String()
	if len(token) == 0 {
		return lang.Value{}, fmt.Errorf("unexpected token")
	}
	if val, ok := trySExprNumber(token); ok {
		return val, nil
	}
	return lang.SymbolValue(token), nil
}

func readSExprString(sc *sExprScanner) (lang.Value, error) {
	var builder strings.Builder
	for {
		r, _, err := sc.read()
		if err != nil {
			return lang.Value{}, errors.New("unterminated string")
		}
		if r == '"' {
			break
		}
		if r == '\\' {
			esc, _, err := sc.read()
			if err != nil {
				return lang.Value{}, errors.New("unterminated escape sequence")
			}
			switch esc {
			case 'n':
				builder.WriteRune('\n')
			case 't':
				builder.WriteRune('\t')
			case '\\':
				builder.WriteRune('\\')
			case '"':
				builder.WriteRune('"')
			default:
				builder.WriteRune(esc)
			}
			continue
		}
		builder.WriteRune(r)
	}
	return lang.StringValue(builder.String()), nil
}

func trySExprNumber(token string) (lang.Value, bool) {
	if i, err := strconv.ParseInt(token, 10, 64); err == nil {
		return lang.IntValue(i), true
	}
	if f, err := strconv.ParseFloat(token, 64); err == nil {
		return lang.RealValue(f), true
	}
	return lang.Value{}, false
}
