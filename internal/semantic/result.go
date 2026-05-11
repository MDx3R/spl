package semantic

// AnalysisResult is the immutable output of the semantic analysis pipeline.
// It is passed to subsequent compiler phases such as IR (triad) generation,
// giving them access to the resolved scope tree and symbol table without
// requiring them to repeat any analysis.
type AnalysisResult struct {
	// RootScope is the root of the persistent scope tree built during the Collect pass.
	// Callers may traverse it to look up symbols by name at any scope level.
	RootScope *Scope

	// Symbols lists every declared symbol in declaration order, mirroring the
	// rows of FormatTable. Methods registered via impl blocks are excluded from
	// this list (they are accessible through the scope tree).
	Symbols []*Symbol

	// Errors collects all semantic diagnostics accumulated during all passes.
	Errors []SemanticError
}
