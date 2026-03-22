package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestExtractCompletionTokenHandlesEscapedQuote(t *testing.T) {
	line := `make "a path with\" quote/quo`
	line = strings.Replace(line, `\"a`, `"a`, 1)
	token, prefix, raw := extractCompletionToken(line)

	if prefix != `"` {
		t.Fatalf("expected quoted prefix, got %q", prefix)
	}
	if token != `a path with\" quote/quo` {
		t.Fatalf("expected token to preserve escaped quote context, got %q", token)
	}
	if raw != `a path with\" quote/quo` {
		t.Fatalf("expected raw token to preserve escaped quote context, got %q", raw)
	}
}
