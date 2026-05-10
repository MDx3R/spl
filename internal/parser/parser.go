package parser

import (
	"fmt"
	"slices"

	"github.com/MDx3R/spl/internal/scanner"
)

const MaxArgs = 65535

// stmtStart is used for synchronising inside block bodies.
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

// declStart is used for synchronising at the top-level declaration boundary.
var declStart = map[scanner.TokenKind]bool{
	scanner.Let:    true,
	scanner.Fn:     true,
	scanner.Struct: true,
	scanner.Trait:  true,
	scanner.Impl:   true,
	scanner.Pub:    true,
}

// itemStart is the recovery set for the top-level declaration loop.
// Excludes Let which is only valid inside blocks.
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

// rangeTerminators are tokens that cannot begin the RHS of a range expression.
var rangeTerminators = map[scanner.TokenKind]bool{
	scanner.Semi:   true,
	scanner.Rbrace: true,
	scanner.Rparen: true,
	scanner.Rbrack: true,
	scanner.Comma:  true,
}

var compoundAssignOps = []scanner.TokenKind{
	scanner.PlusEq, scanner.MinusEq, scanner.StarEq,
	scanner.SlashEq, scanner.PercentEq, scanner.AndEq,
	scanner.OrEq, scanner.CaretEq, scanner.ShlEq, scanner.ShrEq,
}

// Parser holds the scanning cursor and error handler.
type Parser struct {
	scanner *scanner.Scanner
	tok     scanner.Token
	errh    func(tok scanner.Token, msg string)
}

func NewParser(sc *scanner.Scanner, errh func(tok scanner.Token, msg string)) *Parser {
	return &Parser{scanner: sc, errh: errh}
}

func (p *Parser) Parse() *File {
	p.scanner.Init()
	p.tok = p.scanner.Next()

	var decls []Decl
	for !p.isAtEnd() {
		start := p.current()
		decls = append(decls, p.topLevelDecl())
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
		return p.structDeclaration(vis)
	case p.match(scanner.Trait):
		return p.traitDeclaration(vis)
	case p.match(scanner.Impl):
		return p.implDeclaration()
	default:
		p.errorf("Expected top-level declaration (fn, struct, trait, impl).")
		p.recover(itemStart)
		return BadDecl{From: tok, To: p.current()}
	}
}

func (p *Parser) errorf(format string, args ...any) {
	p.errh(p.tok, fmt.Sprintf(format, args...))
}

// parseType parses a simple type expression: Ident, Self, &Type, &mut Type, [Type].
func (p *Parser) parseType() Expr {
	tok := p.current()

	// &Type or &mut Type
	if p.match(scanner.And) {
		p.match(scanner.Mut) // consume optional mut; distinction not stored yet
		inner := p.parseType()
		return UnaryExpr{Op: tok, Right: inner}
	}

	if p.match(scanner.SelfUpper) {
		return Ident{Name: "Self"}
	}
	if p.match(scanner.SelfLower) {
		return Ident{Name: "self"}
	}
	if p.match(scanner.Name) {
		return Ident{Name: tok.Lit}
	}
	if p.match(scanner.Lbrack) {
		inner := p.parseType()
		p.expect(scanner.Rbrack, "Expect ']' after array type.")
		return ArrayExpr{Elems: []Expr{inner}}
	}

	p.errorf("Expect type.")
	return BadExpr{From: tok, To: p.current()}
}

func (p *Parser) varDeclaration() Decl {
	mut := p.match(scanner.Mut)
	nameTok := p.current()
	if !p.expect(scanner.Name, "Expect variable name.") {
		p.recover(stmtStart)
		return BadDecl{From: nameTok, To: p.current()}
	}
	name := nameTok.Lit

	var typ Expr
	if p.match(scanner.Colon) {
		typ = p.parseType()
	}

	if !p.expect(scanner.Eq, "Expect '=' after variable name.") {
		p.recover(stmtStart)
		return BadDecl{From: nameTok, To: p.current()}
	}

	value := p.expression()

	if !p.expect(scanner.Semi, "Expect ';' after variable declaration.") {
		p.recover(stmtStart)
		return BadDecl{From: nameTok, To: p.current()}
	}

	return VarDecl{Name: name, Type: typ, Value: value, Mut: mut}
}

func (p *Parser) funcDeclaration(vis Visibility) Decl {
	nameTok := p.current()
	if !p.expect(scanner.Name, "Expect function name.") {
		p.recover(declStart)
		return BadDecl{From: nameTok, To: p.current()}
	}

	if !p.expect(scanner.Lparen, "Expect '(' after function name.") {
		p.recover(declStart)
		return BadDecl{From: nameTok, To: p.current()}
	}

	params, ok := p.parseFuncParams()
	if !ok {
		p.recover(declStart)
		return BadDecl{From: nameTok, To: p.current()}
	}

	var returnType Expr
	if p.match(scanner.ThinArrow) {
		returnType = p.parseType()
	}

	if !p.expect(scanner.Lbrace, "Expect '{' before function body.") {
		p.recover(declStart)
		return BadDecl{From: nameTok, To: p.current()}
	}

	body := p.block()

	return FuncDecl{
		Name:       nameTok.Lit,
		Params:     params,
		ReturnType: returnType,
		Body:       body,
		Visibility: vis,
	}
}

// parseFuncParams parses a parenthesised parameter list.
// The opening '(' must already have been consumed.
// Returns (params, ok); ok is false when a hard error made the params unusable.
func (p *Parser) parseFuncParams() ([]Param, bool) {
	params := []Param{}
	if p.match(scanner.Rparen) {
		return params, true
	}

	for {
		switch {
		case p.current().Kind == scanner.SelfLower:
			// bare self
			p.consume()
			params = append(params, Param{Name: "self"})

		case p.current().Kind == scanner.And:
			// &self or &mut self
			p.consume()
			mut := p.match(scanner.Mut)
			if !p.expect(scanner.SelfLower, "Expect 'self' after '&' in parameter list.") {
				p.recover(paramRecover)
				return params, false
			}
			params = append(params, Param{Name: "self", Ref: true, Mut: mut})

		default:
			pNameTok := p.current()
			if !p.expect(scanner.Name, "Expect parameter name.") {
				p.recover(paramRecover)
				return params, false
			}
			if !p.expect(scanner.Colon, "Expect ':' after parameter name.") {
				p.recover(paramRecover)
				return params, false
			}
			pType := p.parseType()
			params = append(params, Param{Name: pNameTok.Lit, Type: pType})
		}

		if !p.match(scanner.Comma) {
			break
		}
		if p.current().Kind == scanner.Rparen {
			break // trailing comma
		}
	}

	if !p.expect(scanner.Rparen, "Expect ')' after parameters.") {
		return params, false
	}
	return params, true
}

// parseFuncSignature parses "name ( params ) -> ReturnType" without a body.
func (p *Parser) parseFuncSignature() (FuncSignature, bool) {
	nameTok := p.current()
	if !p.expect(scanner.Name, "Expect method name.") {
		return FuncSignature{}, false
	}
	if !p.expect(scanner.Lparen, "Expect '(' after method name.") {
		return FuncSignature{}, false
	}

	params, ok := p.parseFuncParams()
	if !ok {
		return FuncSignature{}, false
	}

	var returnType Expr
	if p.match(scanner.ThinArrow) {
		returnType = p.parseType()
	}

	return FuncSignature{Name: nameTok.Lit, Params: params, ReturnType: returnType}, true
}

func (p *Parser) structDeclaration(vis Visibility) Decl {
	nameTok := p.current()
	if !p.expect(scanner.Name, "Expect struct name.") {
		p.recover(itemStart)
		return BadDecl{From: nameTok, To: p.current()}
	}

	if !p.expect(scanner.Lbrace, "Expect '{' after struct name.") {
		p.recover(itemStart)
		return BadDecl{From: nameTok, To: p.current()}
	}

	fields := []FieldDef{}
	for !p.isAtEnd() && p.current().Kind != scanner.Rbrace {
		fieldTok := p.current()
		if !p.expect(scanner.Name, "Expect field name.") {
			p.recover(fieldRecover)
			if p.current().Kind == scanner.Rbrace {
				break
			}
			continue
		}

		if !p.expect(scanner.Colon, "Expect ':' after field name.") {
			p.recover(fieldRecover)
			if p.current().Kind == scanner.Rbrace {
				break
			}
			continue
		}

		fieldType := p.parseType()
		fields = append(fields, FieldDef{Name: fieldTok.Lit, Type: fieldType})

		if !p.match(scanner.Comma) {
			break
		}
	}

	if !p.expect(scanner.Rbrace, "Expect '}' after struct body.") {
		return BadDecl{From: nameTok, To: p.current()}
	}

	return StructDecl{Name: nameTok.Lit, Fields: fields, Visibility: vis}
}

func (p *Parser) traitDeclaration(vis Visibility) Decl {
	nameTok := p.current()
	if !p.expect(scanner.Name, "Expect trait name.") {
		p.recover(itemStart)
		return BadDecl{From: nameTok, To: p.current()}
	}

	if !p.expect(scanner.Lbrace, "Expect '{' after trait name.") {
		p.recover(itemStart)
		return BadDecl{From: nameTok, To: p.current()}
	}

	methods := []TraitMethod{}
	for !p.isAtEnd() && p.current().Kind != scanner.Rbrace {
		for p.current().IsComment() {
			p.consume()
		}
		if p.current().Kind == scanner.Rbrace {
			break
		}

		if !p.expect(scanner.Fn, "Expect 'fn' in trait body.") {
			p.recover(traitBodyRecover)
			if p.current().Kind == scanner.Rbrace {
				break
			}
			continue
		}

		sig, ok := p.parseFuncSignature()
		if !ok {
			p.recover(traitBodyRecover)
			continue
		}

		switch {
		case p.match(scanner.Semi):
			methods = append(methods, TraitMethod{Sig: sig})
		case p.match(scanner.Lbrace):
			body := p.block()
			methods = append(methods, TraitMethod{Sig: sig, Body: &body})
		default:
			p.errorf("Expect ';' or '{' after trait method signature.")
			p.recover(traitBodyRecover)
		}
	}

	if !p.expect(scanner.Rbrace, "Expect '}' after trait body.") {
		return BadDecl{From: nameTok, To: p.current()}
	}

	return TraitDecl{Name: nameTok.Lit, Methods: methods, Visibility: vis}
}

func (p *Parser) implDeclaration() Decl {
	tok := p.current()

	firstTok := p.current()
	if !p.expect(scanner.Name, "Expect type or trait name after 'impl'.") {
		p.recover(itemStart)
		return BadDecl{From: tok, To: p.current()}
	}
	first := firstTok.Lit

	// impl Trait for Type  vs  impl Type
	var traitName, typeName string
	if p.match(scanner.For) {
		traitName = first
		typeTok := p.current()
		if !p.expect(scanner.Name, "Expect type name after 'for'.") {
			p.recover(itemStart)
			return BadDecl{From: tok, To: p.current()}
		}
		typeName = typeTok.Lit
	} else {
		typeName = first
	}

	if !p.expect(scanner.Lbrace, "Expect '{' after impl header.") {
		p.recover(itemStart)
		return BadDecl{From: tok, To: p.current()}
	}

	methods := []FuncDecl{}
	for !p.isAtEnd() && p.current().Kind != scanner.Rbrace {
		for p.current().IsComment() {
			p.consume()
		}
		if p.current().Kind == scanner.Rbrace {
			break
		}

		vis := p.parseVisibility()
		if !p.expect(scanner.Fn, "Expect 'fn' in impl body.") {
			p.recover(traitBodyRecover)
			if p.current().Kind == scanner.Rbrace {
				break
			}
			continue
		}

		decl := p.funcDeclaration(vis)
		if fd, ok := decl.(FuncDecl); ok {
			methods = append(methods, fd)
		}
	}

	if !p.expect(scanner.Rbrace, "Expect '}' after impl body.") {
		return BadDecl{From: tok, To: p.current()}
	}

	return ImplDecl{Trait: traitName, Type: typeName, Methods: methods}
}

// block parses a block body after the opening '{' has already been consumed.
// It returns BlockExpr directly; errors are reported via errh and parsing continues.
func (p *Parser) block() BlockExpr {
	var stmts []Stmt
	var tail Expr

	for !p.isAtEnd() {
		// skip doc-comment tokens and empty semicolons
		if p.current().IsComment() {
			p.consume()
			continue
		}
		if p.match(scanner.Semi) {
			continue
		}

		if p.current().Kind == scanner.Rbrace {
			p.consume()
			return BlockExpr{Stmts: stmts, Tail: tail}
		}

		start := p.current()

		switch p.current().Kind {
		case scanner.Let:
			p.consume()
			stmts = append(stmts, p.varDeclaration())

		case scanner.Fn:
			p.consume()
			stmts = append(stmts, p.funcDeclaration(Visibility{Kind: VisPrivate}))

		// Do I even need this here? default clause is going to fail on this regardless
		case scanner.Const, scanner.Type, scanner.Mod, scanner.Use,
			scanner.Struct, scanner.Enum, scanner.Union, scanner.Static,
			scanner.Trait, scanner.Impl, scanner.Extern:
			p.errorf("'%s' is not yet supported inside blocks.", p.current().Kind)
			p.consume()
			p.recover(stmtStart)
			stmts = append(stmts, BadStmt{From: start, To: p.current()})

		// TODO: refactor
		default:
			// Expression - could be a regular statement or the block's tail value.
			expr := p.expression()

			if p.isExprWithBlock(expr) {
				// ExpressionWithBlock: trailing ; is optional.
				if p.current().Kind == scanner.Rbrace {
					// No semicolon before }: treat as tail.
					tail = expr
					p.consume()
					return BlockExpr{Stmts: stmts, Tail: tail}
				}
				p.match(scanner.Semi)
				stmts = append(stmts, ExprStmt{Expr: expr})
			} else if p.current().Kind == scanner.Rbrace {
				// Non-block expr with no semicolon before }: tail expression.
				tail = expr
				p.consume()
				return BlockExpr{Stmts: stmts, Tail: tail}
			} else if p.match(scanner.Semi) {
				stmts = append(stmts, ExprStmt{Expr: expr})
			} else {
				p.errorf("Expect ';' after expression.")
				p.recover(stmtStart)
				stmts = append(stmts, ExprStmt{Expr: expr})
			}
		}

		// Safety: prevent infinite loop if no token was consumed.
		if !p.isAtEnd() && p.current() == start {
			p.consume()
		}
	}

	p.errorf("Expect '}' after block.")
	return BlockExpr{Stmts: stmts, Tail: tail}
}

func (p *Parser) isExprWithBlock(e Expr) bool {
	switch e.(type) {
	case BlockExpr, IfExpr, WhileExpr, ForExpr, LoopExpr:
		return true
	}
	return false
}

// Precedence chain (lowest → highest):
//   expression  return, break, continue
//   assignment  =  +=  -=  *=  /=  %=  &=  |=  ^=  <<=  >>=
//   range       ..  ..=
//   or          ||
//   and         &&
//   comparison  ==  !=  <  >  <=  >=   (no chaining)
//   bitor       |
//   bitxor      ^
//   bitand      &
//   shift       <<  >>
//   term        +  -
//   factor      *  /  %
//   unary       !  -   (prefix)
//   postfix     .field  (args)  [index]
//   primary     literals, ident, self, Self, if, while, for, loop, {block},
//               (group), [array], macro!()

// isReturnTerminator returns true when `return`/`break` should produce
// a bare (no-value) variant: only at an unambiguous statement boundary.
func (p *Parser) isReturnTerminator() bool {
	k := p.current().Kind
	return k == scanner.Semi || k == scanner.Rbrace ||
		k == scanner.Rparen || k == scanner.Rbrack
}

func (p *Parser) expression() Expr {
	if p.match(scanner.Return) {
		if p.isAtEnd() || p.isReturnTerminator() {
			return ReturnExpr{}
		}
		return ReturnExpr{Expr: p.expression()}
	}
	if p.match(scanner.Break) {
		if p.isAtEnd() || p.isReturnTerminator() {
			return BreakExpr{}
		}
		return BreakExpr{Value: p.expression()}
	}
	if p.match(scanner.Continue) {
		return ContinueExpr{}
	}
	return p.assignment()
}

func isValidLValue(e Expr) bool {
	switch e.(type) {
	case Ident, FieldExpr, IndexExpr:
		return true
	}
	return false
}

func (p *Parser) assignment() Expr {
	expr := p.rangeExpr()

	op := p.current()
	if p.match(scanner.Eq) {
		value := p.assignment() // right-associative
		if isValidLValue(expr) {
			return AssignExpr{Target: expr, Value: value}
		}
		p.errh(op, "Invalid assignment target.")
		return BadExpr{From: op, To: p.current()}
	}

	op = p.current()
	if p.matchMany(compoundAssignOps...) {
		value := p.expression()
		return CompoundAssignExpr{Target: expr, Op: op, Value: value}
	}

	return expr
}

func (p *Parser) rangeExpr() Expr {
	// Open-ended left: ..hi or ..=hi
	if p.current().Kind == scanner.DotDot || p.current().Kind == scanner.DotDotEq {
		op := p.current()
		inclusive := op.Kind == scanner.DotDotEq
		p.consume()
		var hi Expr
		if !p.isAtEnd() && !rangeTerminators[p.current().Kind] {
			hi = p.or()
		}
		return RangeExpr{Lo: nil, Hi: hi, Inclusive: inclusive}
	}

	lo := p.or()

	op := p.current()
	if op.Kind == scanner.DotDot || op.Kind == scanner.DotDotEq {
		inclusive := op.Kind == scanner.DotDotEq
		p.consume()
		if p.isAtEnd() || rangeTerminators[p.current().Kind] {
			return RangeExpr{Lo: lo, Hi: nil, Inclusive: inclusive}
		}
		hi := p.or()
		return RangeExpr{Lo: lo, Hi: hi, Inclusive: inclusive}
	}

	return lo
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
	expr := p.comparison()
	for {
		op := p.current()
		if !p.match(scanner.AndAnd) {
			break
		}
		expr = BinaryExpr{Left: expr, Op: op, Right: p.comparison()}
	}
	return expr
}

// comparison handles ==, !=, <, >, <=, >= with no chaining (Rust spec).
func (p *Parser) comparison() Expr {
	expr := p.bitor()
	op := p.current()
	if p.matchMany(scanner.EqEq, scanner.NotEq, scanner.Lt, scanner.LtEq, scanner.Gt, scanner.GtEq) {
		return BinaryExpr{Left: expr, Op: op, Right: p.bitor()}
	}
	return expr
}

func (p *Parser) bitor() Expr {
	expr := p.bitxor()
	for {
		op := p.current()
		if !p.match(scanner.Or) {
			break
		}
		expr = BinaryExpr{Left: expr, Op: op, Right: p.bitxor()}
	}
	return expr
}

func (p *Parser) bitxor() Expr {
	expr := p.bitand()
	for {
		op := p.current()
		if !p.match(scanner.Caret) {
			break
		}
		expr = BinaryExpr{Left: expr, Op: op, Right: p.bitand()}
	}
	return expr
}

func (p *Parser) bitand() Expr {
	expr := p.shift()
	for {
		op := p.current()
		if !p.match(scanner.And) {
			break
		}
		expr = BinaryExpr{Left: expr, Op: op, Right: p.shift()}
	}
	return expr
}

func (p *Parser) shift() Expr {
	expr := p.term()
	for {
		op := p.current()
		if !p.matchMany(scanner.Shl, scanner.Shr) {
			break
		}
		expr = BinaryExpr{Left: expr, Op: op, Right: p.term()}
	}
	return expr
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
		if !p.matchMany(scanner.Star, scanner.Slash, scanner.Percent) {
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
			if p.current().Kind == scanner.Rparen {
				break // trailing comma
			}
		}
		if !p.expect(scanner.Rparen, "Expect ')' after arguments.") {
			return BadExpr{From: tok, To: p.current()}
		}
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

	// Name or macro invocation (name!)
	if p.match(scanner.Name) {
		name := tok.Lit
		if p.match(scanner.Not) {
			return p.macroExpr(name)
		}
		return Ident{Name: name}
	}

	if p.match(scanner.SelfLower) {
		return Ident{Name: "self"}
	}
	if p.match(scanner.SelfUpper) {
		return Ident{Name: "Self"}
	}

	// Array literal: [expr, expr, ...]
	if p.match(scanner.Lbrack) {
		return p.arrayExpr()
	}

	// Grouped expression: (expr)
	if p.match(scanner.Lparen) {
		expr := p.expression()
		if !p.expect(scanner.Rparen, "Expect ')' after group expression.") {
			return BadExpr{From: tok, To: p.current()}
		}
		return GroupingExpr{Expr: expr}
	}

	// Block expression: { stmts... }
	if p.match(scanner.Lbrace) {
		return p.block()
	}

	if p.match(scanner.If) {
		return p.ifExpr()
	}
	if p.match(scanner.While) {
		return p.whileExpr()
	}
	if p.match(scanner.For) {
		return p.forExpr()
	}
	if p.match(scanner.Loop) {
		return p.loopExpr()
	}

	p.errorf("Expect expression.")
	return BadExpr{From: tok, To: p.current()}
}

// ─── control-flow expressions ─────────────────────────────────────────────────

func (p *Parser) ifExpr() Expr {
	tok := p.current()

	cond := p.expression()

	if !p.expect(scanner.Lbrace, "Expect '{' after if condition.") {
		p.recover(stmtStart)
		return BadExpr{From: tok, To: p.current()}
	}
	then := p.block()

	var elseBranch Expr
	if p.match(scanner.Else) {
		if p.match(scanner.If) {
			elseBranch = p.ifExpr()
		} else {
			if !p.expect(scanner.Lbrace, "Expect '{' after else.") {
				p.recover(stmtStart)
				return BadExpr{From: tok, To: p.current()}
			}
			elseBranch = p.block()
		}
	}

	return IfExpr{Cond: cond, Then: then, Else: elseBranch}
}

func (p *Parser) whileExpr() Expr {
	tok := p.current()

	cond := p.expression()

	if !p.expect(scanner.Lbrace, "Expect '{' after while condition.") {
		p.recover(stmtStart)
		return BadExpr{From: tok, To: p.current()}
	}
	body := p.block()

	return WhileExpr{Cond: cond, Body: body}
}

func (p *Parser) forExpr() Expr {
	tok := p.current()

	bindingTok := p.current()
	if !p.expect(scanner.Name, "Expect binding name after 'for'.") {
		p.recover(stmtStart)
		return BadExpr{From: tok, To: p.current()}
	}

	if !p.expect(scanner.In, "Expect 'in' after for binding.") {
		p.recover(stmtStart)
		return BadExpr{From: tok, To: p.current()}
	}

	iter := p.expression()

	if !p.expect(scanner.Lbrace, "Expect '{' after for iterator.") {
		p.recover(stmtStart)
		return BadExpr{From: tok, To: p.current()}
	}
	body := p.block()

	return ForExpr{Binding: bindingTok.Lit, Iter: iter, Body: body}
}

func (p *Parser) loopExpr() Expr {
	tok := p.current()

	if !p.expect(scanner.Lbrace, "Expect '{' after 'loop'.") {
		p.recover(stmtStart)
		return BadExpr{From: tok, To: p.current()}
	}
	body := p.block()

	return LoopExpr{Body: body}
}

// ─── helper expressions ───────────────────────────────────────────────────────

// macroExpr parses ident!(args...) -- name and '!' already consumed.
func (p *Parser) macroExpr(name string) Expr {
	tok := p.current()
	if !p.expect(scanner.Lparen, "Expect '(' after macro name '!'.") {
		p.recover(argRecover)
		return BadExpr{From: tok, To: p.current()}
	}

	args := []Expr{}
	if !p.match(scanner.Rparen) {
		for {
			args = append(args, p.expression())
			if !p.match(scanner.Comma) {
				break
			}
			if p.current().Kind == scanner.Rparen {
				break
			}
		}
		if !p.expect(scanner.Rparen, "Expect ')' after macro arguments.") {
			return BadExpr{From: tok, To: p.current()}
		}
	}

	return MacroExpr{Name: name, Args: args}
}

// arrayExpr parses [expr, expr, ...] -- '[' already consumed.
func (p *Parser) arrayExpr() Expr {
	tok := p.current()
	elems := []Expr{}

	if !p.match(scanner.Rbrack) {
		for {
			elems = append(elems, p.expression())
			if !p.match(scanner.Comma) {
				break
			}
			if p.current().Kind == scanner.Rbrack {
				break // trailing comma
			}
		}
		if !p.expect(scanner.Rbrack, "Expect ']' after array elements.") {
			return BadExpr{From: tok, To: p.current()}
		}
	}

	return ArrayExpr{Elems: elems}
}

// ─── scanner interface ────────────────────────────────────────────────────────

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
