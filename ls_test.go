package kamishell

import (
	"bytes"
	"strings"
	"testing"
)

func TestLs(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := NewEmptyEnvironment()

	// Test basic ls on current directory
	exitCode := Ls([]string{"."}, env, nil, stdout, stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "go.mod") {
		t.Errorf("expected stdout to contain go.mod, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()

	// Test ls -a on current directory
	exitCode = Ls([]string{"-a", "."}, env, nil, stdout, stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	// Since we are in the root, it should contain .gitignore if it exists or other hidden files
	// Actually let's check for README.md instead which is always there
	if !strings.Contains(stdout.String(), "README.md") {
		t.Errorf("expected stdout to contain README.md, got %s", stdout.String())
	}
}
