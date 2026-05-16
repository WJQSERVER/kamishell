package core

import (
	"testing"
)

// ============================================================
// 1. Command flags: ls -la, grep -i pattern file
// ============================================================

func TestCmdArg_ShortFlag(t *testing.T) {
	input := `ls -la`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Expected: IDENT("ls"), IDENT("-la"), SEMICOLON, EOF
	// Actual:   IDENT("ls"), MINUS("-"), IDENT("la"), SEMICOLON, EOF
	// The flag is split into two tokens: MINUS and IDENT
	assertTokenTypes(t, tokens, []TokenType{IDENT, MINUS, IDENT, SEMICOLON, EOF})
}

func TestCmdArg_ShortFlag_ParserArgs(t *testing.T) {
	input := `ls -la`
	stmt := parseSingleCommand(t, input)

	t.Logf("Command: %q, args: %d", stmt.Name, len(stmt.Arguments))
	for i, arg := range stmt.Arguments {
		t.Logf("  arg[%d]: %T = %q", i, arg, arg.String())
	}

	// Expected: command="ls", args=["-la"]
	// Actual: args may contain expression for -la (negation)
	if len(stmt.Arguments) != 1 {
		t.Errorf("Expected 1 argument, got %d — flag split into multiple tokens", len(stmt.Arguments))
	}
	if len(stmt.Arguments) >= 1 {
		argStr := stmt.Arguments[0].String()
		if argStr != "\"-la\"" && argStr != "-la" {
			t.Errorf("Expected arg '-la', got %q — flag parsed as expression, not string", argStr)
		}
	}
}

func TestCmdArg_LongFlag(t *testing.T) {
	input := `grep --ignore-case pattern`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Expected: IDENT("grep"), IDENT("--ignore-case"), IDENT("pattern"), SEMICOLON, EOF
	// Actual: IDENT("grep"), MINUS("-"), MINUS("-"), IDENT("ignore"), MINUS("-"), IDENT("case"), ...
	// The long flag is completely shattered
}

func TestCmdArg_MixedFlagsAndArgs(t *testing.T) {
	input := `grep -i "pattern" file.txt`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Expected: IDENT("grep"), IDENT("-i"), STRING("pattern"), IDENT("file.txt"), SEMICOLON, EOF
	// Actual: MINUS breaks -i, DOT breaks file.txt
}

func TestCmdArg_FlagWithSpace(t *testing.T) {
	input := `curl -X POST http://example.com`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Expected: IDENT("curl"), IDENT("-X"), IDENT("POST"), URL, SEMICOLON, EOF
	// Actual: MINUS breaks -X, COLON+// breaks URL
}

// ============================================================
// 2. URL: git clone https://...
// ============================================================

func TestCmdArg_HTTPS_URL(t *testing.T) {
	input := `git clone https://github.com/WJQSERVER/codex.git`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Expected: IDENT("git"), IDENT("clone"), <full URL token>, SEMICOLON, EOF
	// Actual: IDENT("git"), IDENT("clone"), IDENT("https"), COLON(":"), EOF
	// URL content after "://" eaten as comment
	for _, tok := range tokens {
		if tok.Type == EOF {
			break
		}
		t.Logf("  %s: %q", tok.Type, tok.Literal)
	}
}

func TestCmdArg_FTP_URL(t *testing.T) {
	input := `wget ftp://files.example.com/data.tar.gz`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Same issue as HTTPS: "ftp://" → IDENT("ftp") COLON(":") then "//" → comment
}

func TestCmdArg_SSH_URL(t *testing.T) {
	input := `git clone ssh://git@github.com/user/repo.git`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// "@" is ILLEGAL token, "://" eats rest as comment
}

func TestCmdArg_URL_WithPort(t *testing.T) {
	input := `curl http://localhost:8080/api/v1`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// ":" after http triggers COLON, then "//" → comment
	// Even if fixed, the port ":8080" would also break
}

func TestCmdArg_URL_ParserArgs(t *testing.T) {
	input := `git clone https://github.com/WJQSERVER/codex.git`
	stmt := parseSingleCommand(t, input)

	t.Logf("Command: %q, args: %d", stmt.Name, len(stmt.Arguments))
	for i, arg := range stmt.Arguments {
		t.Logf("  arg[%d]: %T = %q", i, arg, arg.String())
	}

	// Expected: args=["clone", "https://github.com/WJQSERVER/codex.git"]
	// Actual: args=["clone", "https", ":"] — URL destroyed
	if len(stmt.Arguments) >= 2 {
		urlArg := stmt.Arguments[1].String()
		if urlArg == "\"https\"" || urlArg == "https" {
			t.Errorf("BUG: URL truncated to %q", urlArg)
		}
	}
}

// ============================================================
// 3. File extensions: script.py, file.txt
// ============================================================

func TestCmdArg_FileWithExtension(t *testing.T) {
	input := `python3 script.py arg1`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Expected: IDENT("python3"), IDENT("script.py"), IDENT("arg1"), SEMICOLON, EOF
	// Actual: IDENT("python3"), IDENT("script"), DOT("."), IDENT("py"), IDENT("arg1"), SEMICOLON, EOF
	// "script.py" split into IDENT DOT IDENT — becomes member access expression
}

func TestCmdArg_FileWithExtension_ParserArgs(t *testing.T) {
	input := `python3 script.py arg1`
	stmt := parseSingleCommand(t, input)

	t.Logf("Command: %q, args: %d", stmt.Name, len(stmt.Arguments))
	for i, arg := range stmt.Arguments {
		t.Logf("  arg[%d]: %T = %q", i, arg, arg.String())
	}

	// Expected: args=["script.py", "arg1"]
	// Actual: args may be ["script", <member access expression>] or similar
	if len(stmt.Arguments) != 2 {
		t.Errorf("Expected 2 args, got %d — filename split by DOT", len(stmt.Arguments))
	}
}

func TestCmdArg_MultipleDots(t *testing.T) {
	input := `cat archive.tar.gz`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Expected: IDENT("cat"), IDENT("archive.tar.gz"), SEMICOLON, EOF
	// Actual: IDENT("cat"), IDENT("archive"), DOT, IDENT("tar"), DOT, IDENT("gz"), ...
}

func TestCmdArg_DotPath(t *testing.T) {
	input := `./myscript arg`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Expected: IDENT("./myscript"), IDENT("arg")
	// Actual: DOT, SLASH, IDENT("myscript"), IDENT("arg")
	// "./" is split into DOT + SLASH, not recognized as path start
}

func TestCmdArg_DotDotPath(t *testing.T) {
	input := `../build/output`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Expected: IDENT("../build/output")
	// Actual: DOT, DOT, SLASH, IDENT("build"), SLASH, IDENT("output")
}

// ============================================================
// 4. Email addresses
// ============================================================

func TestCmdArg_EmailAddress(t *testing.T) {
	input := `mail user@example.com`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Expected: IDENT("mail"), IDENT("user@example.com"), SEMICOLON, EOF
	// Actual: IDENT("mail"), IDENT("user"), ILLEGAL("@") — lexer error
	// "@" is not a recognized character
}

func TestCmdArg_SSHUserHost(t *testing.T) {
	input := `ssh user@hostname`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Expected: IDENT("ssh"), IDENT("user@hostname"), SEMICOLON, EOF
	// Actual: "@" is ILLEGAL
}

func TestCmdArg_GitSSH(t *testing.T) {
	input := `git clone git@github.com:user/repo.git`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// "@" → ILLEGAL, ":" → COLON, "." → DOT — completely broken
}

// ============================================================
// 5. Regex patterns in arguments
// ============================================================

func TestCmdArg_RegexPattern(t *testing.T) {
	input := `grep foo.*bar file.txt`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Expected: IDENT("grep"), IDENT("foo.*bar"), IDENT("file.txt"), SEMICOLON, EOF
	// Actual: IDENT("grep"), IDENT("foo"), DOT, ASTERISK, IDENT("bar"), IDENT("file"), DOT, IDENT("txt")
	// Regex pattern shattered into expression tokens
}

func TestCmdArg_RegexWithSpecialChars(t *testing.T) {
	input := `grep "^[a-z]+$" file`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Quoted regex works: STRING("^[a-z]+$"), IDENT("file")
	// This is fine — quotes protect the pattern
}

func TestCmdArg_BareRegex_Brackets(t *testing.T) {
	input := `grep [0-9] file`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Expected: IDENT("grep"), IDENT("[0-9]"), IDENT("file")
	// Actual: LBRACKET, NUMBER(0), MINUS, NUMBER(9), RBRACKET — completely broken
}

// ============================================================
// 6. file:line format
// ============================================================

func TestCmdArg_FileColonLine(t *testing.T) {
	input := `vim main.go:42`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Expected: IDENT("vim"), IDENT("main.go:42"), SEMICOLON, EOF
	// Actual: IDENT("vim"), IDENT("main"), DOT, IDENT("go"), COLON, NUMBER(42)
	// file:line shattered into DOT + COLON + NUMBER
}

func TestCmdArg_FileColonLine_ParserArgs(t *testing.T) {
	input := `vim main.go:42`
	stmt := parseSingleCommand(t, input)

	t.Logf("Command: %q, args: %d", stmt.Name, len(stmt.Arguments))
	for i, arg := range stmt.Arguments {
		t.Logf("  arg[%d]: %T = %q", i, arg, arg.String())
	}

	// Expected: args=["main.go:42"]
	// Actual: args likely contain multiple expressions from DOT/COLON/NUMBER
	if len(stmt.Arguments) != 1 {
		t.Errorf("Expected 1 arg, got %d — file:line format broken by DOT and COLON", len(stmt.Arguments))
	}
}

func TestCmdArg_GrepFileLine(t *testing.T) {
	input := `grep -n "TODO" src/main.go:10`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Combined: flag (-n) + file:line — both broken
}

// ============================================================
// 7. Compound scenarios
// ============================================================

func TestCmdArg_GitCloneSSH(t *testing.T) {
	input := `git clone git@github.com:user/repo.git`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Multiple issues: @ ILLEGAL, : COLON, . DOT
}

func TestCmdArg_CurlWithData(t *testing.T) {
	input := `curl -X POST -H "Content-Type: application/json" -d '{"key":"value"}' http://api.example.com/data`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// Issues: -X flag, URL with ://
}

func TestCmdArg_PipRedirect(t *testing.T) {
	input := `grep -i "error" /var/log/syslog | wc -l`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// -i and -l flags broken, PIPE should work as terminator
}

func TestCmdArg_BackgroundExec(t *testing.T) {
	input := `sleep 10 &`
	l := NewLexer(input)
	tokens := collectTokens(t, l)

	t.Logf("Tokens for %q:", input)
	dumpTokens(t, tokens)

	// AMPERSAND should work as background operator
}

// ============================================================
// 8. Edge cases that SHOULD work
// ============================================================

func TestCmdArg_QuotedArgs(t *testing.T) {
	input := `grep "-i" "pattern" "file.txt"`
	stmt := parseSingleCommand(t, input)

	t.Logf("Command: %q, args: %d", stmt.Name, len(stmt.Arguments))
	for i, arg := range stmt.Arguments {
		t.Logf("  arg[%d]: %T = %q", i, arg, arg.String())
	}

	// Quoted args should work correctly
	if len(stmt.Arguments) != 3 {
		t.Errorf("Expected 3 args, got %d", len(stmt.Arguments))
	}
}

func TestCmdArg_KeyValueArg(t *testing.T) {
	input := `env GOOS=linux GOARCH=amd64`
	stmt := parseSingleCommand(t, input)

	t.Logf("Command: %q, args: %d", stmt.Name, len(stmt.Arguments))
	for i, arg := range stmt.Arguments {
		t.Logf("  arg[%d]: %T = %q", i, arg, arg.String())
	}

	// Key=value args should work via tryParseKeyValueArgument
	if len(stmt.Arguments) != 2 {
		t.Errorf("Expected 2 args, got %d", len(stmt.Arguments))
	}
}

func TestCmdArg_SimplePath(t *testing.T) {
	input := `cd /tmp/test`
	stmt := parseSingleCommand(t, input)

	t.Logf("Command: %q, args: %d", stmt.Name, len(stmt.Arguments))
	for i, arg := range stmt.Arguments {
		t.Logf("  arg[%d]: %T = %q", i, arg, arg.String())
	}

	// Absolute paths should work (recent fix)
	if len(stmt.Arguments) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(stmt.Arguments))
	}
}

func TestCmdArg_NestedPath(t *testing.T) {
	input := `cd /data/github/WJQSERVER/`
	stmt := parseSingleCommand(t, input)

	t.Logf("Command: %q, args: %d", stmt.Name, len(stmt.Arguments))
	for i, arg := range stmt.Arguments {
		t.Logf("  arg[%d]: %T = %q", i, arg, arg.String())
	}

	// Nested paths should work
	if len(stmt.Arguments) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(stmt.Arguments))
	}
}

// ============================================================
// Helpers
// ============================================================

func collectTokens(t *testing.T, l *Lexer) []Token {
	t.Helper()
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}
	return tokens
}

func dumpTokens(t *testing.T, tokens []Token) {
	t.Helper()
	for i, tok := range tokens {
		t.Logf("  [%d] Type=%-15s Literal=%q", i, tok.Type, tok.Literal)
	}
}

func parseSingleCommand(t *testing.T, input string) *CommandStatement {
	t.Helper()
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) == 0 {
		t.Fatal("No statements parsed")
	}

	stmt, ok := program.Statements[0].(*CommandStatement)
	if !ok {
		t.Fatalf("Expected CommandStatement, got %T", program.Statements[0])
	}
	return stmt
}

func assertTokenTypes(t *testing.T, tokens []Token, expected []TokenType) {
	t.Helper()
	if len(tokens) != len(expected) {
		t.Errorf("Token count mismatch: got %d, expected %d", len(tokens), len(expected))
		return
	}
	for i := range expected {
		if tokens[i].Type != expected[i] {
			t.Errorf("token[%d]: got %s, expected %s", i, tokens[i].Type, expected[i])
		}
	}
}
