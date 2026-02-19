package kamishell

import (
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

var precedences = map[TokenType]int{
	EQ:       EQUALS,
	NEQ:      EQUALS,
	GREATER:  LESSGREATER,
	LESS:     LESSGREATER,
	PLUS:     SUM,
}

type (
	prefixParseFn func() Expression
	infixParseFn  func(Expression) Expression
)

type Parser struct {
	l         *Lexer
	curToken  Token
	peekToken Token

	prefixParseFns map[TokenType]prefixParseFn
	infixParseFns  map[TokenType]infixParseFn
}

func NewParser(l *Lexer) *Parser {
	p := &Parser{l: l}

	p.prefixParseFns = make(map[TokenType]prefixParseFn)
	p.registerPrefix(IDENT, p.parseIdentifier)
	p.registerPrefix(NUMBER, p.parseIntegerLiteral)
	p.registerPrefix(STRING, p.parseStringLiteral)
	p.registerPrefix(TRUE_TOK, p.parseBooleanLiteral)
	p.registerPrefix(FALSE_TOK, p.parseBooleanLiteral)
	p.registerPrefix(DOLLAR, p.parseInterpolation)
	p.registerPrefix(NIL, p.parseNilLiteral)

	p.infixParseFns = make(map[TokenType]infixParseFn)
	p.registerInfix(EQ, p.parseInfixExpression)
	p.registerInfix(NEQ, p.parseInfixExpression)
	p.registerInfix(GREATER, p.parseInfixExpression)
	p.registerInfix(LESS, p.parseInfixExpression)
	p.registerInfix(PLUS, p.parseInfixExpression)

	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) registerPrefix(tokenType TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseProgram() *Program {
	program := &Program{}
	program.Statements = []Statement{}

	for p.curToken.Type != EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() Statement {
    stmt := p.parsePipeOrRedirectStatement()
    for p.peekToken.Type == AND || p.peekToken.Type == OR {
        operator := p.peekToken
        p.nextToken() // move to && or ||
        p.nextToken() // move to next command
        right := p.parsePipeOrRedirectStatement()
        stmt = &LogicalStatement{
            Token:    operator,
            Left:     stmt,
            Operator: operator.Literal,
            Right:    right,
        }
    }
    if p.peekToken.Type == AMPERSAND {
        p.nextToken() // move to &
        stmt = &BackgroundStatement{
            Token: p.curToken,
            Stmt:  stmt,
        }
    }
    return stmt
}

func (p *Parser) parsePipeOrRedirectStatement() Statement {
	var stmt Statement
	switch p.curToken.Type {
	case SEMICOLON:
		return nil
	case PRINT:
		stmt = p.parsePrintStatement()
	case EXEC:
		stmt = p.parseExecStatement()
	case IF:
		stmt = p.parseIfStatement()
	case FOR:
		stmt = p.parseForStatement()
	case FUNC:
		stmt = p.parseFunctionStatement()
	case GO:
		stmt = p.parseGoStatement()
	case IDENT:
		if p.peekToken.Type == COLON_ASSIGN || p.peekToken.Type == ASSIGN {
			stmt = p.parseAssignStatement()
		} else {
			stmt = p.parseCommandStatement()
		}
	case LBRACE:
		stmt = p.parseBlockStatement()
	case NUMBER, STRING, TRUE_TOK, FALSE_TOK, DOLLAR:
		stmt = p.parseExpressionStatement()
	default:
		stmt = p.parseCommandStatement()
	}

	for {
		if p.peekToken.Type == PIPE {
			stmt = p.parsePipeStatement(stmt)
		} else if p.peekToken.Type == GREATER || p.peekToken.Type == APPEND {
			stmt = p.parseRedirectStatement(stmt)
		} else {
			break
		}
	}

	return stmt
}

func (p *Parser) parsePipeStatement(left Statement) *PipeStatement {
	ps := &PipeStatement{Token: p.peekToken, Commands: []Statement{left}}
	for p.peekToken.Type == PIPE {
		p.nextToken() // move to |
		p.nextToken() // move to start of next command
		cmd := p.parseCommandStatement()
		ps.Commands = append(ps.Commands, cmd)
	}
	return ps
}

func (p *Parser) parseRedirectStatement(left Statement) *RedirectStatement {
	stmt := &RedirectStatement{Token: p.peekToken, Source: left}
	stmt.Append = p.peekToken.Type == APPEND

	p.nextToken() // move to > or >>
	p.nextToken() // move to target

	stmt.Target = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseExpressionStatement() *ExpressionStatement {
	stmt := &ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)
	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parsePrintStatement() *PrintStatement {
	stmt := &PrintStatement{Token: p.curToken}
	p.nextToken()
	stmt.Expression = p.parseExpression(LESSGREATER)
	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseExecStatement() *ExecStatement {
	stmt := &ExecStatement{Token: p.curToken}
	p.nextToken()
	stmt.CommandStr = p.parseExpression(LOWEST)
	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseAssignStatement() *AssignStatement {
	stmt := &AssignStatement{Token: p.peekToken}
	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	p.nextToken() // cur is :=
	p.nextToken() // cur is start of expression

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseIfStatement() *IfStatement {
	stmt := &IfStatement{Token: p.curToken}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}

	if p.peekToken.Type == LBRACE {
		p.nextToken()
		stmt.Consequence = p.parseBlockStatement()
	}

	if p.peekToken.Type == ELSE {
		p.nextToken()
		if p.peekToken.Type == SEMICOLON {
			p.nextToken()
		}
		if p.peekToken.Type == LBRACE {
			p.nextToken()
			stmt.Alternative = p.parseBlockStatement()
		}
	}

	return stmt
}

func (p *Parser) parseForStatement() *ForStatement {
	stmt := &ForStatement{Token: p.curToken}

	p.nextToken()
	if p.curToken.Type != LBRACE {
		stmt.Condition = p.parseExpression(LOWEST)
	}

	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}

	if p.peekToken.Type == LBRACE {
		p.nextToken()
		stmt.Consequence = p.parseBlockStatement()
	}

	return stmt
}

func (p *Parser) parseBlockStatement() *BlockStatement {
	block := &BlockStatement{Token: p.curToken}
	block.Statements = []Statement{}

	p.nextToken()

	for p.curToken.Type != RBRACE && p.curToken.Type != EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

func (p *Parser) parseCommandStatement() *CommandStatement {
	stmt := &CommandStatement{Token: p.curToken, Name: p.curToken.Literal}

	for p.peekToken.Type != SEMICOLON && p.peekToken.Type != EOF && p.peekToken.Type != RBRACE && p.peekToken.Type != PIPE && p.peekToken.Type != GREATER && p.peekToken.Type != APPEND && p.peekToken.Type != AND && p.peekToken.Type != OR && p.peekToken.Type != AMPERSAND {
		p.nextToken()
		if p.curToken.Type == IDENT {
			// In command context, treat bare words as strings
			stmt.Arguments = append(stmt.Arguments, &StringLiteral{Token: p.curToken, Value: p.curToken.Literal})
		} else {
			stmt.Arguments = append(stmt.Arguments, p.parseExpression(CALL))
		}
	}

	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpression(precedence int) Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		return nil
	}
	leftExp := prefix()

	for p.peekToken.Type != SEMICOLON && p.peekToken.Type != LBRACE && p.peekToken.Type != GREATER && p.peekToken.Type != APPEND && p.peekToken.Type != AND && p.peekToken.Type != OR && p.peekToken.Type != AMPERSAND && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() Expression {
	return &Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseInterpolation() Expression {
	p.nextToken() // consume $
	return &Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() Expression {
	lit := &IntegerLiteral{Token: p.curToken}
	val, _ := strconv.ParseInt(p.curToken.Literal, 0, 64)
	lit.Value = val
	return lit
}

func (p *Parser) parseStringLiteral() Expression {
	return &StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseNilLiteral() Expression {
	return &NilLiteral{Token: p.curToken}
}

func (p *Parser) parseBooleanLiteral() Expression {
	return &BooleanLiteral{Token: p.curToken, Value: p.curToken.Type == TRUE_TOK}
}

func (p *Parser) parseInfixExpression(left Expression) Expression {
	expression := &InfixExpression{
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

func (p *Parser) parseFunctionStatement() *FunctionStatement {
	stmt := &FunctionStatement{Token: p.curToken}

	p.nextToken() // move to name
	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if p.peekToken.Type == LPAREN {
		p.nextToken() // move to (
		stmt.Parameters = p.parseFunctionParameters()
	}

	if p.peekToken.Type == LBRACE {
		p.nextToken() // move to {
		stmt.Body = p.parseBlockStatement()
	}

	return stmt
}

func (p *Parser) parseFunctionParameters() []*Identifier {
	identifiers := []*Identifier{}

	if p.peekToken.Type == RPAREN {
		p.nextToken()
		return identifiers
	}

	p.nextToken()

	ident := &Identifier{Token: p.curToken, Value: p.curToken.Literal}
	identifiers = append(identifiers, ident)

	for p.peekToken.Type == COMMA {
		p.nextToken()
		p.nextToken()
		ident := &Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)
	}

	if p.peekToken.Type == RPAREN {
		p.nextToken()
	}

	return identifiers
}

func (p *Parser) parseGoStatement() *GoStatement {
	stmt := &GoStatement{Token: p.curToken}
	p.nextToken()
	if p.curToken.Type == LBRACE {
		stmt.Node = p.parseBlockStatement()
	} else {
		stmt.Node = p.parseCommandStatement()
	}
	return stmt
}
