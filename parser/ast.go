package parser

import "github.com/sergev/gisp/lang"

// Position tracks a source location within a Gisp source file.
type Position struct {
	Offset int // zero-based byte offset
	Line   int // one-based line number
	Column int // one-based column number (rune count)
}

// Node represents any AST node with a source position.
type Node interface {
	Pos() Position
}

// Program is the root of a parsed Gisp file.
type Program struct {
	Decls []Decl
}

// Decl represents a top-level declaration.
type Decl interface {
	Node
	declNode()
}

// Stmt represents a statement inside a block.
type Stmt interface {
	Node
	stmtNode()
}

// Expr represents an expression.
type Expr interface {
	Node
	exprNode()
}

// IdentifierExpr refers to a variable or function name.
type IdentifierExpr struct {
	Name string
	Posn Position
}

func (e *IdentifierExpr) Pos() Position { return e.Posn }
func (*IdentifierExpr) exprNode()       {}

// NumberExpr represents an integer or floating literal in source form.
type NumberExpr struct {
	Value string
	Posn  Position
}

func (e *NumberExpr) Pos() Position { return e.Posn }
func (*NumberExpr) exprNode()       {}

// StringExpr is a double-quoted string literal.
type StringExpr struct {
	Value string
	Posn  Position
}

func (e *StringExpr) Pos() Position { return e.Posn }
func (*StringExpr) exprNode()       {}

// BoolExpr is a boolean literal.
type BoolExpr struct {
	Value bool
	Posn  Position
}

func (e *BoolExpr) Pos() Position { return e.Posn }
func (*BoolExpr) exprNode()       {}

// ListExpr is a literal list [a, b, ...].
type ListExpr struct {
	Elements []Expr
	Posn     Position
}

func (e *ListExpr) Pos() Position { return e.Posn }
func (*ListExpr) exprNode()       {}

// LambdaExpr is an anonymous function.
type LambdaExpr struct {
	Params []string
	Body   *BlockStmt
	Posn   Position
}

func (e *LambdaExpr) Pos() Position { return e.Posn }
func (*LambdaExpr) exprNode()       {}

// CallExpr invokes an expression with arguments.
type CallExpr struct {
	Callee Expr
	Args   []Expr
	Posn   Position
}

func (e *CallExpr) Pos() Position { return e.Posn }
func (*CallExpr) exprNode()       {}

// UnaryExpr represents prefix operator application.
type UnaryExpr struct {
	Op   TokenType
	Expr Expr
	Posn Position
}

func (e *UnaryExpr) Pos() Position { return e.Posn }
func (*UnaryExpr) exprNode()       {}

// BinaryExpr represents infix operator application.
type BinaryExpr struct {
	Op          TokenType
	Left, Right Expr
	Posn        Position
}

func (e *BinaryExpr) Pos() Position { return e.Posn }
func (*BinaryExpr) exprNode()       {}

// SExprLiteral embeds a raw Scheme expression parsed via the existing reader.
type SExprLiteral struct {
	Value lang.Value
	Posn  Position
}

func (e *SExprLiteral) Pos() Position { return e.Posn }
func (*SExprLiteral) exprNode()       {}

// FuncDecl introduces a top-level function with optional name export.
type FuncDecl struct {
	Name   string
	Params []string
	Body   *BlockStmt
	Posn   Position
}

func (d *FuncDecl) Pos() Position { return d.Posn }
func (*FuncDecl) declNode()       {}

// VarDecl declares a mutable binding, optionally initialised.
type VarDecl struct {
	Name  string
	Init  Expr // may be nil
	Posn  Position
	Const bool
}

func (d *VarDecl) Pos() Position { return d.Posn }
func (*VarDecl) declNode()       {}
func (*VarDecl) stmtNode()       {}

// BlockStmt is a braced block.
type BlockStmt struct {
	Stmts []Stmt
	Posn  Position
}

func (s *BlockStmt) Pos() Position { return s.Posn }
func (*BlockStmt) stmtNode()       {}

// ExprStmt evaluates an expression for side-effects.
type ExprStmt struct {
	Expr Expr
	Posn Position
}

func (s *ExprStmt) Pos() Position { return s.Posn }
func (*ExprStmt) stmtNode()       {}

// ExprDecl represents a top-level expression evaluated for side-effects.
type ExprDecl struct {
	Expr Expr
	Posn Position
}

func (d *ExprDecl) Pos() Position { return d.Posn }
func (*ExprDecl) declNode()       {}

// AssignStmt mutates an existing binding.
type AssignStmt struct {
	Name string
	Expr Expr
	Posn Position
}

func (s *AssignStmt) Pos() Position { return s.Posn }
func (*AssignStmt) stmtNode()       {}

// IfStmt conditionally executes branches.
type IfStmt struct {
	Cond Expr
	Then *BlockStmt
	Else *BlockStmt // may be nil
	Posn Position
}

func (s *IfStmt) Pos() Position { return s.Posn }
func (*IfStmt) stmtNode()       {}

// WhileStmt repeats while condition is truthy.
type WhileStmt struct {
	Cond Expr
	Body *BlockStmt
	Posn Position
}

func (s *WhileStmt) Pos() Position { return s.Posn }
func (*WhileStmt) stmtNode()       {}

// ReturnStmt exits the current function, optionally with a value.
type ReturnStmt struct {
	Result Expr // may be nil
	Posn   Position
}

func (s *ReturnStmt) Pos() Position { return s.Posn }
func (*ReturnStmt) stmtNode()       {}
