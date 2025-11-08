package reader

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/sergev/gisp/lang"
)

// ReadString parses all expressions from a string.
func ReadString(src string) ([]lang.Value, error) {
	return ReadAll(strings.NewReader(src))
}

// ReadAll parses all expressions from the provided reader.
func ReadAll(r io.Reader) ([]lang.Value, error) {
	rd := newRuneReader(r)
	var values []lang.Value
	for {
		if err := rd.skipWhitespace(); err != nil {
			if errors.Is(err, io.EOF) {
				return values, nil
			}
			return nil, err
		}
		if rd.peekEOF() {
			return values, nil
		}
		val, err := readExpr(rd)
		if err != nil {
			return nil, err
		}
		values = append(values, val)
	}
}

type runeReader struct {
	br   *bufio.Reader
	undo []rune
}

func newRuneReader(r io.Reader) *runeReader {
	return &runeReader{br: bufio.NewReader(r)}
}

func (rr *runeReader) read() (rune, error) {
	if len(rr.undo) > 0 {
		r := rr.undo[len(rr.undo)-1]
		rr.undo = rr.undo[:len(rr.undo)-1]
		return r, nil
	}
	ch, _, err := rr.br.ReadRune()
	return ch, err
}

func (rr *runeReader) unread(r rune) {
	rr.undo = append(rr.undo, r)
}

func (rr *runeReader) peek() (rune, error) {
	r, err := rr.read()
	if err != nil {
		return 0, err
	}
	rr.unread(r)
	return r, nil
}

func (rr *runeReader) peekEOF() bool {
	_, err := rr.peek()
	return errors.Is(err, io.EOF)
}

func (rr *runeReader) skipWhitespace() error {
	for {
		r, err := rr.read()
		if err != nil {
			return err
		}
		if unicode.IsSpace(r) {
			continue
		}
		if r == ';' {
			if err := rr.skipLine(); err != nil {
				return err
			}
			continue
		}
		rr.unread(r)
		return nil
	}
}

func (rr *runeReader) skipLine() error {
	for {
		r, err := rr.read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if r == '\n' {
			return nil
		}
	}
}

func readExpr(rr *runeReader) (lang.Value, error) {
	r, err := rr.read()
	if err != nil {
		return lang.Value{}, err
	}
	switch r {
	case '(':
		return readList(rr)
	case '\'':
		expr, err := readExpr(rr)
		if err != nil {
			return lang.Value{}, err
		}
		return lang.List(lang.SymbolValue("quote"), expr), nil
	case '`':
		expr, err := readExpr(rr)
		if err != nil {
			return lang.Value{}, err
		}
		return lang.List(lang.SymbolValue("quasiquote"), expr), nil
	case ',':
		next, err := rr.peek()
		if err != nil {
			return lang.Value{}, err
		}
		if next == '@' {
			if _, err := rr.read(); err != nil {
				return lang.Value{}, err
			}
			expr, err := readExpr(rr)
			if err != nil {
				return lang.Value{}, err
			}
			return lang.List(lang.SymbolValue("unquote-splicing"), expr), nil
		}
		expr, err := readExpr(rr)
		if err != nil {
			return lang.Value{}, err
		}
		return lang.List(lang.SymbolValue("unquote"), expr), nil
	case '"':
		return readString(rr)
	case '#':
		return readDispatch(rr)
	default:
		if unicode.IsSpace(r) {
			return readExpr(rr)
		}
		if r == ')' {
			return lang.Value{}, fmt.Errorf("unexpected )")
		}
		rr.unread(r)
		return readAtom(rr)
	}
}

func readDispatch(rr *runeReader) (lang.Value, error) {
	r, err := rr.read()
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

func readList(rr *runeReader) (lang.Value, error) {
	if err := rr.skipWhitespace(); err != nil {
		if errors.Is(err, io.EOF) {
			return lang.Value{}, errors.New("unterminated list")
		}
		return lang.Value{}, err
	}

	r, err := rr.peek()
	if err != nil {
		return lang.Value{}, err
	}
	if r == ')' {
		if _, err := rr.read(); err != nil {
			return lang.Value{}, err
		}
		return lang.EmptyList, nil
	}

	var elems []lang.Value
	for {
		if err := rr.skipWhitespace(); err != nil {
			return lang.Value{}, err
		}
		r, err := rr.peek()
		if err != nil {
			return lang.Value{}, err
		}
		if r == ')' {
			if _, err := rr.read(); err != nil {
				return lang.Value{}, err
			}
			break
		}
		if r == '.' {
			if _, err := rr.read(); err != nil {
				return lang.Value{}, err
			}
			if err := rr.skipWhitespace(); err != nil {
				return lang.Value{}, err
			}
			cdr, err := readExpr(rr)
			if err != nil {
				return lang.Value{}, err
			}
			if err := rr.skipWhitespace(); err != nil {
				return lang.Value{}, err
			}
			if next, err := rr.read(); err != nil || next != ')' {
				if err == nil {
					return lang.Value{}, fmt.Errorf("expected ) after dotted pair, got %q", next)
				}
				return lang.Value{}, err
			}
			return buildDottedList(elems, cdr), nil
		}
		elem, err := readExpr(rr)
		if err != nil {
			return lang.Value{}, err
		}
		elems = append(elems, elem)
	}

	return lang.List(elems...), nil
}

func buildDottedList(elems []lang.Value, tail lang.Value) lang.Value {
	result := tail
	for i := len(elems) - 1; i >= 0; i-- {
		result = lang.PairValue(elems[i], result)
	}
	return result
}

func readAtom(rr *runeReader) (lang.Value, error) {
	var builder strings.Builder
	for {
		r, err := rr.read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return lang.Value{}, err
		}
		if unicode.IsSpace(r) || r == '(' || r == ')' || r == '"' || r == ';' {
			rr.unread(r)
			break
		}
		builder.WriteRune(r)
	}
	token := builder.String()
	if len(token) == 0 {
		return lang.Value{}, fmt.Errorf("unexpected token")
	}
	if val, ok := tryNumber(token); ok {
		return val, nil
	}
	return lang.SymbolValue(token), nil
}

func readString(rr *runeReader) (lang.Value, error) {
	var builder strings.Builder
	for {
		r, err := rr.read()
		if err != nil {
			return lang.Value{}, errors.New("unterminated string")
		}
		if r == '"' {
			break
		}
		if r == '\\' {
			esc, err := rr.read()
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

func tryNumber(token string) (lang.Value, bool) {
	if i, err := strconv.ParseInt(token, 10, 64); err == nil {
		return lang.IntValue(i), true
	}
	if f, err := strconv.ParseFloat(token, 64); err == nil {
		return lang.RealValue(f), true
	}
	return lang.Value{}, false
}
