package parser

import "github.com/MDx3R/spl/internal/scanner"

// Node is the base interface for all AST nodes.
// Visitor is defined in visitor.go; both files are in the same package.
type Node interface {
	Accept(v Visitor)
}

// Expr is implemented by all expression nodes.
type Expr interface {
	Node
	exprNode()
}

// Stmt is implemented by all statement nodes.
type Stmt interface {
	Node
	stmtNode()
}

// Decl is implemented by all declaration nodes.
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
		Target Expr // lvalue: Ident, FieldExpr, IndexExpr
		Value  Expr
	}

	CompoundAssignExpr struct {
		Target Expr
		Op     scanner.Token
		Value  Expr
	}

	BlockExpr struct {
		Stmts []Stmt
		Tail  Expr // nil when block has no tail expression
	}

	CallExpr struct {
		Fun  Expr
		Args []Expr
	}

	FieldExpr struct {
		Obj   Expr
		Field string
	}

	IndexExpr struct {
		Obj   Expr
		Index Expr
	}

	// if cond { then } else { else_ }
	// Else is nil, BlockExpr, or IfExpr (else-if chain).
	IfExpr struct {
		Cond Expr
		Then BlockExpr
		Else Expr
	}

	WhileExpr struct {
		Cond Expr
		Body BlockExpr
	}

	ForExpr struct {
		Binding string
		Iter    Expr
		Body    BlockExpr
	}

	LoopExpr struct {
		Body BlockExpr
	}

	// 0..5 (Inclusive=false) or 0..=5 (Inclusive=true).
	// Lo or Hi may be nil for open-ended ranges (..5, 0..).
	RangeExpr struct {
		Lo, Hi    Expr
		Inclusive bool
	}

	// ident!(...) macro invocation
	MacroExpr struct {
		Name string
		Args []Expr
	}

	// [1, 2, 3]
	ArrayExpr struct {
		Elems []Expr
	}

	BreakExpr struct {
		Value Expr // may be nil
	}

	ContinueExpr struct{}

	ReturnExpr struct {
		Expr Expr // may be nil
	}
)

func (e BadExpr) exprNode()            {}
func (e Ident) exprNode()              {}
func (e BinaryExpr) exprNode()         {}
func (e UnaryExpr) exprNode()          {}
func (e GroupingExpr) exprNode()       {}
func (e LiteralExpr) exprNode()        {}
func (e AssignExpr) exprNode()         {}
func (e CompoundAssignExpr) exprNode() {}
func (e BlockExpr) exprNode()          {}
func (e CallExpr) exprNode()           {}
func (e FieldExpr) exprNode()          {}
func (e IndexExpr) exprNode()          {}
func (e IfExpr) exprNode()             {}
func (e WhileExpr) exprNode()          {}
func (e ForExpr) exprNode()            {}
func (e LoopExpr) exprNode()           {}
func (e RangeExpr) exprNode()          {}
func (e MacroExpr) exprNode()          {}
func (e ArrayExpr) exprNode()          {}
func (e BreakExpr) exprNode()          {}
func (e ContinueExpr) exprNode()       {}
func (e ReturnExpr) exprNode()         {}

type (
	BadStmt struct {
		From, To scanner.Token
	}

	ExprStmt struct {
		Expr Expr
	}
)

func (s BadStmt) stmtNode()  {}
func (s ExprStmt) stmtNode() {}

type VisibilityKind uint

const (
	VisPrivate VisibilityKind = iota
	VisPublic
)

type Visibility struct {
	Kind VisibilityKind
}

// Param is a function parameter.
// For self parameters: Name is "self"; Ref indicates & prefix; Mut indicates mut.
type Param struct {
	Name string
	Type Expr // nil for bare self / &self
	Ref  bool // true for &self and &mut self
	Mut  bool // true for mut self and &mut self
}

// FuncSignature is a function declaration without a body (used in trait declarations).
type FuncSignature struct {
	Name       string
	Params     []Param
	ReturnType Expr // nil when no return type is specified
}

// TraitMethod is either abstract (Body == nil) or has a default implementation.
type TraitMethod struct {
	Sig  FuncSignature
	Body *BlockExpr // nil = abstract
}

// FieldDef is a field in a struct declaration: name: Type
type FieldDef struct {
	Name string
	Type Expr
}

type (
	BadDecl struct {
		From, To scanner.Token
	}

	VarDecl struct {
		Name  string
		Type  Expr // nil when no type annotation
		Value Expr
		Mut   bool
	}

	FuncDecl struct {
		Name       string
		Params     []Param
		ReturnType Expr // nil when no return type
		Body       BlockExpr
		IsAsync    bool
		IsUnsafe   bool
		Const      bool
		Visibility Visibility
	}

	StructDecl struct {
		Name       string
		Fields     []FieldDef
		Visibility Visibility
	}

	TraitDecl struct {
		Name       string
		Methods    []TraitMethod
		Visibility Visibility
	}

	ImplDecl struct {
		Trait   string // "" for an inherent impl
		Type    string
		Methods []FuncDecl
	}
)

func (d BadDecl) stmtNode()    {}
func (d BadDecl) declNode()    {}
func (d VarDecl) stmtNode()    {}
func (d VarDecl) declNode()    {}
func (d FuncDecl) stmtNode()   {}
func (d FuncDecl) declNode()   {}
func (d StructDecl) stmtNode() {}
func (d StructDecl) declNode() {}
func (d TraitDecl) stmtNode()  {}
func (d TraitDecl) declNode()  {}
func (d ImplDecl) stmtNode()   {}
func (d ImplDecl) declNode()   {}
