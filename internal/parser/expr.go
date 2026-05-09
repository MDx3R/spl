package parser

import "github.com/MDx3R/spl/internal/scanner"

// All node types implement the Node interface.
type Node interface{}

// All expression nodes implement the Expr interface.
type Expr interface {
	Node
	exprNode()
}

// All statement nodes implement the Stmt interface.
type Stmt interface {
	Node
	stmtNode()
}

// All declaration nodes implement the Decl interface.
type Decl interface {
	Stmt
	declNode()
}

type File struct {
	Decls []Decl
}

type (
	BadExpr struct {
		From, To scanner.Token
	}

	Ident struct {
		Name string
	}

	BinaryExpr struct {
		Left  Expr
		Op    scanner.Token
		Right Expr
	}

	UnaryExpr struct {
		Op    scanner.Token
		Right Expr
	}

	GroupingExpr struct {
		Expr Expr
	}

	LiteralExpr struct {
		Value any
	}

	AssignExpr struct {
		Name  string
		Value Expr
	}

	BlockExpr struct {
		Stmts []Stmt
		Tail  Expr
	}

	CallExpr struct {
		Fun  Expr
		Args []Expr
	}

	ReturnExpr struct {
		Expr Expr
	}
)

func (e BadExpr) exprNode()      {}
func (e Ident) exprNode()        {}
func (e BinaryExpr) exprNode()   {}
func (e UnaryExpr) exprNode()    {}
func (e GroupingExpr) exprNode() {}
func (e LiteralExpr) exprNode()  {}
func (e AssignExpr) exprNode()   {}
func (e BlockExpr) exprNode()    {}
func (e CallExpr) exprNode()     {}
func (e ReturnExpr) exprNode()   {}

type (
	BadStmt struct {
		From, To scanner.Token
	}

	ExprStmt struct {
		Expr Expr
	}
)

func (e BadStmt) stmtNode()  {}
func (e ExprStmt) stmtNode() {}

type VisibilityKind uint

const (
	VisPrivate VisibilityKind = iota // no keyword
	VisPublic                        // pub
)

type Visibility struct {
	Kind VisibilityKind
}

type Param struct {
	Name string // TODO: extend to support "patters", e.g. mut x
	Type Expr
}

type (
	BadDecl struct {
		From, To scanner.Token
	}

	VarDecl struct {
		Name  string
		Value Expr
		Mut   bool
	}

	FuncDecl struct {
		Name       string
		Params     []Param
		ReturnType Expr
		Body       BlockExpr

		IsAsync    bool
		IsUnsafe   bool
		Const      bool
		Visibility Visibility
	}
)

func (d BadDecl) stmtNode()  {}
func (d BadDecl) declNode()  {}
func (d VarDecl) stmtNode()  {}
func (d VarDecl) declNode()  {}
func (d FuncDecl) stmtNode() {}
func (d FuncDecl) declNode() {}
