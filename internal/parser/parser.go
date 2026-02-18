package parser

import (
	"kamishell/internal/ast"
	"kamishell/internal/lexer"
)

type Parser struct {
	l      *lexer.Lexer
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
	default:
		return p.parseCommandStatement()
	}
}

func (p *Parser) parsePrintStatement() *ast.PrintStatement {
	stmt := &ast.PrintStatement{Token: p.curToken}

	p.nextToken()

	stmt.Expression = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

	if p.peekToken.Type == lexer.SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseCommandStatement() *ast.CommandStatement {
	stmt := &ast.CommandStatement{Token: p.curToken, Name: p.curToken.Literal}

	for p.peekToken.Type != lexer.SEMICOLON && p.peekToken.Type != lexer.EOF {
		p.nextToken()
		stmt.Arguments = append(stmt.Arguments, p.curToken.Literal)
	}

	if p.peekToken.Type == lexer.SEMICOLON {
		p.nextToken()
	}

	return stmt
}
