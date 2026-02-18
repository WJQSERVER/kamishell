package parser

import (
	"kamishell/internal/ast"
	"kamishell/internal/lexer"
	"strconv"
)

const (
	_ int = iota
	LOWEST
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // myFunction(X)
)

var precedences = map[lexer.TokenType]int{
	lexer.EQ:       EQUALS,
	lexer.NEQ:      EQUALS,
	lexer.GREATER:  LESSGREATER,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	l         *lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}

	p.prefixParseFns = make(map[lexer.TokenType]prefixParseFn)
	p.registerPrefix(lexer.IDENT, p.parseIdentifier)
	p.registerPrefix(lexer.NUMBER, p.parseIntegerLiteral)
	p.registerPrefix(lexer.STRING, p.parseStringLiteral)
	p.registerPrefix(lexer.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.FALSE, p.parseBooleanLiteral)

	p.infixParseFns = make(map[lexer.TokenType]infixParseFn)
	p.registerInfix(lexer.EQ, p.parseInfixExpression)
	p.registerInfix(lexer.NEQ, p.parseInfixExpression)
	p.registerInfix(lexer.GREATER, p.parseInfixExpression)

	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) registerPrefix(tokenType lexer.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType lexer.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for p.curToken.Type != lexer.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer.SEMICOLON:
		return nil
	case lexer.PRINT:
		return p.parsePrintStatement()
	case lexer.EXEC:
		return p.parseExecStatement()
	case lexer.IF:
		return p.parseIfStatement()
	case lexer.IDENT:
		if p.peekToken.Type == lexer.COLON_ASSIGN {
			return p.parseAssignStatement()
		}
		return p.parseCommandStatement()
	case lexer.LBRACE:
		return p.parseBlockStatement()
	case lexer.NUMBER, lexer.STRING, lexer.TRUE, lexer.FALSE:
		return p.parseExpressionStatement()
	default:
		return p.parseCommandStatement()
	}
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)
	if p.peekToken.Type == lexer.SEMICOLON {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parsePrintStatement() *ast.PrintStatement {
	stmt := &ast.PrintStatement{Token: p.curToken}
	p.nextToken()
	stmt.Expression = p.parseExpression(LOWEST)
	if p.peekToken.Type == lexer.SEMICOLON {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseExecStatement() *ast.ExecStatement {
	stmt := &ast.ExecStatement{Token: p.curToken}
	p.nextToken()
	stmt.CommandStr = p.parseExpression(LOWEST)
	if p.peekToken.Type == lexer.SEMICOLON {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseAssignStatement() *ast.AssignStatement {
	stmt := &ast.AssignStatement{Token: p.peekToken}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	p.nextToken() // cur is :=
	p.nextToken() // cur is start of expression

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekToken.Type == lexer.SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseIfStatement() *ast.IfStatement {
	stmt := &ast.IfStatement{Token: p.curToken}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	// In Go, the { must be on the same line or before semicolon insertion.
	// Our Lexer doesn't insert semicolon after if condition because expressions don't necessarily end.
	// But if the user put a newline, the Lexer might have inserted a semicolon if the condition was completable.
	if p.peekToken.Type == lexer.SEMICOLON {
		p.nextToken()
	}

	if p.peekToken.Type == lexer.LBRACE {
		p.nextToken()
		stmt.Consequence = p.parseBlockStatement()
	}

	if p.peekToken.Type == lexer.ELSE {
		p.nextToken()
		if p.peekToken.Type == lexer.SEMICOLON {
			p.nextToken()
		}
		if p.peekToken.Type == lexer.LBRACE {
			p.nextToken()
			stmt.Alternative = p.parseBlockStatement()
		}
	}

	return stmt
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for p.curToken.Type != lexer.RBRACE && p.curToken.Type != lexer.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

func (p *Parser) parseCommandStatement() *ast.CommandStatement {
	stmt := &ast.CommandStatement{Token: p.curToken, Name: p.curToken.Literal}

	for p.peekToken.Type != lexer.SEMICOLON && p.peekToken.Type != lexer.EOF && p.peekToken.Type != lexer.RBRACE {
		p.nextToken()
		stmt.Arguments = append(stmt.Arguments, p.curToken.Literal)
	}

	if p.peekToken.Type == lexer.SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		return nil
	}
	leftExp := prefix()

	for p.peekToken.Type != lexer.SEMICOLON && p.peekToken.Type != lexer.LBRACE && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}
	val, _ := strconv.ParseInt(p.curToken.Literal, 0, 64)
	lit.Value = val
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BooleanLiteral{Token: p.curToken, Value: p.curToken.Type == lexer.TRUE}
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}
