package semantic

import (
	"fmt"
	"strings"

	"github.com/MDx3R/spl/internal/parser"
)

// Analyzer orchestrates the three-pass semantic analysis pipeline:
//
//	Pass 1 (Collect)  – builds the scope tree and registers placeholder symbols.
//	Pass 2 (Resolve)  – resolves all type names and fills in signatures.
//	Pass 2.5          – detects infinitely recursive struct types.
//	Pass 3 (Check)    – bottom-up type checking of all expressions and statements.
//
// Results are exposed via the public API and through Result() for downstream passes.
type Analyzer struct {
	globalScope  *Scope
	currentScope *Scope

	// symbols collects every declared value symbol in declaration order.
	// This is the flat list returned by Symbols() and formatted by FormatTable().
	symbols []*Symbol
	// errors accumulates semantic diagnostics across all passes.
	errors []SemanticError

	// fnStack tracks the active function contexts for return-type checking.
	fnStack []*fnContext
	// implType is the name of the type currently being impl'd; used to resolve Self.
	implType string
	// loopDepth counts loop nesting for break/continue validation.
	loopDepth int

	// scopeID is a monotonically increasing counter used to assign unique IDs to scopes.
	scopeID int
	// nextSymID is a monotonically increasing counter used to assign unique IDs to symbols.
	nextSymID int
}

// fnContext carries information about the function currently being type-checked.
type fnContext struct {
	Name    string
	RetType Type
}

// NewAnalyzer creates an Analyzer with an empty global scope.
func NewAnalyzer() *Analyzer {
	a := &Analyzer{}
	a.globalScope = a.newScope("global", ScopeGlobal, nil)
	a.currentScope = a.globalScope
	return a
}

// --- Public API ---

// Symbols returns all declared value symbols in declaration order.
func (a *Analyzer) Symbols() []*Symbol { return a.symbols }

// Errors returns all semantic diagnostics produced by the analysis.
func (a *Analyzer) Errors() []SemanticError { return a.errors }

// Result returns the complete analysis result for consumption by later passes (e.g., IR generation).
func (a *Analyzer) Result() *AnalysisResult {
	return &AnalysisResult{
		RootScope: a.globalScope,
		Symbols:   a.symbols,
		Errors:    a.errors,
	}
}

// Analyze runs the full semantic analysis pipeline on f.
func (a *Analyzer) Analyze(f *parser.File) {
	a.collectFile(f)
	a.resolveFile(f)
	a.checkRecursiveStructs()
	a.checkFile(f)
}

// AnalyzeFile is a backward-compatibility alias for Analyze.
func (a *Analyzer) AnalyzeFile(f *parser.File) { a.Analyze(f) }

// FormatTable produces a human-readable symbol table.
func (a *Analyzer) FormatTable() string {
	var sb strings.Builder
	fmt.Fprintln(&sb, "Symbol Table:")
	header := fmt.Sprintf("%-20s | %-8s | %-5s | %-5s | %s", "Name", "Type", "Mut", "Init", "Scope")
	fmt.Fprintln(&sb, header)
	fmt.Fprintln(&sb, strings.Repeat("-", len(header)))
	for _, sym := range a.symbols {
		typ := "unknown"
		if sym.Type != nil {
			typ = sym.Type.String()
		}
		fmt.Fprintf(&sb, "%-20s | %-8s | %-5s | %-5s | %s\n",
			sym.Name, typ, boolStr(sym.Mutable), boolStr(sym.Initialized), sym.ScopeName)
	}
	return sb.String()
}

// FormatErrors produces a human-readable semantic analysis summary.
func (a *Analyzer) FormatErrors() string {
	var sb strings.Builder
	if len(a.errors) == 0 {
		fmt.Fprintln(&sb, "Semantic Analysis: OK")
	} else {
		fmt.Fprintln(&sb, "Semantic Analysis:")
		for _, e := range a.errors {
			fmt.Fprintf(&sb, "  %s: %s\n", e.Kind, e.Message)
		}
	}
	return sb.String()
}

// --- Internal helpers shared across passes ---

// newScope allocates a new child scope linked to parent with a unique ID.
func (a *Analyzer) newScope(name string, kind ScopeKind, parent *Scope) *Scope {
	a.scopeID++
	return newChildScope(a.scopeID, name, kind, parent)
}

// pushScope creates a new child of currentScope and makes it current.
func (a *Analyzer) pushScope(name string, kind ScopeKind) {
	a.currentScope = a.newScope(name, kind, a.currentScope)
}

// popScope restores the parent scope, panicking if already at root.
func (a *Analyzer) popScope() {
	if a.currentScope.Parent != nil {
		a.currentScope = a.currentScope.Parent
	}
}

// declareValue registers sym in the current scope's value namespace and appends it
// to the flat symbol list. Records an ErrDuplicate diagnostic on collision.
// Returns false on collision.
func (a *Analyzer) declareValue(sym *Symbol) bool {
	sym.ScopeName = a.currentScope.Name
	_, ok := a.currentScope.DeclareValue(sym)
	if !ok {
		a.addError(ErrDuplicate,
			fmt.Sprintf("identifier '%s' is already declared in this scope", sym.Name))
		return false
	}
	a.symbols = append(a.symbols, sym)
	return true
}

// declareType registers t in the current scope's type namespace.
// Records an ErrDuplicate diagnostic on collision. Returns false on collision.
func (a *Analyzer) declareType(name string, t Type) bool {
	_, ok := a.currentScope.DeclareType(name, t)
	if !ok {
		a.addError(ErrDuplicate,
			fmt.Sprintf("type '%s' is already declared in this scope", name))
		return false
	}
	return true
}

// addError appends a semantic diagnostic.
func (a *Analyzer) addError(kind ErrorKind, msg string) {
	a.errors = append(a.errors, SemanticError{Kind: kind, Message: msg})
}

// newSymbol allocates a Symbol with a unique ID and the given kind+name.
func (a *Analyzer) newSymbol(kind SymbolKind, name string) *Symbol {
	a.nextSymID++
	return &Symbol{ID: a.nextSymID, Kind: kind, Name: name}
}

// currentFunc returns the innermost function context, or nil if not inside a function.
func (a *Analyzer) currentFunc() *fnContext {
	if len(a.fnStack) == 0 {
		return nil
	}
	return a.fnStack[len(a.fnStack)-1]
}

// boolStr formats a bool for the symbol table.
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
