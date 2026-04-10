package core

import (
	"strconv"
	"strings"
)

const (
	_ int = iota
	LOWEST
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	MEMBER      // obj.prop
	CALL        // myFunction(X)
)

type Parser struct {
	l         *Lexer
	curToken  Token
	peekToken Token
}

func NewParser(l *Lexer) *Parser {
	p := &Parser{l: l}

	p.nextToken()
	p.nextToken()
	return p
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
	case VAR:
		stmt = p.parseVarStatement()
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
			if p.peekToken.Type == ASSIGN && p.curToken.End == p.peekToken.Start {
				stmt = p.parseInvalidTightAssignStatement()
				break
			}
			stmt = p.parseAssignStatement()
		} else if p.peekToken.Type == DOT || p.peekToken.Type == LPAREN {
			stmt = p.parseExpressionStatement()
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

	if p.curToken.Type == IDENT {
		stmt.Target = &StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
		p.nextToken()
	} else {
		stmt.Target = p.parseExpression(LOWEST)
	}

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

func (p *Parser) parseInvalidTightAssignStatement() *InvalidStatement {
	stmt := &InvalidStatement{Token: p.peekToken, Message: "syntax error: assignments with '=' require spaces around the operator"}

	p.nextToken() // move to =
	if p.peekToken.Type != SEMICOLON && p.peekToken.Type != EOF {
		p.nextToken() // move to start of expression
		_ = p.parseExpression(LOWEST)
	}

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
		if merged, ok := p.tryParseKeyValueArgument(); ok {
			stmt.Arguments = append(stmt.Arguments, merged)
			continue
		}
		p.nextToken()
		if p.curToken.Type == IDENT {
			// In command context, treat bare words as strings
			stmt.Arguments = append(stmt.Arguments, &StringLiteral{Token: p.curToken, Value: p.curToken.Literal})
		} else {
			arg := p.parseExpression(CALL)
			if arg != nil {
				stmt.Arguments = append(stmt.Arguments, arg)
			} else {
				// Fallback: treat unknown operators in command context as literal strings
				stmt.Arguments = append(stmt.Arguments, &StringLiteral{Token: p.curToken, Value: p.curToken.Literal})
			}
		}
	}

	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) tryParseKeyValueArgument() (Expression, bool) {
	if p.peekToken.Type != IDENT {
		return nil, false
	}
	left := p.peekToken

	// Need at least IDENT '=' <value> with no spaces around '='.
	if p.l == nil {
		return nil, false
	}
	if p.peekToken.End >= len(p.l.input) || p.l.input[p.peekToken.End] != '=' {
		return nil, false
	}

	savedCur := p.curToken
	savedPeek := p.peekToken
	p.nextToken() // current becomes left ident
	if p.peekToken.Type != ASSIGN || p.curToken.End != p.peekToken.Start {
		p.curToken = savedCur
		p.peekToken = savedPeek
		return nil, false
	}
	p.nextToken() // current becomes =
	if p.peekToken.Type != IDENT && p.peekToken.Type != NUMBER && p.peekToken.Type != STRING && p.peekToken.Type != TRUE_TOK && p.peekToken.Type != FALSE_TOK && p.peekToken.Type != NIL {
		p.curToken = savedCur
		p.peekToken = savedPeek
		return nil, false
	}
	p.nextToken() // current becomes value
	merged := left.Literal + "=" + p.curToken.Literal
	return &StringLiteral{Token: left, Value: merged, Obj: &String{Value: merged}}, true
}

func (p *Parser) parseExpression(precedence int) Expression {
	var leftExp Expression
	switch p.curToken.Type {
	case IDENT:
		leftExp = p.parseIdentifier()
	case NUMBER:
		leftExp = p.parseIntegerLiteral()
	case STRING:
		leftExp = p.parseStringLiteral()
	case TRUE_TOK, FALSE_TOK:
		leftExp = p.parseBooleanLiteral()
	case DOLLAR:
		leftExp = p.parseInterpolation()
	case NIL:
		leftExp = p.parseNilLiteral()
	case LPAREN:
		leftExp = p.parseGroupedExpression()
	default:
		return nil
	}

	for p.peekToken.Type != SEMICOLON && p.peekToken.Type != LBRACE && p.peekToken.Type != GREATER && p.peekToken.Type != APPEND && p.peekToken.Type != AND && p.peekToken.Type != OR && p.peekToken.Type != AMPERSAND && precedence < p.peekPrecedence() {
		p.nextToken()
		switch p.curToken.Type {
		case EQ, NEQ, GREATER, LESS, PLUS:
			leftExp = p.parseInfixExpression(leftExp)
		case DOT:
			leftExp = p.parseMemberExpression(leftExp)
		case LPAREN:
			leftExp = p.parseCallExpression(leftExp)
		default:
			return leftExp
		}
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
	val, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		lit.Err = "invalid integer literal: " + p.curToken.Literal
		return lit
	}
	lit.Value = val
	lit.Obj = getIntegerObject(val)
	return lit
}

func (p *Parser) parseStringLiteral() Expression {
	lit := &StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	if strings.IndexByte(lit.Value, '$') < 0 {
		lit.Obj = &String{Value: lit.Value}
	}
	return lit
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
	return precedenceForToken(p.peekToken.Type)
}

func (p *Parser) curPrecedence() int {
	return precedenceForToken(p.curToken.Type)
}

func precedenceForToken(tokenType TokenType) int {
	switch tokenType {
	case EQ, NEQ:
		return EQUALS
	case GREATER, LESS:
		return LESSGREATER
	case PLUS:
		return SUM
	case DOT:
		return MEMBER
	case LPAREN:
		return CALL
	default:
		return LOWEST
	}
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
	if p.peekToken.Type == RPAREN {
		p.nextToken()
		return nil
	}

	identifiers := make([]*Identifier, 0, 4)

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

func (p *Parser) parseVarStatement() *VarStatement {
	stmt := &VarStatement{Token: p.curToken}

	if p.peekToken.Type != IDENT {
		return nil
	}
	p.nextToken()
	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Optional type
	if p.peekToken.Type == IDENT {
		p.nextToken()
		stmt.Type = &Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	// Optional value
	if p.peekToken.Type == ASSIGN {
		p.nextToken() // move to =
		p.nextToken() // move to expression
		stmt.Value = p.parseExpression(LOWEST)
	}

	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseGroupedExpression() Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if p.peekToken.Type != RPAREN {
		return nil
	}
	p.nextToken()

	return exp
}

func (p *Parser) parseMemberExpression(left Expression) Expression {
	exp := &MemberExpression{Token: p.curToken, Object: left}

	p.nextToken()
	if p.curToken.Type != IDENT {
		return nil
	}

	exp.Property = &Identifier{Token: p.curToken, Value: p.curToken.Literal}
	return exp
}

func (p *Parser) parseCallExpression(function Expression) Expression {
	exp := &CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(RPAREN)
	return exp
}

func (p *Parser) parseExpressionList(end TokenType) []Expression {
	if p.peekToken.Type == end {
		p.nextToken()
		return nil
	}

	args := make([]Expression, 0, 4)

	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))

	for p.peekToken.Type == COMMA {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	if p.peekToken.Type == end {
		p.nextToken()
	}

	return args
}
