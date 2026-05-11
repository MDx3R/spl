package semantic

import "github.com/MDx3R/spl/internal/parser"

// SymbolKind classifies a symbol by its declaration context.
type SymbolKind int

const (
	SymVar    SymbolKind = iota // let / let mut binding
	SymFunc                     // fn declaration
	SymType                     // struct / trait declaration (value-namespace entry)
	SymParam                    // function or method parameter
	SymMethod                   // impl method
)

// Symbol records all semantic metadata for a declared identifier.
type Symbol struct {
	// ID is a monotonically increasing unique identifier, useful for debugging.
	ID int
	// Kind classifies how the symbol was declared.
	Kind SymbolKind
	// Name is the source-level identifier text.
	Name string
	// Type is the resolved semantic type (nil before resolution, unknown after a failed resolution).
	Type Type
	// Mutable is true for let-mut bindings.
	Mutable bool
	// Initialized is true when the binding is guaranteed to hold a value.
	Initialized bool
	// ScopeName is the display name of the scope where this symbol was declared.
	// Used by FormatTable.
	ScopeName string
	// DeclNode is the AST node of the declaration site (may be nil for synthetic symbols).
	DeclNode parser.Node
}
