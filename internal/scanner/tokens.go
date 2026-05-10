package scanner

type TokenKind uint

const (
	EOF TokenKind = iota

	// Comments
	LineComment
	BlockComment
	OuterLineDoc
	InnerLineDoc
	OuterBlockDoc
	InnerBlockDoc

	// Identifiers and literals
	Name
	IntLit
	FloatLit
	StrLit
	CharLit

	// Delimiters
	Lparen
	Rparen
	Lbrace
	Rbrace
	Lbrack
	Rbrack

	// Structural symbols
	Comma
	Semi
	Colon
	Dot
	DotDot
	DotDotEq
	ThinArrow
	ThickArrow
	Pound
	Question

	// Assignment operators
	Eq
	MinusEq
	PlusEq
	StarEq
	SlashEq
	PercentEq
	AndEq
	OrEq
	CaretEq
	ShlEq
	ShrEq

	// Comparison operators
	EqEq
	NotEq
	Lt
	LtEq
	Gt
	GtEq

	// Arithmetic operators
	Minus
	Plus
	Star
	Slash
	Percent

	// Bitwise operators
	And
	Or
	Caret
	Shl
	Shr

	// Logical operators
	Not
	AndAnd
	OrOr

	// Keywords
	Break
	Const
	Continue
	Else
	Enum
	Extern
	False
	Fn
	For
	If
	Impl
	In
	Let
	Loop
	Match
	Mod
	Move
	Mut
	Pub
	Return
	SelfLower
	SelfUpper
	Static
	Struct
	Super
	Trait
	True
	Type
	Use
	While
	Union

	Invalid
)

var tokenKindNames = [...]string{
	EOF:           "EOF",
	LineComment:   "LineComment",
	BlockComment:  "BlockComment",
	OuterLineDoc:  "OuterLineDoc",
	InnerLineDoc:  "InnerLineDoc",
	OuterBlockDoc: "OuterBlockDoc",
	InnerBlockDoc: "InnerBlockDoc",
	Name:          "Name",
	IntLit:        "IntLit",
	FloatLit:      "FloatLit",
	StrLit:        "StrLit",
	CharLit:       "CharLit",
	Lparen:        "(",
	Rparen:        ")",
	Lbrace:        "{",
	Rbrace:        "}",
	Lbrack:        "[",
	Rbrack:        "]",
	Comma:         ",",
	Semi:          ";",
	Colon:         ":",
	Dot:           ".",
	DotDot:        "..",
	DotDotEq:      "..=",
	ThinArrow:     "->",
	ThickArrow:    "=>",
	Pound:         "#",
	Question:      "?",
	Eq:            "=",
	MinusEq:       "-=",
	PlusEq:        "+=",
	StarEq:        "*=",
	SlashEq:       "/=",
	PercentEq:     "%=",
	AndEq:         "&=",
	OrEq:          "|=",
	CaretEq:       "^=",
	ShlEq:         "<<=",
	ShrEq:         ">>=",
	EqEq:          "==",
	NotEq:         "!=",
	Lt:            "<",
	LtEq:          "<=",
	Gt:            ">",
	GtEq:          ">=",
	Minus:         "-",
	Plus:          "+",
	Star:          "*",
	Slash:         "/",
	Percent:       "%",
	And:           "&",
	Or:            "|",
	Caret:         "^",
	Shl:           "<<",
	Shr:           ">>",
	Not:           "!",
	AndAnd:        "&&",
	OrOr:          "||",
	Break:         "break",
	Const:         "const",
	Continue:      "continue",
	Else:          "else",
	Enum:          "enum",
	Extern:        "extern",
	False:         "false",
	Fn:            "fn",
	For:           "for",
	If:            "if",
	Impl:          "impl",
	In:            "in",
	Let:           "let",
	Loop:          "loop",
	Match:         "match",
	Mod:           "mod",
	Move:          "move",
	Mut:           "mut",
	Pub:           "pub",
	Return:        "return",
	SelfLower:     "self",
	SelfUpper:     "Self",
	Static:        "static",
	Struct:        "struct",
	Super:         "super",
	Trait:         "trait",
	True:          "true",
	Type:          "type",
	Use:           "use",
	While:         "while",
	Union:         "union",
	Invalid:       "Invalid",
}

func (k TokenKind) String() string {
	if int(k) < len(tokenKindNames) {
		return tokenKindNames[k]
	}
	return "unknown"
}

type Token struct {
	Kind      TokenKind
	Lit       string
	Line, Col uint
}

func (t Token) IsComment() bool {
	return (t.Kind >= LineComment && t.Kind <= InnerBlockDoc)
}
