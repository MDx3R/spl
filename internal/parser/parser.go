package parser

import (
	"fmt"
	"slices"

	"github.com/MDx3R/spl/internal/scanner"
)

const MaxArgs = 65535

var stmtStart = map[scanner.TokenKind]bool{
	scanner.Break: true,
	// TODO: add more statement start tokens
}

var declStart = map[scanner.TokenKind]bool{
	scanner.Let: true,
	// TODO: add more declaration start tokens
}
var exprStart = map[scanner.TokenKind]bool{
	scanner.Lbrace: true,
	// TODO: add more expression end tokens
}
var exprEnd = map[scanner.TokenKind]bool{
	scanner.Comma: true,
	// TODO: add more expression end tokens
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

	return nil
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
		return p.funcDeclaration()
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
	tok := p.current()
	if !p.expect(scanner.Name, "Expect variable name.") {
		p.recover(stmtStart)
		return BadDecl{From: tok, To: p.current()}
	}

	mut := p.match(scanner.Mut)

	if !p.expect(scanner.Eq, "Expect '=' after variable name.") {
		p.recover(stmtStart)
		return BadDecl{From: tok, To: p.current()}
	}

	value := p.expression()

	if !p.expect(scanner.Semi, "Expect ';' after variable declaration.") {
		p.recover(stmtStart)
		return BadDecl{From: tok, To: p.current()}
	}

	return VarDecl{Name: tok.Lit, Value: value, Mut: mut}
}

func (p *Parser) funcDeclaration() Decl {
	tok := p.current()
	if !p.expect(scanner.Name, "Expect function name.") {
		p.recover(declStart)
		return BadDecl{From: tok, To: p.current()}
	}

	name := p.current().Lit

	if !p.expect(scanner.Lparen, "Expect '(' after function name.") {
		p.recover(declStart)
		return BadDecl{From: tok, To: p.current()}
	}

	params := []Param{}
	if !p.match(scanner.Rparen) {
		for {
			tok := p.current()
			if !p.expect(scanner.Name, "Expect parameter name.") {
				p.recover(declStart)
				return BadDecl{From: tok, To: p.current()}
			}

			pName := p.current().Lit

			if !p.expect(scanner.Colon, "Expect ':' after parameter name.") {
				p.recover(declStart)
				return BadDecl{From: tok, To: p.current()}
			}
			if !p.expect(scanner.Name, "Expect parameter type.") {
				p.recover(declStart)
				return BadDecl{From: tok, To: p.current()}
			}

			pType := p.expression()

			_, isIdent := pType.(Ident)
			if !isIdent {
				p.errorf("Expect valid parameter type.")
				p.recover(declStart)
				return BadDecl{From: tok, To: p.current()}
			}

			params = append(params, Param{Name: pName, Type: pType})

			if !p.match(scanner.Comma) {
				break
			}
		}
	}

	if !p.expect(scanner.Rparen, "Expect ')' after parameters.") {
		p.recover(declStart)
		return BadDecl{From: tok, To: p.current()}
	}

	var returnType Expr
	if p.match(scanner.ThinArrow) { // Arrow = '->'
		returnType = p.expression()
	}

	if !p.expect(scanner.Lbrace, "Expect '{' before function body.") {
		p.recover(declStart)
		return BadDecl{From: tok, To: p.current()}
	}

	body := p.block()

	return FuncDecl{
		Name:       name,
		Params:     params,
		ReturnType: returnType,
		Body:       body.(BlockExpr),
		Visibility: Visibility{Kind: VisPrivate},
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
		return nil
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
	for !p.match(scanner.Rbrace) {
		// TODO: support tail expressions
		stmts = append(stmts, p.statement())
	}

	if p.isAtEnd() {
		p.errorf("Expect '}' after block.")
		return BadExpr{From: tok, To: p.current()}
	}

	return BlockExpr{Stmts: stmts, Tail: tail}
}

func (p *Parser) assignment() Expr {
	expr := p.equality()

	if p.match(scanner.Eq) {
		equals := p.current()
		value := p.assignment()

		if val, ok := expr.(Ident); ok {
			return AssignExpr{Name: val.Name, Value: value}
		}

		p.errh(equals, "Invalid assignment target.")
		return BadExpr{From: equals, To: p.current()}
	}

	return expr
}

func (p *Parser) equality() Expr {
	expr := p.comparison()

	if p.matchMany(scanner.EqEq, scanner.NotEq) {
		return BinaryExpr{Left: expr, Op: p.current(), Right: p.comparison()}
	}

	return expr
}

func (p *Parser) comparison() Expr {
	expr := p.term()

	if p.matchMany(scanner.Gt, scanner.Lt, scanner.GtEq, scanner.LtEq) {
		return BinaryExpr{Left: expr, Op: p.current(), Right: p.term()}
	}

	return expr
}

func (p *Parser) term() Expr {
	expr := p.factor()

	if p.matchMany(scanner.Plus, scanner.Minus) {
		return BinaryExpr{Left: expr, Op: p.current(), Right: p.factor()}
	}

	return expr
}

func (p *Parser) factor() Expr {
	expr := p.unary()

	if p.matchMany(scanner.Star, scanner.Slash, scanner.Percent, scanner.Caret) {
		return BinaryExpr{Left: expr, Op: p.current(), Right: p.unary()}
	}

	return expr
}

func (p *Parser) unary() Expr {
	if p.matchMany(scanner.Not, scanner.Minus) {
		return UnaryExpr{Op: p.current(), Right: p.unary()}
	}

	return p.call()
}

func (p *Parser) call() Expr {
	expr := p.primary()

	for p.match(scanner.Lparen) {
		expr = p.callWithArgs(expr)
	}

	return expr
}

func (p *Parser) callWithArgs(fun Expr) Expr {
	tok := p.current()
	args := []Expr{}

	if !p.match(scanner.Rparen) {
		for {
			args = append(args, p.expression())

			if !p.match(scanner.Comma) {
				break
			}
		}
	}

	if len(args) > MaxArgs {
		p.errorf("Can't have more than %d arguments.", MaxArgs)
		return BadExpr{From: tok, To: p.current()}
	}

	if !p.expect(scanner.Rparen, "Expect ')' after arguments.") {
		return BadExpr{From: p.current(), To: p.current()}
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
	if p.matchMany(scanner.IntLit, scanner.FloatLit, scanner.StrLit, scanner.CharLit) {
		return LiteralExpr{Value: p.current().Lit}
	}

	if p.match(scanner.Name) {
		return Ident{Name: p.current().Lit}
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
