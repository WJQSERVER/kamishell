package builtin

import (
	"bytes"
	"os"
	"testing"
)

func TestMkdir(t *testing.T) {
	tmpDir := "test_mkdir_dir"
	defer os.RemoveAll(tmpDir)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Basic mkdir
	Mkdir([]string{tmpDir}, &rmMockEnv{}, nil, stdout, stderr)
	if info, err := os.Stat(tmpDir); err != nil || !info.IsDir() {
		t.Errorf("expected directory to be created")
	}

	// Mkdir existing (should fail)
	code := Mkdir([]string{tmpDir}, &rmMockEnv{}, nil, stdout, stderr)
	if code == 0 {
		t.Errorf("expected failure when creating existing directory")
	}

	// Mkdir -p existing (should succeed)
	code = Mkdir([]string{"-p", tmpDir}, &rmMockEnv{}, nil, stdout, stderr)
	if code != 0 {
		t.Errorf("expected success with -p on existing directory")
	}

	// Mkdir -p nested
	nested := tmpDir + "/a/b/c"
	Mkdir([]string{"-p", nested}, &rmMockEnv{}, nil, stdout, stderr)
	if info, err := os.Stat(nested); err != nil || !info.IsDir() {
		t.Errorf("expected nested directory to be created")
	}
}
