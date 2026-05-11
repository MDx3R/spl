package semantic

import (
	"fmt"
	"strings"

	"github.com/MDx3R/spl/internal/parser"
	"github.com/MDx3R/spl/internal/scanner"
)

// --- Pass 2: Resolve ---
//
// resolveFile fills in the placeholder types registered by the Collect pass.
// For each FuncDecl it resolves parameter types and return types; for each
// StructDecl it resolves field types; for each ImplDecl it resolves method signatures.
// Name uses inside expressions are NOT resolved here — that happens implicitly
// during the Check pass via scope lookups.

func (a *Analyzer) resolveFile(f *parser.File) {
	for _, d := range f.Decls {
		a.resolveDecl(d)
	}
}

func (a *Analyzer) resolveDecl(d parser.Decl) {
	switch d := d.(type) {
	case parser.FuncDecl:
		a.resolveFunc(d)
	case parser.StructDecl:
		a.resolveStruct(d)
	case parser.TraitDecl:
		a.resolveTrait(d)
	case parser.ImplDecl:
		a.resolveImpl(d)
	case parser.VarDecl:
		a.resolveVar(d)
	}
}

func (a *Analyzer) resolveFunc(d parser.FuncDecl) {
	sym := a.currentScope.LookupValue(d.Name)
	if sym == nil {
		return // error already reported in Collect pass
	}
	sym.Type = a.buildFuncType(d)
}

func (a *Analyzer) resolveStruct(d parser.StructDecl) {
	st := a.currentScope.LookupType(d.Name)
	if st == nil {
		return
	}
	s, ok := st.(*Struct)
	if !ok {
		return
	}
	fields := make([]FieldDef, 0, len(d.Fields))
	for _, f := range d.Fields {
		ft := a.resolveTypeExpr(f.Type)
		fields = append(fields, FieldDef{Name: f.Name, Type: ft})
	}
	s.Fields = fields
}

func (a *Analyzer) resolveTrait(_ parser.TraitDecl) {
	// Trait method signatures are recorded but trait bounds are out of scope.
}

func (a *Analyzer) resolveImpl(d parser.ImplDecl) {
	prev := a.implType
	a.implType = d.Type
	defer func() { a.implType = prev }()

	for _, method := range d.Methods {
		name := fmt.Sprintf("%s::%s", d.Type, method.Name)
		sym := a.currentScope.LookupValue(name)
		if sym == nil {
			continue
		}
		sym.Type = a.buildFuncType(method)
	}
}

func (a *Analyzer) resolveVar(d parser.VarDecl) {
	sym := a.currentScope.LookupValue(d.Name)
	if sym == nil {
		return
	}
	// If there is a type annotation, resolve it now so that the Check pass
	// can compare the initializer type against it.
	if d.Type != nil {
		sym.Type = a.resolveTypeExpr(d.Type)
	}
	// When there is no annotation, type inference runs in the Check pass.
}

// resolveLocalItems fills in the types of item declarations found among block statements.
// Must be called after collectLocalItems for the same statement list.
func (a *Analyzer) resolveLocalItems(stmts []parser.Stmt) {
	for _, s := range stmts {
		if decl, ok := stmtAsDecl(s); ok {
			a.resolveDecl(decl)
		}
	}
}

// buildFuncType constructs a resolved *Function type from a FuncDecl, consulting
// the current scope for user-defined type names.
func (a *Analyzer) buildFuncType(d parser.FuncDecl) *Function {
	var params []Type
	for _, p := range d.Params {
		if p.Name == "self" {
			continue // self is implicit; tracked via implType
		}
		params = append(params, a.resolveTypeExpr(p.Type))
	}
	ret := Type(unit)
	if d.ReturnType != nil {
		if rt := a.resolveTypeExpr(d.ReturnType); rt != nil {
			ret = rt
		}
	}
	return &Function{Params: params, Ret: ret}
}

// resolveTypeExpr converts a type-position Expr into a semantic Type.
//
// The parser reuses expression nodes for types:
//   - Ident               → builtin scalar or user-defined type
//   - ArrayExpr{1 elem}   → [T] (unsized array / slice)
//   - UnaryExpr{&, T}     → &T (reference; simplified to the inner type)
//   - Ident "Self"/"self" → current impl receiver type
func (a *Analyzer) resolveTypeExpr(e parser.Expr) Type {
	if e == nil {
		return unknown
	}
	switch ex := e.(type) {
	case parser.Ident:
		return a.resolveTypeName(ex.Name)

	case parser.ArrayExpr:
		if len(ex.Elems) == 1 {
			elem := a.resolveTypeExpr(ex.Elems[0])
			return &Array{Elem: elem}
		}
		return unknown

	case parser.UnaryExpr:
		// &T and &mut T: we simplify references to their inner type for now.
		if ex.Op.Kind == scanner.And {
			return a.resolveTypeExpr(ex.Right)
		}
		return unknown
	}
	return unknown
}

// resolveTypeName maps a single identifier to a built-in or user-defined type.
func (a *Analyzer) resolveTypeName(name string) Type {
	if name == "Self" || name == "self" {
		if a.implType != "" {
			if t := a.currentScope.LookupType(a.implType); t != nil {
				return t
			}
		}
		return unknown
	}
	if t := resolveTypeName(name); t != nil {
		return t
	}
	if t := a.currentScope.LookupType(name); t != nil {
		return t
	}
	a.addError(ErrUndeclared, fmt.Sprintf("unknown type '%s'", name))
	return unknown
}

// --- Pass 2.5: Recursive struct detection ---

// checkRecursiveStructs detects cycles in the struct field-type dependency graph.
// It uses pointer identity as the graph key to correctly handle same-named structs
// declared in different scopes, and copies path slices before recursion to prevent
// aliasing.
func (a *Analyzer) checkRecursiveStructs() {
	deps := make(map[*Struct][]*Struct)

	var gather func(sc *Scope)
	gather = func(sc *Scope) {
		for _, t := range sc.types {
			if st, ok := t.(*Struct); ok {
				if _, exists := deps[st]; !exists {
					deps[st] = nil
				}
				for _, f := range st.Fields {
					for _, dep := range structsInType(f.Type) {
						deps[st] = append(deps[st], dep)
					}
				}
			}
		}
		for _, child := range sc.Children {
			gather(child)
		}
	}
	gather(a.globalScope)

	color := make(map[*Struct]int) // 0=unvisited 1=in-progress 2=done

	var detect func(st *Struct, path []*Struct)
	detect = func(st *Struct, path []*Struct) {
		switch color[st] {
		case 2:
			return
		case 1:
			// Back edge: build the cycle name list.
			cyc := make([]*Struct, len(path)+1)
			copy(cyc, path)
			cyc[len(path)] = st
			names := make([]string, len(cyc))
			for i, s := range cyc {
				names[i] = s.Name
			}
			a.addError(ErrRecursiveType, "recursive type: "+strings.Join(names, " -> "))
			return
		}

		color[st] = 1
		for _, dep := range deps[st] {
			next := make([]*Struct, len(path)+1)
			copy(next, path)
			next[len(path)] = st
			detect(dep, next)
		}
		color[st] = 2
	}

	for st := range deps {
		if color[st] == 0 {
			detect(st, nil)
		}
	}
}
