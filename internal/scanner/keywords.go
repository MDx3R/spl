package scanner

var keywords = map[string]TokenKind{
	"break":    Break,
	"const":    Const,
	"continue": Continue,
	"else":     Else,
	"enum":     Enum,
	"extern":   Extern,
	"false":    False,
	"fn":       Fn,
	"for":      For,
	"if":       If,
	"impl":     Impl,
	"in":       In,
	"let":      Let,
	"loop":     Loop,
	"match":    Match,
	"mod":      Mod,
	"move":     Move,
	"mut":      Mut,
	"pub":      Pub,
	"return":   Return,
	"self":     SelfLower,
	"Self":     SelfUpper,
	"static":   Static,
	"struct":   Struct,
	"super":    Super,
	"trait":    Trait,
	"true":     True,
	"type":     Type,
	"use":      Use,
	"while":    While,
}

func LookupKeyword(ident string) (TokenKind, bool) {
	k, ok := keywords[ident]
	return k, ok
}
