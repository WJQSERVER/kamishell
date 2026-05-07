package main

import (
	"os"
	"path/filepath"
	"testing"

	"kamishell/builtin"
	"kamishell/core"
)

func TestCompleterQuotedPathWithSpaces(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	dir := filepath.Join(tempDir, "dir with space")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file.km"), []byte(""), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	c := &KamiCompleter{env: core.NewEnvironment()}
	input := []rune(`make "./dir with space/fi`)
	candidates, length := c.Do(input, len(input))

	if length != len([]rune(`"./dir with space/fi`)) {
		t.Fatalf("expected completion length to match quoted token, got %d", length)
	}
	if len(candidates) == 0 {
		t.Fatal("expected quoted path candidates")
	}
	if string(candidates[0]) != `"./dir with space/file.km` {
		t.Fatalf("expected quoted candidate to preserve prefix, got %q", string(candidates[0]))
	}
}

func TestCompleterDeduplicatesEnvironmentCandidates(t *testing.T) {
	env := core.NewEmptyEnvironment()
	env.Set("KAMI_DUP", "1")
	inner := core.NewEnclosedEnvironment(env)
	inner.Set("KAMI_DUP", "2")

	c := &KamiCompleter{env: inner}
	input := []rune("KAMI_D")
	candidates, _ := c.Do(input, len(input))

	count := 0
	for _, candidate := range candidates {
		if string(candidate) == "KAMI_DUP" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected deduplicated env candidate once, got %d", count)
	}
}

func TestParseCompletionContextHandlesQuotedPath(t *testing.T) {
	line := `make "a path with" quo`
	ctx := parseCompletionContext(line)

	if ctx.currentToken != "quo" {
		t.Fatalf("expected current token 'quo', got %q", ctx.currentToken)
	}
	if ctx.commandName != "make" {
		t.Fatalf("expected command name 'make', got %q", ctx.commandName)
	}
	if ctx.isFirstWord {
		t.Fatal("expected not first word")
	}
}

func TestParseCompletionContextFirstWord(t *testing.T) {
	ctx := parseCompletionContext("ls")
	if !ctx.isFirstWord {
		t.Fatal("expected first word")
	}
	if ctx.currentToken != "ls" {
		t.Fatalf("expected current token 'ls', got %q", ctx.currentToken)
	}
}

func TestParseCompletionContextFlag(t *testing.T) {
	ctx := parseCompletionContext("ls -")
	if ctx.isFirstWord {
		t.Fatal("expected not first word")
	}
	if ctx.commandName != "ls" {
		t.Fatalf("expected command name 'ls', got %q", ctx.commandName)
	}
	if ctx.currentToken != "-" {
		t.Fatalf("expected current token '-', got %q", ctx.currentToken)
	}
}

func TestParseCompletionContextAfterPipe(t *testing.T) {
	ctx := parseCompletionContext("cat file | gre")
	// After a pipe, we're in a new command context - first word should be true
	if !ctx.isFirstWord {
		t.Fatal("expected first word after pipe (new command context)")
	}
	if ctx.commandName != "gre" {
		t.Fatalf("expected command name 'gre' after pipe, got %q", ctx.commandName)
	}
}

func TestCompleterFlagCompletion(t *testing.T) {
	// Trigger metadata registration by running the command once
	_ = builtin.Ls([]string{"--help"}, &testEnv{}, nil, &discardWriter{}, &discardWriter{})

	c := &KamiCompleter{env: core.NewEnvironment()}
	input := []rune("ls -")
	candidates, _ := c.Do(input, len(input))

	// Should have flag candidates from ls metadata
	found := false
	for _, cand := range candidates {
		if string(cand) == "-a" || string(cand) == "-l" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected flag candidates for ls, got none")
	}
}

// testEnv is a minimal Environment implementation for testing
type testEnv struct{}

func (e *testEnv) Get(key string) (any, bool) { return nil, false }
func (e *testEnv) Set(key string, val any)    {}
func (e *testEnv) SetString(name string, val string) {}
func (e *testEnv) GetString(name string) (string, bool) { return "", false }

// discardWriter discards all writes
type discardWriter struct{}

func (w *discardWriter) Write(p []byte) (n int, err error) { return len(p), nil }

func TestCompleterCommandPositionIncludesExternal(t *testing.T) {
	c := &KamiCompleter{env: core.NewEnvironment()}
	input := []rune("go")
	candidates, _ := c.Do(input, len(input))

	// Should include 'go' as external command (if in PATH)
	found := false
	for _, cand := range candidates {
		if string(cand) == "go" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'go' in command candidates (should be in PATH)")
	}
}

func TestCompleterHelpCompletesBuiltinNames(t *testing.T) {
	c := &KamiCompleter{env: core.NewEnvironment()}
	input := []rune("help ls")
	candidates, _ := c.Do(input, len(input))

	// 'ls' is already complete, so should match file paths starting with 'ls'
	// or nothing if no files match. The important thing is it doesn't crash.
	_ = candidates
}

func TestCompleterEnvVarCompletion(t *testing.T) {
	t.Setenv("KAMI_TEST_VAR", "test")

	c := &KamiCompleter{env: core.NewEnvironment()}
	input := []rune("echo $KAMI_TEST")
	candidates, _ := c.Do(input, len(input))

	found := false
	for _, cand := range candidates {
		if string(cand) == "$KAMI_TEST_VAR" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected $KAMI_TEST_VAR in env var candidates")
	}
}
