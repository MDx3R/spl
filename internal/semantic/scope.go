package semantic

// ScopeKind classifies the kind of a lexical scope.
type ScopeKind int

const (
	ScopeGlobal   ScopeKind = iota // top-level file scope
	ScopeFunction                   // function body scope
	ScopeBlock                      // anonymous block expression scope
	ScopeFor                        // for-loop scope (holds loop binding)
)

// Scope is a node in the persistent scope tree. It holds value (variable/function/method)
// and type (struct/trait) bindings for one lexical region. The tree is built during
// the Collect pass and remains available throughout all subsequent passes and IR generation.
type Scope struct {
	// ID is a unique monotonic counter assigned during scope creation.
	ID int
	// Name is the human-readable scope label used in diagnostics and FormatTable.
	Name string
	// Kind classifies the scope.
	Kind ScopeKind

	// Parent is the immediately enclosing scope (nil for the global scope).
	Parent *Scope
	// Children are all directly enclosed scopes, in creation order.
	Children []*Scope

	values map[string]*Symbol
	types  map[string]Type
}

// newChildScope creates a Scope, links it to parent as a child, and returns it.
func newChildScope(id int, name string, kind ScopeKind, parent *Scope) *Scope {
	s := &Scope{
		ID:     id,
		Name:   name,
		Kind:   kind,
		Parent: parent,
		values: make(map[string]*Symbol),
		types:  make(map[string]Type),
	}
	if parent != nil {
		parent.Children = append(parent.Children, s)
	}
	return s
}

// DeclareValue inserts sym into this scope's value namespace.
// Returns (existing, false) when the name is already declared in this exact scope.
func (s *Scope) DeclareValue(sym *Symbol) (*Symbol, bool) {
	if existing, ok := s.values[sym.Name]; ok {
		return existing, false
	}
	s.values[sym.Name] = sym
	return sym, true
}

// DeclareType inserts a named type into this scope's type namespace.
// Returns (existing, false) when the name is already declared in this exact scope.
func (s *Scope) DeclareType(name string, t Type) (Type, bool) {
	if existing, ok := s.types[name]; ok {
		return existing, false
	}
	s.types[name] = t
	return t, true
}

// LookupValue searches for a value symbol by walking from this scope up to the root.
// Returns nil when the name is not found in any enclosing scope.
func (s *Scope) LookupValue(name string) *Symbol {
	for sc := s; sc != nil; sc = sc.Parent {
		if sym, ok := sc.values[name]; ok {
			return sym
		}
	}
	return nil
}

// LookupType searches for a type by walking from this scope up to the root.
// Returns nil when the name is not found in any enclosing scope.
func (s *Scope) LookupType(name string) Type {
	for sc := s; sc != nil; sc = sc.Parent {
		if t, ok := sc.types[name]; ok {
			return t
		}
	}
	return nil
}
