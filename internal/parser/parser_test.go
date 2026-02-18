package parser

import (
	"kamishell/internal/ast"
	"kamishell/internal/lexer"
	"testing"
)

func TestParseAssignStatement(t *testing.T) {
	input := `x := 5;
	name := "kami";
	valid := true;`
	l := lexer.New(input)
	p := New(l)
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
		assignStmt, ok := stmt.(*ast.AssignStatement)
		if !ok {
			t.Fatalf("test[%d] - stmt is not *ast.AssignStatement. got=%T", i, stmt)
		}

		if assignStmt.Name.Value != tt.expectedIdentifier {
			t.Errorf("test[%d] - assignStmt.Name.Value not %s. got=%s", i, tt.expectedIdentifier, assignStmt.Name.Value)
		}

		if assignStmt.Value.String() != tt.expectedValue {
			t.Errorf("test[%d] - assignStmt.Value.String() not %s. got=%s", i, tt.expectedValue, assignStmt.Value.String())
		}
	}
}

func TestParseCommandStatement(t *testing.T) {
	input := `ls -la;`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.CommandStatement)
	if !ok {
		t.Fatalf("stmt is not *ast.CommandStatement. got=%T", program.Statements[0])
	}

	if stmt.Name != "ls" {
		t.Errorf("stmt.Name not %s. got=%s", "ls", stmt.Name)
	}

	if len(stmt.Arguments) != 1 {
		t.Errorf("len(stmt.Arguments) not 1. got=%d", len(stmt.Arguments))
	}

	if stmt.Arguments[0] != "-la" {
		t.Errorf("stmt.Arguments[0] not %s. got=%s", "-la", stmt.Arguments[0])
	}
}
