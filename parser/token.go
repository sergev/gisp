package parser

// TokenType enumerates lexical categories recognised by the Gisp lexer.
type TokenType int

const (
	tokenEOF TokenType = iota
	tokenIllegal

	tokenIdentifier
	tokenNumber
	tokenString
	tokenSExpr

	// Keywords
	tokenFunc
	tokenVar
	tokenConst
	tokenIf
	tokenElse
	tokenWhile
	tokenSwitch
	tokenCase
	tokenDefault
	tokenReturn
	tokenTrue
	tokenFalse
	tokenNil

	// Operators and punctuation
	tokenAssign               // =
	tokenPlusAssign           // +=
	tokenMinusAssign          // -=
	tokenStarAssign           // *=
	tokenSlashAssign          // /=
	tokenPercentAssign        // %=
	tokenShiftLeftAssign      // <<=
	tokenShiftRightAssign     // >>=
	tokenAmpersandAssign      // &=
	tokenPipeAssign           // |=
	tokenCaretAssign          // ^=
	tokenAmpersandCaretAssign // &^=
	tokenEqualEqual           // ==
	tokenBangEqual            // !=
	tokenPlus                 // +
	tokenMinus                // -
	tokenPlusPlus             // ++
	tokenMinusMinus           // --
	tokenStar                 // *
	tokenSlash                // /
	tokenPercent              // %
	tokenCaret                // ^
	tokenAmpersand            // &
	tokenPipe                 // |
	tokenShiftLeft            // <<
	tokenShiftRight           // >>
	tokenAmpersandCaret       // &^
	tokenLess                 // <
	tokenLessEqual            // <=
	tokenGreater              // >
	tokenGreaterEqual         // >=
	tokenBang                 // !
	tokenAndAnd               // &&
	tokenOrOr                 // ||

	tokenComma       // ,
	tokenSemicolon   // ;
	tokenColon       // :
	tokenLParen      // (
	tokenRParen      // )
	tokenVectorStart // #[
	tokenLBrace      // {
	tokenRBrace      // }
	tokenLBracket    // [
	tokenRBracket    // ]
)

func (tt TokenType) String() string {
	switch tt {
	case tokenEOF:
		return "EOF"
	case tokenIllegal:
		return "illegal"
	case tokenIdentifier:
		return "identifier"
	case tokenNumber:
		return "number"
	case tokenString:
		return "string"
	case tokenSExpr:
		return "sexpr"
	case tokenFunc:
		return "func"
	case tokenVar:
		return "var"
	case tokenConst:
		return "const"
	case tokenIf:
		return "if"
	case tokenElse:
		return "else"
	case tokenWhile:
		return "while"
	case tokenSwitch:
		return "switch"
	case tokenCase:
		return "case"
	case tokenDefault:
		return "default"
	case tokenReturn:
		return "return"
	case tokenTrue:
		return "true"
	case tokenFalse:
		return "false"
	case tokenNil:
		return "nil"
	case tokenAssign:
		return "="
	case tokenPlusAssign:
		return "+="
	case tokenMinusAssign:
		return "-="
	case tokenStarAssign:
		return "*="
	case tokenSlashAssign:
		return "/="
	case tokenPercentAssign:
		return "%="
	case tokenShiftLeftAssign:
		return "<<="
	case tokenShiftRightAssign:
		return ">>="
	case tokenAmpersandAssign:
		return "&="
	case tokenPipeAssign:
		return "|="
	case tokenCaretAssign:
		return "^="
	case tokenAmpersandCaretAssign:
		return "&^="
	case tokenEqualEqual:
		return "=="
	case tokenBangEqual:
		return "!="
	case tokenPlus:
		return "+"
	case tokenMinus:
		return "-"
	case tokenPlusPlus:
		return "++"
	case tokenMinusMinus:
		return "--"
	case tokenStar:
		return "*"
	case tokenSlash:
		return "/"
	case tokenPercent:
		return "%"
	case tokenCaret:
		return "^"
	case tokenAmpersand:
		return "&"
	case tokenPipe:
		return "|"
	case tokenShiftLeft:
		return "<<"
	case tokenShiftRight:
		return ">>"
	case tokenAmpersandCaret:
		return "&^"
	case tokenLess:
		return "<"
	case tokenLessEqual:
		return "<="
	case tokenGreater:
		return ">"
	case tokenGreaterEqual:
		return ">="
	case tokenBang:
		return "!"
	case tokenAndAnd:
		return "&&"
	case tokenOrOr:
		return "||"
	case tokenComma:
		return ","
	case tokenSemicolon:
		return ";"
	case tokenColon:
		return ":"
	case tokenLParen:
		return "("
	case tokenRParen:
		return ")"
	case tokenVectorStart:
		return "#["
	case tokenLBrace:
		return "{"
	case tokenRBrace:
		return "}"
	case tokenLBracket:
		return "["
	case tokenRBracket:
		return "]"
	default:
		return "unknown"
	}
}

// Token is a single lexical unit produced by the lexer.
type Token struct {
	Type   TokenType
	Lexeme string      // raw lexeme when useful (identifiers, numbers)
	Value  interface{} // decoded literal value for strings and s-expr literals
	Pos    Position
}
