package builtin

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCd(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := &rmMockEnv{store: map[string]any{}}

	origWD, _ := os.Getwd()
	defer os.Chdir(origWD)

	tmpDir, _ := os.MkdirTemp("", "cd_test")
	defer os.RemoveAll(tmpDir)

	// Test cd to tmpDir
	Cd([]string{tmpDir}, env, nil, stdout, stderr)
	newWD, _ := os.Getwd()

	evalNewWD, _ := filepath.EvalSymlinks(newWD)
	evalTmpDir, _ := filepath.EvalSymlinks(tmpDir)
	if evalNewWD != evalTmpDir {
		t.Errorf("expected %q, got %q", evalTmpDir, evalNewWD)
	}

	// Check OLDPWD
	if val, ok := env.Get("OLDPWD"); !ok || val != origWD {
		t.Errorf("expected OLDPWD to be %q, got %v", origWD, val)
	}

	// Test cd -
	stdout.Reset()
	Cd([]string{"-"}, env, nil, stdout, stderr)
	finalWD, _ := os.Getwd()
	if finalWD != origWD {
		t.Errorf("expected to return to %q, got %q", origWD, finalWD)
	}
}
