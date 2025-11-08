package parser

import (
	"fmt"

	"github.com/sergev/gisp/lang"
)

// Parse translates source text into a Program AST.
func Parse(src string) (*Program, error) {
	p := &parser{
		lx: newLexer(src),
	}
	if err := p.advance(); err != nil {
		return nil, err
	}
	return p.parseProgram()
}

type parser struct {
	lx      *lexer
	curr    Token
	peekTok Token
	hasPeek bool
}

func (p *parser) advance() error {
	if p.hasPeek {
		p.curr = p.peekTok
		p.hasPeek = false
		return nil
	}
	tok, err := p.lx.nextToken()
	if err != nil {
		return err
	}
	p.curr = tok
	return nil
}

func (p *parser) peek() (Token, error) {
	if !p.hasPeek {
		tok, err := p.lx.nextToken()
		if err != nil {
			return Token{}, err
		}
		p.peekTok = tok
		p.hasPeek = true
	}
	return p.peekTok, nil
}

func (p *parser) expect(tt TokenType) (Token, error) {
	if p.curr.Type != tt {
		return Token{}, p.errorf(p.curr.Pos, "expected %s, found %s", tt, p.curr.Type)
	}
	tok := p.curr
	if err := p.advance(); err != nil {
		return Token{}, err
	}
	return tok, nil
}

func (p *parser) parseProgram() (*Program, error) {
	var decls []Decl
	for p.curr.Type != tokenEOF {
		decl, err := p.parseTopLevelDecl()
		if err != nil {
			return nil, err
		}
		decls = append(decls, decl)
	}
	return &Program{Decls: decls}, nil
}

func (p *parser) parseTopLevelDecl() (Decl, error) {
	switch p.curr.Type {
	case tokenFunc:
		return p.parseFuncDecl()
	case tokenVar:
		return p.parseVarDecl(true)
	case tokenConst:
		return p.parseConstDecl(true)
	default:
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokenSemicolon); err != nil {
			return nil, err
		}
		return &ExprDecl{
			Expr: expr,
			Posn: expr.Pos(),
		}, nil
	}
}

func (p *parser) parseFuncDecl() (Decl, error) {
	funcTok, err := p.expect(tokenFunc)
	if err != nil {
		return nil, err
	}
	nameTok, err := p.expect(tokenIdentifier)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(tokenLParen); err != nil {
		return nil, err
	}
	params, err := p.parseParamNames()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &FuncDecl{
		Name:   nameTok.Lexeme,
		Params: params,
		Body:   body,
		Posn:   posFromToken(funcTok),
	}, nil
}

func (p *parser) parseVarDecl(isTopLevel bool) (Decl, error) {
	varTok, err := p.expect(tokenVar)
	if err != nil {
		return nil, err
	}
	return p.finishBindingDecl(varTok, false, isTopLevel)
}

func (p *parser) parseConstDecl(isTopLevel bool) (Decl, error) {
	constTok, err := p.expect(tokenConst)
	if err != nil {
		return nil, err
	}
	return p.finishBindingDecl(constTok, true, isTopLevel)
}

func (p *parser) finishBindingDecl(start Token, isConst bool, expectSemi bool) (Decl, error) {
	nameTok, err := p.expect(tokenIdentifier)
	if err != nil {
		return nil, err
	}
	var init Expr
	if p.curr.Type == tokenAssign {
		if _, err := p.expect(tokenAssign); err != nil {
			return nil, err
		}
		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		init = value
	}
	if expectSemi {
		if _, err := p.expect(tokenSemicolon); err != nil {
			return nil, err
		}
	}
	return &VarDecl{
		Name:  nameTok.Lexeme,
		Init:  init,
		Const: isConst,
		Posn:  posFromToken(start),
	}, nil
}

func (p *parser) parseBlock() (*BlockStmt, error) {
	braceTok, err := p.expect(tokenLBrace)
	if err != nil {
		return nil, err
	}
	var stmts []Stmt
	for p.curr.Type != tokenRBrace && p.curr.Type != tokenEOF {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
	}
	if p.curr.Type != tokenRBrace {
		return nil, p.errorf(p.curr.Pos, "expected } to close block")
	}
	if _, err := p.expect(tokenRBrace); err != nil {
		return nil, err
	}
	return &BlockStmt{
		Stmts: stmts,
		Posn:  posFromToken(braceTok),
	}, nil
}

func (p *parser) parseStatement() (Stmt, error) {
	switch p.curr.Type {
	case tokenVar:
		decl, err := p.parseVarDecl(false)
		if err != nil {
			return nil, err
		}
		return decl.(Stmt), nil
	case tokenConst:
		decl, err := p.parseConstDecl(false)
		if err != nil {
			return nil, err
		}
		return decl.(Stmt), nil
	case tokenIf:
		return p.parseIfStmt()
	case tokenWhile:
		return p.parseWhileStmt()
	case tokenReturn:
		return p.parseReturnStmt()
	case tokenLBrace:
		block, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		return block, nil
	case tokenIdentifier:
		if stmt, ok, err := p.tryParseAssignmentStmt(); err != nil {
			return nil, err
		} else if ok {
			return stmt, nil
		}
		fallthrough
	default:
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokenSemicolon); err != nil {
			return nil, err
		}
		return &ExprStmt{
			Expr: expr,
			Posn: expr.Pos(),
		}, nil
	}
}

func (p *parser) tryParseAssignmentStmt() (Stmt, bool, error) {
	nameTok := p.curr
	peek, err := p.peek()
	if err != nil {
		return nil, false, err
	}
	if peek.Type != tokenAssign {
		return nil, false, nil
	}
	if _, err := p.expect(tokenIdentifier); err != nil {
		return nil, false, err
	}
	if _, err := p.expect(tokenAssign); err != nil {
		return nil, false, err
	}
	value, err := p.parseExpression()
	if err != nil {
		return nil, false, err
	}
	if _, err := p.expect(tokenSemicolon); err != nil {
		return nil, false, err
	}
	return &AssignStmt{
		Name: nameTok.Lexeme,
		Expr: value,
		Posn: posFromToken(nameTok),
	}, true, nil
}

func (p *parser) parseIfStmt() (Stmt, error) {
	ifTok, err := p.expect(tokenIf)
	if err != nil {
		return nil, err
	}
	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	thenBlock, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	var elseBlock *BlockStmt
	if p.curr.Type == tokenElse {
		if _, err := p.expect(tokenElse); err != nil {
			return nil, err
		}
		block, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		elseBlock = block
	}
	return &IfStmt{
		Cond: cond,
		Then: thenBlock,
		Else: elseBlock,
		Posn: posFromToken(ifTok),
	}, nil
}

func (p *parser) parseWhileStmt() (Stmt, error) {
	whTok, err := p.expect(tokenWhile)
	if err != nil {
		return nil, err
	}
	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &WhileStmt{
		Cond: cond,
		Body: body,
		Posn: posFromToken(whTok),
	}, nil
}

func (p *parser) parseReturnStmt() (Stmt, error) {
	retTok, err := p.expect(tokenReturn)
	if err != nil {
		return nil, err
	}
	var result Expr
	if p.curr.Type != tokenSemicolon {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		result = expr
	}
	if _, err := p.expect(tokenSemicolon); err != nil {
		return nil, err
	}
	return &ReturnStmt{
		Result: result,
		Posn:   posFromToken(retTok),
	}, nil
}

func (p *parser) parseExpression() (Expr, error) {
	return p.parseLogicalOr()
}

func (p *parser) parseLogicalOr() (Expr, error) {
	left, err := p.parseLogicalAnd()
	if err != nil {
		return nil, err
	}
	for p.curr.Type == tokenOrOr {
		opTok, _ := p.expect(tokenOrOr)
		right, err := p.parseLogicalAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{
			Op:    opTok.Type,
			Left:  left,
			Right: right,
			Posn:  posFromToken(opTok),
		}
	}
	return left, nil
}

func (p *parser) parseLogicalAnd() (Expr, error) {
	left, err := p.parseEquality()
	if err != nil {
		return nil, err
	}
	for p.curr.Type == tokenAndAnd {
		opTok, _ := p.expect(tokenAndAnd)
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{
			Op:    opTok.Type,
			Left:  left,
			Right: right,
			Posn:  posFromToken(opTok),
		}
	}
	return left, nil
}

func (p *parser) parseEquality() (Expr, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for p.curr.Type == tokenEqualEqual || p.curr.Type == tokenBangEqual {
		opTok := p.curr
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{
			Op:    opTok.Type,
			Left:  left,
			Right: right,
			Posn:  posFromToken(opTok),
		}
	}
	return left, nil
}

func (p *parser) parseComparison() (Expr, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}
	for p.curr.Type == tokenLess || p.curr.Type == tokenLessEqual ||
		p.curr.Type == tokenGreater || p.curr.Type == tokenGreaterEqual {
		opTok := p.curr
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{
			Op:    opTok.Type,
			Left:  left,
			Right: right,
			Posn:  posFromToken(opTok),
		}
	}
	return left, nil
}

func (p *parser) parseTerm() (Expr, error) {
	left, err := p.parseFactor()
	if err != nil {
		return nil, err
	}
	for p.curr.Type == tokenPlus || p.curr.Type == tokenMinus {
		opTok := p.curr
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{
			Op:    opTok.Type,
			Left:  left,
			Right: right,
			Posn:  posFromToken(opTok),
		}
	}
	return left, nil
}

func (p *parser) parseFactor() (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.curr.Type == tokenStar || p.curr.Type == tokenSlash {
		opTok := p.curr
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{
			Op:    opTok.Type,
			Left:  left,
			Right: right,
			Posn:  posFromToken(opTok),
		}
	}
	return left, nil
}

func (p *parser) parseUnary() (Expr, error) {
	if p.curr.Type == tokenBang || p.curr.Type == tokenMinus {
		opTok := p.curr
		if err := p.advance(); err != nil {
			return nil, err
		}
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{
			Op:   opTok.Type,
			Expr: expr,
			Posn: posFromToken(opTok),
		}, nil
	}
	return p.parsePostfix()
}

func (p *parser) parsePostfix() (Expr, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		switch p.curr.Type {
		case tokenLParen:
			callTok, _ := p.expect(tokenLParen)
			args, err := p.parseArgumentList()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(tokenRParen); err != nil {
				return nil, err
			}
			expr = &CallExpr{
				Callee: expr,
				Args:   args,
				Posn:   posFromToken(callTok),
			}
		default:
			return expr, nil
		}
	}
}

func (p *parser) parseArgumentList() ([]Expr, error) {
	var args []Expr
	if p.curr.Type == tokenRParen {
		return args, nil
	}
	for {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, expr)
		if p.curr.Type != tokenComma {
			break
		}
		if _, err := p.expect(tokenComma); err != nil {
			return nil, err
		}
	}
	return args, nil
}

func (p *parser) parsePrimary() (Expr, error) {
	switch p.curr.Type {
	case tokenIdentifier:
		tok, err := p.expect(tokenIdentifier)
		if err != nil {
			return nil, err
		}
		return &IdentifierExpr{
			Name: tok.Lexeme,
			Posn: posFromToken(tok),
		}, nil
	case tokenNumber:
		tok, err := p.expect(tokenNumber)
		if err != nil {
			return nil, err
		}
		return &NumberExpr{
			Value: tok.Lexeme,
			Posn:  posFromToken(tok),
		}, nil
	case tokenString:
		tok, err := p.expect(tokenString)
		if err != nil {
			return nil, err
		}
		strVal, _ := tok.Value.(string)
		return &StringExpr{
			Value: strVal,
			Posn:  posFromToken(tok),
		}, nil
	case tokenTrue, tokenFalse:
		tok := p.curr
		if err := p.advance(); err != nil {
			return nil, err
		}
		return &BoolExpr{
			Value: tok.Type == tokenTrue,
			Posn:  posFromToken(tok),
		}, nil
	case tokenSExpr:
		tok, err := p.expect(tokenSExpr)
		if err != nil {
			return nil, err
		}
		val, _ := tok.Value.(lang.Value)
		return &SExprLiteral{
			Value: val,
			Posn:  posFromToken(tok),
		}, nil
	case tokenFunc:
		return p.parseLambdaExpr()
	case tokenLParen:
		if _, err := p.expect(tokenLParen); err != nil {
			return nil, err
		}
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokenRParen); err != nil {
			return nil, err
		}
		return expr, nil
	case tokenLBracket:
		return p.parseListLiteral()
	default:
		return nil, p.errorf(p.curr.Pos, "unexpected token %s in expression", p.curr.Type)
	}
}

func (p *parser) parseLambdaExpr() (Expr, error) {
	funcTok, err := p.expect(tokenFunc)
	if err != nil {
		return nil, err
	}
	if p.curr.Type != tokenLParen {
		return nil, p.errorf(p.curr.Pos, "expected ( after func")
	}
	if _, err := p.expect(tokenLParen); err != nil {
		return nil, err
	}
	params, err := p.parseParamNames()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &LambdaExpr{
		Params: params,
		Body:   body,
		Posn:   posFromToken(funcTok),
	}, nil
}

func (p *parser) parseListLiteral() (Expr, error) {
	startTok, err := p.expect(tokenLBracket)
	if err != nil {
		return nil, err
	}
	var elems []Expr
	for p.curr.Type != tokenRBracket {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		elems = append(elems, expr)
		if p.curr.Type == tokenComma {
			if _, err := p.expect(tokenComma); err != nil {
				return nil, err
			}
			continue
		}
		break
	}
	if _, err := p.expect(tokenRBracket); err != nil {
		return nil, err
	}
	return &ListExpr{
		Elements: elems,
		Posn:     posFromToken(startTok),
	}, nil
}

func (p *parser) parseParamNames() ([]string, error) {
	var params []string
	if p.curr.Type == tokenRParen {
		return params, nil
	}
	for {
		tok, err := p.expect(tokenIdentifier)
		if err != nil {
			return nil, err
		}
		params = append(params, tok.Lexeme)
		if p.curr.Type != tokenComma {
			break
		}
		if _, err := p.expect(tokenComma); err != nil {
			return nil, err
		}
	}
	return params, nil
}

func (p *parser) errorf(pos Position, format string, args ...interface{}) error {
	return fmt.Errorf("%s:%d:%d: %s", "input", pos.Line, pos.Column, fmt.Sprintf(format, args...))
}

func posFromToken(tok Token) Position {
	return tok.Pos
}
