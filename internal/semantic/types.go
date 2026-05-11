package semantic

import "github.com/MDx3R/spl/internal/scanner"

// Type is the base interface for all semantic types in the type system.
type Type interface {
	String() string
	Equals(other Type) bool
}

// --- Scalar types ---

// ScalarKind enumerates all primitive scalar types.
type ScalarKind uint

const (
	ScalarUnknown ScalarKind = iota // unresolved / inferred-as-unknown
	ScalarI8
	ScalarI16
	ScalarI32
	ScalarI64
	ScalarI128
	ScalarIsize
	ScalarU8
	ScalarU16
	ScalarU32
	ScalarU64
	ScalarU128
	ScalarUsize
	ScalarF32
	ScalarF64
	ScalarBool
	ScalarChar
	ScalarString
	ScalarUnit
)

func (k ScalarKind) String() string {
	switch k {
	case ScalarI8:
		return "i8"
	case ScalarI16:
		return "i16"
	case ScalarI32:
		return "i32"
	case ScalarI64:
		return "i64"
	case ScalarI128:
		return "i128"
	case ScalarIsize:
		return "isize"
	case ScalarU8:
		return "u8"
	case ScalarU16:
		return "u16"
	case ScalarU32:
		return "u32"
	case ScalarU64:
		return "u64"
	case ScalarU128:
		return "u128"
	case ScalarUsize:
		return "usize"
	case ScalarF32:
		return "f32"
	case ScalarF64:
		return "f64"
	case ScalarBool:
		return "bool"
	case ScalarChar:
		return "char"
	case ScalarString:
		return "string"
	case ScalarUnit:
		return "unit"
	default:
		return "unknown"
	}
}

// Scalar represents a primitive scalar type.
type Scalar struct {
	Kind ScalarKind
}

func (s *Scalar) String() string { return s.Kind.String() }

func (s *Scalar) Equals(other Type) bool {
	o, ok := other.(*Scalar)
	return ok && s.Kind == o.Kind
}

// --- Invalid type (error sentinel) ---

// Invalid is assigned to expressions after a type error to prevent cascade diagnostics.
// Any check involving Invalid is skipped silently.
type Invalid struct{}

func (i *Invalid) String() string         { return "invalid" }
func (i *Invalid) Equals(_ Type) bool     { return false }

// --- Struct type ---

// Struct is a named product type with ordered named fields.
type Struct struct {
	Name   string
	Fields []FieldDef
}

// FieldDef is one field of a Struct type.
type FieldDef struct {
	Name string
	Type Type
}

func (s *Struct) String() string { return s.Name }

func (s *Struct) Equals(other Type) bool {
	o, ok := other.(*Struct)
	return ok && s.Name == o.Name
}

// --- Function type ---

// Function is the type of a function or method.
type Function struct {
	Params []Type
	Ret    Type
}

func (f *Function) String() string {
	ret := "<unresolved>"
	if f.Ret != nil {
		ret = f.Ret.String()
	}
	if len(f.Params) == 0 {
		return "fn() -> " + ret
	}
	s := "fn("
	for i, p := range f.Params {
		if i > 0 {
			s += ", "
		}
		s += p.String()
	}
	return s + ") -> " + ret
}

func (f *Function) Equals(other Type) bool {
	o, ok := other.(*Function)
	if !ok || len(f.Params) != len(o.Params) {
		return false
	}
	for i, p := range f.Params {
		if !p.Equals(o.Params[i]) {
			return false
		}
	}
	switch {
	case f.Ret == nil && o.Ret == nil:
		return true
	case f.Ret == nil || o.Ret == nil:
		return false
	default:
		return f.Ret.Equals(o.Ret)
	}
}

// --- Array type ---

// Array is a homogeneous sequence type with an element type.
type Array struct {
	Elem Type
}

func (a *Array) String() string { return "[" + a.Elem.String() + "]" }

func (a *Array) Equals(other Type) bool {
	o, ok := other.(*Array)
	return ok && a.Elem.Equals(o.Elem)
}

// --- Range type ---

// Range is the type produced by a range expression (lo..hi or lo..=hi).
type Range struct {
	Elem Type // element type inferred from bounds
}

func (r *Range) String() string { return "range" }

func (r *Range) Equals(other Type) bool { _, ok := other.(*Range); return ok }

// --- Trait type ---

// Trait is a trait type (used as a placeholder; trait bounds are out of scope).
type Trait struct {
	Name string
}

func (t *Trait) String() string { return t.Name }

func (t *Trait) Equals(other Type) bool {
	o, ok := other.(*Trait)
	return ok && t.Name == o.Name
}

// --- Predefined type values ---

var (
	unit    = &Scalar{Kind: ScalarUnit}
	unknown = &Scalar{Kind: ScalarUnknown}
	invalid = &Invalid{}
)

// --- Type predicates ---

// isUnknown returns true for the unresolved scalar type.
func isUnknown(t Type) bool {
	s, ok := t.(*Scalar)
	return ok && s.Kind == ScalarUnknown
}

// isInvalid returns true for the error-sentinel type.
func isInvalid(t Type) bool { _, ok := t.(*Invalid); return ok }

// isUnknownOrInvalid absorbs both unresolved and error states.
func isUnknownOrInvalid(t Type) bool { return t == nil || isUnknown(t) || isInvalid(t) }

// isNumericKind returns true for all integer and floating-point scalar kinds.
func isNumericKind(k ScalarKind) bool {
	switch k {
	case ScalarI8, ScalarI16, ScalarI32, ScalarI64, ScalarI128, ScalarIsize,
		ScalarU8, ScalarU16, ScalarU32, ScalarU64, ScalarU128, ScalarUsize,
		ScalarF32, ScalarF64, ScalarChar:
		return true
	}
	return false
}

// isNumeric returns true for all numeric (integer/float) scalar types.
func isNumeric(t Type) bool {
	s, ok := t.(*Scalar)
	return ok && isNumericKind(s.Kind)
}

// isBool returns true for the bool scalar type.
func isBool(t Type) bool {
	s, ok := t.(*Scalar)
	return ok && s.Kind == ScalarBool
}

// --- Type resolution from names ---

// resolveTypeName maps a built-in type name to a concrete Type.
// Returns nil for user-defined or unknown names.
func resolveTypeName(name string) Type {
	switch name {
	case "i8":
		return &Scalar{Kind: ScalarI8}
	case "i16":
		return &Scalar{Kind: ScalarI16}
	case "i32":
		return &Scalar{Kind: ScalarI32}
	case "i64":
		return &Scalar{Kind: ScalarI64}
	case "i128":
		return &Scalar{Kind: ScalarI128}
	case "isize":
		return &Scalar{Kind: ScalarIsize}
	case "u8":
		return &Scalar{Kind: ScalarU8}
	case "u16":
		return &Scalar{Kind: ScalarU16}
	case "u32":
		return &Scalar{Kind: ScalarU32}
	case "u64":
		return &Scalar{Kind: ScalarU64}
	case "u128":
		return &Scalar{Kind: ScalarU128}
	case "usize":
		return &Scalar{Kind: ScalarUsize}
	case "f32":
		return &Scalar{Kind: ScalarF32}
	case "f64":
		return &Scalar{Kind: ScalarF64}
	case "bool":
		return &Scalar{Kind: ScalarBool}
	case "char":
		return &Scalar{Kind: ScalarChar}
	case "str", "String":
		return &Scalar{Kind: ScalarString}
	case "()":
		return &Scalar{Kind: ScalarUnit}
	}
	return nil
}

// inferLiteralType infers the Type from a literal token kind.
// Integer literals default to i32 (like Rust); float literals default to f64.
func inferLiteralType(kind scanner.TokenKind, v any) Type {
	switch kind {
	case scanner.True, scanner.False:
		return &Scalar{Kind: ScalarBool}
	case scanner.StrLit:
		return &Scalar{Kind: ScalarString}
	case scanner.CharLit:
		return &Scalar{Kind: ScalarChar}
	case scanner.IntLit:
		return &Scalar{Kind: ScalarI32}
	case scanner.FloatLit:
		return &Scalar{Kind: ScalarF64}
	}
	switch v.(type) {
	case bool:
		return &Scalar{Kind: ScalarBool}
	case float64:
		return &Scalar{Kind: ScalarF64}
	}
	return &Scalar{Kind: ScalarI32}
}

// --- Arithmetic helpers ---

// arithmeticResult returns the result type of an arithmetic binary expression.
// If either operand is unknown, returns unknown (suppresses cascade errors).
// If operands are incompatible, returns invalid.
func arithmeticResult(left, right Type) Type {
	if isUnknownOrInvalid(left) || isUnknownOrInvalid(right) {
		return unknown
	}
	ls, lok := left.(*Scalar)
	rs, rok := right.(*Scalar)
	if lok && rok && isNumericKind(ls.Kind) && ls.Kind == rs.Kind {
		return left
	}
	return invalid
}

// bitwiseResult returns the result type of a bitwise binary expression.
func bitwiseResult(left, right Type) Type {
	return arithmeticResult(left, right)
}

// --- Operator classification ---

// isArithmeticOp returns true for +, -, *, /, %.
func isArithmeticOp(k scanner.TokenKind) bool {
	switch k {
	case scanner.Plus, scanner.Minus, scanner.Star, scanner.Slash, scanner.Percent:
		return true
	}
	return false
}

// isComparisonOp returns true for ==, !=, <, >, <=, >=.
func isComparisonOp(k scanner.TokenKind) bool {
	switch k {
	case scanner.EqEq, scanner.NotEq, scanner.Lt, scanner.LtEq, scanner.Gt, scanner.GtEq:
		return true
	}
	return false
}

// isLogicalOp returns true for && and ||.
func isLogicalOp(k scanner.TokenKind) bool {
	return k == scanner.AndAnd || k == scanner.OrOr
}

// isBitwiseOp returns true for &, |, ^, <<, >>.
func isBitwiseOp(k scanner.TokenKind) bool {
	switch k {
	case scanner.And, scanner.Or, scanner.Caret, scanner.Shl, scanner.Shr:
		return true
	}
	return false
}

// compoundBaseOp returns the arithmetic operator for a compound assignment operator.
func compoundBaseOp(k scanner.TokenKind) string {
	switch k {
	case scanner.PlusEq:
		return "+"
	case scanner.MinusEq:
		return "-"
	case scanner.StarEq:
		return "*"
	case scanner.SlashEq:
		return "/"
	case scanner.PercentEq:
		return "%"
	case scanner.AndEq:
		return "&"
	case scanner.OrEq:
		return "|"
	case scanner.CaretEq:
		return "^"
	case scanner.ShlEq:
		return "<<"
	case scanner.ShrEq:
		return ">>"
	default:
		return k.String()
	}
}

// structsInType recursively extracts *Struct types embedded in t (for recursive struct detection).
func structsInType(t Type) []*Struct {
	switch t := t.(type) {
	case *Struct:
		return []*Struct{t}
	case *Array:
		return structsInType(t.Elem)
	}
	return nil
}
