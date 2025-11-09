package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sergev/gisp/lang"
)

// CompileProgram rewrites the parsed AST into Scheme s-expressions consumable by the evaluator.
func CompileProgram(prog *Program) ([]lang.Value, error) {
	if prog == nil {
		return nil, nil
	}
	b := &builder{}
	var results []lang.Value
	ctx := compileContext{}
	for _, decl := range prog.Decls {
		forms, err := compileDecl(b, decl, ctx)
		if err != nil {
			return nil, err
		}
		results = append(results, forms...)
	}
	return results, nil
}

type compileContext struct {
	returnSym string
}

func (c compileContext) withReturn(sym string) compileContext {
	c.returnSym = sym
	return c
}

func compileDecl(b *builder, decl Decl, ctx compileContext) ([]lang.Value, error) {
	switch d := decl.(type) {
	case *FuncDecl:
		form, err := compileFuncDecl(b, d, ctx)
		if err != nil {
			return nil, err
		}
		return []lang.Value{form}, nil
	case *VarDecl:
		form, err := compileTopLevelBinding(b, d, ctx)
		if err != nil {
			return nil, err
		}
		return []lang.Value{form}, nil
	case *ExprDecl:
		expr, err := compileExpr(b, d.Expr, ctx)
		if err != nil {
			return nil, err
		}
		return []lang.Value{expr}, nil
	default:
		return nil, fmt.Errorf("unsupported top-level declaration %T", decl)
	}
}

func compileTopLevelBinding(b *builder, decl *VarDecl, ctx compileContext) (lang.Value, error) {
	var value lang.Value
	var err error
	if decl.Init != nil {
		value, err = compileExpr(b, decl.Init, ctx)
		if err != nil {
			return lang.Value{}, err
		}
	} else {
		value = lang.EmptyList
	}
	return b.list(
		b.symbol("define"),
		b.symbol(decl.Name),
		value,
	), nil
}

func compileFuncDecl(b *builder, decl *FuncDecl, ctx compileContext) (lang.Value, error) {
	retSym := b.gensym("return")
	bodyCtx := ctx.withReturn(retSym)
	body, err := compileBlock(b, decl.Body, bodyCtx)
	if err != nil {
		return lang.Value{}, err
	}
	paramList := lang.EmptyList
	for i := len(decl.Params) - 1; i >= 0; i-- {
		paramList = lang.PairValue(b.symbol(decl.Params[i]), paramList)
	}
	callCC := b.list(
		b.symbol("call/cc"),
		b.list(
			b.symbol("lambda"),
			lang.List(b.symbol(retSym)),
			body,
		),
	)
	lambda := b.list(
		b.symbol("lambda"),
		paramList,
		callCC,
	)
	return b.list(
		b.symbol("define"),
		b.symbol(decl.Name),
		lambda,
	), nil
}

func compileBlock(b *builder, block *BlockStmt, ctx compileContext) (lang.Value, error) {
	if block == nil {
		return lang.EmptyList, nil
	}
	return compileStmts(b, block.Stmts, ctx)
}

func compileStmts(b *builder, stmts []Stmt, ctx compileContext) (lang.Value, error) {
	if len(stmts) == 0 {
		return lang.EmptyList, nil
	}
	first := stmts[0]
	rest := stmts[1:]
	restExpr, err := compileStmts(b, rest, ctx)
	if err != nil {
		return lang.Value{}, err
	}
	return compileStmtWithRest(b, first, restExpr, ctx)
}

func compileStmtWithRest(b *builder, stmt Stmt, rest lang.Value, ctx compileContext) (lang.Value, error) {
	switch s := stmt.(type) {
	case *VarDecl:
		initVal := lang.EmptyList
		if s.Init != nil {
			val, err := compileExpr(b, s.Init, ctx)
			if err != nil {
				return lang.Value{}, err
			}
			initVal = val
		}
		return b.let([]binding{{name: s.Name, value: initVal}}, rest), nil
	case *AssignStmt:
		expr, err := compileExpr(b, s.Expr, ctx)
		if err != nil {
			return lang.Value{}, err
		}
		setExpr := b.list(
			b.symbol("set!"),
			b.symbol(s.Name),
			expr,
		)
		return b.begin([]lang.Value{setExpr, rest}), nil
	case *ExprStmt:
		expr, err := compileExpr(b, s.Expr, ctx)
		if err != nil {
			return lang.Value{}, err
		}
		return b.begin([]lang.Value{expr, rest}), nil
	case *BlockStmt:
		body, err := compileBlock(b, s, ctx)
		if err != nil {
			return lang.Value{}, err
		}
		return b.begin([]lang.Value{body, rest}), nil
	case *IfStmt:
		cond, err := compileExpr(b, s.Cond, ctx)
		if err != nil {
			return lang.Value{}, err
		}
		thenExpr, err := compileBlock(b, s.Then, ctx)
		if err != nil {
			return lang.Value{}, err
		}
		var elseExpr lang.Value
		if s.Else != nil {
			elseExpr, err = compileBlock(b, s.Else, ctx)
			if err != nil {
				return lang.Value{}, err
			}
		} else {
			elseExpr = lang.EmptyList
		}
		ifExpr := b.list(
			b.symbol("if"),
			cond,
			thenExpr,
			elseExpr,
		)
		return b.begin([]lang.Value{ifExpr, rest}), nil
	case *WhileStmt:
		cond, err := compileExpr(b, s.Cond, ctx)
		if err != nil {
			return lang.Value{}, err
		}
		body, err := compileBlock(b, s.Body, ctx)
		if err != nil {
			return lang.Value{}, err
		}
		loopSym := b.gensym("loop")
		loopBody := b.list(
			b.symbol("if"),
			cond,
			b.begin([]lang.Value{
				body,
				b.list(b.symbol(loopSym)),
			}),
			lang.EmptyList,
		)
		loopLambda := b.list(
			b.symbol("lambda"),
			lang.EmptyList,
			loopBody,
		)
		loopSet := b.list(
			b.symbol("set!"),
			b.symbol(loopSym),
			loopLambda,
		)
		loopCall := b.list(b.symbol(loopSym))
		loopLetBody := b.begin([]lang.Value{loopSet, loopCall})
		loopLet := b.let([]binding{{name: loopSym, value: lang.EmptyList}}, loopLetBody)
		return b.begin([]lang.Value{loopLet, rest}), nil
	case *ReturnStmt:
		if ctx.returnSym == "" {
			return lang.Value{}, fmt.Errorf("return not allowed in this context")
		}
		var value lang.Value
		if s.Result != nil {
			val, err := compileExpr(b, s.Result, ctx)
			if err != nil {
				return lang.Value{}, err
			}
			value = val
		} else {
			value = lang.EmptyList
		}
		return b.list(
			b.symbol(ctx.returnSym),
			value,
		), nil
	default:
		return lang.Value{}, fmt.Errorf("unsupported statement %T", stmt)
	}
}

func compileExpr(b *builder, expr Expr, ctx compileContext) (lang.Value, error) {
	switch e := expr.(type) {
	case *IdentifierExpr:
		return b.symbol(e.Name), nil
	case *NumberExpr:
		return parseNumber(e.Value)
	case *StringExpr:
		return lang.StringValue(e.Value), nil
	case *BoolExpr:
		return lang.BoolValue(e.Value), nil
	case *ListExpr:
		elems := make([]lang.Value, 0, len(e.Elements)+1)
		elems = append(elems, b.symbol("list"))
		for _, el := range e.Elements {
			val, err := compileExpr(b, el, ctx)
			if err != nil {
				return lang.Value{}, err
			}
			elems = append(elems, val)
		}
		return lang.List(elems...), nil
	case *LambdaExpr:
		return compileLambdaExpr(b, e, ctx)
	case *CallExpr:
		callee, err := compileExpr(b, e.Callee, ctx)
		if err != nil {
			return lang.Value{}, err
		}
		args := make([]lang.Value, 0, len(e.Args)+1)
		args = append(args, callee)
		for _, arg := range e.Args {
			val, err := compileExpr(b, arg, ctx)
			if err != nil {
				return lang.Value{}, err
			}
			args = append(args, val)
		}
		return lang.List(args...), nil
	case *UnaryExpr:
		return compileUnaryExpr(b, e, ctx)
	case *BinaryExpr:
		return compileBinaryExpr(b, e, ctx)
	case *SExprLiteral:
		return e.Value, nil
	default:
		return lang.Value{}, fmt.Errorf("unsupported expression %T", expr)
	}
}

func compileLambdaExpr(b *builder, expr *LambdaExpr, ctx compileContext) (lang.Value, error) {
	retSym := b.gensym("return")
	bodyCtx := ctx.withReturn(retSym)
	body, err := compileBlock(b, expr.Body, bodyCtx)
	if err != nil {
		return lang.Value{}, err
	}
	paramList := lang.EmptyList
	for i := len(expr.Params) - 1; i >= 0; i-- {
		paramList = lang.PairValue(b.symbol(expr.Params[i]), paramList)
	}
	callCC := b.list(
		b.symbol("call/cc"),
		b.list(
			b.symbol("lambda"),
			lang.List(b.symbol(retSym)),
			body,
		),
	)
	return b.list(
		b.symbol("lambda"),
		paramList,
		callCC,
	), nil
}

func compileUnaryExpr(b *builder, expr *UnaryExpr, ctx compileContext) (lang.Value, error) {
	val, err := compileExpr(b, expr.Expr, ctx)
	if err != nil {
		return lang.Value{}, err
	}
	switch expr.Op {
	case tokenMinus:
		return lang.List(
			b.symbol("-"),
			val,
		), nil
	case tokenBang:
		return lang.List(
			b.symbol("not"),
			val,
		), nil
	default:
		return lang.Value{}, fmt.Errorf("unsupported unary operator %s", expr.Op)
	}
}

func compileBinaryExpr(b *builder, expr *BinaryExpr, ctx compileContext) (lang.Value, error) {
	left, err := compileExpr(b, expr.Left, ctx)
	if err != nil {
		return lang.Value{}, err
	}
	right, err := compileExpr(b, expr.Right, ctx)
	if err != nil {
		return lang.Value{}, err
	}
	switch expr.Op {
	case tokenPlus:
		return lang.List(b.symbol("+"), left, right), nil
	case tokenMinus:
		return lang.List(b.symbol("-"), left, right), nil
	case tokenStar:
		return lang.List(b.symbol("*"), left, right), nil
	case tokenSlash:
		return lang.List(b.symbol("/"), left, right), nil
	case tokenEqualEqual:
		return lang.List(b.symbol("="), left, right), nil
	case tokenBangEqual:
		return lang.List(
			b.symbol("not"),
			lang.List(b.symbol("="), left, right),
		), nil
	case tokenLess:
		return lang.List(b.symbol("<"), left, right), nil
	case tokenLessEqual:
		return lang.List(b.symbol("<="), left, right), nil
	case tokenGreater:
		return lang.List(b.symbol(">"), left, right), nil
	case tokenGreaterEqual:
		return lang.List(b.symbol(">="), left, right), nil
	case tokenAndAnd:
		return lang.List(b.symbol("and"), left, right), nil
	case tokenOrOr:
		return lang.List(b.symbol("or"), left, right), nil
	default:
		return lang.Value{}, fmt.Errorf("unsupported binary operator %s", expr.Op)
	}
}

func parseNumber(src string) (lang.Value, error) {
	if strings.ContainsAny(src, ".eE") {
		f, err := strconv.ParseFloat(src, 64)
		if err != nil {
			return lang.Value{}, fmt.Errorf("invalid float literal %q: %w", src, err)
		}
		return lang.RealValue(f), nil
	}
	i, err := strconv.ParseInt(src, 10, 64)
	if err != nil {
		return lang.Value{}, fmt.Errorf("invalid integer literal %q: %w", src, err)
	}
	return lang.IntValue(i), nil
}
