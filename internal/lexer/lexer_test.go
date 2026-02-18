package lexer

import (
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `print "hello";
	files := ls -la;
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
		{RBRACE, "}"},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}
