package core

import (
	"bytes"
	"github.com/WJQSERVER/kamishell/builtin"
	"os"
	"path/filepath"
	"testing"
)

// getEnvString extracts a string value from the core.Environment store.
func getEnvString(env *Environment, key string) string {
	v, ok := env.Get(key)
	if !ok {
		return "<not set>"
	}
	if s, ok := v.(*String); ok {
		return s.Value
	}
	return "<unexpected type>"
}

// --------------------------------------------------------------------------
// Bug Confirmation Test 1: Parser does NOT produce CommandStatement for "cd .."
//
// Root cause: when the parser sees IDENT ("cd") followed by DOT ("."), it
// enters the expression/method-call branch (line 180 of parser.go) instead of
// parseCommandStatement().  The lexer produces two DOT tokens for "..", so the
// parser interprets "cd" as an identifier expression and ".." as a dot-access,
// yielding an ExpressionStatement whose String() is empty.  This means "cd .."
// is a no-op in interactive mode.
// --------------------------------------------------------------------------

func TestCdDotDot_LexerTokenization(t *testing.T) {
	input := `cd ..`
	l := NewLexer(input)

	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}

	t.Logf("Tokens for %q:", input)
	for i, tok := range tokens {
		t.Logf("  [%d] Type=%-15s Literal=%q", i, tok.Type, tok.Literal)
	}

	// Lexer splits ".." into two DOT tokens: ["cd", ".", ".", EOF]
	if len(tokens) != 4 {
		t.Errorf("Expected 4 tokens (IDENT, DOT, DOT, EOF), got %d", len(tokens))
	}
	if tokens[0].Type != IDENT || tokens[0].Literal != "cd" {
		t.Errorf("tokens[0] = %q %q; want IDENT 'cd'", tokens[0].Type, tokens[0].Literal)
	}
	if tokens[1].Type != DOT || tokens[2].Type != DOT {
		t.Errorf("tokens[1..2] should be DOT DOT, got %q %q", tokens[1].Type, tokens[2].Type)
	}
}

func TestCdDotDot_ParserProducesExpressionNotCommand(t *testing.T) {
	input := `cd ..`
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(program.Statements) == 0 {
		t.Fatal("No statements parsed")
	}

	stmt := program.Statements[0]
	t.Logf("Statement type : %T", stmt)
	t.Logf("Statement repr : %q", stmt.String())

	// BUG: the parser produces ExpressionStatement instead of CommandStatement.
	// Because peekToken after IDENT("cd") is DOT, the parser takes the
	// parseExpressionStatement path (parser.go line ~199).
	if cs, ok := stmt.(*CommandStatement); ok {
		t.Logf("CommandStatement name=%q args=%d", cs.Name, len(cs.Arguments))
	} else {
		t.Errorf("EXPECTED BUG: 'cd ..' parsed as %T, not *CommandStatement. "+
			"The parser sees IDENT then DOT and takes the expression path.", stmt)
	}
}

func TestCdDotDot_MoreDots(t *testing.T) {
	// Verify the same problem with other dot-containing paths
	cases := []struct {
		input     string
		wantIsCmd bool
		wantNArgs int
	}{
		{`cd ..`, true, 1},
		{`cd ../foo`, true, 1},
		{`cd /tmp`, true, 1},
		{`cd .`, true, 1},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			l := NewLexer(tc.input)
			p := NewParser(l)
			program := p.ParseProgram()
			if len(program.Statements) == 0 {
				t.Fatal("No statements parsed")
			}
			stmt := program.Statements[0]
			cs, ok := stmt.(*CommandStatement)
			if ok != tc.wantIsCmd {
				t.Errorf("input %q: isCommand=%v, want %v (type=%T)",
					tc.input, ok, tc.wantIsCmd, stmt)
				return
			}
			if ok && tc.wantNArgs >= 0 && len(cs.Arguments) != tc.wantNArgs {
				t.Errorf("input %q: args=%d, want %d", tc.input, len(cs.Arguments), tc.wantNArgs)
			}
		})
	}
}

// --------------------------------------------------------------------------
// Bug Confirmation Test 2: filepath.Abs("..") resolves AFTER os.Chdir
//
// In builtin/cd.go, when -L mode (the default) is used, the new PWD is
// computed via filepath.Abs(dir) AFTER os.Chdir(dir) has already changed the
// process working directory.  Because filepath.Abs resolves relative paths
// against the current working directory, "cd .." resolves ".." one level too
// far.  Example:
//   cwd = /a/b/c → cd .. → os.Chdir("..") → cwd = /a/b
//   filepath.Abs("..")  resolves against /a/b → returns /a  (WRONG, should be /a/b)
// --------------------------------------------------------------------------

func TestCdDotDot_BuiltinDirect_PwdBug(t *testing.T) {
	// Create: tmpDir/a/b/c
	tmpDir, err := os.MkdirTemp("", "cd_dotdot_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	deep := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatal(err)
	}

	origWD, _ := os.Getwd()
	defer os.Chdir(origWD)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := NewEnvironment()

	cmd := builtin.Builtins["cd"]

	// Step 1: cd into deep dir
	exitCode := cmd.Action([]string{deep}, env, nil, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("cd to deep dir failed: %s", stderr.String())
	}

	t.Logf("After 'cd %s':", deep)
	t.Logf("  cwd: %q", mustGetwd(t))
	t.Logf("  PWD: %q", getEnvString(env, "PWD"))

	// Step 2: cd ..
	stdout.Reset()
	stderr.Reset()
	exitCode = cmd.Action([]string{".."}, env, nil, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("cd .. failed: %s", stderr.String())
	}

	cwd := mustGetwd(t)
	pwd := getEnvString(env, "PWD")
	expected := symlinkEval(filepath.Join(tmpDir, "a", "b"))

	t.Logf("After 'cd ..':")
	t.Logf("  expected: %q", expected)
	t.Logf("  cwd:     %q", symlinkEval(cwd))
	t.Logf("  PWD:     %q", pwd)

	if symlinkEval(cwd) != expected {
		t.Errorf("Physical cwd wrong: expected %q, got %q", expected, symlinkEval(cwd))
	}
	if pwd != expected {
		t.Errorf("BUG CONFIRMED: PWD is %q, expected %q. "+
			"filepath.Abs(\"..\") is computed after os.Chdir, resolving against the new cwd.", pwd, expected)
	}

	// Step 3: cd .. again
	stdout.Reset()
	stderr.Reset()
	exitCode = cmd.Action([]string{".."}, env, nil, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("2nd cd .. failed: %s", stderr.String())
	}

	cwd2 := mustGetwd(t)
	pwd2 := getEnvString(env, "PWD")
	expected2 := symlinkEval(filepath.Join(tmpDir, "a"))

	t.Logf("After 2nd 'cd ..':")
	t.Logf("  expected: %q", expected2)
	t.Logf("  cwd:     %q", symlinkEval(cwd2))
	t.Logf("  PWD:     %q", pwd2)

	if symlinkEval(cwd2) != expected2 {
		t.Errorf("Physical cwd wrong: expected %q, got %q", expected2, symlinkEval(cwd2))
	}
	if pwd2 != expected2 {
		t.Errorf("BUG CONFIRMED: PWD is %q, expected %q", pwd2, expected2)
	}
}

// --------------------------------------------------------------------------
// Bug Confirmation Test 3: cd .. via script engine is a complete no-op
//
// Because the parser produces an ExpressionStatement instead of a
// CommandStatement, evaluating "cd .." in the script engine does nothing —
// the working directory and PWD are completely unchanged.
// --------------------------------------------------------------------------

func TestCdDotDot_ViaScriptEngine_NoOp(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cd_dotdot_script")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	deep := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatal(err)
	}

	origWD, _ := os.Getwd()
	defer os.Chdir(origWD)

	env := NewEnvironment()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	eval := func(script string) {
		t.Helper()
		l := NewLexer(script)
		p := NewParser(l)
		program := p.ParseProgram()
		Fold(program)
		Resolve(program)
		result := EvalWithIO(program, env, nil, stdout, stderr)
		if result != nil && result.Type() == ERROR_OBJ {
			t.Errorf("Script %q error: %s", script, result.Inspect())
		}
	}

	// cd into deep dir
	eval("cd " + deep)

	after1 := symlinkEval(mustGetwd(t))
	t.Logf("After 'cd %s': cwd=%q, PWD=%q", deep, after1, getEnvString(env, "PWD"))
	if after1 != symlinkEval(deep) {
		t.Fatalf("cd to deep dir failed")
	}

	// cd .. — should go to tmpDir/a/b but parser bug makes it a no-op
	eval("cd ..")

	after2 := symlinkEval(mustGetwd(t))
	pwd2 := getEnvString(env, "PWD")
	expected2 := symlinkEval(filepath.Join(tmpDir, "a", "b"))

	t.Logf("After 'cd ..': expected=%q, cwd=%q, PWD=%q", expected2, after2, pwd2)

	// BUG: neither cwd nor PWD changed
	if after2 == after1 && pwd2 == getEnvString(env, "PWD") {
		// Verify it didn't move at all
		if after2 != expected2 {
			t.Errorf("BUG CONFIRMED: 'cd ..' via script engine is a complete no-op. "+
				"cwd stayed at %q instead of moving to %q", after2, expected2)
		}
	}
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func mustGetwd(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd(): %v", err)
	}
	return dir
}

func symlinkEval(path string) string {
	resolved, _ := filepath.EvalSymlinks(path)
	return resolved
}

func TestCdDotDot_TokenPositions(t *testing.T) {
	cases := []string{"cd ..", "cd..", "cd .", "cd.", "cd ../foo"}
	for _, input := range cases {
		l := NewLexer(input)
		var tokens []Token
		for {
			tok := l.NextToken()
			tokens = append(tokens, tok)
			if tok.Type == EOF {
				break
			}
		}
		t.Logf("Input %q:", input)
		for i, tok := range tokens {
			t.Logf("  [%d] Type=%-10s Literal=%-15q Start=%d End=%d", i, tok.Type, tok.Literal, tok.Start, tok.End)
		}
		// Check gap between first two tokens
		if len(tokens) >= 2 {
			gap := tokens[1].Start - tokens[0].End
			t.Logf("  Gap between tokens[0] and tokens[1] = %d (0=no space, >0=has space)", gap)
		}
		t.Log("")
	}
}
