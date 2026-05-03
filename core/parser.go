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
	INDEX       // arr[i]
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
	case RETURN:
		stmt = p.parseReturnStatement()
	case GO:
		stmt = p.parseGoStatement()
	case IMPORT:
		stmt = p.parseImportStatement()
	case WAIT:
		stmt = p.parseWaitStatement()
	case SWITCH:
		stmt = p.parseSwitchStatement()
	case BREAK:
		stmt = &BreakStatement{Token: p.curToken}
	case CONTINUE:
		stmt = &ContinueStatement{Token: p.curToken}
	case ASTERISK:
		// *p = val (pointer dereference assignment)
		if p.peekToken.Type == IDENT {
			// Look ahead to see if this is *p = val
			// Save state including lexer
			savedCur := p.curToken
			savedPeek := p.peekToken
			savedLexer := p.l.GetPosition()
			p.nextToken() // move to p
			isAssign := p.peekToken.Type == ASSIGN
			// Restore state
			p.curToken = savedCur
			p.peekToken = savedPeek
			p.l.SetPosition(savedLexer)
			if isAssign {
				// This is *p = val
				stmt = p.parsePointerAssignStatement()
			} else {
				// Not an assignment, parse as expression
				stmt = p.parseExpressionStatement()
			}
		} else {
			stmt = p.parseExpressionStatement()
		}
	case IDENT:
		if p.peekToken.Type == COLON_ASSIGN || p.peekToken.Type == ASSIGN {
			if p.peekToken.Type == ASSIGN && p.curToken.End == p.peekToken.Start {
				stmt = p.parseInvalidTightAssignStatement()
				break
			}
			stmt = p.parseAssignStatement()
		} else if p.peekToken.Type == LBRACKET {
			stmt = p.parseIndexAssignOrCommand()
		} else if p.peekToken.Type == DOT {
			// Check for IDENT.IDENT { pattern (method call with block)
			if p.isMethodCallWithBlock() {
				mcbStmt := p.parseMethodCallBlockStatement()
				if mcbStmt != nil {
					stmt = mcbStmt
				} else {
					stmt = p.parseExpressionStatement()
				}
			} else {
				stmt = p.parseExpressionStatement()
			}
		} else if p.peekToken.Type == LPAREN {
			stmt = p.parseExpressionStatement()
		} else {
			stmt = p.parseCommandStatement()
		}
	case LBRACE:
		stmt = p.parseBlockStatement()
	case NUMBER, FLOAT, STRING, TRUE_TOK, FALSE_TOK, DOLLAR:
		stmt = p.parseExpressionStatement()
	default:
		stmt = p.parseCommandStatement()
	}

	for {
		if stmt == nil {
			break
		}
		if p.peekToken.Type == PIPE {
			stmt = p.parsePipeStatement(stmt)
		} else if p.peekToken.Type == REDIRECT || p.peekToken.Type == APPEND {
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

	p.nextToken() // move to -> or >>
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
	stmt.Expression = p.parseExpression(LOWEST)
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
	stmt.Name = p.curToken.Literal

	p.nextToken() // cur is :=
	p.nextToken() // cur is start of expression

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parsePointerAssignStatement() *PointerAssignStatement {
	stmt := &PointerAssignStatement{Token: p.curToken}
	
	// curToken is *, peekToken is p
	// Parse *p as the target
	target := &PrefixExpression{Token: p.curToken, Operator: "*"}
	p.nextToken() // move to p
	target.Right = p.parseIdentifier()
	stmt.Target = target
	
	p.nextToken() // move to =
	p.nextToken() // move to start of expression
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
	if p.curToken.Type == LBRACE {
		stmt.Consequence = p.parseBlockStatement()
		p.classifyForIncrement(stmt)
		return stmt
	}

	// Detect range patterns early: `for range arr`, `for i := range arr`, `for i, v := range arr`
	if p.curToken.Type == RANGE {
		p.nextToken() // consume range
		rangeExpr := p.parseExpression(LOWEST)
		return p.buildRangeFromExpr(stmt, nil, rangeExpr)
	}

	firstExpr := p.parseExpression(LOWEST)

	isAssign := p.peekToken.Type == COLON_ASSIGN || p.peekToken.Type == ASSIGN
	hasInit := isAssign || p.peekToken.Type == SEMICOLON

	// Detect comma pattern: `for i, v := range arr` — peekToken is COMMA after first ident
	if !isAssign && p.peekToken.Type == COMMA {
		if ident, ok := firstExpr.(*Identifier); ok {
			// lookahead: i , v := range arr
			savedCur := p.curToken
			savedPeek := p.peekToken
			savedLexer := p.l.GetPosition()
			p.nextToken() // ,
			p.nextToken() // v
			secondIdent := p.curToken.Literal
			p.nextToken() // :=
			isColonAssign := p.curToken.Type == COLON_ASSIGN
			p.nextToken() // range
			isRange := p.curToken.Type == RANGE
			p.curToken = savedCur
			p.peekToken = savedPeek
			p.l.SetPosition(savedLexer)

			if isColonAssign && isRange {
				p.nextToken() // ,
				p.nextToken() // v
				secondIdent = p.curToken.Literal
				p.nextToken() // :=
				p.nextToken() // range
				p.nextToken() // arr
				rangeExpr := p.parseExpression(LOWEST)
				return p.buildRangeFromExpr(stmt, []string{ident.Value, secondIdent}, rangeExpr)
			}
		}
	}

	if isAssign {
		// Check if this is `ident := range expr`
		if ident, ok := firstExpr.(*Identifier); ok {
			p.nextToken() // consume :=, curToken = :=, peekToken = next
			// Check for := range (peekToken is what follows :=)
			if p.peekToken.Type == RANGE {
				p.nextToken() // curToken = RANGE, peekToken = arr
				p.nextToken() // curToken = arr, peekToken = {
				rangeExpr := p.parseExpression(LOWEST)
				return p.buildRangeFromExpr(stmt, []string{ident.Value}, rangeExpr)
			}
			// Not range, normal three-clause: curToken is :=, parse value from peekToken
			p.nextToken() // move to value
			val := p.parseExpression(LOWEST)
			stmt.Init = &AssignStatement{Token: Token{Type: COLON_ASSIGN, Literal: ":="}, Name: ident.Value, Value: val}
		} else {
			stmt.Init = p.buildForClauseStatement(firstExpr)
		}
	} else if hasInit {
		stmt.Init = &ExpressionStatement{Expression: firstExpr}
	}

	if hasInit {
		if p.curToken.Type != SEMICOLON {
			if p.peekToken.Type == SEMICOLON {
				p.nextToken()
			}
		}
		if p.curToken.Type == SEMICOLON {
			p.nextToken()
		}
		if p.curToken.Type != LBRACE && p.curToken.Type != RBRACE && p.curToken.Type != EOF {
			stmt.Condition = p.parseExpression(LOWEST)
		}
		if p.peekToken.Type == SEMICOLON {
			p.nextToken()
			if p.peekToken.Type != LBRACE {
				p.nextToken()
				if p.curToken.Type != LBRACE && p.curToken.Type != RBRACE && p.curToken.Type != EOF {
					postExpr := p.parseExpression(LOWEST)
					if p.peekToken.Type == COLON_ASSIGN || p.peekToken.Type == ASSIGN {
						stmt.Post = p.buildForClauseStatement(postExpr)
					} else {
						stmt.Post = &ExpressionStatement{Expression: postExpr}
					}
				}
			}
		}
	} else {
		stmt.Condition = firstExpr
	}

	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}
	if p.peekToken.Type == LBRACE {
		p.nextToken()
		stmt.Consequence = p.parseBlockStatement()
	}

	p.classifyForIncrement(stmt)
	return stmt
}

func (p *Parser) buildRangeFromExpr(stmt *ForStatement, vars []string, rangeExpr Expression) *ForStatement {
	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}
	if p.peekToken.Type == LBRACE {
		p.nextToken()
	}

	body := p.parseBlockStatement()

	// Iterator range: rangeExpr is a function call → for v := range iter(args) { ... }
	if _, ok := rangeExpr.(*CallExpression); ok {
		stmt.IsIterRange = true
		stmt.IterCall = rangeExpr
		stmt.IterVars = vars
		stmt.Consequence = body
		return stmt
	}

	// Array range: degrade to three-clause for
	if len(vars) == 0 {
		vars = []string{"_i"}
	}

	initName := vars[0]

	// i := 0
	stmt.Init = &AssignStatement{
		Token: Token{Type: COLON_ASSIGN, Literal: ":="},
		Name:  initName,
		Value: &IntegerLiteral{Value: 0},
	}

	// i < len(arr)
	stmt.Condition = &InfixExpression{
		Token:    Token{Type: LESS, Literal: "<"},
		Left:     &Identifier{Value: initName},
		Operator: "<",
		Right: &CallExpression{
			Function:  &Identifier{Value: "len"},
			Arguments: []Expression{rangeExpr},
		},
	}

	// i = i + 1
	stmt.Post = &AssignStatement{
		Token: Token{Type: ASSIGN, Literal: "="},
		Name:  initName,
		Value: &InfixExpression{
			Left:     &Identifier{Value: initName},
			Operator: "+",
			Right:    &IntegerLiteral{Value: 1},
		},
	}

	// If two variables, prepend v := arr[i] to body
	if len(vars) >= 2 {
		valAssign := &AssignStatement{
			Token: Token{Type: COLON_ASSIGN, Literal: ":="},
			Name:  vars[1],
			Value: &IndexExpression{
				Left:  rangeExpr,
				Index: &Identifier{Value: initName},
			},
		}
		body.Statements = append([]Statement{valAssign}, body.Statements...)
	}

	stmt.Consequence = body
	return stmt
}

func (p *Parser) buildForClauseStatement(expr Expression) Statement {
	if ident, ok := expr.(*Identifier); ok {
		if p.peekToken.Type == COLON_ASSIGN || p.peekToken.Type == ASSIGN {
			op := p.peekToken
			p.nextToken()
			p.nextToken()
			val := p.parseExpression(LOWEST)
			return &AssignStatement{Token: op, Name: ident.Value, Value: val}
		}
	}
	return &ExpressionStatement{Expression: expr}
}

func (p *Parser) classifyForIncrement(stmt *ForStatement) {
	body := stmt.Consequence
	if body == nil || len(body.Statements) != 1 {
		return
	}
	assign, ok := body.Statements[0].(*AssignStatement)
	if !ok || assign.Token.Literal != "=" {
		return
	}
	infix, ok := assign.Value.(*InfixExpression)
	if !ok {
		return
	}
	leftIdent, leftIsIdent := infix.Left.(*Identifier)
	if !leftIsIdent || leftIdent.Value != assign.Name {
		return
	}
	rightLit, rightIsLit := infix.Right.(*IntegerLiteral)
	if !rightIsLit || rightLit.Err != "" {
		return
	}
	if rightLit.Value != 1 && rightLit.Value != -1 {
		return
	}
	switch infix.Operator {
	case "+":
		stmt.IncVarName = assign.Name
		stmt.IncDelta = rightLit.Value
		stmt.HasInc = true
	case "-":
		stmt.IncVarName = assign.Name
		stmt.IncDelta = -rightLit.Value
		stmt.HasInc = true
	}
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

	for p.peekToken.Type != SEMICOLON && p.peekToken.Type != EOF && p.peekToken.Type != RBRACE && p.peekToken.Type != PIPE && p.peekToken.Type != REDIRECT && p.peekToken.Type != APPEND && p.peekToken.Type != AND && p.peekToken.Type != OR && p.peekToken.Type != AMPERSAND {
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
	case FLOAT:
		leftExp = p.parseFloatLiteral()
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
	case GO:
		leftExp = p.parseGoExpression()
	case AMPERSAND:
		leftExp = p.parseAddressExpression()
	case ASTERISK:
		leftExp = p.parseDereferenceExpression()
	case NOT:
		leftExp = p.parseNotExpression()
	case LBRACKET:
		leftExp = p.parseArrayLiteral()
	case FUNC:
		leftExp = p.parseAnonymousFunction()
	default:
		return nil
	}

	for p.peekToken.Type != SEMICOLON && p.peekToken.Type != LBRACE && p.peekToken.Type != APPEND && p.peekToken.Type != AND && p.peekToken.Type != OR && p.peekToken.Type != AMPERSAND && precedence < p.peekPrecedence() {
		p.nextToken()
		switch p.curToken.Type {
		case EQ, NEQ, GREATER, LESS, PLUS, MINUS:
			leftExp = p.parseInfixExpression(leftExp)
		case DOT:
			leftExp = p.parseMemberExpression(leftExp)
		case LPAREN:
			leftExp = p.parseCallExpression(leftExp)
		case LBRACKET:
			leftExp = p.parseIndexExpression(leftExp)
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

func (p *Parser) parseFloatLiteral() Expression {
	lit := &FloatLiteral{Token: p.curToken}
	val, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		lit.Err = "invalid float literal: " + p.curToken.Literal
		return lit
	}
	lit.Value = val
	lit.Obj = &Float{Value: val}
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
	case PLUS, MINUS:
		return SUM
	case DOT:
		return MEMBER
	case LPAREN:
		return CALL
	case LBRACKET:
		return INDEX
	default:
		return LOWEST
	}
}

func (p *Parser) parseArrayLiteral() Expression {
	array := &ArrayLiteral{Token: p.curToken}
	array.Elements = p.parseExpressionList(RBRACKET)
	return array
}

func (p *Parser) parseAnonymousFunction() Expression {
	// curToken = FUNC
	// Parse as a function literal: func(params) { body }
	stmt := &FunctionStatement{Token: p.curToken}

	p.nextToken() // move to name or (
	if p.curToken.Type == LPAREN {
		// Anonymous function with no name
		stmt.Parameters = p.parseFunctionParameters()
	} else {
		// func name(params) — treat name as part of the function
		stmt.Name = p.curToken.Literal
		if p.peekToken.Type == LPAREN {
			p.nextToken()
			stmt.Parameters = p.parseFunctionParameters()
		}
	}

	if p.peekToken.Type == LBRACE {
		p.nextToken()
		stmt.Body = p.parseBlockStatement()
	}

	// Return as FunctionLiteral expression
	return &FunctionLiteral{
		Token:      stmt.Token,
		Parameters: stmt.Parameters,
		Body:       stmt.Body,
	}
}

func (p *Parser) parseIndexExpression(left Expression) Expression {
	exp := &IndexExpression{Token: p.curToken, Left: left}
	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)
	if p.peekToken.Type == RBRACKET {
		p.nextToken()
	}
	return exp
}

func (p *Parser) parseIndexAssignOrCommand() Statement {
	// curToken = IDENT, peekToken = LBRACKET
	// Look ahead: IDENT [ expr ] = val → index assignment
	// Otherwise: fall back to command statement
	savedCur := p.curToken
	savedPeek := p.peekToken
	savedLexer := p.l.GetPosition()

	ident := p.curToken.Literal
	p.nextToken() // move to [
	p.nextToken() // move to expr
	// Skip the index expression (could be complex)
	// We just need to check if after ] there's an =
	depth := 1
	for depth > 0 && p.curToken.Type != EOF {
		if p.curToken.Type == LBRACKET {
			depth++
		} else if p.curToken.Type == RBRACKET {
			depth--
		}
		if depth > 0 {
			p.nextToken()
		}
	}
	// curToken should be ]
	isAssign := p.peekToken.Type == ASSIGN

	// Restore state
	p.curToken = savedCur
	p.peekToken = savedPeek
	p.l.SetPosition(savedLexer)

	if isAssign {
		// Parse as: ident [ indexExpr ] = value
		p.nextToken() // move to [
		p.nextToken() // move to index
		indexExpr := p.parseExpression(LOWEST)
		if p.peekToken.Type == RBRACKET {
			p.nextToken() // consume ]
		}
		p.nextToken() // move to =
		p.nextToken() // move to value
		val := p.parseExpression(LOWEST)
		if p.peekToken.Type == SEMICOLON {
			p.nextToken()
		}
		target := &IndexExpression{
			Token: savedPeek,
			Left:  &Identifier{Token: savedCur, Value: ident},
			Index: indexExpr,
		}
		return &AssignStatement{Token: p.curToken, Target: target, Value: val}
	}

	// Not an assignment, fall back to command statement
	return p.parseCommandStatement()
}

func (p *Parser) parseFunctionStatement() *FunctionStatement {
	stmt := &FunctionStatement{Token: p.curToken}

	p.nextToken() // move to name
	stmt.Name = p.curToken.Literal

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

func (p *Parser) parseFunctionParameters() []string {
	if p.peekToken.Type == RPAREN {
		p.nextToken()
		return nil
	}

	identifiers := make([]string, 0, 4)

	p.nextToken()
	identifiers = append(identifiers, p.curToken.Literal)

	for p.peekToken.Type == COMMA {
		p.nextToken()
		p.nextToken()
		identifiers = append(identifiers, p.curToken.Literal)
	}

	if p.peekToken.Type == RPAREN {
		p.nextToken()
	}

	return identifiers
}

func (p *Parser) parseReturnStatement() *ReturnStatement {
	stmt := &ReturnStatement{Token: p.curToken}

	if p.peekToken.Type != SEMICOLON && p.peekToken.Type != RBRACE && p.peekToken.Type != EOF {
		p.nextToken()
		stmt.ReturnValue = p.parseExpression(LOWEST)
		if p.peekToken.Type == SEMICOLON {
			p.nextToken()
		}
	}

	return stmt
}

func (p *Parser) parseGoStatement() *GoStatement {
	stmt := &GoStatement{Token: p.curToken}
	p.nextToken()
	if p.curToken.Type == LBRACE {
		stmt.Node = p.parseBlockStatement()
	} else if p.curToken.Type == IDENT && p.peekToken.Type == LPAREN {
		stmt.Node = p.parseExpressionStatement()
	} else {
		stmt.Node = p.parseCommandStatement()
	}
	return stmt
}

func (p *Parser) parseGoExpression() *GoExpression {
	expr := &GoExpression{Token: p.curToken}
	p.nextToken()
	if p.curToken.Type == LBRACE {
		expr.Node = p.parseBlockStatement()
	} else if p.curToken.Type == IDENT && p.peekToken.Type == LPAREN {
		expr.Node = p.parseExpressionStatement()
	} else {
		expr.Node = p.parseCommandStatement()
	}
	return expr
}

func (p *Parser) parseWaitStatement() *WaitStatement {
	stmt := &WaitStatement{Token: p.curToken}
	
	// Check if there's a timeout argument: wait(10)
	if p.peekToken.Type == LPAREN {
		p.nextToken() // move to (
		p.nextToken() // move to timeout value
		stmt.Timeout = p.parseExpression(LOWEST)
		if p.peekToken.Type == RPAREN {
			p.nextToken() // move to )
		}
	}
	
	return stmt
}

func (p *Parser) parseSwitchStatement() *SwitchStatement {
	stmt := &SwitchStatement{Token: p.curToken}

	p.nextToken()

	// tagless switch: switch { ... }
	// tagged switch: switch expr { ... }
	if p.curToken.Type == LBRACE {
		// tagless switch, curToken is already {
	} else {
		stmt.Tag = p.parseExpression(LOWEST)
		if p.peekToken.Type == SEMICOLON {
			p.nextToken()
		}
		// expect LBRACE
		if p.peekToken.Type != LBRACE {
			return stmt
		}
		p.nextToken() // move to {
	}
	p.nextToken() // move past {

	for p.curToken.Type != RBRACE && p.curToken.Type != EOF {
		switch p.curToken.Type {
		case CASE:
			clause := p.parseCaseClause()
			stmt.Cases = append(stmt.Cases, clause)
		case DEFAULT:
			clause := p.parseDefaultClause()
			stmt.Cases = append(stmt.Cases, clause)
		default:
			p.nextToken()
		}
	}

	p.classifySwitch(stmt)
	return stmt
}

func (p *Parser) parseCaseClause() CaseClause {
	clause := CaseClause{Token: p.curToken}

	// parse case value list (comma separated)
	p.nextToken()
	clause.Values = append(clause.Values, p.parseExpression(LOWEST))
	p.nextToken()
	for p.curToken.Type == COMMA {
		p.nextToken()
		clause.Values = append(clause.Values, p.parseExpression(LOWEST))
		p.nextToken()
	}

	// curToken should be COLON
	if p.curToken.Type == COLON {
		p.nextToken()
	}

	clause.Body = p.parseCaseBody()
	return clause
}

func (p *Parser) classifySwitch(stmt *SwitchStatement) {
	if len(stmt.Cases) == 0 {
		return
	}
	allInt := true
	allStr := true
	for i := range stmt.Cases {
		c := &stmt.Cases[i]
		if c.Values == nil {
			continue
		}
		caseInts := make([]int64, 0, len(c.Values))
		caseStrs := make([]string, 0, len(c.Values))
		for _, v := range c.Values {
			switch lit := v.(type) {
			case *IntegerLiteral:
				if lit.Err != "" {
					allInt = false
					allStr = false
					caseInts = nil
					caseStrs = nil
					break
				}
				allStr = false
				caseInts = append(caseInts, lit.Value)
			case *StringLiteral:
				if strings.IndexByte(lit.Value, '$') >= 0 {
					allInt = false
					allStr = false
					caseInts = nil
					caseStrs = nil
					break
				}
				allInt = false
				caseStrs = append(caseStrs, lit.Value)
			default:
				allInt = false
				allStr = false
				caseInts = nil
				caseStrs = nil
			}
		}
		if len(caseInts) == len(c.Values) {
			c.IntConsts = caseInts
			c.HasConstVals = true
		} else if len(caseStrs) == len(c.Values) {
			c.StringConsts = caseStrs
			c.HasConstVals = true
		}
	}
	stmt.IntSwitch = allInt
	stmt.StringSwitch = allStr
}

func (p *Parser) parseDefaultClause() CaseClause {
	clause := CaseClause{Token: p.curToken}

	// expect COLON
	if p.peekToken.Type == COLON {
		p.nextToken()
	}
	p.nextToken()

	clause.Body = p.parseCaseBody()
	return clause
}

func (p *Parser) parseCaseBody() *BlockStatement {
	block := &BlockStatement{}
	block.Statements = []Statement{}

	for p.curToken.Type != CASE && p.curToken.Type != DEFAULT &&
		p.curToken.Type != RBRACE && p.curToken.Type != EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

func (p *Parser) isMethodCallWithBlock() bool {
	// Check for IDENT.IDENT { pattern without consuming tokens
	// curToken = IDENT (object), peekToken = DOT
	if p.curToken.Type != IDENT || p.peekToken.Type != DOT {
		return false
	}
	// Save state
	savedCur := p.curToken
	savedPeek := p.peekToken
	savedLexerPos := p.l.GetPosition()
	// Look ahead: DOT IDENT LBRACE
	p.nextToken() // move to DOT
	p.nextToken() // move to method name
	isBlock := p.curToken.Type == IDENT && p.peekToken.Type == LBRACE
	// Restore state
	p.curToken = savedCur
	p.peekToken = savedPeek
	p.l.SetPosition(savedLexerPos)
	return isBlock
}

func (p *Parser) parseMethodCallBlockStatement() *MethodCallBlockStatement {
	stmt := &MethodCallBlockStatement{Token: p.curToken}
	stmt.Object = &Identifier{Token: p.curToken, Value: p.curToken.Literal} // object name

	p.nextToken() // move to .
	p.nextToken() // move to method name

	// Verify it's actually IDENT.IDENT { pattern
	if p.curToken.Type != IDENT || p.peekToken.Type != LBRACE {
		// Not a method call with block, fallback to expression
		// This shouldn't happen if isMethodCallWithBlock worked correctly
		return nil
	}

	stmt.Method = p.curToken.Literal
	p.nextToken() // move to {
	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseImportStatement() *ImportStatement {
	stmt := &ImportStatement{Token: p.curToken}
	
	// Expect a string literal for the import path
	if p.peekToken.Type != STRING {
		return nil
	}
	p.nextToken()
	stmt.Path = p.curToken.Literal
	
	return stmt
}

func (p *Parser) parseVarStatement() *VarStatement {
	stmt := &VarStatement{Token: p.curToken}

	if p.peekToken.Type != IDENT {
		return nil
	}
	p.nextToken()
	stmt.Name = p.curToken.Literal

	// Type is required for var
	if p.peekToken.Type != IDENT {
		return nil  // var x without type is error
	}
	p.nextToken()
	stmt.TypeName = p.curToken.Literal

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

func (p *Parser) parseAddressExpression() Expression {
	exp := &PrefixExpression{Token: p.curToken, Operator: "&"}
	p.nextToken()
	exp.Right = p.parseExpression(PREFIX)
	return exp
}

func (p *Parser) parseDereferenceExpression() Expression {
	exp := &PrefixExpression{Token: p.curToken, Operator: "*"}
	p.nextToken()
	exp.Right = p.parseExpression(PREFIX)
	return exp
}

func (p *Parser) parseNotExpression() Expression {
	exp := &PrefixExpression{Token: p.curToken, Operator: "!"}
	p.nextToken()
	exp.Right = p.parseExpression(PREFIX)
	return exp
}

func (p *Parser) parseMemberExpression(left Expression) Expression {
	exp := &MemberExpression{Token: p.curToken, Object: left}

	p.nextToken()
	if p.curToken.Type != IDENT {
		return nil
	}

	exp.Property = p.curToken.Literal
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
