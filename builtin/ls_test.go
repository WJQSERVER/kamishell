package builtin

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLs(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := &rmMockEnv{}

	// Test basic ls on current directory
	exitCode := Ls([]string{"."}, env, nil, stdout, stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ls.go") {
		t.Errorf("expected stdout to contain ls.go, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()

	// Test ls -R
	exitCode = Ls([]string{"-R", "."}, env, nil, stdout, stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), ".:") {
		t.Errorf("expected stdout to contain recursive header, got %s", stdout.String())
	}

	stdout.Reset()
	// Test ls -d
	exitCode = Ls([]string{"-d", "."}, env, nil, stdout, stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if strings.TrimSpace(stdout.String()) != "." {
		t.Errorf("expected '.', got %q", strings.TrimSpace(stdout.String()))
	}
}

func TestLsClassifyUsesTargetPath(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	filePath := filepath.Join(targetDir, "tool")
	if err := os.WriteFile(filePath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := Ls([]string{"-F", targetDir}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "tool*") {
		t.Fatalf("expected executable marker for target dir entry, got %q", stdout.String())
	}
}
