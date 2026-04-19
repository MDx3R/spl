package scanner

type TokenKind uint

const (
	EOF TokenKind = iota

	// comments
	LineComment
	BlockComment
	OuterLineDoc
	InnerLineDoc
	OuterBlockDoc
	InnerBlockDoc

	// names and literals
	Name
	IntLit
	FloatLit
	StrLit
	CharLit

	// delimiters
	Lparen
	Rparen
	Lbrace
	Rbrace
	Lbrack
	Rbrack
	Comma
	Dot
	Semi
	Colon

	// operators
	Eq
	EqEq
	Lt
	LtEq
	Gt
	GtEq
	Not
	NotEq

	Minus
	Plus
	Star
	Slash
	Percent

	And
	AndAnd
	Or
	OrOr
	Caret

	// assignments
	MinusEq
	PlusEq
	StarEq
	SlashEq
	PercentEq

	AndEq
	OrEq
	CaretEq

	// keywords
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
	Dot:           ".",
	Semi:          ";",
	Colon:         ":",
	Eq:            "=",
	EqEq:          "==",
	Lt:            "<",
	LtEq:          "<=",
	Gt:            ">",
	GtEq:          ">=",
	Not:           "!",
	NotEq:         "!=",
	Minus:         "-",
	Plus:          "+",
	Star:          "*",
	Slash:         "/",
	Percent:       "%",
	And:           "&",
	AndAnd:        "&&",
	Or:            "|",
	OrOr:          "||",
	Caret:         "^",
	MinusEq:       "-=",
	PlusEq:        "+=",
	StarEq:        "*=",
	SlashEq:       "/=",
	PercentEq:     "%=",
	AndEq:         "&=",
	OrEq:          "|=",
	CaretEq:       "^=",
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
