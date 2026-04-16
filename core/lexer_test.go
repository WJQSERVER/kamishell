package core

import (
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `// This is a comment
	print "hello"
	/*
	   Multi-line
	   comment
	*/
	files := ls -la
	if err != nil {
		exit 1
	}`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{PRINT, "print"},
		{STRING, "hello"},
		{SEMICOLON, ";"},
		{IDENT, "files"},
		{COLON_ASSIGN, ":="},
		{IDENT, "ls"},
		{IDENT, "-la"},
		{SEMICOLON, ";"},
		{IF, "if"},
		{IDENT, "err"},
		{NEQ, "!="},
		{NIL, "nil"},
		{LBRACE, "{"},
		{IDENT, "exit"},
		{NUMBER, "1"},
		{SEMICOLON, ";"},
		{RBRACE, "}"},
		{SEMICOLON, ";"},
		{EOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q. literal=%q",
				i, tt.expectedType, tok.Type, tok.Literal)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestShebangSupport(t *testing.T) {
	input := `#!/usr/bin/env kami
	print "hello"`

	l := NewLexer(input)
	tok := l.NextToken()

	if tok.Type != PRINT {
		t.Fatalf("Expected PRINT token after shebang, got %q", tok.Type)
	}
}

func TestSemicolonInsertion(t *testing.T) {
	input := `ls
	print "hi"`

	l := NewLexer(input)

	tok := l.NextToken()
	if tok.Type != IDENT || tok.Literal != "ls" {
		t.Fatalf("expected ls, got %v", tok)
	}

	tok = l.NextToken()
	if tok.Type != SEMICOLON {
		t.Fatalf("expected semicolon, got %v", tok)
	}

	tok = l.NextToken()
	if tok.Type != PRINT {
		t.Fatalf("expected print, got %v", tok)
	}
}

func TestAssignWithoutSpacesDoesNotBecomeSingleIdentifier(t *testing.T) {
	l := NewLexer("x=1")

	tok := l.NextToken()
	if tok.Type != IDENT || tok.Literal != "x" {
		t.Fatalf("expected first token IDENT x, got type=%q literal=%q", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != ASSIGN || tok.Literal != "=" {
		t.Fatalf("expected second token ASSIGN =, got type=%q literal=%q", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != NUMBER || tok.Literal != "1" {
		t.Fatalf("expected third token NUMBER 1, got type=%q literal=%q", tok.Type, tok.Literal)
	}
}

func TestUnicodeIdentifierTokenization(t *testing.T) {
	l := NewLexer("变量 := 1")

	tok := l.NextToken()
	if tok.Type != IDENT || tok.Literal != "变量" {
		t.Fatalf("expected unicode IDENT 变量, got type=%q literal=%q", tok.Type, tok.Literal)
	}
}

func TestASCIIIdentifierTokenizationWithPathCharacters(t *testing.T) {
	l := NewLexer("cmd/sub-command_1 := 1")

	tok := l.NextToken()
	if tok.Type != IDENT || tok.Literal != "cmd/sub-command_1" {
		t.Fatalf("expected path-like IDENT, got type=%q literal=%q", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != COLON_ASSIGN {
		t.Fatalf("expected := after identifier, got %v", tok)
	}
}

func TestFloatLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"3.14", "3.14"},
		{"0.5", "0.5"},
		{".5", "0.5"},
		{"123.456", "123.456"},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		tok := l.NextToken()

		if tok.Type != FLOAT {
			t.Errorf("input %q: expected FLOAT, got %q", tt.input, tok.Type)
		}
		if tok.Literal != tt.expected {
			t.Errorf("input %q: expected literal %q, got %q", tt.input, tt.expected, tok.Literal)
		}
	}
}

func TestFloatVsInteger(t *testing.T) {
	tests := []struct {
		input       string
		expectType  TokenType
		expectValue string
	}{
		{"123", NUMBER, "123"},
		{"123.0", FLOAT, "123.0"},
		{"123.456", FLOAT, "123.456"},
		{".5", FLOAT, "0.5"},
	}

	for _, tt := range tests {
		l := NewLexer(tt.input)
		tok := l.NextToken()

		if tok.Type != tt.expectType {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.expectType, tok.Type)
		}
		if tok.Literal != tt.expectValue {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.expectValue, tok.Literal)
		}
	}
}
