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
	// Parser treats `else if` as two statements, not a chain
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

	// Parser produces 2 statements: first if (no else), second if-else
	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(program.Statements))
	}

	stmt0, ok := program.Statements[0].(*IfStatement)
	if !ok {
		t.Fatalf("stmt0 is not *IfStatement. got=%T", program.Statements[0])
	}
	if stmt0.Alternative != nil {
		t.Error("first if should have no alternative")
	}

	stmt1, ok := program.Statements[1].(*IfStatement)
	if !ok {
		t.Fatalf("stmt1 is not *IfStatement. got=%T", program.Statements[1])
	}
	if stmt1.Alternative == nil {
		t.Error("second if should have alternative")
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
