package core

import (
	"testing"
)

func TestParseAssignStatement(t *testing.T) {
	input := `x := 5
	name := "kami"
	valid := true`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 3 {
		t.Fatalf("program.Statements does not contain 3 statements. got=%d", len(program.Statements))
	}

	tests := []struct {
		expectedIdentifier string
		expectedValue      string
	}{
		{"x", "5"},
		{"name", "\"kami\""},
		{"valid", "true"},
	}

	for i, tt := range tests {
		stmt := program.Statements[i]
		assignStmt, ok := stmt.(*AssignStatement)
		if !ok {
			t.Fatalf("test[%d] - stmt is not *AssignStatement. got=%T", i, stmt)
		}

		if assignStmt.Names[0] != tt.expectedIdentifier {
			t.Errorf("test[%d] - assignStmt.Names[0] not %s. got=%s", i, tt.expectedIdentifier, assignStmt.Names[0])
		}

		if assignStmt.Value.String() != tt.expectedValue {
			t.Errorf("test[%d] - assignStmt.Value.String() not %s. got=%s", i, tt.expectedValue, assignStmt.Value.String())
		}
	}
}

func TestParseCommandStatement(t *testing.T) {
	input := `ls "-la"`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*CommandStatement)
	if !ok {
		t.Fatalf("stmt is not *CommandStatement. got=%T", program.Statements[0])
	}

	if stmt.Name != "ls" {
		t.Errorf("stmt.Name not %s. got=%s", "ls", stmt.Name)
	}

	if len(stmt.Arguments) != 1 {
		t.Errorf("len(stmt.Arguments) not 1. got=%d", len(stmt.Arguments))
	}

	arg, ok := stmt.Arguments[0].(*StringLiteral)
	if !ok {
		t.Fatalf("argument is not *StringLiteral. got=%T", stmt.Arguments[0])
	}

	if arg.Value != "-la" {
		t.Errorf("argument value not %s. got=%s", "-la", arg.Value)
	}
}

func TestParseVarStatement(t *testing.T) {
	input := `var count int = 42
var name string`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 2 {
		t.Fatalf("program.Statements does not contain 2 statements. got=%d", len(program.Statements))
	}

	stmt0, ok := program.Statements[0].(*VarStatement)
	if !ok {
		t.Fatalf("stmt0 is not *VarStatement. got=%T", program.Statements[0])
	}
	if stmt0.Name != "count" || stmt0.TypeName != "int" || stmt0.Value.String() != "42" {
		t.Fatalf("unexpected first var statement: %#v", stmt0)
	}

	stmt1, ok := program.Statements[1].(*VarStatement)
	if !ok {
		t.Fatalf("stmt1 is not *VarStatement. got=%T", program.Statements[1])
	}
	if stmt1.Name != "name" || stmt1.TypeName != "string" || stmt1.Value != nil {
		t.Fatalf("unexpected second var statement: %#v", stmt1)
	}
}

func TestParseCommandStatementKeepsKeyValueArgument(t *testing.T) {
	input := `target_env "app" GOOS=linux`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*CommandStatement)
	if !ok {
		t.Fatalf("stmt is not *CommandStatement. got=%T", program.Statements[0])
	}

	if len(stmt.Arguments) != 2 {
		t.Fatalf("expected 2 arguments, got=%d", len(stmt.Arguments))
	}
	if stmt.Arguments[1].String() != `"GOOS=linux"` {
		t.Fatalf("expected key=value to stay one argument, got %s", stmt.Arguments[1].String())
	}
}

func TestParseCommandStatementKeepsMultipleKeyValueArguments(t *testing.T) {
	input := `target_env "app" GOOS=linux GOARCH=amd64 CGO_ENABLED=0`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*CommandStatement)
	if !ok {
		t.Fatalf("stmt is not *CommandStatement. got=%T", program.Statements[0])
	}

	if len(stmt.Arguments) != 4 {
		t.Fatalf("expected 4 arguments, got=%d", len(stmt.Arguments))
	}

	for i, want := range []string{`"app"`, `"GOOS=linux"`, `"GOARCH=amd64"`, `"CGO_ENABLED=0"`} {
		if got := stmt.Arguments[i].String(); got != want {
			t.Fatalf("argument[%d] expected %s, got %s", i, want, got)
		}
	}
}

func TestParsePipelineWithLogicalAndBackgroundStatements(t *testing.T) {
	input := `print "a" | cat && print "b" &`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	background, ok := program.Statements[0].(*BackgroundStatement)
	if !ok {
		t.Fatalf("stmt is not *BackgroundStatement. got=%T", program.Statements[0])
	}

	logical, ok := background.Stmt.(*LogicalStatement)
	if !ok {
		t.Fatalf("background stmt is not *LogicalStatement. got=%T", background.Stmt)
	}

	if logical.Operator != "&&" {
		t.Fatalf("expected logical operator &&, got %q", logical.Operator)
	}

	pipe, ok := logical.Left.(*PipeStatement)
	if !ok {
		t.Fatalf("left stmt is not *PipeStatement. got=%T", logical.Left)
	}

	if len(pipe.Commands) != 2 {
		t.Fatalf("expected 2 pipeline commands, got=%d", len(pipe.Commands))
	}

	right, ok := logical.Right.(*PrintStatement)
	if !ok {
		t.Fatalf("right stmt is not *PrintStatement. got=%T", logical.Right)
	}

	if right.TokenLiteral() != "print" {
		t.Fatalf("expected right command print, got %q", right.TokenLiteral())
	}
}

// --- Else-if chains ---

func TestParseElseIfChain(t *testing.T) {
	input := `if x > 10 {
    print "big"
} else if x > 5 {
    print "medium"
} else {
    print "small"
}`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	// else if is now a single chained IfStatement
	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*IfStatement)
	if !ok {
		t.Fatalf("stmt is not *IfStatement. got=%T", program.Statements[0])
	}

	if stmt.Alternative == nil {
		t.Fatal("expected alternative for else-if")
	}

	block := stmt.Alternative
	if len(block.Statements) != 1 {
		t.Fatalf("expected 1 statement in alternative block, got %d", len(block.Statements))
	}

	innerIf, ok := block.Statements[0].(*IfStatement)
	if !ok {
		t.Fatalf("inner statement is not *IfStatement. got=%T", block.Statements[0])
	}

	// The inner if should have its own alternative (final else)
	if innerIf.Alternative == nil {
		t.Fatal("expected alternative for final else")
	}
}

// --- Switch ---

func TestParseSwitchStatement(t *testing.T) {
	input := `switch x {
case 1:
    print "one"
case 2:
    print "two"
default:
    print "other"
}`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*SwitchStatement)
	if !ok {
		t.Fatalf("stmt is not *SwitchStatement. got=%T", program.Statements[0])
	}

	if stmt.Tag == nil {
		t.Fatal("expected switch tag")
	}

	if len(stmt.Cases) != 3 {
		t.Fatalf("expected 3 cases, got %d", len(stmt.Cases))
	}

	// First case: 1
	if stmt.Cases[0].Values == nil {
		t.Fatal("expected case values for first case")
	}

	// Default case
	if stmt.Cases[2].Values != nil {
		t.Fatal("expected nil values for default case")
	}
}

// --- Iterator range ---

func TestParseForRange(t *testing.T) {
	// Array range degrades to three-clause for, not IsIterRange
	input := `for i, v := range arr {
    print v
}`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ForStatement)
	if !ok {
		t.Fatalf("stmt is not *ForStatement. got=%T", program.Statements[0])
	}

	// Array range is NOT IsIterRange; it's degraded to three-clause for
	if stmt.IsIterRange {
		t.Fatal("expected IsIterRange to be false for array range")
	}

	// Init should be the range variable declaration
	if stmt.Init == nil {
		t.Fatal("expected init statement")
	}
}

// --- Prefix & and * ---

func TestParsePrefixAddressOf(t *testing.T) {
	input := `p := &x`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*AssignStatement)
	if !ok {
		t.Fatalf("stmt is not *AssignStatement. got=%T", program.Statements[0])
	}

	prefix, ok := stmt.Value.(*PrefixExpression)
	if !ok {
		t.Fatalf("value is not *PrefixExpression. got=%T", stmt.Value)
	}

	if prefix.Operator != "&" {
		t.Errorf("expected operator &, got %q", prefix.Operator)
	}
}

func TestParsePrefixDereference(t *testing.T) {
	input := `x := *p`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*AssignStatement)
	if !ok {
		t.Fatalf("stmt is not *AssignStatement. got=%T", program.Statements[0])
	}

	prefix, ok := stmt.Value.(*PrefixExpression)
	if !ok {
		t.Fatalf("value is not *PrefixExpression. got=%T", stmt.Value)
	}

	if prefix.Operator != "*" {
		t.Errorf("expected operator *, got %q", prefix.Operator)
	}
}

// --- Import ---

func TestParseImportStatement(t *testing.T) {
	input := `import "Go/fmt"`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ImportStatement)
	if !ok {
		t.Fatalf("stmt is not *ImportStatement. got=%T", program.Statements[0])
	}

	if stmt.Path != "Go/fmt" {
		t.Errorf("expected path 'Go/fmt', got %q", stmt.Path)
	}
}

// --- Exec ---

func TestParseExecStatement(t *testing.T) {
	input := `exec "echo hello"`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExecStatement)
	if !ok {
		t.Fatalf("stmt is not *ExecStatement. got=%T", program.Statements[0])
	}

	if stmt.CommandStr == nil {
		t.Fatal("expected command string expression")
	}
}

// --- >= and <= ---

func TestParseGeqLeqOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"if x >= 5 { print 1 }", ">="},
		{"if x <= 5 { print 1 }", "<="},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		program := p.ParseProgram()

		if len(program.Statements) != 1 {
			t.Fatalf("input %q: expected 1 statement, got %d", tt.input, len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*IfStatement)
		if !ok {
			t.Fatalf("input %q: stmt is not *IfStatement. got=%T", tt.input, program.Statements[0])
		}

		infix, ok := stmt.Condition.(*InfixExpression)
		if !ok {
			t.Fatalf("input %q: condition is not *InfixExpression. got=%T", tt.input, stmt.Condition)
		}

		if infix.Operator != tt.expected {
			t.Errorf("input %q: expected operator %q, got %q", tt.input, tt.expected, infix.Operator)
		}
	}
}

// --- Prefix - (negation) ---

func TestParsePrefixNegation(t *testing.T) {
	input := `x := -5`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*AssignStatement)
	if !ok {
		t.Fatalf("stmt is not *AssignStatement. got=%T", program.Statements[0])
	}

	prefix, ok := stmt.Value.(*PrefixExpression)
	if !ok {
		t.Fatalf("value is not *PrefixExpression. got=%T", stmt.Value)
	}

	if prefix.Operator != "-" {
		t.Errorf("expected operator -, got %q", prefix.Operator)
	}
}

// --- Modulo ---

func TestParseModuloOperator(t *testing.T) {
	input := `x := 10 % 3`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*AssignStatement)
	if !ok {
		t.Fatalf("stmt is not *AssignStatement. got=%T", program.Statements[0])
	}

	infix, ok := stmt.Value.(*InfixExpression)
	if !ok {
		t.Fatalf("value is not *InfixExpression. got=%T", stmt.Value)
	}

	if infix.Operator != "%" {
		t.Errorf("expected operator %%, got %q", infix.Operator)
	}
}

// --- Grouped expression as statement ---

func TestParseGroupedExpressionStatement(t *testing.T) {
	input := `(5 + 3)`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("stmt is not *ExpressionStatement. got=%T", program.Statements[0])
	}

	infix, ok := stmt.Expression.(*InfixExpression)
	if !ok {
		t.Fatalf("expression is not *InfixExpression. got=%T", stmt.Expression)
	}

	if infix.Operator != "+" {
		t.Errorf("expected operator +, got %q", infix.Operator)
	}
}

func TestParsePrintGroupedExpression(t *testing.T) {
	input := `print (5 + 3)`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*PrintStatement)
	if !ok {
		t.Fatalf("stmt is not *PrintStatement. got=%T", program.Statements[0])
	}

	infix, ok := stmt.Expression.(*InfixExpression)
	if !ok {
		t.Fatalf("expression is not *InfixExpression. got=%T", stmt.Expression)
	}

	if infix.Operator != "+" {
		t.Errorf("expected operator +, got %q", infix.Operator)
	}
}

func TestParseIfGroupedCondition(t *testing.T) {
	input := `if (x > 5) { print "big" }`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*IfStatement)
	if !ok {
		t.Fatalf("stmt is not *IfStatement. got=%T", program.Statements[0])
	}

	// Condition should be a grouped expression (InfixExpression)
	infix, ok := stmt.Condition.(*InfixExpression)
	if !ok {
		t.Fatalf("condition is not *InfixExpression. got=%T", stmt.Condition)
	}

	if infix.Operator != ">" {
		t.Errorf("expected operator >, got %q", infix.Operator)
	}
}

func TestParseNilLiteralStatement(t *testing.T) {
	input := `nil`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("stmt is not *ExpressionStatement. got=%T", program.Statements[0])
	}

	_, ok = stmt.Expression.(*NilLiteral)
	if !ok {
		t.Fatalf("expression is not *NilLiteral. got=%T", stmt.Expression)
	}
}

func TestParseArrayLiteralStatement(t *testing.T) {
	input := `[1, 2, 3]`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("stmt is not *ExpressionStatement. got=%T", program.Statements[0])
	}

	arr, ok := stmt.Expression.(*ArrayLiteral)
	if !ok {
		t.Fatalf("expression is not *ArrayLiteral. got=%T", stmt.Expression)
	}

	if len(arr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr.Elements))
	}
}

func TestParsePrefixNotStatement(t *testing.T) {
	input := `!true`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("stmt is not *ExpressionStatement. got=%T", program.Statements[0])
	}

	prefix, ok := stmt.Expression.(*PrefixExpression)
	if !ok {
		t.Fatalf("expression is not *PrefixExpression. got=%T", stmt.Expression)
	}

	if prefix.Operator != "!" {
		t.Errorf("expected operator !, got %q", prefix.Operator)
	}
}

func TestParsePrefixNegationStatement(t *testing.T) {
	input := `-5`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatalf("stmt is not *ExpressionStatement. got=%T", program.Statements[0])
	}

	prefix, ok := stmt.Expression.(*PrefixExpression)
	if !ok {
		t.Fatalf("expression is not *PrefixExpression. got=%T", stmt.Expression)
	}

	if prefix.Operator != "-" {
		t.Errorf("expected operator -, got %q", prefix.Operator)
	}
}

// --- Exec Keyword Form (target behavior) ---
// These tests constrain the expected behavior of the new exec implementation.
// They should FAIL or produce compilation errors until the implementation is done.

// 关键字形式：基本裸词
func TestParseExecBareWordBasic(t *testing.T) {
	input := `exec echo hello`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExecStatement)
	if !ok {
		t.Fatalf("stmt is not *ExecStatement. got=%T", program.Statements[0])
	}

	if len(stmt.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(stmt.Args))
	}

	arg0, ok := stmt.Args[0].(*StringLiteral)
	if !ok {
		t.Fatalf("arg[0] is not *StringLiteral. got=%T", stmt.Args[0])
	}
	if arg0.Value != "echo" {
		t.Errorf("expected arg[0] %q, got %q", "echo", arg0.Value)
	}

	arg1, ok := stmt.Args[1].(*StringLiteral)
	if !ok {
		t.Fatalf("arg[1] is not *StringLiteral. got=%T", stmt.Args[1])
	}
	if arg1.Value != "hello" {
		t.Errorf("expected arg[1] %q, got %q", "hello", arg1.Value)
	}
}

// 关键字形式：多参数
func TestParseExecBareWordMultipleArgs(t *testing.T) {
	input := `exec ls -la /tmp`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	stmt, ok := program.Statements[0].(*ExecStatement)
	if !ok {
		t.Fatalf("stmt is not *ExecStatement. got=%T", program.Statements[0])
	}

	if len(stmt.Args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(stmt.Args))
	}

	expected := []string{"ls", "-la", "/tmp"}
	for i, exp := range expected {
		arg, ok := stmt.Args[i].(*StringLiteral)
		if !ok {
			t.Fatalf("arg[%d] is not *StringLiteral. got=%T", i, stmt.Args[i])
		}
		if arg.Value != exp {
			t.Errorf("expected arg[%d] %q, got %q", i, exp, arg.Value)
		}
	}
}

// 关键字形式：无参数
func TestParseExecBareWordNoArgs(t *testing.T) {
	input := `exec`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	stmt, ok := program.Statements[0].(*ExecStatement)
	if !ok {
		t.Fatalf("stmt is not *ExecStatement. got=%T", program.Statements[0])
	}

	if len(stmt.Args) != 0 {
		t.Fatalf("expected 0 args, got %d", len(stmt.Args))
	}
}

// 关键字形式：空参数后跟分号
func TestParseExecBareWordEmptyWithSemicolon(t *testing.T) {
	input := `exec; print 1`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExecStatement)
	if !ok {
		t.Fatalf("stmt is not *ExecStatement. got=%T", program.Statements[0])
	}

	if len(stmt.Args) != 0 {
		t.Fatalf("expected 0 args, got %d", len(stmt.Args))
	}
}

// 关键字形式：双引号剥离
func TestParseExecBareWordDoubleQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`exec echo "my document.txt"`, []string{"echo", "my document.txt"}},
		{`exec grep "pattern with spaces" file.txt`, []string{"grep", "pattern with spaces", "file.txt"}},
		{`exec curl -H "Content-Type: application/json" url`, []string{"curl", "-H", "Content-Type: application/json", "url"}},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		program := p.ParseProgram()

		stmt, ok := program.Statements[0].(*ExecStatement)
		if !ok {
			t.Fatalf("stmt is not *ExecStatement. got=%T", program.Statements[0])
		}

		if len(stmt.Args) != len(tt.expected) {
			t.Fatalf("expected %d args, got %d", len(tt.expected), len(stmt.Args))
		}

		for i, expected := range tt.expected {
			arg, ok := stmt.Args[i].(*StringLiteral)
			if !ok {
				t.Fatalf("arg[%d] is not *StringLiteral. got=%T", i, stmt.Args[i])
			}
			if arg.Value != expected {
				t.Errorf("expected arg[%d] %q, got %q", i, expected, arg.Value)
			}
		}
	}
}

// 关键字形式：单引号剥离
func TestParseExecBareWordSingleQuotes(t *testing.T) {
	input := `exec echo 'my document.txt'`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	stmt, ok := program.Statements[0].(*ExecStatement)
	if !ok {
		t.Fatalf("stmt is not *ExecStatement. got=%T", program.Statements[0])
	}

	if len(stmt.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(stmt.Args))
	}

	arg1, ok := stmt.Args[1].(*StringLiteral)
	if !ok {
		t.Fatalf("arg[1] is not *StringLiteral. got=%T", stmt.Args[1])
	}
	if arg1.Value != "my document.txt" {
		t.Errorf("expected arg[1] %q, got %q", "my document.txt", arg1.Value)
	}
}

// 关键字形式：分号终止
func TestParseExecBareWordSemicolon(t *testing.T) {
	input := `exec echo hello; print 1`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ExecStatement)
	if !ok {
		t.Fatalf("stmt is not *ExecStatement. got=%T", program.Statements[0])
	}

	if len(stmt.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(stmt.Args))
	}
}

// 关键字形式：&& 终止
func TestParseExecBareWordLogicalAnd(t *testing.T) {
	input := `exec echo hello && print 1`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	logicStmt, ok := program.Statements[0].(*LogicalStatement)
	if !ok {
		t.Fatalf("stmt is not *LogicalStatement. got=%T", program.Statements[0])
	}

	execStmt, ok := logicStmt.Left.(*ExecStatement)
	if !ok {
		t.Fatalf("left is not *ExecStatement. got=%T", logicStmt.Left)
	}

	if len(execStmt.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(execStmt.Args))
	}
}

// 关键字形式：|| 终止
func TestParseExecBareWordLogicalOr(t *testing.T) {
	input := `exec echo hello || print 1`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	logicStmt, ok := program.Statements[0].(*LogicalStatement)
	if !ok {
		t.Fatalf("stmt is not *LogicalStatement. got=%T", program.Statements[0])
	}

	execStmt, ok := logicStmt.Left.(*ExecStatement)
	if !ok {
		t.Fatalf("left is not *ExecStatement. got=%T", logicStmt.Left)
	}

	if len(execStmt.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(execStmt.Args))
	}
}

// 关键字形式：管道终止
func TestParseExecBareWordPipe(t *testing.T) {
	input := `exec echo hello | cat`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	pipeStmt, ok := program.Statements[0].(*PipeStatement)
	if !ok {
		t.Fatalf("stmt is not *PipeStatement. got=%T", program.Statements[0])
	}

	if len(pipeStmt.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(pipeStmt.Commands))
	}

	execStmt, ok := pipeStmt.Commands[0].(*ExecStatement)
	if !ok {
		t.Fatalf("command[0] is not *ExecStatement. got=%T", pipeStmt.Commands[0])
	}

	if len(execStmt.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(execStmt.Args))
	}
}

// 关键字形式：重定向终止（当前行为：-> 作为分隔符返回）
func TestParseExecBareWordRedirect(t *testing.T) {
	input := `exec echo hello -> out.txt`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	// scanCommandWords returns at -> delimiter, parser sees peekToken=REDIRECT
	// and wraps in RedirectStatement, but the target parsing is imperfect
	// because lexer position is after the delimiter.
	// This is a known limitation shared with parseCommandStatement.
	if len(program.Statements) < 1 {
		t.Fatalf("expected at least 1 statement, got %d", len(program.Statements))
	}

	// The first statement should be a RedirectStatement
	redirectStmt, ok := program.Statements[0].(*RedirectStatement)
	if !ok {
		t.Fatalf("stmt is not *RedirectStatement. got=%T", program.Statements[0])
	}

	execStmt, ok := redirectStmt.Source.(*ExecStatement)
	if !ok {
		t.Fatalf("source is not *ExecStatement. got=%T", redirectStmt.Source)
	}

	if len(execStmt.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(execStmt.Args))
	}
}

// 关键字形式：追加重定向终止
func TestParseExecBareWordAppend(t *testing.T) {
	input := `exec echo hello >> out.txt`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) < 1 {
		t.Fatalf("expected at least 1 statement, got %d", len(program.Statements))
	}

	redirectStmt, ok := program.Statements[0].(*RedirectStatement)
	if !ok {
		t.Fatalf("stmt is not *RedirectStatement. got=%T", program.Statements[0])
	}

	execStmt, ok := redirectStmt.Source.(*ExecStatement)
	if !ok {
		t.Fatalf("source is not *ExecStatement. got=%T", redirectStmt.Source)
	}

	if len(execStmt.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(execStmt.Args))
	}
}

// 关键字形式：后台执行终止
func TestParseExecBareWordBackground(t *testing.T) {
	input := `exec echo hello &`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	bgStmt, ok := program.Statements[0].(*BackgroundStatement)
	if !ok {
		t.Fatalf("stmt is not *BackgroundStatement. got=%T", program.Statements[0])
	}

	execStmt, ok := bgStmt.Stmt.(*ExecStatement)
	if !ok {
		t.Fatalf("stmt is not *ExecStatement. got=%T", bgStmt.Stmt)
	}

	if len(execStmt.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(execStmt.Args))
	}
}

// 关键字形式：变量插值（$HOME）
func TestParseExecBareWordVariableInterpolation(t *testing.T) {
	input := `x := "world"; exec echo $x`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[1].(*ExecStatement)
	if !ok {
		t.Fatalf("stmt is not *ExecStatement. got=%T", program.Statements[1])
	}

	if len(stmt.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(stmt.Args))
	}

	arg1, ok := stmt.Args[1].(*StringLiteral)
	if !ok {
		t.Fatalf("arg[1] is not *StringLiteral. got=%T", stmt.Args[1])
	}

	// $x should be detected for interpolation
	if arg1.Parts == nil && arg1.Obj == nil {
		t.Fatal("expected $x to be detected for interpolation")
	}
}

// 关键字形式：双引号内变量展开
func TestParseExecBareWordDoubleQuoteInterpolation(t *testing.T) {
	input := `x := "world"; exec echo "$x"`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[1].(*ExecStatement)
	if !ok {
		t.Fatalf("stmt is not *ExecStatement. got=%T", program.Statements[1])
	}

	if len(stmt.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(stmt.Args))
	}

	arg1, ok := stmt.Args[1].(*StringLiteral)
	if !ok {
		t.Fatalf("arg[1] is not *StringLiteral. got=%T", stmt.Args[1])
	}

	// "$x" should be detected for interpolation
	if arg1.Parts == nil && arg1.Obj == nil {
		t.Fatal("expected $x in double quotes to be detected for interpolation")
	}
}

// 关键字形式：URL 正确处理
func TestParseExecBareWordURL(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`exec curl http://localhost:8080`, []string{"curl", "http://localhost:8080"}},
		{`exec wget https://example.com/file.txt`, []string{"wget", "https://example.com/file.txt"}},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		p := NewParser(l)
		program := p.ParseProgram()

		stmt, ok := program.Statements[0].(*ExecStatement)
		if !ok {
			t.Fatalf("stmt is not *ExecStatement. got=%T", program.Statements[0])
		}

		if len(stmt.Args) != len(tt.expected) {
			t.Fatalf("expected %d args, got %d", len(tt.expected), len(stmt.Args))
		}

		for i, expected := range tt.expected {
			arg, ok := stmt.Args[i].(*StringLiteral)
			if !ok {
				t.Fatalf("arg[%d] is not *StringLiteral. got=%T", i, stmt.Args[i])
			}
			if arg.Value != expected {
				t.Errorf("expected arg[%d] %q, got %q", i, expected, arg.Value)
			}
		}
	}
}

// 关键字形式：不展开 glob
func TestParseExecBareWordNoGlobExpansion(t *testing.T) {
	input := `exec echo *.go`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	stmt, ok := program.Statements[0].(*ExecStatement)
	if !ok {
		t.Fatalf("stmt is not *ExecStatement. got=%T", program.Statements[0])
	}

	if len(stmt.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(stmt.Args))
	}

	arg1, ok := stmt.Args[1].(*StringLiteral)
	if !ok {
		t.Fatalf("arg[1] is not *StringLiteral. got=%T", stmt.Args[1])
	}
	if arg1.Value != "*.go" {
		t.Errorf("expected arg[1] %q, got %q", "*.go", arg1.Value)
	}
}

// 关键字形式：不展开 home
func TestParseExecBareWordNoHomeExpansion(t *testing.T) {
	input := `exec echo ~`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	stmt, ok := program.Statements[0].(*ExecStatement)
	if !ok {
		t.Fatalf("stmt is not *ExecStatement. got=%T", program.Statements[0])
	}

	if len(stmt.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(stmt.Args))
	}

	arg1, ok := stmt.Args[1].(*StringLiteral)
	if !ok {
		t.Fatalf("arg[1] is not *StringLiteral. got=%T", stmt.Args[1])
	}
	if arg1.Value != "~" {
		t.Errorf("expected arg[1] %q, got %q", "~", arg1.Value)
	}
}

// 关键字形式：单引号内变量不展开
func TestParseExecBareWordSingleQuoteNoInterpolation(t *testing.T) {
	input := `exec echo '$HOME'`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	stmt, ok := program.Statements[0].(*ExecStatement)
	if !ok {
		t.Fatalf("stmt is not *ExecStatement. got=%T", program.Statements[0])
	}

	if len(stmt.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(stmt.Args))
	}

	arg1, ok := stmt.Args[1].(*StringLiteral)
	if !ok {
		t.Fatalf("arg[1] is not *StringLiteral. got=%T", stmt.Args[1])
	}
	if arg1.Value != "$HOME" {
		t.Errorf("expected arg[1] %q, got %q", "$HOME", arg1.Value)
	}
	// Single quotes should NOT trigger interpolation
	if arg1.Parts != nil {
		t.Fatal("expected no interpolation in single quotes")
	}
}

// 弃用形式：exec "..." 应该报错或产生警告
func TestParseExecDeprecatedStringForm(t *testing.T) {
	input := `exec "echo hello"`
	l := NewLexer(input)
	p := NewParser(l)
	_ = p.ParseProgram()

	// Should have parser errors
	if len(p.Errors()) == 0 {
		t.Fatal("expected parser errors for deprecated exec \"...\" form")
	}
}

// 弃用形式：exec "" 应该报错
func TestParseExecDeprecatedEmptyString(t *testing.T) {
	input := `exec ""`
	l := NewLexer(input)
	p := NewParser(l)
	_ = p.ParseProgram()

	// Should have parser errors
	if len(p.Errors()) == 0 {
		t.Fatal("expected parser errors for deprecated exec \"\" form")
	}
}
