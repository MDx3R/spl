package parser

import (
	"fmt"
	"slices"

	"github.com/MDx3R/spl/internal/scanner"
)

const MaxArgs = 65535

var stmtStart = map[scanner.TokenKind]bool{
	scanner.Break:    true,
	scanner.Continue: true,
	scanner.Return:   true,
	scanner.Let:      true,
	scanner.If:       true,
	scanner.While:    true,
	scanner.For:      true,
	scanner.Loop:     true,
	scanner.Lbrace:   true,
	scanner.Semi:     true,
}

var declStart = map[scanner.TokenKind]bool{
	scanner.Let:    true,
	scanner.Fn:     true,
	scanner.Struct: true,
	scanner.Trait:  true,
	scanner.Impl:   true,
	scanner.Pub:    true,
}

// itemStart is the recovery set for top-level declarations only.
// Deliberately excludes Let (valid only inside blocks) to avoid
// stopping recover() at a token topLevelDecl() cannot handle.
var itemStart = map[scanner.TokenKind]bool{
	scanner.Fn:     true,
	scanner.Struct: true,
	scanner.Trait:  true,
	scanner.Impl:   true,
	scanner.Pub:    true,
}

var paramRecover = map[scanner.TokenKind]bool{
	scanner.Comma:  true,
	scanner.Rparen: true,
	scanner.Fn:     true,
	scanner.Struct: true,
}

var fieldRecover = map[scanner.TokenKind]bool{
	scanner.Comma:  true,
	scanner.Rbrace: true,
	scanner.Fn:     true,
	scanner.Struct: true,
}

var argRecover = map[scanner.TokenKind]bool{
	scanner.Comma:  true,
	scanner.Rparen: true,
	scanner.Semi:   true,
}

var traitBodyRecover = map[scanner.TokenKind]bool{
	scanner.Fn:     true,
	scanner.Rbrace: true,
}

type Parser struct {
	scanner *scanner.Scanner
	tok     scanner.Token
	errh    func(tok scanner.Token, msg string)
}

func NewParser(scanner *scanner.Scanner, errh func(tok scanner.Token, msg string)) *Parser {
	return &Parser{scanner: scanner, errh: errh}
}

func (p *Parser) Parse() *File {
	p.scanner.Init()
	p.tok = p.scanner.Next()

	var decls []Decl
	for !p.isAtEnd() {
		start := p.current()
		decls = append(decls, p.topLevelDecl())
		// safety: prevent infinite loop if topLevelDecl made no progress
		if !p.isAtEnd() && p.current() == start {
			p.consume()
		}
	}
	return &File{Decls: decls}
}

func (p *Parser) parseVisibility() Visibility {
	if p.match(scanner.Pub) {
		return Visibility{Kind: VisPublic}
	}
	return Visibility{Kind: VisPrivate}
}

func (p *Parser) topLevelDecl() Decl {
	for p.current().IsComment() {
		p.consume()
	}

	vis := p.parseVisibility()

	tok := p.current()
	switch {
	case p.match(scanner.Fn):
		return p.funcDeclaration(vis)
	case p.match(scanner.Struct):
		// stub -- implemented later
		p.recover(declStart)
		return BadDecl{From: tok, To: p.current()}
	case p.match(scanner.Trait):
		// stub -- implemented later
		p.recover(declStart)
		return BadDecl{From: tok, To: p.current()}
	case p.match(scanner.Impl):
		// stub -- implemented later
		p.recover(declStart)
		return BadDecl{From: tok, To: p.current()}
	default:
		p.errorf("Expected top-level declaration (fn, struct, trait, impl).")
		p.recover(itemStart)
		return BadDecl{From: tok, To: p.current()}
	}
}

func (p *Parser) errorf(format string, args ...any) {
	p.errh(p.tok, fmt.Sprintf(format, args...))
}

func (p *Parser) statement() Stmt {
	if p.match(scanner.Let) {
		return p.varDeclaration()
	}

	if p.match(scanner.Const) {
		return nil
	}
	if p.match(scanner.Type) {
		return nil
	}
	if p.match(scanner.Mod) {
		return nil
	}
	if p.match(scanner.Use) {
		return nil
	}
	if p.match(scanner.Fn) {
		return p.funcDeclaration(Visibility{Kind: VisPrivate})
	}
	if p.match(scanner.Struct) {
		return nil
	}
	if p.match(scanner.Enum) {
		return nil
	}
	if p.match(scanner.Union) {
		return nil
	}
	if p.match(scanner.Static) {
		return nil
	}
	if p.match(scanner.Trait) {
		return nil
	}
	if p.match(scanner.Impl) {
		return nil
	}
	if p.match(scanner.Extern) {
		return nil
	}
	return p.expressionStatement()
}

func (p *Parser) varDeclaration() Decl {
	// fix: match optional "mut" first, then expect the name
	mut := p.match(scanner.Mut)

	nameTok := p.current()
	if !p.expect(scanner.Name, "Expect variable name.") {
		p.recover(stmtStart)
		return BadDecl{From: nameTok, To: p.current()}
	}
	name := nameTok.Lit

	if !p.expect(scanner.Eq, "Expect '=' after variable name.") {
		p.recover(stmtStart)
		return BadDecl{From: nameTok, To: p.current()}
	}

	value := p.expression()

	if !p.expect(scanner.Semi, "Expect ';' after variable declaration.") {
		p.recover(stmtStart)
		return BadDecl{From: nameTok, To: p.current()}
	}

	return VarDecl{Name: name, Value: value, Mut: mut}
}

func (p *Parser) funcDeclaration(vis Visibility) Decl {
	nameTok := p.current()
	if !p.expect(scanner.Name, "Expect function name.") {
		p.recover(declStart)
		return BadDecl{From: nameTok, To: p.current()}
	}
	name := nameTok.Lit

	if !p.expect(scanner.Lparen, "Expect '(' after function name.") {
		p.recover(declStart)
		return BadDecl{From: nameTok, To: p.current()}
	}

	params := []Param{}
	if !p.match(scanner.Rparen) {
		for {
			pNameTok := p.current()
			if !p.expect(scanner.Name, "Expect parameter name.") {
				p.recover(paramRecover)
				return BadDecl{From: nameTok, To: p.current()}
			}
			pName := pNameTok.Lit

			if !p.expect(scanner.Colon, "Expect ':' after parameter name.") {
				p.recover(paramRecover)
				return BadDecl{From: nameTok, To: p.current()}
			}

			pTypeTok := p.current()
			if !p.expect(scanner.Name, "Expect parameter type.") {
				p.recover(paramRecover)
				return BadDecl{From: nameTok, To: p.current()}
			}
			pType := Ident{Name: pTypeTok.Lit}

			params = append(params, Param{Name: pName, Type: pType})

			if !p.match(scanner.Comma) {
				break
			}
		}

		if !p.expect(scanner.Rparen, "Expect ')' after parameters.") {
			p.recover(declStart)
			return BadDecl{From: nameTok, To: p.current()}
		}
	}

	var returnType Expr
	if p.match(scanner.ThinArrow) {
		returnType = p.expression()
	}

	if !p.expect(scanner.Lbrace, "Expect '{' before function body.") {
		p.recover(declStart)
		return BadDecl{From: nameTok, To: p.current()}
	}

	body := p.block()

	return FuncDecl{
		Name:       name,
		Params:     params,
		ReturnType: returnType,
		Body:       body.(BlockExpr),
		Visibility: vis,
	}
}

func (p *Parser) expressionStatement() Stmt {
	tok := p.current()
	expr := p.expression()

	_, isBlock := expr.(BlockExpr)
	if !isBlock {
		if !p.expect(scanner.Semi, "Expect ';' after expression.") {
			return BadStmt{From: tok, To: p.current()}
		}
	} else {
		p.match(scanner.Semi)
	}

	return ExprStmt{Expr: expr}
}

func (p *Parser) expression() Expr {
	if p.match(scanner.Lbrace) {
		return p.block()
	}
	if p.match(scanner.Lbrack) {
		return nil
	}
	if p.match(scanner.Return) {
		return ReturnExpr{}
	}
	if p.match(scanner.Break) {
		return nil
	}

	return p.assignment()
}

func (p *Parser) block() Expr {
	var tail Expr
	stmts := []Stmt{}

	tok := p.current()
	for !p.match(scanner.Rbrace) && !p.isAtEnd() {
		start := p.current()
		stmts = append(stmts, p.statement())
		// safety: prevent infinite loop if no progress was made
		if p.current() == start {
			p.consume()
		}
	}

	if p.isAtEnd() {
		p.errorf("Expect '}' after block.")
		return BadExpr{From: tok, To: p.current()}
	}

	return BlockExpr{Stmts: stmts, Tail: tail}
}

func (p *Parser) assignment() Expr {
	expr := p.or()

	op := p.current()
	if p.match(scanner.Eq) {
		value := p.assignment()

		if val, ok := expr.(Ident); ok {
			return AssignExpr{Name: val.Name, Value: value}
		}

		p.errh(op, "Invalid assignment target.")
		return BadExpr{From: op, To: p.current()}
	}

	return expr
}

func (p *Parser) or() Expr {
	expr := p.and()

	for {
		op := p.current()
		if !p.match(scanner.OrOr) {
			break
		}
		expr = BinaryExpr{Left: expr, Op: op, Right: p.and()}
	}

	return expr
}

func (p *Parser) and() Expr {
	expr := p.equality()

	for {
		op := p.current()
		if !p.match(scanner.AndAnd) {
			break
		}
		expr = BinaryExpr{Left: expr, Op: op, Right: p.equality()}
	}

	return expr
}

func (p *Parser) equality() Expr {
	expr := p.comparison()

	for {
		op := p.current()
		if !p.matchMany(scanner.EqEq, scanner.NotEq) {
			break
		}
		expr = BinaryExpr{Left: expr, Op: op, Right: p.comparison()}
	}

	return expr
}

func (p *Parser) comparison() Expr {
	expr := p.rangeExpr()

	for {
		op := p.current()
		if !p.matchMany(scanner.Gt, scanner.Lt, scanner.GtEq, scanner.LtEq) {
			break
		}
		expr = BinaryExpr{Left: expr, Op: op, Right: p.rangeExpr()}
	}

	return expr
}

func (p *Parser) rangeExpr() Expr {
	return p.term()
}

func (p *Parser) term() Expr {
	expr := p.factor()

	for {
		op := p.current()
		if !p.matchMany(scanner.Plus, scanner.Minus) {
			break
		}
		expr = BinaryExpr{Left: expr, Op: op, Right: p.factor()}
	}

	return expr
}

func (p *Parser) factor() Expr {
	expr := p.unary()

	for {
		op := p.current()
		if !p.matchMany(scanner.Star, scanner.Slash, scanner.Percent, scanner.Caret) {
			break
		}
		expr = BinaryExpr{Left: expr, Op: op, Right: p.unary()}
	}

	return expr
}

func (p *Parser) unary() Expr {
	op := p.current()
	if p.matchMany(scanner.Not, scanner.Minus) {
		return UnaryExpr{Op: op, Right: p.unary()}
	}

	return p.postfix()
}

func (p *Parser) postfix() Expr {
	expr := p.primary()

	for {
		switch {
		case p.match(scanner.Lparen):
			expr = p.finishCall(expr)
		case p.match(scanner.Lbrack):
			idx := p.expression()
			if !p.expect(scanner.Rbrack, "Expect ']' after index expression.") {
				return BadExpr{From: p.current(), To: p.current()}
			}
			expr = IndexExpr{Obj: expr, Index: idx}
		case p.match(scanner.Dot):
			nameTok := p.current()
			if !p.expect(scanner.Name, "Expect field or method name after '.'.") {
				return BadExpr{From: p.current(), To: p.current()}
			}
			expr = FieldExpr{Obj: expr, Field: nameTok.Lit}
		default:
			return expr
		}
	}
}

func (p *Parser) finishCall(fun Expr) Expr {
	tok := p.current()
	args := []Expr{}

	if !p.match(scanner.Rparen) {
		for {
			if len(args) >= MaxArgs {
				p.errorf("Can't have more than %d arguments.", MaxArgs)
				p.recover(argRecover)
				return BadExpr{From: tok, To: p.current()}
			}
			args = append(args, p.expression())

			if !p.match(scanner.Comma) {
				break
			}
		}
	}

	if !p.expect(scanner.Rparen, "Expect ')' after arguments.") {
		return BadExpr{From: tok, To: p.current()}
	}

	return CallExpr{Fun: fun, Args: args}
}

func (p *Parser) primary() Expr {
	if p.match(scanner.False) {
		return LiteralExpr{Value: false}
	}
	if p.match(scanner.True) {
		return LiteralExpr{Value: true}
	}

	tok := p.current()
	if p.matchMany(scanner.IntLit, scanner.FloatLit, scanner.StrLit, scanner.CharLit) {
		return LiteralExpr{Value: tok.Lit}
	}

	if p.match(scanner.Name) {
		return Ident{Name: tok.Lit}
	}

	if p.match(scanner.Lparen) {
		expr := p.expression()

		if !p.expect(scanner.Rparen, "Expect ')' after group expression.") {
			return BadExpr{From: p.current(), To: p.current()}
		}

		return GroupingExpr{Expr: expr}
	}

	p.errorf("Expect expression.")
	return BadExpr{From: p.current(), To: p.current()}
}

func (p *Parser) isAtEnd() bool          { return p.tok.Kind == scanner.EOF }
func (p *Parser) current() scanner.Token { return p.tok }

func (p *Parser) matchMany(kinds ...scanner.TokenKind) bool {
	return slices.ContainsFunc(kinds, p.match)
}

func (p *Parser) match(kind scanner.TokenKind) bool {
	if p.isAtEnd() || p.current().Kind != kind {
		return false
	}

	p.consume()
	return true
}

func (p *Parser) expect(kind scanner.TokenKind, msg string) bool {
	if p.match(kind) {
		return true
	}

	p.errorf("%s", msg)

	// advance regardless
	p.consume()
	return false
}

func (p *Parser) consume() {
	if p.isAtEnd() {
		return
	}

	p.tok = p.scanner.Next()
}

func (p *Parser) recover(to map[scanner.TokenKind]bool) {
	for {
		if p.isAtEnd() {
			return
		}

		if to[p.tok.Kind] {
			return
		}

		p.consume()
	}
}

func (p *Parser) isDeclStart() bool {
	return declStart[p.current().Kind]
}
