package builtin

import (
	"bytes"
	"strings"
	"testing"
)

type mockEnv struct{}
func (m *mockEnv) Set(name string, val interface{}) {}
func (m *mockEnv) Get(name string) (interface{}, bool) { return nil, false }

func TestLs(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := &mockEnv{}

	// Test basic ls on root
	exitCode := Ls([]string{"../.."}, env, nil, stdout, stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "go.mod") {
		t.Errorf("expected stdout to contain go.mod, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()

	// Test ls -a on root
	exitCode = Ls([]string{"-a", "../.."}, env, nil, stdout, stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), ".gitignore") {
		t.Errorf("expected stdout to contain .gitignore, got %s", stdout.String())
	}
}
