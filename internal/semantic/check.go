package semantic

import (
	"fmt"

	"github.com/MDx3R/spl/internal/parser"
	"github.com/MDx3R/spl/internal/scanner"
)

// --- Pass 3: Check ---
//
// checkFile performs bottom-up type checking of all declaration bodies.
// It fills in symbol types inferred from initializers, validates semantic rules,
// and records diagnostics without modifying the AST.

func (a *Analyzer) checkFile(f *parser.File) {
	for _, d := range f.Decls {
		a.checkDecl(d)
	}
}

func (a *Analyzer) checkDecl(d parser.Decl) {
	switch d := d.(type) {
	case parser.FuncDecl:
		a.checkFuncBody(d)
	case parser.VarDecl:
		// Top-level let bindings: evaluate initializer for side effects and type inference.
		if d.Value != nil {
			initType := a.checkExpr(d.Value)
			sym := a.currentScope.LookupValue(d.Name)
			if sym != nil && sym.Type == nil {
				sym.Type = initType
			}
		}
	case parser.ImplDecl:
		prev := a.implType
		a.implType = d.Type
		for _, method := range d.Methods {
			a.checkMethod(d.Type, method)
		}
		a.implType = prev
	}
}

// checkFuncBody type-checks one function declaration (free or local).
func (a *Analyzer) checkFuncBody(d parser.FuncDecl) {
	sym := a.currentScope.LookupValue(d.Name)
	if sym == nil {
		return
	}
	fnType, ok := sym.Type.(*Function)
	if !ok {
		return
	}

	a.fnStack = append(a.fnStack, &fnContext{Name: d.Name, RetType: fnType.Ret})
	defer func() { a.fnStack = a.fnStack[:len(a.fnStack)-1] }()

	a.pushScope(d.Name, ScopeFunction)
	defer a.popScope()

	a.registerParams(d.Params, fnType)
	a.checkBlockBodyInScope(d.Body)
}

// checkMethod type-checks one impl method.
func (a *Analyzer) checkMethod(typeName string, d parser.FuncDecl) {
	name := fmt.Sprintf("%s::%s", typeName, d.Name)
	sym := a.currentScope.LookupValue(name)

	var fnType *Function
	if sym != nil {
		fnType, _ = sym.Type.(*Function)
	}
	if fnType == nil {
		fnType = a.buildFuncType(d)
	}

	a.fnStack = append(a.fnStack, &fnContext{Name: d.Name, RetType: fnType.Ret})
	defer func() { a.fnStack = a.fnStack[:len(a.fnStack)-1] }()

	a.pushScope(d.Name, ScopeFunction)
	defer a.popScope()

	// Register self with the impl receiver type.
	paramIdx := 0
	for _, p := range d.Params {
		if p.Name == "self" {
			selfSym := a.newSymbol(SymParam, "self")
			selfSym.Type = a.currentScope.LookupType(typeName)
			if selfSym.Type == nil {
				selfSym.Type = unknown
			}
			selfSym.Initialized = true
			a.currentScope.DeclareValue(selfSym)
			continue
		}
		var pt Type = unknown
		if paramIdx < len(fnType.Params) {
			pt = fnType.Params[paramIdx]
		}
		paramIdx++
		ps := a.newSymbol(SymParam, p.Name)
		ps.Type = pt
		ps.Initialized = true
		a.currentScope.DeclareValue(ps)
	}

	a.checkBlockBodyInScope(d.Body)
}

// registerParams declares function parameters in the current scope,
// using the correct paramIdx (skipping self) to match the resolved Function.Params.
func (a *Analyzer) registerParams(params []parser.Param, fnType *Function) {
	paramIdx := 0
	for _, p := range params {
		if p.Name == "self" {
			continue
		}
		var pt Type = unknown
		if paramIdx < len(fnType.Params) {
			pt = fnType.Params[paramIdx]
		}
		paramIdx++
		sym := a.newSymbol(SymParam, p.Name)
		sym.Type = pt
		sym.Initialized = true
		a.declareValue(sym)
	}
}

// checkBlockBodyInScope processes a BlockExpr's statements and tail expression
// within the already-active scope (used by checkFuncBody and checkFor).
// It runs the local-items mini-pipeline before processing statements.
func (a *Analyzer) checkBlockBodyInScope(b parser.BlockExpr) Type {
	a.collectLocalItems(b.Stmts)
	a.resolveLocalItems(b.Stmts)

	for _, s := range b.Stmts {
		a.checkStmt(s)
	}

	if b.Tail != nil {
		tailType := a.checkExpr(b.Tail)
		if fn := a.currentFunc(); fn != nil && !isUnknownOrInvalid(tailType) {
			if fn.RetType != nil && !fn.RetType.Equals(tailType) {
				a.addError(ErrTypeMismatch, fmt.Sprintf(
					"return type mismatch in '%s': expected %s, got %s",
					fn.Name, fn.RetType, tailType))
			}
		}
		return tailType
	}
	return unit
}

// checkBlock creates a new block scope, runs the local-items mini-pipeline,
// checks all statements, and returns the tail expression type (or unit).
func (a *Analyzer) checkBlock(b parser.BlockExpr) Type {
	a.pushScope("block", ScopeBlock)
	defer a.popScope()

	a.collectLocalItems(b.Stmts)
	a.resolveLocalItems(b.Stmts)

	for _, s := range b.Stmts {
		a.checkStmt(s)
	}

	if b.Tail != nil {
		return a.checkExpr(b.Tail)
	}
	return unit
}

// checkStmt dispatches statement type-checking. Item declarations (fn, struct,
// trait, impl) appearing as statements are handled via the mini-pipeline in
// checkBlock/checkBlockBodyInScope; here we check their bodies.
func (a *Analyzer) checkStmt(s parser.Stmt) {
	switch s := s.(type) {
	case parser.ExprStmt:
		a.checkExpr(s.Expr)
	case parser.VarDecl:
		a.checkVarDecl(s)
	case parser.FuncDecl:
		// Local function: body was collected/resolved in mini-pipeline; check it now.
		a.checkFuncBody(s)
	case parser.StructDecl:
		// Struct body was already resolved in mini-pipeline; nothing more to check.
	case parser.TraitDecl:
		// Same for traits.
	case parser.ImplDecl:
		prev := a.implType
		a.implType = s.Type
		for _, method := range s.Methods {
			a.checkMethod(s.Type, method)
		}
		a.implType = prev
	case parser.EmptyStmt, parser.BadStmt:
		// nothing
	}
}

// checkVarDecl evaluates the initializer, checks type compatibility with any
// annotation, and registers the symbol in the current scope.
func (a *Analyzer) checkVarDecl(d parser.VarDecl) {
	var initType Type
	if d.Value != nil {
		initType = a.checkExpr(d.Value)
	}
	if initType == nil {
		initType = unknown
	}

	varType := initType
	if d.Type != nil {
		annotated := a.resolveTypeExpr(d.Type)
		if annotated != nil {
			// Only report a mismatch when both types are concrete (non-unknown, non-invalid).
			if !isUnknownOrInvalid(initType) && !isUnknownOrInvalid(annotated) {
				if !annotated.Equals(initType) {
					a.addError(ErrTypeMismatch, fmt.Sprintf(
						"type mismatch in declaration of '%s': annotation is %s but initializer is %s",
						d.Name, annotated, initType))
				}
			}
			varType = annotated
		}
	}

	sym := a.newSymbol(SymVar, d.Name)
	sym.Type = varType
	sym.Mutable = d.Mut
	sym.Initialized = d.Value != nil
	sym.DeclNode = d
	a.declareValue(sym)
}

// --- Expression type-checking ---

// checkExpr returns the semantic type of e, recording any diagnostics encountered.
// It never returns nil; unknown is used as the fallback for unresolvable expressions.
func (a *Analyzer) checkExpr(e parser.Expr) Type {
	if e == nil {
		return unit
	}

	var t Type
	switch e := e.(type) {
	case parser.Ident:
		t = a.checkIdent(e)
	case parser.LiteralExpr:
		t = a.checkLiteral(e)
	case parser.BinaryExpr:
		t = a.checkBinary(e)
	case parser.UnaryExpr:
		t = a.checkUnary(e)
	case parser.GroupingExpr:
		t = a.checkExpr(e.Expr)
	case parser.AssignExpr:
		t = a.checkAssign(e)
	case parser.CompoundAssignExpr:
		t = a.checkCompoundAssign(e)
	case parser.BlockExpr:
		t = a.checkBlock(e)
	case parser.CallExpr:
		t = a.checkCall(e)
	case parser.FieldExpr:
		t = a.checkField(e)
	case parser.IndexExpr:
		t = a.checkIndex(e)
	case parser.ArrayExpr:
		t = a.checkArray(e)
	case parser.StructLitExpr:
		t = a.checkStructLit(e)
	case parser.RangeExpr:
		t = a.checkRange(e)
	case parser.IfExpr:
		t = a.checkIf(e)
	case parser.WhileExpr:
		t = a.checkWhile(e)
	case parser.ForExpr:
		t = a.checkFor(e)
	case parser.LoopExpr:
		t = a.checkLoop(e)
	case parser.MacroExpr:
		t = a.checkMacro(e)
	case parser.BreakExpr:
		t = a.checkBreak(e)
	case parser.ContinueExpr:
		t = a.checkContinue(e)
	case parser.ReturnExpr:
		t = a.checkReturn(e)
	case parser.BadExpr:
		t = unknown
	default:
		t = unknown
	}

	if t == nil {
		t = unknown
	}
	return t
}

func (a *Analyzer) checkIdent(e parser.Ident) Type {
	sym := a.currentScope.LookupValue(e.Name)
	if sym == nil {
		a.addError(ErrUndeclared, fmt.Sprintf("identifier '%s' is not declared", e.Name))
		return unknown
	}
	if !sym.Initialized {
		a.addError(ErrUninitialized, fmt.Sprintf("variable '%s' is used before initialization", e.Name))
	}
	if sym.Type == nil {
		return unknown
	}
	return sym.Type
}

func (a *Analyzer) checkLiteral(e parser.LiteralExpr) Type {
	return inferLiteralType(e.Kind, e.Value)
}

func (a *Analyzer) checkBinary(e parser.BinaryExpr) Type {
	leftType := a.checkExpr(e.Left)
	rightType := a.checkExpr(e.Right)

	switch {
	case isArithmeticOp(e.Op.Kind):
		if !isNumeric(leftType) && !isUnknownOrInvalid(leftType) {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"arithmetic operator '%s' requires numeric type, got %s", e.Op.Kind, leftType))
		}
		if !isNumeric(rightType) && !isUnknownOrInvalid(rightType) {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"arithmetic operator '%s' requires numeric type, got %s", e.Op.Kind, rightType))
		}
		res := arithmeticResult(leftType, rightType)
		if res == invalid {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"type mismatch in arithmetic: %s and %s", leftType, rightType))
			return unknown
		}
		return res

	case isBitwiseOp(e.Op.Kind):
		if !isNumeric(leftType) && !isUnknownOrInvalid(leftType) {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"bitwise operator '%s' requires integer type, got %s", e.Op.Kind, leftType))
		}
		if !isNumeric(rightType) && !isUnknownOrInvalid(rightType) {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"bitwise operator '%s' requires integer type, got %s", e.Op.Kind, rightType))
		}
		res := bitwiseResult(leftType, rightType)
		if res == invalid {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"type mismatch in bitwise op: %s and %s", leftType, rightType))
			return unknown
		}
		return res

	case isComparisonOp(e.Op.Kind):
		if !isUnknownOrInvalid(leftType) && !isUnknownOrInvalid(rightType) {
			if !leftType.Equals(rightType) {
				a.addError(ErrTypeMismatch, fmt.Sprintf(
					"comparison requires operands of the same type, got %s and %s",
					leftType, rightType))
			}
		}
		return &Scalar{Kind: ScalarBool}

	case isLogicalOp(e.Op.Kind):
		if !isBool(leftType) && !isUnknownOrInvalid(leftType) {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"logical operator '%s' requires bool, got %s", e.Op.Kind, leftType))
		}
		if !isBool(rightType) && !isUnknownOrInvalid(rightType) {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"logical operator '%s' requires bool, got %s", e.Op.Kind, rightType))
		}
		return &Scalar{Kind: ScalarBool}

	default:
		// Fallback for any other binary operator: treat as arithmetic.
		return arithmeticResult(leftType, rightType)
	}
}

func (a *Analyzer) checkUnary(e parser.UnaryExpr) Type {
	rightType := a.checkExpr(e.Right)

	switch e.Op.Kind {
	case scanner.Not:
		if !isBool(rightType) && !isUnknownOrInvalid(rightType) {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"unary '!' requires bool, got %s", rightType))
		}
		return &Scalar{Kind: ScalarBool}

	case scanner.Minus:
		if !isNumeric(rightType) && !isUnknownOrInvalid(rightType) {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"unary '-' requires numeric type, got %s", rightType))
		}
		return rightType

	case scanner.And:
		// &expr or &mut expr — reference; simplified to inner type.
		return rightType

	default:
		return rightType
	}
}

func (a *Analyzer) checkAssign(e parser.AssignExpr) Type {
	valType := a.checkExpr(e.Value)

	switch target := e.Target.(type) {
	case parser.Ident:
		sym := a.currentScope.LookupValue(target.Name)
		if sym == nil {
			a.addError(ErrUndeclared, fmt.Sprintf("identifier '%s' is not declared", target.Name))
		} else {
			if !sym.Mutable {
				a.addError(ErrImmutable, fmt.Sprintf(
					"cannot assign to immutable variable '%s'", target.Name))
			}
			if sym.Type != nil && !isUnknownOrInvalid(valType) {
				if !isUnknown(sym.Type) && !sym.Type.Equals(valType) {
					a.addError(ErrTypeMismatch, fmt.Sprintf(
						"cannot assign %s value to variable '%s' of type %s",
						valType, target.Name, sym.Type))
				} else if isUnknown(sym.Type) && !isUnknown(valType) {
					// Assigning a concrete type to an unknown-typed variable is suspicious.
					a.addError(ErrTypeMismatch, fmt.Sprintf(
						"cannot assign %s value to variable '%s' of type %s",
						valType, target.Name, sym.Type))
				}
			}
			sym.Initialized = true
		}

	case parser.FieldExpr:
		a.checkExpr(target.Obj)
		// Mutability of field targets is checked implicitly through the root object.

	default:
		a.checkExpr(e.Target)
	}

	return unit
}

func (a *Analyzer) checkCompoundAssign(e parser.CompoundAssignExpr) Type {
	valType := a.checkExpr(e.Value)

	switch target := e.Target.(type) {
	case parser.Ident:
		sym := a.currentScope.LookupValue(target.Name)
		if sym == nil {
			a.addError(ErrUndeclared, fmt.Sprintf("identifier '%s' is not declared", target.Name))
		} else {
			if !sym.Mutable {
				a.addError(ErrImmutable, fmt.Sprintf(
					"cannot assign to immutable variable '%s'", target.Name))
			}
			if !isNumeric(sym.Type) && !isUnknownOrInvalid(sym.Type) {
				a.addError(ErrTypeMismatch, fmt.Sprintf(
					"compound assignment target '%s' must be numeric, got %s",
					target.Name, sym.Type))
			}
			if !isNumeric(valType) && !isUnknownOrInvalid(valType) {
				a.addError(ErrTypeMismatch, fmt.Sprintf(
					"compound assignment value must be numeric, got %s", valType))
			}
		}
	default:
		a.checkExpr(e.Target)
	}

	return unit
}

func (a *Analyzer) checkCall(e parser.CallExpr) Type {
	calleeType := a.checkExpr(e.Fun)

	argTypes := make([]Type, len(e.Args))
	for i, arg := range e.Args {
		argTypes[i] = a.checkExpr(arg)
	}

	fn, ok := calleeType.(*Function)
	if !ok {
		// Callee resolved to unknown (e.g., undeclared identifier) — no cascade error.
		return unknown
	}

	if len(e.Args) != len(fn.Params) {
		a.addError(ErrTypeMismatch, fmt.Sprintf(
			"function called with %d argument(s) but expects %d",
			len(e.Args), len(fn.Params)))
	} else {
		for i, got := range argTypes {
			want := fn.Params[i]
			if !isUnknownOrInvalid(got) && !isUnknownOrInvalid(want) && !want.Equals(got) {
				a.addError(ErrTypeMismatch, fmt.Sprintf(
					"argument %d: expected %s, got %s", i+1, want, got))
			}
		}
	}

	if fn.Ret != nil {
		return fn.Ret
	}
	return unit
}

func (a *Analyzer) checkField(e parser.FieldExpr) Type {
	objType := a.checkExpr(e.Obj)
	if isUnknownOrInvalid(objType) {
		return unknown
	}
	st, ok := objType.(*Struct)
	if !ok {
		a.addError(ErrTypeMismatch, fmt.Sprintf(
			"field access on non-struct type %s", objType))
		return unknown
	}
	for _, f := range st.Fields {
		if f.Name == e.Field {
			return f.Type
		}
	}
	a.addError(ErrUndeclared, fmt.Sprintf(
		"struct '%s' has no field '%s'", st.Name, e.Field))
	return unknown
}

func (a *Analyzer) checkIndex(e parser.IndexExpr) Type {
	objType := a.checkExpr(e.Obj)
	a.checkExpr(e.Index)
	if isUnknownOrInvalid(objType) {
		return unknown
	}
	if arr, ok := objType.(*Array); ok {
		return arr.Elem
	}
	a.addError(ErrTypeMismatch, fmt.Sprintf("index operation on non-array type %s", objType))
	return unknown
}

func (a *Analyzer) checkArray(e parser.ArrayExpr) Type {
	if len(e.Elems) == 0 {
		return &Array{Elem: unknown}
	}
	first := a.checkExpr(e.Elems[0])
	for _, el := range e.Elems[1:] {
		elType := a.checkExpr(el)
		if !isUnknownOrInvalid(first) && !isUnknownOrInvalid(elType) && !first.Equals(elType) {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"array elements have inconsistent types: %s and %s", first, elType))
		}
	}
	return &Array{Elem: first}
}

func (a *Analyzer) checkStructLit(e parser.StructLitExpr) Type {
	st, ok := a.currentScope.LookupType(e.Name).(*Struct)
	if !ok {
		if a.currentScope.LookupType(e.Name) == nil {
			a.addError(ErrUndeclared, fmt.Sprintf("unknown struct '%s'", e.Name))
		} else {
			a.addError(ErrTypeMismatch, fmt.Sprintf("'%s' is not a struct", e.Name))
		}
		// Still check field expressions for side effects.
		for _, f := range e.Fields {
			if f.Value != nil {
				a.checkExpr(f.Value)
			}
		}
		if e.Spread != nil {
			a.checkExpr(e.Spread)
		}
		return unknown
	}

	// Build lookup map for O(1) field resolution.
	fieldTypes := make(map[string]Type, len(st.Fields))
	for _, f := range st.Fields {
		fieldTypes[f.Name] = f.Type
	}

	provided := make(map[string]bool, len(e.Fields))
	for _, lit := range e.Fields {
		provided[lit.Name] = true
		want, exists := fieldTypes[lit.Name]
		if !exists {
			a.addError(ErrUndeclared, fmt.Sprintf(
				"struct '%s' has no field '%s'", e.Name, lit.Name))
			if lit.Value != nil {
				a.checkExpr(lit.Value)
			}
			continue
		}
		var got Type
		if lit.Value != nil {
			got = a.checkExpr(lit.Value)
		} else {
			// Shorthand: field name == local variable name.
			got = a.checkIdent(parser.Ident{Name: lit.Name})
		}
		if !isUnknownOrInvalid(got) && !isUnknownOrInvalid(want) && !want.Equals(got) {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"field '%s' of struct '%s': expected %s, got %s", lit.Name, e.Name, want, got))
		}
	}

	if e.Spread == nil {
		for _, f := range st.Fields {
			if !provided[f.Name] {
				a.addError(ErrTypeMismatch, fmt.Sprintf(
					"missing field '%s' in struct literal '%s'", f.Name, e.Name))
			}
		}
	} else {
		a.checkExpr(e.Spread)
	}

	return st
}

func (a *Analyzer) checkRange(e parser.RangeExpr) Type {
	var elemType Type = &Scalar{Kind: ScalarI32}
	if e.Lo != nil {
		elemType = a.checkExpr(e.Lo)
	}
	if e.Hi != nil {
		hiType := a.checkExpr(e.Hi)
		if !isUnknownOrInvalid(elemType) && !isUnknownOrInvalid(hiType) && !elemType.Equals(hiType) {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"range bounds must have the same type, got %s and %s", elemType, hiType))
		}
	}
	return &Range{Elem: elemType}
}

func (a *Analyzer) checkIf(e parser.IfExpr) Type {
	condType := a.checkExpr(e.Cond)
	if !isBool(condType) && !isUnknownOrInvalid(condType) {
		a.addError(ErrTypeMismatch, fmt.Sprintf(
			"if condition must be bool, got %s", condType))
	}

	thenType := a.checkBlock(e.Then)

	if e.Else == nil {
		// if without else always has type unit in Rust.
		return unit
	}

	var elseType Type
	switch elseExpr := e.Else.(type) {
	case parser.BlockExpr:
		elseType = a.checkBlock(elseExpr)
	default:
		elseType = a.checkExpr(e.Else)
	}

	if !isUnknownOrInvalid(thenType) && !isUnknownOrInvalid(elseType) {
		if !thenType.Equals(elseType) {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"if/else branches have different types: %s vs %s", thenType, elseType))
		}
	}

	return thenType
}

func (a *Analyzer) checkWhile(e parser.WhileExpr) Type {
	condType := a.checkExpr(e.Cond)
	if !isBool(condType) && !isUnknownOrInvalid(condType) {
		a.addError(ErrTypeMismatch, fmt.Sprintf(
			"while condition must be bool, got %s", condType))
	}
	a.loopDepth++
	a.checkBlock(e.Body)
	a.loopDepth--
	return unit
}

func (a *Analyzer) checkFor(e parser.ForExpr) Type {
	// Infer loop binding type from the iterator / range bounds.
	bindingType := a.inferIterType(e.Iter)

	a.loopDepth++
	a.pushScope("for", ScopeFor)
	defer func() {
		a.popScope()
		a.loopDepth--
	}()

	bindSym := a.newSymbol(SymVar, e.Binding)
	bindSym.Type = bindingType
	bindSym.Initialized = true
	a.declareValue(bindSym)

	// Process body stmts in the same "for" scope (avoids gratuitous nesting).
	a.collectLocalItems(e.Body.Stmts)
	a.resolveLocalItems(e.Body.Stmts)
	for _, s := range e.Body.Stmts {
		a.checkStmt(s)
	}
	if e.Body.Tail != nil {
		a.checkExpr(e.Body.Tail)
	}

	return unit
}

// inferIterType returns the element type produced by an iterator expression.
func (a *Analyzer) inferIterType(iter parser.Expr) Type {
	if rng, ok := iter.(parser.RangeExpr); ok {
		if rng.Lo != nil {
			return a.checkExpr(rng.Lo)
		}
		if rng.Hi != nil {
			return a.checkExpr(rng.Hi)
		}
		return &Scalar{Kind: ScalarI32}
	}
	iterType := a.checkExpr(iter)
	if arr, ok := iterType.(*Array); ok {
		return arr.Elem
	}
	if r, ok := iterType.(*Range); ok {
		return r.Elem
	}
	return unknown
}

func (a *Analyzer) checkLoop(e parser.LoopExpr) Type {
	a.loopDepth++
	a.checkBlock(e.Body)
	a.loopDepth--
	return unit
}

func (a *Analyzer) checkMacro(e parser.MacroExpr) Type {
	for _, arg := range e.Args {
		a.checkExpr(arg)
	}
	return unit
}

func (a *Analyzer) checkBreak(e parser.BreakExpr) Type {
	if a.loopDepth == 0 {
		a.addError(ErrInvalidControl, "break outside of loop")
	}
	if e.Value != nil {
		a.checkExpr(e.Value)
	}
	return unit
}

func (a *Analyzer) checkContinue(_ parser.ContinueExpr) Type {
	if a.loopDepth == 0 {
		a.addError(ErrInvalidControl, "continue outside of loop")
	}
	return unit
}

func (a *Analyzer) checkReturn(e parser.ReturnExpr) Type {
	fn := a.currentFunc()
	if fn == nil {
		a.addError(ErrInvalidControl, "return outside of function")
		if e.Expr != nil {
			a.checkExpr(e.Expr)
		}
		return unit
	}

	if e.Expr != nil {
		got := a.checkExpr(e.Expr)
		if !isUnknownOrInvalid(got) && fn.RetType != nil && !fn.RetType.Equals(got) {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"return type mismatch: expected %s, got %s", fn.RetType, got))
		}
	} else {
		// Bare return: valid only in functions returning unit.
		if fn.RetType != nil && !fn.RetType.Equals(unit) {
			a.addError(ErrTypeMismatch, fmt.Sprintf(
				"missing return value in function returning %s", fn.RetType))
		}
	}
	return unit
}
