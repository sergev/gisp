package sexpr

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sergev/gisp/lang"
)

var errUnexpectedEOF = errors.New("unexpected EOF")

type runeWidth struct {
	r rune
	w int
}

type runeSource interface {
	read() (rune, int, error)
}

type scanner struct {
	src               runeSource
	undo              []runeWidth
	isEOF             func(error) bool
	allowEOFInComment bool
}

func newScanner(src runeSource, isEOF func(error) bool, allowEOFInComment bool) *scanner {
	return &scanner{
		src:               src,
		isEOF:             isEOF,
		allowEOFInComment: allowEOFInComment,
	}
}

func (s *scanner) read() (rune, int, error) {
	if len(s.undo) > 0 {
		last := s.undo[len(s.undo)-1]
		s.undo = s.undo[:len(s.undo)-1]
		return last.r, last.w, nil
	}
	r, w, err := s.src.read()
	if err != nil {
		return 0, 0, err
	}
	return r, w, nil
}

func (s *scanner) unread(r rune, w int) {
	s.undo = append(s.undo, runeWidth{r: r, w: w})
}

func (s *scanner) peek() (rune, int, error) {
	r, w, err := s.read()
	if err != nil {
		return 0, 0, err
	}
	s.unread(r, w)
	return r, w, nil
}

func (s *scanner) peekEOF() bool {
	_, _, err := s.peek()
	return err != nil && s.isEOF(err)
}

func (s *scanner) skipWhitespace() error {
	for {
		r, w, err := s.read()
		if err != nil {
			return err
		}
		switch {
		case unicode.IsSpace(r):
			continue
		case r == ';':
			if err := s.skipLine(); err != nil {
				return err
			}
			continue
		default:
			s.unread(r, w)
			return nil
		}
	}
}

func (s *scanner) skipLine() error {
	for {
		r, _, err := s.read()
		if err != nil {
			if s.allowEOFInComment && s.isEOF(err) {
				return nil
			}
			return err
		}
		if r == '\n' {
			return nil
		}
	}
}

func readExpr(sc *scanner) (lang.Value, error) {
	r, w, err := sc.read()
	if err != nil {
		return lang.Value{}, err
	}
	switch r {
	case '(':
		return readList(sc)
	case '\'':
		expr, err := readExpr(sc)
		if err != nil {
			return lang.Value{}, err
		}
		return lang.List(lang.SymbolValue("quote"), expr), nil
	case '`':
		expr, err := readExpr(sc)
		if err != nil {
			return lang.Value{}, err
		}
		return lang.List(lang.SymbolValue("quasiquote"), expr), nil
	case ',':
		next, _, err := sc.peek()
		if err != nil {
			return lang.Value{}, err
		}
		if next == '@' {
			if _, _, err := sc.read(); err != nil {
				return lang.Value{}, err
			}
			expr, err := readExpr(sc)
			if err != nil {
				return lang.Value{}, err
			}
			return lang.List(lang.SymbolValue("unquote-splicing"), expr), nil
		}
		expr, err := readExpr(sc)
		if err != nil {
			return lang.Value{}, err
		}
		return lang.List(lang.SymbolValue("unquote"), expr), nil
	case '"':
		return readString(sc)
	case '#':
		return readDispatch(sc)
	default:
		if unicode.IsSpace(r) {
			return readExpr(sc)
		}
		if r == ')' {
			return lang.Value{}, fmt.Errorf("unexpected )")
		}
		sc.unread(r, w)
		return readAtom(sc)
	}
}

func readDispatch(sc *scanner) (lang.Value, error) {
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

func readList(sc *scanner) (lang.Value, error) {
	if err := sc.skipWhitespace(); err != nil {
		if sc.isEOF(err) {
			return lang.Value{}, errors.New("unterminated list")
		}
		return lang.Value{}, err
	}
	r, _, err := sc.peek()
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
		next, _, err := sc.peek()
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
			cdr, err := readExpr(sc)
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
			return buildDottedList(elems, cdr), nil
		}
		elem, err := readExpr(sc)
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

func readAtom(sc *scanner) (lang.Value, error) {
	var builder strings.Builder
	for {
		r, w, err := sc.read()
		if err != nil {
			if sc.isEOF(err) {
				break
			}
			return lang.Value{}, err
		}
		if unicode.IsSpace(r) || r == '(' || r == ')' || r == '"' || r == ';' ||
			r == ',' || r == ']' || r == '}' {
			sc.unread(r, w)
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

func readString(sc *scanner) (lang.Value, error) {
	var builder strings.Builder
	for {
		r, _, err := sc.read()
		if err != nil {
			if sc.isEOF(err) {
				return lang.Value{}, errors.New("unterminated string")
			}
			return lang.Value{}, err
		}
		if r == '"' {
			break
		}
		if r == '\\' {
			esc, _, err := sc.read()
			if err != nil {
				if sc.isEOF(err) {
					return lang.Value{}, errors.New("unterminated escape sequence")
				}
				return lang.Value{}, err
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

// ParseAll reads all s-expressions from the provided reader.
func ParseAll(r io.Reader) ([]lang.Value, error) {
	sc := newScanner(newReaderSource(r), func(err error) bool { return errors.Is(err, io.EOF) }, true)
	return parseAll(sc)
}

func parseAll(sc *scanner) ([]lang.Value, error) {
	var values []lang.Value
	for {
		if err := sc.skipWhitespace(); err != nil {
			if sc.isEOF(err) {
				return values, nil
			}
			return nil, err
		}
		if sc.peekEOF() {
			return values, nil
		}
		val, err := readExpr(sc)
		if err != nil {
			return nil, err
		}
		values = append(values, val)
	}
}

// ReadString parses all expressions from a string.
func ReadString(src string) ([]lang.Value, error) {
	return ParseAll(strings.NewReader(src))
}

// Reader incrementally reads s-expressions from an input stream.
type Reader struct {
	sc *scanner
}

// NewReader constructs a Reader over r.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		sc: newScanner(newReaderSource(r), func(err error) bool { return errors.Is(err, io.EOF) }, true),
	}
}

// Read parses and returns the next s-expression from the stream.
// It returns io.EOF when no more expressions are available.
func (rd *Reader) Read() (lang.Value, error) {
	if rd == nil || rd.sc == nil {
		return lang.Value{}, io.EOF
	}
	if err := rd.sc.skipWhitespace(); err != nil {
		if rd.sc.isEOF(err) {
			return lang.Value{}, io.EOF
		}
		return lang.Value{}, err
	}
	if rd.sc.peekEOF() {
		return lang.Value{}, io.EOF
	}
	val, err := readExpr(rd.sc)
	if err != nil {
		if rd.sc.isEOF(err) {
			return lang.Value{}, io.EOF
		}
		return lang.Value{}, err
	}
	return val, nil
}

// ParseLiteral parses a single s-expression literal from the source string starting at the given byte offset.
// It returns the parsed value and the index immediately following the expression.
func ParseLiteral(src string, start int) (lang.Value, int, error) {
	source := newStringSource(src, start)
	sc := newScanner(source, func(err error) bool { return errors.Is(err, errUnexpectedEOF) }, false)
	if err := sc.skipWhitespace(); err != nil {
		return lang.Value{}, 0, err
	}
	val, err := readExpr(sc)
	if err != nil {
		return lang.Value{}, 0, err
	}
	next := source.pos
	for _, rw := range sc.undo {
		next -= rw.w
	}
	return val, next, nil
}

type readerSource struct {
	br *bufio.Reader
}

func newReaderSource(r io.Reader) *readerSource {
	return &readerSource{br: bufio.NewReader(r)}
}

func (rs *readerSource) read() (rune, int, error) {
	r, w, err := rs.br.ReadRune()
	return r, w, err
}

type stringSource struct {
	src string
	pos int
}

func newStringSource(src string, start int) *stringSource {
	return &stringSource{src: src, pos: start}
}

func (ss *stringSource) read() (rune, int, error) {
	if ss.pos >= len(ss.src) {
		return 0, 0, errUnexpectedEOF
	}
	r, w := utf8.DecodeRuneInString(ss.src[ss.pos:])
	if r == utf8.RuneError && w == 1 {
		return 0, 0, fmt.Errorf("invalid UTF-8 encoding at byte %d", ss.pos)
	}
	ss.pos += w
	return r, w, nil
}
