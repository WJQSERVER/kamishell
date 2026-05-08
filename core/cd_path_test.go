package core

import (
	"bytes"
	"kamishell/builtin"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCdAbsolutePathParsing(t *testing.T) {
	input := `cd /tmp/testpath`
	l := NewLexer(input)

	tokens := []Token{}
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}

	t.Logf("Tokens for input %q:", input)
	for i, tok := range tokens {
		t.Logf("  [%d] Type=%-15s Literal=%q", i, tok.Type, tok.Literal)
	}

	// Lexer produces: IDENT("cd"), SLASH("/"), IDENT("tmp/testpath"), SEMICOLON, EOF
	// The parser handles merging SLASH+IDENT into a path argument
	if len(tokens) != 5 {
		t.Errorf("Expected 5 tokens (IDENT, SLASH, IDENT, SEMICOLON, EOF), got %d", len(tokens))
	}
	if tokens[0].Type != IDENT || tokens[0].Literal != "cd" {
		t.Errorf("tokens[0] should be IDENT 'cd', got %q %q", tokens[0].Type, tokens[0].Literal)
	}
	if tokens[1].Type != SLASH {
		t.Errorf("tokens[1] should be SLASH, got %q", tokens[1].Type)
	}
	if tokens[2].Type != IDENT || tokens[2].Literal != "tmp/testpath" {
		t.Errorf("tokens[2] should be IDENT 'tmp/testpath', got %q %q", tokens[2].Type, tokens[2].Literal)
	}
}

func TestCdAbsolutePathParsing_TrailingSlash(t *testing.T) {
	input := `cd /tmp/testpath/`
	l := NewLexer(input)

	tokens := []Token{}
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}

	t.Logf("Tokens for input %q:", input)
	for i, tok := range tokens {
		t.Logf("  [%d] Type=%-15s Literal=%q", i, tok.Type, tok.Literal)
	}

	// Lexer produces: IDENT("cd"), SLASH("/"), IDENT("tmp/testpath/"), SEMICOLON, EOF
	if len(tokens) != 5 {
		t.Errorf("Expected 5 tokens, got %d", len(tokens))
	}
	if tokens[1].Type != SLASH {
		t.Errorf("tokens[1] should be SLASH, got %q", tokens[1].Type)
	}
	if tokens[2].Type != IDENT || tokens[2].Literal != "tmp/testpath/" {
		t.Errorf("tokens[2] should be IDENT 'tmp/testpath/', got %q %q", tokens[2].Type, tokens[2].Literal)
	}
}

func TestCdAbsolutePathParsing_DeepPath(t *testing.T) {
	input := `cd /data/github/WJQSERVER/`
	l := NewLexer(input)

	tokens := []Token{}
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}

	t.Logf("Tokens for input %q:", input)
	for i, tok := range tokens {
		t.Logf("  [%d] Type=%-15s Literal=%q", i, tok.Type, tok.Literal)
	}

	// Lexer produces: IDENT("cd"), SLASH("/"), IDENT("data/github/WJQSERVER/"), SEMICOLON, EOF
	// The parser merges these into a single path argument
	if len(tokens) != 5 {
		t.Errorf("Expected 5 tokens, got %d", len(tokens))
	}
	if tokens[1].Type != SLASH {
		t.Errorf("tokens[1] should be SLASH, got %q", tokens[1].Type)
	}
	if tokens[2].Type != IDENT || tokens[2].Literal != "data/github/WJQSERVER/" {
		t.Errorf("tokens[2] should be IDENT 'data/github/WJQSERVER/', got %q %q", tokens[2].Type, tokens[2].Literal)
	}
}

func TestCdAbsolutePathParsing_ParserArgs(t *testing.T) {
	input := `cd /data/github/WJQSERVER/`

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

	t.Logf("Command name: %q", stmt.Name)
	t.Logf("Number of arguments: %d", len(stmt.Arguments))
	for i, arg := range stmt.Arguments {
		t.Logf("  arg[%d]: %T = %q", i, arg, arg.String())
	}

	// In a correct implementation, there should be exactly 1 argument: the path
	if len(stmt.Arguments) != 1 {
		t.Errorf("Expected 1 argument (the path), got %d", len(stmt.Arguments))
		t.Logf("This confirms the bug: path is split into multiple tokens by the lexer")
	}
}

func TestCdAbsolutePathParsing_ParserArgs_NoTrailingSlash(t *testing.T) {
	input := `cd /tmp`

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

	t.Logf("Command name: %q", stmt.Name)
	t.Logf("Number of arguments: %d", len(stmt.Arguments))
	for i, arg := range stmt.Arguments {
		t.Logf("  arg[%d]: %T = %q", i, arg, arg.String())
	}

	if len(stmt.Arguments) != 1 {
		t.Errorf("Expected 1 argument (the path), got %d", len(stmt.Arguments))
	}
}

func TestCdAbsolutePath_Behavior(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cd_path_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	nested := filepath.Join(tmpDir, "sub1", "sub2")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	origWD, _ := os.Getwd()
	defer os.Chdir(origWD)

	input := "cd " + nested

	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()
	Fold(program)
	Resolve(program)

	env := NewEnvironment()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	result := EvalWithIO(program, env, nil, stdout, stderr)
	if result != nil && result.Type() == ERROR_OBJ {
		t.Logf("Eval result: %s", result.Inspect())
	}

	newWD, _ := os.Getwd()
	evalNew, _ := filepath.EvalSymlinks(newWD)
	evalTarget, _ := filepath.EvalSymlinks(nested)

	t.Logf("Input:     %q", input)
	t.Logf("Target:    %q", evalTarget)
	t.Logf("Actual WD: %q", evalNew)

	if evalNew != evalTarget {
		t.Errorf("cd to absolute path failed: expected %q, got %q", evalTarget, evalNew)
	}
}

func TestCdAbsolutePath_Behavior_TrailingSlash(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cd_path_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	nested := filepath.Join(tmpDir, "sub1", "sub2") + "/"
	if err := os.MkdirAll(filepath.Dir(nested), 0755); err != nil {
		t.Fatal(err)
	}

	origWD, _ := os.Getwd()
	defer os.Chdir(origWD)

	input := "cd " + nested

	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()
	Fold(program)
	Resolve(program)

	env := NewEnvironment()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	result := EvalWithIO(program, env, nil, stdout, stderr)
	if result != nil && result.Type() == ERROR_OBJ {
		t.Logf("Eval result: %s", result.Inspect())
	}

	newWD, _ := os.Getwd()
	evalNew, _ := filepath.EvalSymlinks(newWD)
	evalTarget, _ := filepath.EvalSymlinks(strings.TrimSuffix(nested, "/"))

	t.Logf("Input:     %q", input)
	t.Logf("Target:    %q", evalTarget)
	t.Logf("Actual WD: %q", evalNew)

	if evalNew != evalTarget {
		t.Errorf("cd to absolute path with trailing slash failed: expected %q, got %q", evalTarget, evalNew)
	}
}

func TestCdDirectBuiltin(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cd_direct_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origWD, _ := os.Getwd()
	defer os.Chdir(origWD)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := NewEnvironment()

	cmd, ok := builtin.Builtins["cd"]
	if !ok {
		t.Fatal("cd builtin not found")
	}

	exitCode := cmd.Action([]string{tmpDir}, env, nil, stdout, stderr)
	if exitCode != 0 {
		t.Errorf("cd failed with exit code %d, stderr: %s", exitCode, stderr.String())
	}

	newWD, _ := os.Getwd()
	evalNew, _ := filepath.EvalSymlinks(newWD)
	evalTarget, _ := filepath.EvalSymlinks(tmpDir)

	if evalNew != evalTarget {
		t.Errorf("Direct cd to %q failed: got %q", evalTarget, evalNew)
	}
}


