package lexer

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

	l := New(input)

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

	l := New(input)
	tok := l.NextToken()

	if tok.Type != PRINT {
		t.Fatalf("Expected PRINT token after shebang, got %q", tok.Type)
	}
}

func TestSemicolonInsertion(t *testing.T) {
	input := `ls
	print "hi"`

	l := New(input)

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
