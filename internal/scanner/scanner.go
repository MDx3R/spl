package scanner

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

type Scanner struct {
	buf  *bufio.Reader
	errh func(line, col uint, msg string)

	ch              rune
	line, col       uint
	tokLine, tokCol uint
}

func NewScanner(src io.Reader) *Scanner {
	s := &Scanner{
		buf:  bufio.NewReader(src),
		line: 1,
	}
	s.errh = func(line, col uint, msg string) {
		fmt.Fprintf(os.Stderr, "%d:%d: %s\n", line, col, msg)
	}
	// TODO: uncomment when scanner client (parser) is implemented
	// s.consume()
	return s
}

func (s *Scanner) Init() {
	s.consume()
}

func (s *Scanner) errorf(format string, args ...any) {
	s.errh(s.line, s.col, fmt.Sprintf(format, args...))
}

func (s *Scanner) Next() Token {
	for {
		s.skipWhitespaces()

		s.tokLine, s.tokCol = s.line, s.col

		if s.isAtEnd() {
			return s.newToken(EOF)
		}

		curr := s.current()

		if s.isIdentStart(curr) {
			return s.ident()
		}
		if s.isDigit(curr) {
			return s.number()
		}

		s.consume()

		switch curr {
		case '(':
			return s.newToken(Lparen)
		case ')':
			return s.newToken(Rparen)
		case '{':
			return s.newToken(Lbrace)
		case '}':
			return s.newToken(Rbrace)
		case '[':
			return s.newToken(Lbrack)
		case ']':
			return s.newToken(Rbrack)
		case ',':
			return s.newToken(Comma)
		case '.':
			return s.newToken(Dot)
		case ';':
			return s.newToken(Semi)
		case ':':
			return s.newToken(Colon)
		case '=':
			return s.switchOp(Eq, EqEq)
		case '<':
			return s.switchOp(Lt, LtEq)
		case '>':
			return s.switchOp(Gt, GtEq)
		case '!':
			return s.switchOp(Not, NotEq)
		case '-':
			return s.switchOp(Minus, MinusEq)
		case '+':
			return s.switchOp(Plus, PlusEq)
		case '*':
			return s.switchOp(Star, StarEq)
		case '%':
			return s.switchOp(Percent, PercentEq)
		case '^':
			return s.switchOp(Caret, CaretEq)
		case '&':
			if s.match('&') {
				return s.newToken(AndAnd)
			}
			return s.switchOp(And, AndEq)
		case '|':
			if s.match('|') {
				return s.newToken(OrOr)
			}
			return s.switchOp(Or, OrEq)
		case '\'':
			return s.stdChar()
		case '"':
			return s.stdString()
		case '/':
			if s.match('/') {
				tok := s.lineComment()
				if tok.Kind == LineComment {
					continue
				}
				return tok
			}
			if s.match('*') {
				tok := s.blockComment()
				if tok.Kind == BlockComment {
					continue
				}
				return tok
			}
			return s.switchOp(Slash, SlashEq)
		default:
			s.errorf("invalid character %#U", curr)
		}
	}
}

func (s *Scanner) skipWhitespaces() {
	for !s.isAtEnd() {
		switch s.current() {
		case ' ', '\t', '\n', '\r':
			s.consume()
		default:
			return
		}
	}
}

func (s *Scanner) consume() {
	if s.ch == '\n' {
		s.line++
		s.col = 0
	}
	r, _, err := s.buf.ReadRune()
	if err != nil {
		s.ch = -1
		return
	}
	s.ch = r
	s.col++
}

func (s *Scanner) match(ch rune) bool {
	if s.isAtEnd() || ch != s.current() {
		return false
	}

	s.consume()
	return true
}

// lineComment is called after "//" has been fully consumed.
// current() is the first character of the comment body (or '/', '!').
func (s *Scanner) lineComment() Token {
	var kind TokenKind
	switch s.current() {
	case '!':
		kind = InnerLineDoc
		s.consume()
	case '/':
		s.consume() // consume third '/'
		if s.current() == '/' {
			// "////" or more — plain comment
			kind = LineComment
		} else {
			// "///" followed by non-'/' — outer doc
			kind = OuterLineDoc
		}
	default:
		kind = LineComment
	}

	var sb strings.Builder
	for !s.atLineEnd() {
		sb.WriteRune(s.current())
		s.consume()
	}

	return s.newLiteralToken(kind, sb.String())
}

// blockComment is called after "/*" has been fully consumed.
// current() is the first character after "/*".
func (s *Scanner) blockComment() Token {
	var kind TokenKind
	switch s.current() {
	case '!':
		s.consume()
		kind = InnerBlockDoc
	case '*':
		s.consume() // consume second '*'
		switch s.current() {
		case '/':
			// "/**/" — empty plain block comment
			s.consume()
			return s.newLiteralToken(BlockComment, "")
		case '*':
			// "/***/..." — plain comment (third '*' means not outer doc)
			// no need to consume the third '*'
			kind = BlockComment
		default:
			// "/**" followed by other — outer doc
			kind = OuterBlockDoc
		}
	default:
		kind = BlockComment
	}

	var sb strings.Builder
	depth := 1
	for depth > 0 {
		if s.isAtEnd() {
			s.errorf("unterminated block comment")
			break
		}
		ch := s.current()
		s.consume()
		if ch == '/' && s.current() == '*' {
			s.consume()
			depth++
			// open nested block comment
			sb.WriteString("/*")
		} else if ch == '*' && s.current() == '/' {
			s.consume()
			depth--
			// close nested block comment
			if depth > 0 {
				sb.WriteString("*/")
			}
		} else {
			sb.WriteRune(ch)
		}
	}

	return s.newLiteralToken(kind, sb.String())
}

// stdString is called after the opening '"' has been consumed.
func (s *Scanner) stdString() Token {
	var sb strings.Builder
	for {
		if s.atLineEnd() {
			s.errorf("unterminated string literal")
			break
		}

		switch s.current() {
		case '"':
			s.consume()
			return s.newLiteralToken(StrLit, sb.String())
		case '\\':
			s.consume()
			sb.WriteRune(s.readEscape())
		default:
			sb.WriteRune(s.current())
			s.consume()
		}
	}

	return s.newLiteralToken(StrLit, sb.String())
}

// stdChar is called after the opening '\” has been consumed.
func (s *Scanner) stdChar() Token {
	if s.atLineEnd() {
		s.errorf("unterminated char literal")
		return s.newToken(CharLit)
	}

	if s.current() == '\'' {
		s.errorf("empty char literal")
		s.consume()
		return s.newToken(CharLit)
	}

	var ch rune
	if s.current() == '\\' {
		s.consume()
		ch = s.readEscape()
	} else {
		ch = s.current()
		s.consume()
	}

	switch {
	case s.current() == '\'':
		s.consume()
	case s.atLineEnd():
		s.errorf("unterminated char literal")
	default:
		s.errorf("char literal contains more than one character")
		// consuming the rest of the invalid char literal
		for !s.atLineEnd() && s.current() != '\'' {
			s.consume()
		}
		if s.current() == '\'' {
			s.consume()
		}
	}

	return s.newLiteralToken(CharLit, string(ch))
}

func (s *Scanner) readEscape() rune {
	if s.isAtEnd() {
		s.errorf("unterminated escape sequence")
		return 0
	}
	ch := s.current()
	s.consume()
	switch ch {
	case 'n':
		return '\n'
	case 't':
		return '\t'
	case 'r':
		return '\r'
	case '\\':
		return '\\'
	case '"':
		return '"'
	case '\'':
		return '\''
	case '0':
		return 0
	default:
		s.errorf("unknown escape sequence \\%c", ch)
		return ch
	}
}

func (s *Scanner) number() Token {
	var sb strings.Builder
	for s.isDigit(s.current()) {
		sb.WriteRune(s.current())
		s.consume()
	}

	isFloat := false

	switch s.current() {
	case '.':
		// Peek at the character after '.'. If it's a digit, this is a float;
		// otherwise treat as IntLit + Dot (preserves "0..5" range syntax).
		if r := s.peek(); r >= '0' && r <= '9' {
			isFloat = true
			sb.WriteRune('.')
			s.consume()
			for s.isDigit(s.current()) {
				sb.WriteRune(s.current())
				s.consume()
			}
			s.collectExponent(&sb)
		}
	case 'e', 'E':
		isFloat = true
		s.collectExponent(&sb)
	}

	kind := IntLit
	if isFloat {
		kind = FloatLit
	}

	return s.newLiteralToken(kind, sb.String())
}

func (s *Scanner) collectExponent(sb *strings.Builder) {
	if s.current() != 'e' && s.current() != 'E' {
		return
	}

	sb.WriteRune(s.current())
	s.consume()
	if s.current() == '+' || s.current() == '-' {
		sb.WriteRune(s.current())
		s.consume()
	}
	if !s.isDigit(s.current()) {
		s.errorf("expected digit in float exponent")
		return
	}

	for s.isDigit(s.current()) {
		sb.WriteRune(s.current())
		s.consume()
	}
}

func (s *Scanner) ident() Token {
	var sb strings.Builder
	for s.isIdentCont(s.current()) {
		sb.WriteRune(s.current())
		s.consume()
	}

	lit := sb.String()
	if kw, ok := LookupKeyword(lit); ok {
		return s.newToken(kw)
	}

	return s.newLiteralToken(Name, lit)
}

// switchOp handles the common (Op, OpEq) pattern.
func (s *Scanner) switchOp(tok, tokEq TokenKind) Token {
	if s.match('=') {
		return s.newToken(tokEq)
	}
	return s.newToken(tok)
}

func (s *Scanner) isAlpha(ch rune) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z')
}

func (s *Scanner) isDigit(ch rune) bool { return '0' <= ch && ch <= '9' }

func (s *Scanner) isIdentCont(ch rune) bool {
	return s.isAlpha(ch) || s.isDigit(ch) || ch == '_'
}

func (s *Scanner) isIdentStart(ch rune) bool {
	return s.isAlpha(ch) || ch == '_'
}

func (s *Scanner) newToken(kind TokenKind) Token {
	return Token{Kind: kind, Line: s.tokLine, Col: s.tokCol}
}

func (s *Scanner) newLiteralToken(kind TokenKind, lit string) Token {
	tok := s.newToken(kind)
	tok.Lit = lit
	return tok
}

func (s *Scanner) atLineEnd() bool { return s.current() == '\n' || s.isAtEnd() }
func (s *Scanner) isAtEnd() bool   { return s.ch == -1 }

// current returns the rune currently under the scanner cursor (already read
// from the underlying reader). Returns -1 when the input is exhausted.
func (s *Scanner) current() rune { return s.ch }

// peek returns the next rune in the input without consuming it, i.e. the rune
// that will become current() after the next consume(). Returns r on
// success, or -1 when the buffer holds no further data.
func (s *Scanner) peek() rune {
	p, _ := s.buf.Peek(utf8.UTFMax)
	if len(p) == 0 {
		return -1
	}
	r, _ := utf8.DecodeRune(p)
	return r
}
