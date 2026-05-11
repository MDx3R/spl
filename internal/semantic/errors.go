package semantic

import "fmt"

// ErrorKind categorises a semantic diagnostic.
type ErrorKind string

const (
	ErrUndeclared    ErrorKind = "undeclared"
	ErrTypeMismatch  ErrorKind = "type_mismatch"
	ErrDuplicate     ErrorKind = "duplicate"
	ErrImmutable     ErrorKind = "immutable_assign"
	ErrRecursiveType ErrorKind = "recursive_type"
	// ErrInvalidControl covers break/continue outside a loop and return outside a function.
	ErrInvalidControl ErrorKind = "invalid_control"
	// ErrUninitialized covers use of a variable before it is initialized.
	ErrUninitialized ErrorKind = "uninitialized"
)

// SemanticError is a single semantic diagnostic produced during analysis.
type SemanticError struct {
	Kind    ErrorKind
	Message string
}

func (e SemanticError) Error() string {
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}
