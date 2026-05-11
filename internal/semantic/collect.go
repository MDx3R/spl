package semantic

import (
	"fmt"

	"github.com/MDx3R/spl/internal/parser"
)

// --- Pass 1: Collect ---
//
// collectFile walks every top-level declaration and registers placeholder symbols
// in the global scope. Types and function signatures remain unresolved; they are
// filled in by the subsequent Resolve pass.

func (a *Analyzer) collectFile(f *parser.File) {
	for _, d := range f.Decls {
		a.collectDecl(d)
	}
}

func (a *Analyzer) collectDecl(d parser.Decl) {
	switch d := d.(type) {
	case parser.FuncDecl:
		a.collectFunc(d)
	case parser.StructDecl:
		a.collectStruct(d)
	case parser.TraitDecl:
		a.collectTrait(d)
	case parser.ImplDecl:
		a.collectImpl(d)
	case parser.VarDecl:
		a.collectVar(d)
	}
}

func (a *Analyzer) collectFunc(d parser.FuncDecl) {
	sym := a.newSymbol(SymFunc, d.Name)
	// Placeholder: params and return type are filled in during the Resolve pass.
	sym.Type = &Function{}
	sym.Initialized = true
	sym.DeclNode = d
	a.declareValue(sym)
}

func (a *Analyzer) collectStruct(d parser.StructDecl) {
	// Placeholder: fields are filled in during the Resolve pass.
	t := &Struct{Name: d.Name}
	a.declareType(d.Name, t)
}

func (a *Analyzer) collectTrait(d parser.TraitDecl) {
	t := &Trait{Name: d.Name}
	a.declareType(d.Name, t)
}

func (a *Analyzer) collectImpl(d parser.ImplDecl) {
	// Methods are namespaced by their receiver type to avoid clashing with free functions.
	// The key "Type::method" is stored in the value namespace of the enclosing scope.
	for _, method := range d.Methods {
		name := fmt.Sprintf("%s::%s", d.Type, method.Name)
		sym := a.newSymbol(SymMethod, name)
		sym.Type = &Function{}
		sym.Initialized = true
		sym.DeclNode = method
		// Use DeclareValue directly so impl methods do not appear in the flat symbol list.
		a.currentScope.DeclareValue(sym)
	}
}

func (a *Analyzer) collectVar(d parser.VarDecl) {
	sym := a.newSymbol(SymVar, d.Name)
	sym.Mutable = d.Mut
	sym.Initialized = d.Value != nil
	sym.DeclNode = d
	a.declareValue(sym)
}

// collectLocalItems registers placeholder symbols for any item declarations
// (fn, struct, trait, impl) that appear as statements inside a block.
// This implements the Rust rule that items declared inside a block are hoisted
// and visible throughout the entire enclosing block.
func (a *Analyzer) collectLocalItems(stmts []parser.Stmt) {
	for _, s := range stmts {
		if decl, ok := stmtAsDecl(s); ok {
			a.collectDecl(decl)
		}
	}
}

// stmtAsDecl type-asserts a Stmt to a Decl for the item-hoisting mini-pipeline.
// In this grammar, FuncDecl, StructDecl, TraitDecl, and ImplDecl all implement Stmt.
func stmtAsDecl(s parser.Stmt) (parser.Decl, bool) {
	switch d := s.(type) {
	case parser.FuncDecl:
		return d, true
	case parser.StructDecl:
		return d, true
	case parser.TraitDecl:
		return d, true
	case parser.ImplDecl:
		return d, true
	}
	return nil, false
}
