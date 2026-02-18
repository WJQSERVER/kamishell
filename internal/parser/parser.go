package parser

import (
	"kamishell/internal/ast"
	"kamishell/internal/lexer"
	"strconv"
)

type Parser struct {
	l         *lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}
	p.nextToken()
	p.nextToken()
	return p
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
	stmt.Expression = p.parseExpression()
	if p.peekToken.Type == lexer.SEMICOLON {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parsePrintStatement() *ast.PrintStatement {
	stmt := &ast.PrintStatement{Token: p.curToken}
	p.nextToken()
	stmt.Expression = p.parseExpression()
	if p.peekToken.Type == lexer.SEMICOLON {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseExecStatement() *ast.ExecStatement {
	stmt := &ast.ExecStatement{Token: p.curToken}
	p.nextToken()
	stmt.CommandStr = p.parseExpression()
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

	stmt.Value = p.parseExpression()

	if p.peekToken.Type == lexer.SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseIfStatement() *ast.IfStatement {
	stmt := &ast.IfStatement{Token: p.curToken}

	p.nextToken()
	stmt.Condition = p.parseExpression()

	if p.peekToken.Type != lexer.LBRACE {
		return nil
	}

	p.nextToken()
	stmt.Consequence = p.parseBlockStatement()

	if p.peekToken.Type == lexer.ELSE {
		p.nextToken()

		if p.peekToken.Type != lexer.LBRACE {
			return nil
		}

		p.nextToken()
		stmt.Alternative = p.parseBlockStatement()
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

func (p *Parser) parseExpression() ast.Expression {
	switch p.curToken.Type {
	case lexer.STRING:
		return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	case lexer.NUMBER:
		val, _ := strconv.ParseInt(p.curToken.Literal, 0, 64)
		return &ast.IntegerLiteral{Token: p.curToken, Value: val}
	case lexer.IDENT:
		return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	case lexer.TRUE, lexer.FALSE:
		return &ast.BooleanLiteral{Token: p.curToken, Value: p.curToken.Type == lexer.TRUE}
	}
	return nil
}
