package builtin

import (
	"bytes"
	"strings"
	"testing"
)

func TestLs(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Test basic ls on root
	exitCode := Ls([]string{"../.."}, stdout, stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "go.mod") {
		t.Errorf("expected stdout to contain go.mod, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()

	// Test ls -a on root
	exitCode = Ls([]string{"-a", "../.."}, stdout, stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), ".gitignore") {
		t.Errorf("expected stdout to contain .gitignore, got %s", stdout.String())
	}
}
