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

	// Test basic ls on current directory
	exitCode := Ls([]string{"."}, env, nil, stdout, stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d. stderr: %s", exitCode, stderr.String())
	}
	// We are running tests from root usually, but when testing the package it might be in builtin/
	// Let's check for a file that is likely in either root or builtin
	if !strings.Contains(stdout.String(), "ls.go") && !strings.Contains(stdout.String(), "go.mod") {
		t.Errorf("expected stdout to contain ls.go or go.mod, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()

	// Test ls -a on current directory
	exitCode = Ls([]string{"-a", "."}, env, nil, stdout, stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "ls_test.go") && !strings.Contains(stdout.String(), "README.md") {
		t.Errorf("expected stdout to contain ls_test.go or README.md, got %s", stdout.String())
	}
}
