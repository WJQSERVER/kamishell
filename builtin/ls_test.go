package builtin

import (
	"bytes"
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
