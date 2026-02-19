package builtin

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPwd(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	dir, _ := os.Getwd()

	// Basic pwd
	Pwd([]string{}, &rmMockEnv{}, nil, stdout, stderr)
	if strings.TrimSpace(stdout.String()) != dir {
		t.Errorf("expected %q, got %q", dir, strings.TrimSpace(stdout.String()))
	}

	stdout.Reset()
	// Test logical with mocked PWD
	env := &rmMockEnv{store: map[string]interface{}{"PWD": "/mocked/path"}}
	// We need to make sure os.Stat("/mocked/path") doesn't fail or matches current dir
	// But in Pwd we check os.SameFile.
	// So let's mock PWD with current dir but maybe a different path (e.g. symlink if possible, but hard in test)
	// For now, just test it falls back to physical if PWD is invalid
	Pwd([]string{"-L"}, env, nil, stdout, stderr)
	if strings.TrimSpace(stdout.String()) != dir {
		t.Errorf("expected fallback to %q, got %q", dir, strings.TrimSpace(stdout.String()))
	}
}
