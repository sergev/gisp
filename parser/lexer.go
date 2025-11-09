package parser

import (
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sergev/gisp/lang"
	"github.com/sergev/gisp/sexpr"
)

type lexer struct {
	src    string
	pos    int
	line   int
	column int
}

func newLexer(src string) *lexer {
	return &lexer{
		src:    src,
		line:   1,
		column: 1,
	}
}

type runeState struct {
	pos    int
	line   int
	column int
}

func (lx *lexer) mark() runeState {
	return runeState{
		pos:    lx.pos,
		line:   lx.line,
		column: lx.column,
	}
}

func (lx *lexer) restore(state runeState) {
	lx.pos = state.pos
	lx.line = state.line
	lx.column = state.column
}

func (lx *lexer) readRune() (rune, int, runeState, error) {
	if lx.pos >= len(lx.src) {
		return 0, 0, lx.mark(), io.EOF
	}
	state := lx.mark()
	r, w := utf8.DecodeRuneInString(lx.src[lx.pos:])
	if r == utf8.RuneError && w == 1 {
		return 0, 0, state, fmt.Errorf("invalid UTF-8 encoding at byte %d", lx.pos)
	}
	lx.pos += w
	if r == '\n' {
		lx.line++
		lx.column = 1
	} else {
		lx.column++
	}
	return r, w, state, nil
}

func (lx *lexer) unread(state runeState) {
	lx.restore(state)
}

func (lx *lexer) skipWhitespace() error {
	for {
		r, _, state, err := lx.readRune()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch {
		case unicode.IsSpace(r):
			continue
		case r == '/':
			next, _, nextState, err := lx.readRune()
			if err == io.EOF {
				lx.unread(state)
				return nil
			}
			if err != nil {
				return err
			}
			if next == '/' {
				if err := lx.skipLine(); err != nil {
					if err == io.EOF {
						return nil
					}
					return err
				}
				continue
			}
			if next == '*' {
				if err := lx.skipBlockComment(); err != nil {
					return err
				}
				continue
			}
			lx.unread(nextState)
			lx.unread(state)
			return nil
		default:
			lx.unread(state)
			return nil
		}
	}
}

func (lx *lexer) skipLine() error {
	for {
		r, _, _, err := lx.readRune()
		if err != nil {
			return err
		}
		if r == '\n' {
			return nil
		}
	}
}

func (lx *lexer) skipBlockComment() error {
	for {
		r, _, _, err := lx.readRune()
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("unterminated block comment")
			}
			return err
		}
		if r == '*' {
			next, _, state, err := lx.readRune()
			if err == io.EOF {
				return fmt.Errorf("unterminated block comment")
			}
			if err != nil {
				return err
			}
			if next == '/' {
				return nil
			}
			lx.unread(state)
		}
	}
}

func (lx *lexer) nextToken() (Token, error) {
	if err := lx.skipWhitespace(); err != nil {
		return Token{}, err
	}
	if lx.pos >= len(lx.src) {
		return Token{
			Type: tokenEOF,
			Pos: Position{
				Offset: lx.pos,
				Line:   lx.line,
				Column: lx.column,
			},
		}, nil
	}

	start := lx.mark()
	r, _, _, err := lx.readRune()
	if err == io.EOF {
		return Token{
			Type: tokenEOF,
			Pos: Position{
				Offset: start.pos,
				Line:   start.line,
				Column: start.column,
			},
		}, nil
	}
	if err != nil {
		return Token{}, err
	}

	switch {
	case isIdentifierStart(r):
		lexeme, err := lx.scanIdentifier(r)
		if err != nil {
			return Token{}, err
		}
		return makeIdentifierToken(lexeme, start), nil
	case unicode.IsDigit(r):
		lexeme, err := lx.scanNumber(r, start)
		if err != nil {
			return Token{}, err
		}
		return Token{
			Type:   tokenNumber,
			Lexeme: lexeme,
			Pos:    positionFromState(start),
		}, nil
	case r == '"':
		value, err := lx.scanString()
		if err != nil {
			return Token{}, err
		}
		return Token{
			Type:  tokenString,
			Value: value,
			Pos:   positionFromState(start),
		}, nil
	case r == '`':
		value, err := lx.scanSExpr(start)
		if err != nil {
			return Token{}, err
		}
		return Token{
			Type:  tokenSExpr,
			Value: value,
			Pos:   positionFromState(start),
		}, nil
	}

	switch r {
	case '+':
		return simpleToken(tokenPlus, start), nil
	case '-':
		return simpleToken(tokenMinus, start), nil
	case '*':
		return simpleToken(tokenStar, start), nil
	case '/':
		return simpleToken(tokenSlash, start), nil
	case '%':
		return Token{
			Type: tokenIllegal,
			Pos:  positionFromState(start),
		}, fmt.Errorf("unsupported operator %%")
	case '(':
		return simpleToken(tokenLParen, start), nil
	case ')':
		return simpleToken(tokenRParen, start), nil
	case '{':
		return simpleToken(tokenLBrace, start), nil
	case '}':
		return simpleToken(tokenRBrace, start), nil
	case '[':
		return simpleToken(tokenLBracket, start), nil
	case ']':
		return simpleToken(tokenRBracket, start), nil
	case ',':
		return simpleToken(tokenComma, start), nil
	case ';':
		return simpleToken(tokenSemicolon, start), nil
	case ':':
		return simpleToken(tokenColon, start), nil
	case '=':
		if lx.match('=') {
			return simpleToken(tokenEqualEqual, start), nil
		}
		return simpleToken(tokenAssign, start), nil
	case '!':
		if lx.match('=') {
			return simpleToken(tokenBangEqual, start), nil
		}
		return simpleToken(tokenBang, start), nil
	case '<':
		if lx.match('=') {
			return simpleToken(tokenLessEqual, start), nil
		}
		return simpleToken(tokenLess, start), nil
	case '>':
		if lx.match('=') {
			return simpleToken(tokenGreaterEqual, start), nil
		}
		return simpleToken(tokenGreater, start), nil
	case '&':
		if lx.match('&') {
			return simpleToken(tokenAndAnd, start), nil
		}
		return illegalToken(start, fmt.Errorf("unexpected '&'"))
	case '|':
		if lx.match('|') {
			return simpleToken(tokenOrOr, start), nil
		}
		return illegalToken(start, fmt.Errorf("unexpected '|'"))
	}

	return illegalToken(start, fmt.Errorf("unexpected character %q", r))
}

func (lx *lexer) match(expected rune) bool {
	state := lx.mark()
	r, _, _, err := lx.readRune()
	if err != nil {
		return false
	}
	if r != expected {
		lx.unread(state)
		return false
	}
	return true
}

func isIdentifierStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentifierPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func (lx *lexer) scanIdentifier(initial rune) (string, error) {
	var builder strings.Builder
	builder.WriteRune(initial)
	for {
		r, _, state, err := lx.readRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if !isIdentifierPart(r) {
			lx.unread(state)
			break
		}
		builder.WriteRune(r)
	}
	return builder.String(), nil
}

func (lx *lexer) scanNumber(initial rune, start runeState) (string, error) {
	var builder strings.Builder
	builder.WriteRune(initial)
	seenDot := false
	seenExponent := false

	for {
		r, _, state, err := lx.readRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if unicode.IsDigit(r) {
			builder.WriteRune(r)
			continue
		}
		if r == '.' && !seenDot && !seenExponent {
			seenDot = true
			builder.WriteRune(r)
			continue
		}
		if (r == 'e' || r == 'E') && !seenExponent {
			seenExponent = true
			builder.WriteRune(r)
			next, _, nextState, err := lx.readRune()
			if err == io.EOF {
				return "", fmt.Errorf("unterminated exponent at line %d column %d", start.line, start.column)
			}
			if err != nil {
				return "", err
			}
			if next == '+' || next == '-' {
				builder.WriteRune(next)
			} else {
				lx.unread(nextState)
			}
			continue
		}
		lx.unread(state)
		break
	}

	return builder.String(), nil
}

func (lx *lexer) scanString() (string, error) {
	var builder strings.Builder
	for {
		r, _, _, err := lx.readRune()
		if err == io.EOF {
			return "", fmt.Errorf("unterminated string literal")
		}
		if err != nil {
			return "", err
		}
		if r == '"' {
			break
		}
		if r == '\\' {
			esc, _, _, err := lx.readRune()
			if err == io.EOF {
				return "", fmt.Errorf("unterminated escape sequence")
			}
			if err != nil {
				return "", err
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
		if r == '\n' {
			return "", fmt.Errorf("newline in string literal")
		}
		builder.WriteRune(r)
	}
	return builder.String(), nil
}

func (lx *lexer) scanSExpr(start runeState) (lang.Value, error) {
	value, end, err := sexpr.ParseLiteral(lx.src, lx.pos)
	if err != nil {
		return lang.Value{}, fmt.Errorf("invalid s-expression literal at line %d column %d: %w", start.line, start.column, err)
	}
	lx.advanceTo(end)
	return value, nil
}

func (lx *lexer) advanceTo(end int) {
	if end < lx.pos {
		return
	}
	segment := lx.src[lx.pos:end]
	for _, r := range segment {
		if r == '\n' {
			lx.line++
			lx.column = 1
		} else {
			lx.column++
		}
	}
	lx.pos = end
}

func makeIdentifierToken(lexeme string, start runeState) Token {
	if keywordType, ok := keywordToken(lexeme); ok {
		return Token{
			Type: keywordType,
			Pos:  positionFromState(start),
		}
	}
	return Token{
		Type:   tokenIdentifier,
		Lexeme: lexeme,
		Pos:    positionFromState(start),
	}
}

func keywordToken(lexeme string) (TokenType, bool) {
	switch lexeme {
	case "func":
		return tokenFunc, true
	case "var":
		return tokenVar, true
	case "const":
		return tokenConst, true
	case "if":
		return tokenIf, true
	case "else":
		return tokenElse, true
	case "while":
		return tokenWhile, true
	case "switch":
		return tokenSwitch, true
	case "case":
		return tokenCase, true
	case "default":
		return tokenDefault, true
	case "return":
		return tokenReturn, true
	case "true":
		return tokenTrue, true
	case "false":
		return tokenFalse, true
	default:
		return tokenIllegal, false
	}
}

func simpleToken(tt TokenType, start runeState) Token {
	return Token{
		Type: tt,
		Pos:  positionFromState(start),
	}
}

func illegalToken(start runeState, err error) (Token, error) {
	return Token{
		Type: tokenIllegal,
		Pos:  positionFromState(start),
	}, err
}

func positionFromState(state runeState) Position {
	return Position{
		Offset: state.pos,
		Line:   state.line,
		Column: state.column,
	}
}
