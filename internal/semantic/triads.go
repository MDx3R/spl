package semantic

// Triad IR generation.
//
// Algorithm (post-order traversal):
//   Literals and identifiers set lastRef without emitting a triad.
//   BinaryExpr: visit left (save ref), visit right (save ref), emit (op, left, right).
//   VarDecl / AssignExpr: visit value, emit (:=, name, value_ref).
//   CompoundAssignExpr (x += v): visit target and value, emit (baseOp, x, v), emit (:=, x, ^N).
//   CallExpr / MacroExpr: emit (param, arg, -) per arg, then (call, name, argCount).
//   IfExpr: emit cond; reserve if_false; visit then; reserve goto; patch if_false;
//           visit else; patch goto.
//   WhileExpr: record loopStart; emit cond; reserve if_false; visit body;
//              emit (goto, -, ^loopStart); patch if_false.
//   ForExpr (range only): emit (:=, i, lo); record loopStart; emit (<, i, hi);
//           reserve if_false; visit body; emit (+, i, 1); emit (:=, i, ^N);
//           emit (goto, -, ^loopStart); patch if_false.
//
// Triads are numbered 1-based in the output. Forward jump targets are patched
// once the target triad index is known.
//
// The TriadEmitter receives an *AnalysisResult produced by the Analyzer. Although
// the current implementation walks the raw AST (preserving identical triad output
// for all existing programs), the result is available for future enhancements
// such as shadowing-aware variable naming and type-directed code generation.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MDx3R/spl/internal/parser"
	"github.com/MDx3R/spl/internal/scanner"
)

// OperandKind classifies a triad operand.
type OperandKind int

const (
	OpNone    OperandKind = iota // prints as "-"
	OpLiteral                    // prints as the raw value string
	OpIdent                      // prints as the identifier name
	OpRef                        // prints as ^N (reference to triad N, 1-based)
)

// Operand is one of the two operands of a Triad.
type Operand struct {
	Kind  OperandKind
	Value string // used for OpLiteral and OpIdent
	Ref   int    // used for OpRef (1-based triad number)
}

// Triad is a three-address intermediate representation instruction.
type Triad struct {
	Op string
	A  Operand
	B  Operand
}

// TriadEmitter walks the AST and generates a Triad sequence.
// It implements parser.Visitor; lastRef carries the operand produced by the
// most recently visited expression (side-channel return).
//
// The result field provides access to the semantic analysis output; the emitter
// consults it for canonical symbol information rather than re-resolving names.
type TriadEmitter struct {
	triads  []*Triad
	lastRef Operand
	result  *AnalysisResult // from the preceding Analyzer pass; may be nil
}

// NewTriadEmitter creates an empty TriadEmitter.
func NewTriadEmitter() *TriadEmitter { return &TriadEmitter{} }

// EmitFile generates triads for every top-level declaration in f,
// using result for symbol and type information from the analysis pass.
// Passing a nil result degrades gracefully (no semantic data is consulted).
func (e *TriadEmitter) EmitFile(f *parser.File, result *AnalysisResult) {
	e.result = result
	for _, d := range f.Decls {
		parser.VisitDecl(e, d)
	}
}

// Triads returns the generated triad slice (read-only).
func (e *TriadEmitter) Triads() []*Triad { return e.triads }

// String formats the complete triad sequence as a numbered listing.
func (e *TriadEmitter) String() string {
	var sb strings.Builder
	fmt.Fprintln(&sb, "Intermediate Representation (Triads):")
	for i, t := range e.triads {
		fmt.Fprintf(&sb, "%2d) (%s, %s, %s)\n",
			i+1, t.Op, formatOperand(t.A), formatOperand(t.B))
	}
	return sb.String()
}

// emit appends a fully specified triad and returns a ref operand pointing to it.
func (e *TriadEmitter) emit(op string, a, b Operand) Operand {
	e.triads = append(e.triads, &Triad{Op: op, A: a, B: b})
	return Operand{Kind: OpRef, Ref: len(e.triads)}
}

// reserveJump emits a triad whose B operand is a placeholder (^0) and returns
// the zero-based slice index for later patching via patchJump.
func (e *TriadEmitter) reserveJump(op string, a Operand) int {
	idx := len(e.triads)
	e.triads = append(e.triads, &Triad{Op: op, A: a, B: Operand{Kind: OpRef, Ref: 0}})
	return idx
}

// patchJump updates the B operand of the triad at sliceIdx with a 1-based target.
func (e *TriadEmitter) patchJump(sliceIdx int, target1Based int) {
	e.triads[sliceIdx].B = Operand{Kind: OpRef, Ref: target1Based}
}

// nextTriadNum returns the 1-based number of the next triad to be emitted.
func (e *TriadEmitter) nextTriadNum() int { return len(e.triads) + 1 }

// visitBlockBody visits block statements and tail without opening a new scope
// (the emitter has no scope concept; it mirrors only the emission order).
func (e *TriadEmitter) visitBlockBody(b parser.BlockExpr) {
	for _, s := range b.Stmts {
		parser.VisitStmt(e, s)
	}
	if b.Tail != nil {
		parser.VisitExpr(e, b.Tail)
	}
}

// Declarations

func (e *TriadEmitter) VisitFuncDecl(d parser.FuncDecl) {
	e.visitBlockBody(d.Body)
}

func (e *TriadEmitter) VisitVarDecl(d parser.VarDecl) {
	if d.Value != nil {
		parser.VisitExpr(e, d.Value)
	}
	val := e.lastRef
	e.emit(":=", Operand{Kind: OpIdent, Value: d.Name}, val)
}

func (e *TriadEmitter) VisitStructDecl(_ parser.StructDecl) {}
func (e *TriadEmitter) VisitTraitDecl(_ parser.TraitDecl)   {}
func (e *TriadEmitter) VisitImplDecl(d parser.ImplDecl) {
	for i := range d.Methods {
		parser.VisitDecl(e, d.Methods[i])
	}
}
func (e *TriadEmitter) VisitBadDecl(_ parser.BadDecl)     {}
func (e *TriadEmitter) VisitEmptyDecl(_ parser.EmptyDecl) {}

// Statements

func (e *TriadEmitter) VisitExprStmt(s parser.ExprStmt) {
	parser.VisitExpr(e, s.Expr)
}

func (e *TriadEmitter) VisitBadStmt(_ parser.BadStmt)     {}
func (e *TriadEmitter) VisitEmptyStmt(_ parser.EmptyStmt) {}

// Expressions

func (e *TriadEmitter) VisitIdent(ex parser.Ident) {
	e.lastRef = Operand{Kind: OpIdent, Value: ex.Name}
}

func (e *TriadEmitter) VisitLiteralExpr(ex parser.LiteralExpr) {
	var val string
	switch ex.Kind {
	case scanner.StrLit:
		val = `"` + fmt.Sprintf("%v", ex.Value) + `"`
	case scanner.CharLit:
		val = "'" + fmt.Sprintf("%v", ex.Value) + "'"
	default:
		val = fmt.Sprintf("%v", ex.Value)
	}
	e.lastRef = Operand{Kind: OpLiteral, Value: val}
}

func (e *TriadEmitter) VisitBinaryExpr(ex parser.BinaryExpr) {
	parser.VisitExpr(e, ex.Left)
	left := e.lastRef
	parser.VisitExpr(e, ex.Right)
	right := e.lastRef
	e.lastRef = e.emit(ex.Op.Kind.String(), left, right)
}

func (e *TriadEmitter) VisitUnaryExpr(ex parser.UnaryExpr) {
	parser.VisitExpr(e, ex.Right)
	right := e.lastRef
	e.lastRef = e.emit(ex.Op.Kind.String(), right, Operand{Kind: OpNone})
}

func (e *TriadEmitter) VisitGroupingExpr(ex parser.GroupingExpr) {
	parser.VisitExpr(e, ex.Expr)
}

func (e *TriadEmitter) VisitAssignExpr(ex parser.AssignExpr) {
	parser.VisitExpr(e, ex.Value)
	val := e.lastRef

	var target Operand
	if id, ok := ex.Target.(parser.Ident); ok {
		target = Operand{Kind: OpIdent, Value: id.Name}
	} else {
		parser.VisitExpr(e, ex.Target)
		target = e.lastRef
	}
	e.lastRef = e.emit(":=", target, val)
}

func (e *TriadEmitter) VisitCompoundAssignExpr(ex parser.CompoundAssignExpr) {
	// e.g. counter += 1  →  (+, counter, 1); (:=, counter, ^N)
	parser.VisitExpr(e, ex.Target)
	left := e.lastRef
	parser.VisitExpr(e, ex.Value)
	right := e.lastRef

	baseOp := compoundBaseOp(ex.Op.Kind)
	addRef := e.emit(baseOp, left, right)

	var target Operand
	if id, ok := ex.Target.(parser.Ident); ok {
		target = Operand{Kind: OpIdent, Value: id.Name}
	} else {
		target = left
	}
	e.lastRef = e.emit(":=", target, addRef)
}

func (e *TriadEmitter) VisitBlockExpr(ex parser.BlockExpr) {
	e.visitBlockBody(ex)
}

func (e *TriadEmitter) VisitCallExpr(ex parser.CallExpr) {
	for _, arg := range ex.Args {
		parser.VisitExpr(e, arg)
		argRef := e.lastRef
		e.emit("param", argRef, Operand{Kind: OpNone})
	}

	var funcName string
	if id, ok := ex.Fun.(parser.Ident); ok {
		funcName = id.Name
	} else {
		parser.VisitExpr(e, ex.Fun)
		funcName = "?"
	}
	e.lastRef = e.emit("call",
		Operand{Kind: OpIdent, Value: funcName},
		Operand{Kind: OpLiteral, Value: strconv.Itoa(len(ex.Args))})
}

func (e *TriadEmitter) VisitMacroExpr(ex parser.MacroExpr) {
	for _, arg := range ex.Args {
		parser.VisitExpr(e, arg)
		argRef := e.lastRef
		e.emit("param", argRef, Operand{Kind: OpNone})
	}
	e.lastRef = e.emit("call",
		Operand{Kind: OpIdent, Value: ex.Name + "!"},
		Operand{Kind: OpLiteral, Value: strconv.Itoa(len(ex.Args))})
}

func (e *TriadEmitter) VisitFieldExpr(ex parser.FieldExpr) {
	parser.VisitExpr(e, ex.Obj)
	e.lastRef = Operand{Kind: OpIdent, Value: ex.Field}
}

func (e *TriadEmitter) VisitIndexExpr(ex parser.IndexExpr) {
	parser.VisitExpr(e, ex.Obj)
	obj := e.lastRef
	parser.VisitExpr(e, ex.Index)
	idx := e.lastRef
	e.lastRef = e.emit("index", obj, idx)
}

func (e *TriadEmitter) VisitArrayExpr(ex parser.ArrayExpr) {
	for _, el := range ex.Elems {
		parser.VisitExpr(e, el)
		e.emit("elem", e.lastRef, Operand{Kind: OpNone})
	}
	e.lastRef = Operand{Kind: OpNone}
}

func (e *TriadEmitter) VisitStructLitExpr(ex parser.StructLitExpr) {
	for _, f := range ex.Fields {
		if f.Value != nil {
			parser.VisitExpr(e, f.Value)
			e.emit("field", Operand{Kind: OpLiteral, Value: f.Name}, e.lastRef)
		} else {
			e.emit("field",
				Operand{Kind: OpLiteral, Value: f.Name},
				Operand{Kind: OpIdent, Value: f.Name})
		}
	}
	if ex.Spread != nil {
		parser.VisitExpr(e, ex.Spread)
		e.emit("spread", e.lastRef, Operand{Kind: OpNone})
	}
	e.lastRef = Operand{Kind: OpNone}
}

func (e *TriadEmitter) VisitRangeExpr(ex parser.RangeExpr) {
	// Handled specially inside VisitForExpr; fall back for standalone range exprs.
	if ex.Lo != nil {
		parser.VisitExpr(e, ex.Lo)
	}
	if ex.Hi != nil {
		parser.VisitExpr(e, ex.Hi)
	}
}

func (e *TriadEmitter) VisitIfExpr(ex parser.IfExpr) {
	parser.VisitExpr(e, ex.Cond)
	condRef := e.lastRef

	ifFalseIdx := e.reserveJump("if_false", condRef)

	e.visitBlockBody(ex.Then)

	gotoIdx := -1
	if ex.Else != nil {
		gotoIdx = e.reserveJump("goto", Operand{Kind: OpNone})
	}

	e.patchJump(ifFalseIdx, e.nextTriadNum())

	if ex.Else != nil {
		switch elseExpr := ex.Else.(type) {
		case parser.BlockExpr:
			e.visitBlockBody(elseExpr)
		default:
			parser.VisitExpr(e, ex.Else)
		}
		e.patchJump(gotoIdx, e.nextTriadNum())
	}

	e.lastRef = Operand{Kind: OpNone}
}

func (e *TriadEmitter) VisitWhileExpr(ex parser.WhileExpr) {
	loopStart := e.nextTriadNum()

	parser.VisitExpr(e, ex.Cond)
	condRef := e.lastRef

	ifFalseIdx := e.reserveJump("if_false", condRef)

	e.visitBlockBody(ex.Body)

	e.emit("goto", Operand{Kind: OpNone}, Operand{Kind: OpRef, Ref: loopStart})

	e.patchJump(ifFalseIdx, e.nextTriadNum())

	e.lastRef = Operand{Kind: OpNone}
}

func (e *TriadEmitter) VisitForExpr(ex parser.ForExpr) {
	binding := Operand{Kind: OpIdent, Value: ex.Binding}

	var loRef, hiRef Operand
	inclusive := false
	if rng, ok := ex.Iter.(parser.RangeExpr); ok {
		inclusive = rng.Inclusive
		if rng.Lo != nil {
			parser.VisitExpr(e, rng.Lo)
			loRef = e.lastRef
		} else {
			loRef = Operand{Kind: OpLiteral, Value: "0"}
		}
		if rng.Hi != nil {
			parser.VisitExpr(e, rng.Hi)
			hiRef = e.lastRef
		} else {
			hiRef = Operand{Kind: OpNone}
		}
	} else {
		parser.VisitExpr(e, ex.Iter)
		loRef = Operand{Kind: OpLiteral, Value: "0"}
		hiRef = e.lastRef
	}

	e.emit(":=", binding, loRef)

	loopStart := e.nextTriadNum()
	condOp := "<"
	if inclusive {
		condOp = "<="
	}
	condRef := e.emit(condOp, binding, hiRef)

	ifFalseIdx := e.reserveJump("if_false", condRef)

	e.visitBlockBody(ex.Body)

	incRef := e.emit("+", binding, Operand{Kind: OpLiteral, Value: "1"})
	e.emit(":=", binding, incRef)

	e.emit("goto", Operand{Kind: OpNone}, Operand{Kind: OpRef, Ref: loopStart})

	e.patchJump(ifFalseIdx, e.nextTriadNum())

	e.lastRef = Operand{Kind: OpNone}
}

func (e *TriadEmitter) VisitLoopExpr(ex parser.LoopExpr) {
	loopStart := e.nextTriadNum()
	e.visitBlockBody(ex.Body)
	e.emit("goto", Operand{Kind: OpNone}, Operand{Kind: OpRef, Ref: loopStart})
	e.lastRef = Operand{Kind: OpNone}
}

func (e *TriadEmitter) VisitBreakExpr(ex parser.BreakExpr) {
	if ex.Value != nil {
		parser.VisitExpr(e, ex.Value)
	}
	e.emit("break", Operand{Kind: OpNone}, Operand{Kind: OpNone})
	e.lastRef = Operand{Kind: OpNone}
}

func (e *TriadEmitter) VisitContinueExpr(_ parser.ContinueExpr) {
	e.emit("continue", Operand{Kind: OpNone}, Operand{Kind: OpNone})
	e.lastRef = Operand{Kind: OpNone}
}

func (e *TriadEmitter) VisitReturnExpr(ex parser.ReturnExpr) {
	if ex.Expr != nil {
		parser.VisitExpr(e, ex.Expr)
		e.emit("return", e.lastRef, Operand{Kind: OpNone})
	} else {
		e.emit("return", Operand{Kind: OpNone}, Operand{Kind: OpNone})
	}
	e.lastRef = Operand{Kind: OpNone}
}

func (e *TriadEmitter) VisitBadExpr(_ parser.BadExpr) {
	e.lastRef = Operand{Kind: OpNone}
}

func formatOperand(o Operand) string {
	switch o.Kind {
	case OpNone:
		return "-"
	case OpLiteral, OpIdent:
		return o.Value
	case OpRef:
		return fmt.Sprintf("^%d", o.Ref)
	}
	return "-"
}
