package parser

import (
	"fmt"
	"strings"
)

type AstPrinter struct {
	depth int
	buf   strings.Builder
}

func NewAstPrinter() *AstPrinter { return &AstPrinter{} }

func (pr *AstPrinter) String() string { return pr.buf.String() }

func (pr *AstPrinter) write(format string, args ...any) {
	pr.buf.WriteString(strings.Repeat("  ", pr.depth))
	fmt.Fprintf(&pr.buf, format, args...)
	pr.buf.WriteByte('\n')
}

func (pr *AstPrinter) indent(fn func()) {
	pr.depth++
	fn()
	pr.depth--
}

func (pr *AstPrinter) VisitFile(f *File) {
	pr.write("File (%d decl(s))", len(f.Decls))
	pr.indent(func() {
		for _, d := range f.Decls {
			VisitDecl(pr, d)
		}
	})
}

// Declarations

func (pr *AstPrinter) VisitBadDecl(d BadDecl) {
	pr.write("BadDecl [%d:%d .. %d:%d]", d.From.Line, d.From.Col, d.To.Line, d.To.Col)
}

func (pr *AstPrinter) VisitVarDecl(d VarDecl) {
	mut := ""
	if d.Mut {
		mut = " mut"
	}
	pr.write("VarDecl%s %q", mut, d.Name)
	pr.indent(func() {
		if d.Type != nil {
			pr.write("type:")
			pr.indent(func() { VisitExpr(pr, d.Type) })
		}
		pr.write("value:")
		pr.indent(func() { VisitExpr(pr, d.Value) })
	})
}

func (pr *AstPrinter) VisitFuncDecl(d FuncDecl) {
	vis := ""
	if d.Visibility.Kind == VisPublic {
		vis = "pub "
	}
	pr.write("FuncDecl %s%q", vis, d.Name)
	pr.indent(func() {
		pr.printParams(d.Params)
		if d.ReturnType != nil {
			pr.write("return:")
			pr.indent(func() { VisitExpr(pr, d.ReturnType) })
		}
		pr.write("body:")
		pr.indent(func() { pr.printBlock(d.Body) })
	})
}

func (pr *AstPrinter) VisitStructDecl(d StructDecl) {
	vis := ""
	if d.Visibility.Kind == VisPublic {
		vis = "pub "
	}
	pr.write("StructDecl %s%q", vis, d.Name)
	pr.indent(func() {
		for _, f := range d.Fields {
			pr.write("field %q:", f.Name)
			pr.indent(func() { VisitExpr(pr, f.Type) })
		}
	})
}

func (pr *AstPrinter) VisitTraitDecl(d TraitDecl) {
	vis := ""
	if d.Visibility.Kind == VisPublic {
		vis = "pub "
	}
	pr.write("TraitDecl %s%q", vis, d.Name)
	pr.indent(func() {
		for _, m := range d.Methods {
			if m.Body == nil {
				pr.write("abstract fn %q", m.Sig.Name)
			} else {
				pr.write("default fn %q", m.Sig.Name)
			}
			pr.indent(func() {
				pr.printParams(m.Sig.Params)
				if m.Sig.ReturnType != nil {
					pr.write("return:")
					pr.indent(func() { VisitExpr(pr, m.Sig.ReturnType) })
				}
				if m.Body != nil {
					pr.write("body:")
					pr.indent(func() { pr.printBlock(*m.Body) })
				}
			})
		}
	})
}

func (pr *AstPrinter) VisitImplDecl(d ImplDecl) {
	if d.Trait == "" {
		pr.write("ImplDecl %q", d.Type)
	} else {
		pr.write("ImplDecl %q for %q", d.Trait, d.Type)
	}
	pr.indent(func() {
		for i := range d.Methods {
			VisitDecl(pr, d.Methods[i])
		}
	})
}

// Statements

func (pr *AstPrinter) VisitBadStmt(s BadStmt) {
	pr.write("BadStmt [%d:%d .. %d:%d]", s.From.Line, s.From.Col, s.To.Line, s.To.Col)
}

func (pr *AstPrinter) VisitExprStmt(s ExprStmt) {
	pr.write("ExprStmt")
	pr.indent(func() { VisitExpr(pr, s.Expr) })
}

// Expressions

func (pr *AstPrinter) VisitBadExpr(e BadExpr) {
	pr.write("BadExpr [%d:%d .. %d:%d]", e.From.Line, e.From.Col, e.To.Line, e.To.Col)
}

func (pr *AstPrinter) VisitIdent(e Ident) {
	pr.write("Ident %q", e.Name)
}

func (pr *AstPrinter) VisitLiteralExpr(e LiteralExpr) {
	pr.write("Literal %v", e.Value)
}

func (pr *AstPrinter) VisitBinaryExpr(e BinaryExpr) {
	pr.write("BinaryExpr %s", e.Op.Kind)
	pr.indent(func() {
		VisitExpr(pr, e.Left)
		VisitExpr(pr, e.Right)
	})
}

func (pr *AstPrinter) VisitUnaryExpr(e UnaryExpr) {
	pr.write("UnaryExpr %s", e.Op.Kind)
	pr.indent(func() { VisitExpr(pr, e.Right) })
}

func (pr *AstPrinter) VisitGroupingExpr(e GroupingExpr) {
	pr.write("GroupingExpr")
	pr.indent(func() { VisitExpr(pr, e.Expr) })
}

func (pr *AstPrinter) VisitAssignExpr(e AssignExpr) {
	pr.write("AssignExpr")
	pr.indent(func() {
		pr.write("target:")
		pr.indent(func() { VisitExpr(pr, e.Target) })
		pr.write("value:")
		pr.indent(func() { VisitExpr(pr, e.Value) })
	})
}

func (pr *AstPrinter) VisitCompoundAssignExpr(e CompoundAssignExpr) {
	pr.write("CompoundAssignExpr %s", e.Op.Kind)
	pr.indent(func() {
		pr.write("target:")
		pr.indent(func() { VisitExpr(pr, e.Target) })
		pr.write("value:")
		pr.indent(func() { VisitExpr(pr, e.Value) })
	})
}

func (pr *AstPrinter) VisitBlockExpr(e BlockExpr) {
	pr.write("BlockExpr")
	pr.indent(func() { pr.printBlock(e) })
}

func (pr *AstPrinter) VisitCallExpr(e CallExpr) {
	pr.write("CallExpr")
	pr.indent(func() {
		pr.write("fun:")
		pr.indent(func() { VisitExpr(pr, e.Fun) })
		if len(e.Args) > 0 {
			pr.write("args:")
			pr.indent(func() {
				for _, a := range e.Args {
					VisitExpr(pr, a)
				}
			})
		}
	})
}

func (pr *AstPrinter) VisitFieldExpr(e FieldExpr) {
	pr.write("FieldExpr .%s", e.Field)
	pr.indent(func() { VisitExpr(pr, e.Obj) })
}

func (pr *AstPrinter) VisitIndexExpr(e IndexExpr) {
	pr.write("IndexExpr")
	pr.indent(func() {
		pr.write("obj:")
		pr.indent(func() { VisitExpr(pr, e.Obj) })
		pr.write("index:")
		pr.indent(func() { VisitExpr(pr, e.Index) })
	})
}

func (pr *AstPrinter) VisitArrayExpr(e ArrayExpr) {
	pr.write("ArrayExpr (%d elem(s))", len(e.Elems))
	pr.indent(func() {
		for _, el := range e.Elems {
			VisitExpr(pr, el)
		}
	})
}

func (pr *AstPrinter) VisitRangeExpr(e RangeExpr) {
	op := ".."
	if e.Inclusive {
		op = "..="
	}
	pr.write("RangeExpr %s", op)
	pr.indent(func() {
		if e.Lo != nil {
			pr.write("lo:")
			pr.indent(func() { VisitExpr(pr, e.Lo) })
		}
		if e.Hi != nil {
			pr.write("hi:")
			pr.indent(func() { VisitExpr(pr, e.Hi) })
		}
	})
}

func (pr *AstPrinter) VisitIfExpr(e IfExpr) {
	pr.write("IfExpr")
	pr.indent(func() {
		pr.write("cond:")
		pr.indent(func() { VisitExpr(pr, e.Cond) })
		pr.write("then:")
		pr.indent(func() { pr.printBlock(e.Then) })
		if e.Else != nil {
			pr.write("else:")
			pr.indent(func() { VisitExpr(pr, e.Else) })
		}
	})
}

func (pr *AstPrinter) VisitWhileExpr(e WhileExpr) {
	pr.write("WhileExpr")
	pr.indent(func() {
		pr.write("cond:")
		pr.indent(func() { VisitExpr(pr, e.Cond) })
		pr.write("body:")
		pr.indent(func() { pr.printBlock(e.Body) })
	})
}

func (pr *AstPrinter) VisitForExpr(e ForExpr) {
	pr.write("ForExpr %q", e.Binding)
	pr.indent(func() {
		pr.write("iter:")
		pr.indent(func() { VisitExpr(pr, e.Iter) })
		pr.write("body:")
		pr.indent(func() { pr.printBlock(e.Body) })
	})
}

func (pr *AstPrinter) VisitLoopExpr(e LoopExpr) {
	pr.write("LoopExpr")
	pr.indent(func() { pr.printBlock(e.Body) })
}

func (pr *AstPrinter) VisitMacroExpr(e MacroExpr) {
	pr.write("MacroExpr %s!", e.Name)
	pr.indent(func() {
		for _, a := range e.Args {
			VisitExpr(pr, a)
		}
	})
}

func (pr *AstPrinter) VisitBreakExpr(e BreakExpr) {
	pr.write("BreakExpr")
	if e.Value != nil {
		pr.indent(func() { VisitExpr(pr, e.Value) })
	}
}

func (pr *AstPrinter) VisitContinueExpr(_ ContinueExpr) {
	pr.write("ContinueExpr")
}

func (pr *AstPrinter) VisitReturnExpr(e ReturnExpr) {
	pr.write("ReturnExpr")
	if e.Expr != nil {
		pr.indent(func() { VisitExpr(pr, e.Expr) })
	}
}

// Helpers

func (pr *AstPrinter) printBlock(b BlockExpr) {
	for _, s := range b.Stmts {
		VisitStmt(pr, s)
	}
	if b.Tail != nil {
		pr.write("tail:")
		pr.indent(func() { VisitExpr(pr, b.Tail) })
	}
}

func (pr *AstPrinter) printParams(params []Param) {
	if len(params) == 0 {
		return
	}
	pr.write("params:")
	pr.indent(func() {
		for _, p := range params {
			switch {
			case p.Ref && p.Mut:
				pr.write("&mut self")
			case p.Ref:
				pr.write("&self")
			case p.Name == "self":
				pr.write("self")
			default:
				pr.write("param %q:", p.Name)
				pr.indent(func() { VisitExpr(pr, p.Type) })
			}
		}
	})
}
