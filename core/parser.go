package core

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
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
	errors    []string
}

func NewParser(l *Lexer) *Parser {
	p := &Parser{l: l}

	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, msg)
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
		// Multi-value assignment: x, y := expr
		if p.peekToken.Type == COMMA {
			savedCur := p.curToken
			savedPeek := p.peekToken
			savedLexer := p.l.GetPosition()
			// Scan ahead: IDENT, IDENT [, IDENT ...] :=
			names := []string{p.curToken.Literal}
			p.nextToken() // move to ,
			for p.peekToken.Type == COMMA || p.peekToken.Type == IDENT {
				if p.curToken.Type == COMMA && p.peekToken.Type == IDENT {
					p.nextToken() // move to IDENT
					names = append(names, p.curToken.Literal)
					if p.peekToken.Type == COLON_ASSIGN || p.peekToken.Type == ASSIGN {
						// Found multi-assign pattern
						op := p.peekToken
						p.nextToken() // move to :=
						p.nextToken() // move to expression
						val := p.parseExpression(LOWEST)
						if p.peekToken.Type == SEMICOLON {
							p.nextToken()
						}
						stmt = &AssignStatement{Token: op, Names: names, Value: val}
						break
					}
					continue
				}
				p.nextToken()
			}
			if stmt == nil {
				// Not a multi-assign, restore and parse as expression
				p.curToken = savedCur
				p.peekToken = savedPeek
				p.l.SetPosition(savedLexer)
				stmt = p.parseExpressionStatement()
			}
			break
		}
		if p.peekToken.Type == COLON_ASSIGN || p.peekToken.Type == ASSIGN {
			if p.peekToken.Type == ASSIGN && p.curToken.End == p.peekToken.Start {
				stmt = p.parseInvalidTightAssignStatement()
				break
			}
			stmt = p.parseAssignStatement()
		} else if p.peekToken.Type == LBRACKET {
			stmt = p.parseIndexAssignOrCommand()
		} else if p.peekToken.Type == DOT {
			// When there is whitespace between IDENT and DOT, treat as a command
			// with dot-starting arguments (e.g. "cd ..", "cd .", "cd ../foo").
			// No space means member access (e.g. "obj.method").
			hasSpace := p.peekToken.Start > p.curToken.End
			if hasSpace {
				stmt = p.parseCommandStatement()
			} else if p.isMethodCallWithBlock() {
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
	case RBRACE:
		return nil
	case NUMBER, FLOAT, STRING, TRUE_TOK, FALSE_TOK, DOLLAR, LPAREN, NIL, LBRACKET, NOT, MINUS:
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

	// Function call form: exec(cmd)
	if p.curToken.Type == LPAREN {
		p.nextToken() // move past (
		if p.curToken.Type == STRING {
			stmt.CommandStr = p.parseExpression(LOWEST)
		} else if p.curToken.Type == RPAREN {
			// exec() with no args
			stmt.CommandStr = nil
		} else {
			stmt.CommandStr = p.parseExpression(LOWEST)
		}
		if p.peekToken.Type == RPAREN {
			p.nextToken() // move past )
		}
		if p.peekToken.Type == SEMICOLON {
			p.nextToken()
		}
		return stmt
	}

	// Deprecated string form: exec "..."
	if p.curToken.Type == STRING {
		p.addError("exec \"...\" is deprecated, use exec <command> <args> or exec(cmd) instead")
		stmt.CommandStr = p.parseExpression(LOWEST)
		if p.peekToken.Type == SEMICOLON {
			p.nextToken()
		}
		return stmt
	}

	// Bare word form: exec echo hello
	words, delim, nextPos := p.scanCommandWordsWithQuotes(p.curToken.Start)
	for _, word := range words {
		lit := &StringLiteral{
			Token: Token{Type: STRING, Literal: word.Value},
			Value: word.Value,
		}
		if word.SingleQuote || strings.IndexByte(word.Value, '$') < 0 {
			lit.Obj = &String{Value: word.Value}
		} else {
			lit.Parts = parseStringParts(word.Value)
		}
		stmt.Args = append(stmt.Args, lit)
	}

	p.peekToken = delim
	p.setLexerPosition(nextPos, delim.Type)
	return stmt
}

func (p *Parser) parseAssignStatement() *AssignStatement {
	stmt := &AssignStatement{Token: p.peekToken}
	stmt.Names = []string{p.curToken.Literal}

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
		if p.peekToken.Type == IF {
			// else if — parse inner if and wrap in a single-statement block
			p.nextToken()
			innerIf := p.parseIfStatement()
			stmt.Alternative = &BlockStatement{
				Token:      innerIf.Token,
				Statements: []Statement{innerIf},
			}
		} else if p.peekToken.Type == LBRACE {
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
			stmt.Init = &AssignStatement{Token: Token{Type: COLON_ASSIGN, Literal: ":="}, Names: []string{ident.Value}, Value: val}
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
		Names: []string{initName},
		Value: &IntegerLiteral{Value: 0},
	}

	// i < len(arr)
	stmt.Condition = &InfixExpression{
		Token:    Token{Type: LESS, Literal: "<"},
		Left:     &Identifier{Value: initName},
		Operator: "<",
		Right: &CallExpression{
			Function:  NewIdentifier(Token{}, "len"),
			Arguments: []Expression{rangeExpr},
		},
	}

	// i = i + 1
	stmt.Post = &AssignStatement{
		Token: Token{Type: ASSIGN, Literal: "="},
		Names: []string{initName},
		Value: &InfixExpression{
			Left:     NewIdentifier(Token{}, initName),
			Operator: "+",
			Right:    &IntegerLiteral{Value: 1},
		},
	}

	// If two variables, prepend v := arr[i] to body
	if len(vars) >= 2 {
		valAssign := &AssignStatement{
			Token: Token{Type: COLON_ASSIGN, Literal: ":="},
			Names: []string{vars[1]},
			Value: &IndexExpression{
				Left:  rangeExpr,
				Index: NewIdentifier(Token{}, initName),
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
			return &AssignStatement{Token: op, Names: []string{ident.Value}, Value: val}
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
	if !leftIsIdent || leftIdent.Value != assign.Names[0] {
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
		stmt.IncVarName = assign.Names[0]
		stmt.IncDelta = rightLit.Value
		stmt.HasInc = true
	case "-":
		stmt.IncVarName = assign.Names[0]
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

	words, delim, nextPos := p.scanCommandWords(p.curToken.Start)
	if len(words) > 0 {
		stmt.Name = words[0]
		for _, arg := range words[1:] {
			stmt.Arguments = append(stmt.Arguments, newCommandArgLiteral(arg))
		}
	}

	p.peekToken = delim
	p.setLexerPosition(nextPos, delim.Type)
	return stmt
}

// commandWord represents a parsed command argument with quote information.
type commandWord struct {
	Value       string
	SingleQuote bool // true if the word was single-quoted (no interpolation)
}

func (p *Parser) scanCommandWords(start int) ([]string, Token, int) {
	if p.l == nil {
		return nil, Token{Type: EOF, Start: p.curToken.End, End: p.curToken.End}, p.curToken.End
	}
	input := p.l.input
	n := len(input)
	i := start
	words := make([]string, 0, 4)

	skipSpaces := func() {
		for i < n {
			switch input[i] {
			case ' ', '\t', '\r':
				i++
			default:
				return
			}
		}
	}

	skipSpaces()
	for i < n {
		delim, nextPos, ok := scanCommandDelimiter(input, i)
		if ok {
			return words, delim, nextPos
		}

		word, nextPos, ok := scanCommandWord(input, i)
		if !ok {
			// Defensive progress in malformed inputs.
			i++
			skipSpaces()
			continue
		}
		words = append(words, word)
		i = nextPos
		skipSpaces()
	}

	return words, Token{Type: EOF, Start: n, End: n}, n
}

func (p *Parser) scanCommandWordsWithQuotes(start int) ([]commandWord, Token, int) {
	if p.l == nil {
		return nil, Token{Type: EOF, Start: p.curToken.End, End: p.curToken.End}, p.curToken.End
	}
	input := p.l.input
	n := len(input)
	i := start
	words := make([]commandWord, 0, 4)

	skipSpaces := func() {
		for i < n {
			switch input[i] {
			case ' ', '\t', '\r':
				i++
			default:
				return
			}
		}
	}

	skipSpaces()
	for i < n {
		delim, nextPos, ok := scanCommandDelimiter(input, i)
		if ok {
			return words, delim, nextPos
		}

		word, nextPos, ok := scanCommandWordWithQuote(input, i)
		if !ok {
			// Defensive progress in malformed inputs.
			i++
			skipSpaces()
			continue
		}
		words = append(words, word)
		i = nextPos
		skipSpaces()
	}

	return words, Token{Type: EOF, Start: n, End: n}, n
}

func scanCommandDelimiter(input string, i int) (Token, int, bool) {
	n := len(input)
	if i >= n {
		return Token{Type: EOF, Start: n, End: n}, n, true
	}

	ch := input[i]
	switch ch {
	case '\n':
		return Token{Type: SEMICOLON, Literal: ";", Start: i, End: i + 1}, i + 1, true
	case ';':
		return Token{Type: SEMICOLON, Literal: ";", Start: i, End: i + 1}, i + 1, true
	case '|':
		if i+1 < n && input[i+1] == '|' {
			return Token{Type: OR, Literal: "||", Start: i, End: i + 2}, i + 2, true
		}
		return Token{Type: PIPE, Literal: "|", Start: i, End: i + 1}, i + 1, true
	case '&':
		if i+1 < n && input[i+1] == '&' {
			return Token{Type: AND, Literal: "&&", Start: i, End: i + 2}, i + 2, true
		}
		return Token{Type: AMPERSAND, Literal: "&", Start: i, End: i + 1}, i + 1, true
	case '>':
		if i+1 < n && input[i+1] == '>' {
			return Token{Type: APPEND, Literal: ">>", Start: i, End: i + 2}, i + 2, true
		}
	case '-':
		if i+1 < n && input[i+1] == '>' {
			return Token{Type: REDIRECT, Literal: "->", Start: i, End: i + 2}, i + 2, true
		}
	case '}':
		return Token{Type: RBRACE, Literal: "}", Start: i, End: i + 1}, i + 1, true
	case '/':
		if i+1 < n && input[i+1] == '/' {
			j := i + 2
			for j < n && input[j] != '\n' {
				j++
			}
			if j < n && input[j] == '\n' {
				return Token{Type: SEMICOLON, Literal: ";", Start: i, End: j + 1}, j + 1, true
			}
			return Token{Type: EOF, Start: n, End: n}, n, true
		}
	}

	return Token{}, i, false
}

func scanCommandWord(input string, i int) (string, int, bool) {
	n := len(input)
	if i >= n {
		return "", i, false
	}

	if input[i] == '"' || input[i] == '\'' {
		return scanQuotedCommandWord(input, i)
	}

	var out strings.Builder
	for i < n {
		if delim, _, ok := scanCommandDelimiter(input, i); ok && delim.Type != EOF {
			break
		}
		ch := input[i]
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			break
		}
		if ch == '\\' && i+1 < n {
			out.WriteByte(input[i+1])
			i += 2
			continue
		}
		out.WriteByte(ch)
		i++
	}

	if out.Len() == 0 {
		return "", i, false
	}
	return out.String(), i, true
}

func scanQuotedCommandWord(input string, i int) (string, int, bool) {
	n := len(input)
	if i >= n {
		return "", i, false
	}
	quote := input[i]
	i++

	var out strings.Builder
	for i < n {
		ch := input[i]
		if ch == quote {
			return out.String(), i + 1, true
		}
		if quote == '"' && ch == '\\' && i+1 < n {
			i++
			switch input[i] {
			case 'n':
				out.WriteByte('\n')
			case 't':
				out.WriteByte('\t')
			case 'r':
				out.WriteByte('\r')
			case '"':
				out.WriteByte('"')
			case '\\':
				out.WriteByte('\\')
			default:
				out.WriteByte(input[i])
			}
			i++
			continue
		}
		if quote == '\'' && ch == '\\' && i+1 < n && input[i+1] == '\'' {
			out.WriteByte('\'')
			i += 2
			continue
		}
		out.WriteByte(ch)
		i++
	}

	// Unterminated quote: keep the accumulated content as a best-effort argument.
	return out.String(), i, true
}

func scanCommandWordWithQuote(input string, i int) (commandWord, int, bool) {
	n := len(input)
	if i >= n {
		return commandWord{}, i, false
	}

	if input[i] == '\'' {
		word, nextPos, ok := scanQuotedCommandWord(input, i)
		if !ok {
			return commandWord{}, i, false
		}
		return commandWord{Value: word, SingleQuote: true}, nextPos, true
	}

	if input[i] == '"' {
		word, nextPos, ok := scanQuotedCommandWord(input, i)
		if !ok {
			return commandWord{}, i, false
		}
		return commandWord{Value: word, SingleQuote: false}, nextPos, true
	}

	var out strings.Builder
	for i < n {
		if delim, _, ok := scanCommandDelimiter(input, i); ok && delim.Type != EOF {
			break
		}
		ch := input[i]
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			break
		}
		if ch == '\\' && i+1 < n {
			out.WriteByte(input[i+1])
			i += 2
			continue
		}
		out.WriteByte(ch)
		i++
	}

	if out.Len() == 0 {
		return commandWord{}, i, false
	}
	return commandWord{Value: out.String(), SingleQuote: false}, i, true
}

func newCommandArgLiteral(value string) *StringLiteral {
	lit := &StringLiteral{
		Token: Token{Type: STRING, Literal: value},
		Value: value,
	}
	if strings.IndexByte(value, '$') < 0 {
		lit.Obj = &String{Value: value}
	} else {
		lit.Parts = parseStringParts(value)
	}
	return lit
}

func (p *Parser) setLexerPosition(pos int, prevToken TokenType) {
	if p.l == nil {
		return
	}
	n := len(p.l.input)
	state := LexerState{PrevToken: prevToken}
	if pos >= n {
		state.Position = n
		state.ReadPosition = n
		state.Ch = 0
		p.l.SetPosition(state)
		return
	}
	state.Position = pos
	state.ReadPosition = pos + 1
	state.Ch = p.l.input[pos]
	p.l.SetPosition(state)
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
	case MINUS:
		leftExp = p.parseNegateExpression()
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
		case EQ, NEQ, GREATER, LESS, GEQ, LEQ, PLUS, MINUS, ASTERISK, SLASH, MODULO:
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
	return NewIdentifier(p.curToken, p.curToken.Literal)
}

func (p *Parser) parseInterpolation() Expression {
	p.nextToken() // consume $
	return NewIdentifier(p.curToken, p.curToken.Literal)
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
	} else {
		lit.Parts = parseStringParts(lit.Value)
	}
	return lit
}

// parseStringParts splits a string containing $var references into segments.
// "hello $name from $HOME" → [{Text:"hello "}, {Var:"name"}, {Text:" from "}, {Var:"HOME"}]
func parseStringParts(s string) []StringPart {
	var parts []StringPart
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '$' {
			if i > start {
				parts = append(parts, StringPart{Text: s[start:i]})
			}
			i++ // skip $
			vstart := i
			for i < len(s) {
				if s[i] < utf8.RuneSelf {
					if !isASCIIIdentChar(s[i]) {
						break
					}
					i++
				} else {
					r, size := utf8.DecodeRuneInString(s[i:])
					if r == utf8.RuneError || !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
						break
					}
					i += size
				}
			}
			parts = append(parts, StringPart{Var: s[vstart:i]})
			start = i
			i-- // will be incremented by for loop
		}
	}
	if start < len(s) {
		parts = append(parts, StringPart{Text: s[start:]})
	}
	return parts
}

func isIdentChar(b byte) bool {
	return isASCIIIdentChar(b)
}

func isASCIIIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
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
	case GREATER, LESS, GEQ, LEQ:
		return LESSGREATER
	case PLUS, MINUS:
		return SUM
	case ASTERISK, SLASH, MODULO:
		return PRODUCT
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
	stmt := &FunctionStatement{Token: p.curToken}

	p.nextToken() // move to name or (
	if p.curToken.Type == LPAREN {
		stmt.Parameters = p.parseFunctionParameters()
	} else {
		stmt.Name = p.curToken.Literal
		if p.peekToken.Type == LPAREN {
			p.nextToken()
			stmt.Parameters = p.parseFunctionParameters()
		}
	}

	stmt.ReturnTypes = p.parseFunctionReturnTypes()

	if p.peekToken.Type == LBRACE {
		p.nextToken()
		stmt.Body = p.parseBlockStatement()
	}

	return &FunctionLiteral{
		Token:       stmt.Token,
		Parameters:  stmt.Parameters,
		ReturnTypes: stmt.ReturnTypes,
		Body:        stmt.Body,
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
			Left:  NewIdentifier(savedCur, ident),
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

	stmt.ReturnTypes = p.parseFunctionReturnTypes()

	if p.peekToken.Type == LBRACE {
		p.nextToken() // move to {
		stmt.Body = p.parseBlockStatement()
		p.nextToken() // move past }
	}

	return stmt
}

func (p *Parser) parseFunctionParameters() []Parameter {
	if p.peekToken.Type == RPAREN {
		p.nextToken()
		return nil
	}

	params := make([]Parameter, 0, 4)
	names := make([]string, 0, 2)

	p.nextToken() // move to first token
	names = append(names, p.curToken.Literal)

	for {
		if p.peekToken.Type == RPAREN {
			// End of params — type must have been provided
			if len(names) > 0 {
				// Names without type — convert to untyped params
				p.nextToken() // consume )
				for _, n := range names {
					params = append(params, Parameter{Name: n})
				}
				return params
			}
			break
		}
		if p.peekToken.Type == LBRACE {
			if len(names) > 0 {
				for _, n := range names {
					params = append(params, Parameter{Name: n})
				}
				return params
			}
			break
		}

		if p.peekToken.Type == COMMA {
			// Another name in the same group
			p.nextToken() // skip comma
			p.nextToken() // move to next name
			names = append(names, p.curToken.Literal)
			continue
		}

		if p.peekToken.Type == IDENT {
			// Type annotation — apply to all collected names
			p.nextToken() // move to type
			typeName := p.curToken.Literal
			for _, n := range names {
				params = append(params, Parameter{Name: n, TypeName: typeName})
			}
			names = names[:0] // reset names

			// Check for comma (more params) or rparen/lbrace (end)
			if p.peekToken.Type == COMMA {
				p.nextToken() // skip comma
				p.nextToken() // move to next param name
				names = append(names, p.curToken.Literal)
				continue
			}
			break
		}

		// Unexpected token
		break
	}

	if p.peekToken.Type == RPAREN {
		p.nextToken()
	}

	return params
}

func (p *Parser) parseFunctionReturnTypes() []string {
	// Called after ) has been consumed
	// peek is either LBRACE (no return type) or IDENT/type (return types)
	if p.peekToken.Type == LBRACE || p.peekToken.Type == SEMICOLON || p.peekToken.Type == EOF {
		return nil
	}

	// Single return type: func foo() int {
	if p.peekToken.Type == IDENT {
		p.nextToken()
		return []string{p.curToken.Literal}
	}

	// Multi return types: func foo() (int, error) {
	if p.peekToken.Type == LPAREN {
		p.nextToken() // skip (
		types := make([]string, 0, 2)
		p.nextToken() // move to first type
		types = append(types, p.curToken.Literal)
		for p.peekToken.Type == COMMA {
			p.nextToken() // skip comma
			p.nextToken() // move to next type
			types = append(types, p.curToken.Literal)
		}
		if p.peekToken.Type == RPAREN {
			p.nextToken() // skip )
		}
		return types
	}

	return nil
}

func (p *Parser) parseReturnStatement() *ReturnStatement {
	stmt := &ReturnStatement{Token: p.curToken}

	if p.peekToken.Type != SEMICOLON && p.peekToken.Type != RBRACE && p.peekToken.Type != EOF {
		p.nextToken()
		stmt.ReturnValues = []Expression{p.parseExpression(LOWEST)}
		for p.peekToken.Type == COMMA {
			p.nextToken() // skip comma
			p.nextToken() // move to next expression
			stmt.ReturnValues = append(stmt.ReturnValues, p.parseExpression(LOWEST))
		}
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
	stmt.Object = NewIdentifier(p.curToken, p.curToken.Literal) // object name

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

	// Type is optional if = follows directly (type inferred from value)
	if p.peekToken.Type == IDENT {
		p.nextToken()
		stmt.TypeName = p.curToken.Literal
	} else if p.peekToken.Type != ASSIGN && p.peekToken.Type != SEMICOLON && p.peekToken.Type != EOF {
		return nil
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
		p.addError("expected ')' to close grouped expression, got " + string(p.peekToken.Type))
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

func (p *Parser) parseNegateExpression() Expression {
	exp := &PrefixExpression{Token: p.curToken, Operator: "-"}
	p.nextToken()
	exp.Right = p.parseExpression(PREFIX)
	return exp
}

func (p *Parser) parseMemberExpression(left Expression) Expression {
	exp := &MemberExpression{Token: p.curToken, Object: left}

	p.nextToken()
	if p.curToken.Type != IDENT {
		p.addError("expected identifier after '.', got " + string(p.curToken.Type))
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
	} else {
		p.addError("expected " + string(end) + " to close expression list, got " + string(p.peekToken.Type))
	}

	return args
}
