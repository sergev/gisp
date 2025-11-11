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

	hasLastToken bool
	lastToken    TokenType
	lastPos      Position
	bufferedTok  *Token
	parenDepth   int
	braceDepth   int
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

func (lx *lexer) skipWhitespace() (bool, error) {
	sawNewline := false
	for {
		r, _, state, err := lx.readRune()
		if err == io.EOF {
			return sawNewline, nil
		}
		if err != nil {
			return false, err
		}
		switch {
		case unicode.IsSpace(r):
			if r == '\n' {
				sawNewline = true
			}
			continue
		case r == '/':
			next, _, nextState, err := lx.readRune()
			if err == io.EOF {
				lx.unread(state)
				return sawNewline, nil
			}
			if err != nil {
				return false, err
			}
			if next == '/' {
				if err := lx.skipLine(); err != nil {
					if err == io.EOF {
						return sawNewline, nil
					}
					return false, err
				}
				sawNewline = true
				continue
			}
			if next == '*' {
				newlineInComment, err := lx.skipBlockComment()
				if err != nil {
					return false, err
				}
				if newlineInComment {
					sawNewline = true
				}
				continue
			}
			lx.unread(nextState)
			lx.unread(state)
			return sawNewline, nil
		default:
			lx.unread(state)
			return sawNewline, nil
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

func (lx *lexer) skipBlockComment() (bool, error) {
	sawNewline := false
	for {
		r, _, _, err := lx.readRune()
		if err != nil {
			if err == io.EOF {
				return sawNewline, newIncompleteError(fmt.Errorf("unterminated block comment"))
			}
			return sawNewline, err
		}
		if r == '\n' {
			sawNewline = true
		}
		if r == '*' {
			next, _, state, err := lx.readRune()
			if err == io.EOF {
				return sawNewline, newIncompleteError(fmt.Errorf("unterminated block comment"))
			}
			if err != nil {
				return sawNewline, err
			}
			if next == '/' {
				return sawNewline, nil
			}
			lx.unread(state)
		}
	}
}

func (lx *lexer) nextToken() (Token, error) {
	if lx.bufferedTok != nil {
		defer func() { lx.bufferedTok = nil }()
		return lx.emit(*lx.bufferedTok), nil
	}

	sawNewline, err := lx.skipWhitespace()
	if err != nil {
		return Token{}, err
	}
	if sawNewline && lx.shouldInsertSemicolon() && lx.canInsertSemicolon() {
		return lx.emit(Token{
			Type: tokenSemicolon,
			Pos:  lx.lastPos,
		}), nil
	}

	if lx.pos >= len(lx.src) {
		if lx.shouldInsertSemicolon() && lx.canInsertSemicolon() {
			return lx.emit(Token{
				Type: tokenSemicolon,
				Pos:  lx.lastPos,
			}), nil
		}
		return lx.emit(Token{
			Type: tokenEOF,
			Pos: Position{
				Offset: lx.pos,
				Line:   lx.line,
				Column: lx.column,
			},
		}), nil
	}

	start := lx.mark()
	r, _, _, err := lx.readRune()
	if err == io.EOF {
		if lx.shouldInsertSemicolon() && lx.canInsertSemicolon() {
			return lx.emit(Token{
				Type: tokenSemicolon,
				Pos:  lx.lastPos,
			}), nil
		}
		return lx.emit(Token{
			Type: tokenEOF,
			Pos: Position{
				Offset: start.pos,
				Line:   start.line,
				Column: start.column,
			},
		}), nil
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
		tok := makeIdentifierToken(lexeme, start)
		return lx.maybeEmitWithBuffer(tok)
	case unicode.IsDigit(r):
		lexeme, err := lx.scanNumber(r, start)
		if err != nil {
			return Token{}, err
		}
		tok := Token{
			Type:   tokenNumber,
			Lexeme: lexeme,
			Pos:    positionFromState(start),
		}
		return lx.maybeEmitWithBuffer(tok)
	case r == '"':
		value, err := lx.scanString()
		if err != nil {
			return Token{}, err
		}
		tok := Token{
			Type:  tokenString,
			Value: value,
			Pos:   positionFromState(start),
		}
		return lx.maybeEmitWithBuffer(tok)
	case r == '`':
		value, err := lx.scanSExpr(start)
		if err != nil {
			return Token{}, err
		}
		tok := Token{
			Type:  tokenSExpr,
			Value: value,
			Pos:   positionFromState(start),
		}
		return lx.maybeEmitWithBuffer(tok)
	case r == '#':
		if lx.match('[') {
			tok := simpleToken(tokenVectorStart, start)
			return lx.maybeEmitWithBuffer(tok)
		}
		illegal, err := illegalToken(start, fmt.Errorf("expected '[' after '#' for vector literal"))
		return lx.emit(illegal), err
	}

	var tok Token
	switch r {
	case '+':
		if lx.match('+') {
			tok = simpleToken(tokenPlusPlus, start)
		} else if lx.match('=') {
			tok = simpleToken(tokenPlusAssign, start)
		} else {
			tok = simpleToken(tokenPlus, start)
		}
	case '-':
		if lx.match('-') {
			tok = simpleToken(tokenMinusMinus, start)
		} else if lx.match('=') {
			tok = simpleToken(tokenMinusAssign, start)
		} else {
			tok = simpleToken(tokenMinus, start)
		}
	case '*':
		if lx.match('=') {
			tok = simpleToken(tokenStarAssign, start)
		} else {
			tok = simpleToken(tokenStar, start)
		}
	case '/':
		if lx.match('=') {
			tok = simpleToken(tokenSlashAssign, start)
		} else {
			tok = simpleToken(tokenSlash, start)
		}
	case '%':
		if lx.match('=') {
			tok = simpleToken(tokenPercentAssign, start)
		} else {
			tok = simpleToken(tokenPercent, start)
		}
	case '^':
		if lx.match('=') {
			tok = simpleToken(tokenCaretAssign, start)
		} else {
			tok = simpleToken(tokenCaret, start)
		}
	case '(':
		tok = simpleToken(tokenLParen, start)
	case ')':
		tok = simpleToken(tokenRParen, start)
	case '{':
		tok = simpleToken(tokenLBrace, start)
	case '}':
		tok = simpleToken(tokenRBrace, start)
	case '[':
		tok = simpleToken(tokenLBracket, start)
	case ']':
		tok = simpleToken(tokenRBracket, start)
	case ',':
		tok = simpleToken(tokenComma, start)
	case ';':
		tok = simpleToken(tokenSemicolon, start)
	case ':':
		tok = simpleToken(tokenColon, start)
	case '=':
		if lx.match('=') {
			tok = simpleToken(tokenEqualEqual, start)
		} else {
			tok = simpleToken(tokenAssign, start)
		}
	case '!':
		if lx.match('=') {
			tok = simpleToken(tokenBangEqual, start)
		} else {
			tok = simpleToken(tokenBang, start)
		}
	case '<':
		if lx.match('<') {
			if lx.match('=') {
				tok = simpleToken(tokenShiftLeftAssign, start)
			} else {
				tok = simpleToken(tokenShiftLeft, start)
			}
		} else if lx.match('=') {
			tok = simpleToken(tokenLessEqual, start)
		} else {
			tok = simpleToken(tokenLess, start)
		}
	case '>':
		if lx.match('>') {
			if lx.match('=') {
				tok = simpleToken(tokenShiftRightAssign, start)
			} else {
				tok = simpleToken(tokenShiftRight, start)
			}
		} else if lx.match('=') {
			tok = simpleToken(tokenGreaterEqual, start)
		} else {
			tok = simpleToken(tokenGreater, start)
		}
	case '&':
		if lx.match('^') {
			if lx.match('=') {
				tok = simpleToken(tokenAmpersandCaretAssign, start)
			} else {
				tok = simpleToken(tokenAmpersandCaret, start)
			}
		} else if lx.match('&') {
			tok = simpleToken(tokenAndAnd, start)
		} else if lx.match('=') {
			tok = simpleToken(tokenAmpersandAssign, start)
		} else {
			tok = simpleToken(tokenAmpersand, start)
		}
	case '|':
		if lx.match('|') {
			tok = simpleToken(tokenOrOr, start)
		} else if lx.match('=') {
			tok = simpleToken(tokenPipeAssign, start)
		} else {
			tok = simpleToken(tokenPipe, start)
		}
	default:
		illegal, err := illegalToken(start, fmt.Errorf("unexpected character %q", r))
		return lx.emit(illegal), err
	}

	return lx.maybeEmitWithBuffer(tok)
}

func (lx *lexer) maybeEmitWithBuffer(tok Token) (Token, error) {
	if tok.Type == tokenRBrace && lx.shouldInsertSemicolon() {
		copied := tok
		lx.bufferedTok = &copied
		return lx.emit(Token{
			Type: tokenSemicolon,
			Pos:  lx.lastPos,
		}), nil
	}
	return lx.emit(tok), nil
}

func (lx *lexer) emit(tok Token) Token {
	lx.adjustParenDepth(tok.Type)
	if tok.Type != tokenIllegal {
		lx.hasLastToken = true
		lx.lastToken = tok.Type
	} else {
		lx.hasLastToken = false
		lx.lastToken = tok.Type
	}
	lx.lastPos = tok.Pos
	return tok
}

func (lx *lexer) adjustParenDepth(tt TokenType) {
	switch tt {
	case tokenLParen, tokenLBracket:
		lx.parenDepth++
	case tokenVectorStart:
		lx.parenDepth++
	case tokenRParen, tokenRBracket:
		if lx.parenDepth > 0 {
			lx.parenDepth--
		}
	case tokenLBrace:
		lx.braceDepth++
	case tokenRBrace:
		if lx.braceDepth > 0 {
			lx.braceDepth--
		}
	}
}

func (lx *lexer) shouldInsertSemicolon() bool {
	if !lx.hasLastToken {
		return false
	}
	switch lx.lastToken {
	case tokenIdentifier,
		tokenNumber,
		tokenString,
		tokenSExpr,
		tokenTrue,
		tokenFalse,
		tokenNil,
		tokenReturn,
		tokenPlusPlus,
		tokenMinusMinus,
		tokenRParen,
		tokenRBracket,
		tokenRBrace:
		return true
	}
	return false
}

func (lx *lexer) canInsertSemicolon() bool {
	if lx.parenDepth == 0 {
		return true
	}
	if lx.braceDepth == 0 {
		return false
	}
	r, err := lx.peekNextRune()
	if err != nil {
		return true
	}
	switch r {
	case ')', ']':
		return false
	}
	return true
}

func (lx *lexer) peekNextRune() (rune, error) {
	state := lx.mark()
	r, _, _, err := lx.readRune()
	if err != nil {
		return 0, err
	}
	lx.unread(state)
	return r, nil
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
				return "", newIncompleteError(fmt.Errorf("unterminated exponent at line %d column %d", start.line, start.column))
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
			return "", newIncompleteError(fmt.Errorf("unterminated string literal"))
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
				return "", newIncompleteError(fmt.Errorf("unterminated escape sequence"))
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
	case "nil":
		return tokenNil, true
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
