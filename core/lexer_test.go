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
	files := ls "-la"
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
		{STRING, "-la"},
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
	l := NewLexer("cmd/sub_command_1 := 1")

	tok := l.NextToken()
	if tok.Type != IDENT || tok.Literal != "cmd/sub_command_1" {
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

// ============================================================
// ASI (Automatic Semicolon Insertion) — P1
// ============================================================

// RBRACKET 后换行应插入分号（当前遗漏）
func TestASIAfterRBRACKET(t *testing.T) {
	input := "arr[0]\nprint \"next\""
	l := NewLexer(input)

	l.NextToken() // IDENT "arr"
	l.NextToken() // LBRACKET
	tok3 := l.NextToken() // NUMBER "0"
	tok4 := l.NextToken() // RBRACKET

	t.Logf("tok3: %q %q", tok3.Type, tok3.Literal)
	t.Logf("tok4: %q %q", tok4.Type, tok4.Literal)

	// After RBRACKET + newline, ASI should insert SEMICOLON
	tok5 := l.NextToken()
	t.Logf("tok5: %q %q (should be SEMICOLON)", tok5.Type, tok5.Literal)

	if tok5.Type != SEMICOLON {
		t.Errorf("expected SEMICOLON after RBRACKET+newline, got %q — ASI missing RBRACKET in isCompletable()", tok5.Type)
	}
}

// RBRACE 后换行应插入分号（已支持）
func TestASIAfterRBRACE(t *testing.T) {
	input := "if true {}\nprint \"next\""
	l := NewLexer(input)

	// Skip to after RBRACE
	for {
		tok := l.NextToken()
		if tok.Type == RBRACE {
			break
		}
	}

	tok := l.NextToken()
	if tok.Type != SEMICOLON {
		t.Errorf("expected SEMICOLON after RBRACE+newline, got %q", tok.Type)
	}
}

// RPAREN 后换行应插入分号（已支持）
func TestASIAfterRPAREN(t *testing.T) {
	input := "print(1)\nprint \"next\""
	l := NewLexer(input)

	// Skip to after RPAREN
	for {
		tok := l.NextToken()
		if tok.Type == RPAREN {
			break
		}
	}

	tok := l.NextToken()
	if tok.Type != SEMICOLON {
		t.Errorf("expected SEMICOLON after RPAREN+newline, got %q", tok.Type)
	}
}

// ============================================================
// Unicode & UTF-8 — P1
// ============================================================

// Unicode 标识符后换行应触发 ASI
func TestASIWithUnicodeIdentifier(t *testing.T) {
	input := "变量\nprint \"next\""
	l := NewLexer(input)

	tok1 := l.NextToken() // IDENT "变量"
	if tok1.Type != IDENT || tok1.Literal != "变量" {
		t.Fatalf("expected IDENT '变量', got %q %q", tok1.Type, tok1.Literal)
	}

	tok2 := l.NextToken() // should be SEMICOLON (ASI)
	if tok2.Type != SEMICOLON {
		t.Errorf("expected SEMICOLON after Unicode IDENT+newline, got %q", tok2.Type)
	}
}

// 字符串中含转义后紧跟多字节 UTF-8 字符
func TestStringEscapeThenMultibyteUTF8(t *testing.T) {
	input := "\"hello\\n世界\""
	l := NewLexer(input)

	tok := l.NextToken()
	if tok.Type != STRING {
		t.Fatalf("expected STRING, got %q", tok.Type)
	}

	expected := "hello\n世界"
	if tok.Literal != expected {
		t.Errorf("expected %q, got %q — readString may corrupt multibyte UTF-8 after escape", expected, tok.Literal)
	}
}

// 字符串中含转义后紧跟多字节 UTF-8（emoji）
func TestStringEscapeThenEmoji(t *testing.T) {
	input := "\"hello\\n😀\""
	l := NewLexer(input)

	tok := l.NextToken()
	if tok.Type != STRING {
		t.Fatalf("expected STRING, got %q", tok.Type)
	}

	expected := "hello\n😀"
	if tok.Literal != expected {
		t.Errorf("expected %q, got %q — readString may corrupt 4-byte UTF-8 emoji after escape", expected, tok.Literal)
	}
}
