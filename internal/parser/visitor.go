package parser

type Visitor interface {
	// Expressions
	VisitBadExpr(e BadExpr)
	VisitIdent(e Ident)
	VisitLiteralExpr(e LiteralExpr)
	VisitBinaryExpr(e BinaryExpr)
	VisitUnaryExpr(e UnaryExpr)
	VisitGroupingExpr(e GroupingExpr)
	VisitAssignExpr(e AssignExpr)
	VisitCompoundAssignExpr(e CompoundAssignExpr)
	VisitBlockExpr(e BlockExpr)
	VisitCallExpr(e CallExpr)
	VisitFieldExpr(e FieldExpr)
	VisitIndexExpr(e IndexExpr)
	VisitArrayExpr(e ArrayExpr)
	VisitRangeExpr(e RangeExpr)
	VisitIfExpr(e IfExpr)
	VisitWhileExpr(e WhileExpr)
	VisitForExpr(e ForExpr)
	VisitLoopExpr(e LoopExpr)
	VisitMacroExpr(e MacroExpr)
	VisitBreakExpr(e BreakExpr)
	VisitContinueExpr(e ContinueExpr)
	VisitReturnExpr(e ReturnExpr)

	// Statements
	VisitBadStmt(s BadStmt)
	VisitExprStmt(s ExprStmt)

	// Declarations
	VisitBadDecl(d BadDecl)
	VisitVarDecl(d VarDecl)
	VisitFuncDecl(d FuncDecl)
	VisitStructDecl(d StructDecl)
	VisitTraitDecl(d TraitDecl)
	VisitImplDecl(d ImplDecl)
}

func VisitExpr(v Visitor, e Expr) {
	if e == nil {
		return
	}
	e.Accept(v)
}

func VisitStmt(v Visitor, s Stmt) {
	if s == nil {
		return
	}
	s.Accept(v)
}

func VisitDecl(v Visitor, d Decl) {
	if d == nil {
		return
	}
	d.Accept(v)
}

// Expressions
func (e BadExpr) Accept(v Visitor)            { v.VisitBadExpr(e) }
func (e Ident) Accept(v Visitor)              { v.VisitIdent(e) }
func (e LiteralExpr) Accept(v Visitor)        { v.VisitLiteralExpr(e) }
func (e BinaryExpr) Accept(v Visitor)         { v.VisitBinaryExpr(e) }
func (e UnaryExpr) Accept(v Visitor)          { v.VisitUnaryExpr(e) }
func (e GroupingExpr) Accept(v Visitor)       { v.VisitGroupingExpr(e) }
func (e AssignExpr) Accept(v Visitor)         { v.VisitAssignExpr(e) }
func (e CompoundAssignExpr) Accept(v Visitor) { v.VisitCompoundAssignExpr(e) }
func (e BlockExpr) Accept(v Visitor)          { v.VisitBlockExpr(e) }
func (e CallExpr) Accept(v Visitor)           { v.VisitCallExpr(e) }
func (e FieldExpr) Accept(v Visitor)          { v.VisitFieldExpr(e) }
func (e IndexExpr) Accept(v Visitor)          { v.VisitIndexExpr(e) }
func (e ArrayExpr) Accept(v Visitor)          { v.VisitArrayExpr(e) }
func (e RangeExpr) Accept(v Visitor)          { v.VisitRangeExpr(e) }
func (e IfExpr) Accept(v Visitor)             { v.VisitIfExpr(e) }
func (e WhileExpr) Accept(v Visitor)          { v.VisitWhileExpr(e) }
func (e ForExpr) Accept(v Visitor)            { v.VisitForExpr(e) }
func (e LoopExpr) Accept(v Visitor)           { v.VisitLoopExpr(e) }
func (e MacroExpr) Accept(v Visitor)          { v.VisitMacroExpr(e) }
func (e BreakExpr) Accept(v Visitor)          { v.VisitBreakExpr(e) }
func (e ContinueExpr) Accept(v Visitor)       { v.VisitContinueExpr(e) }
func (e ReturnExpr) Accept(v Visitor)         { v.VisitReturnExpr(e) }

// Statements
func (s BadStmt) Accept(v Visitor)  { v.VisitBadStmt(s) }
func (s ExprStmt) Accept(v Visitor) { v.VisitExprStmt(s) }

// Declarations
func (d BadDecl) Accept(v Visitor)    { v.VisitBadDecl(d) }
func (d VarDecl) Accept(v Visitor)    { v.VisitVarDecl(d) }
func (d FuncDecl) Accept(v Visitor)   { v.VisitFuncDecl(d) }
func (d StructDecl) Accept(v Visitor) { v.VisitStructDecl(d) }
func (d TraitDecl) Accept(v Visitor)  { v.VisitTraitDecl(d) }
func (d ImplDecl) Accept(v Visitor)   { v.VisitImplDecl(d) }
