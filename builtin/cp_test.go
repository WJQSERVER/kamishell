package builtin

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCp(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "cp_test")
	defer os.RemoveAll(tmpDir)

	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")
	content := "hello"
	os.WriteFile(src, []byte(content), 0644)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := &rmMockEnv{}

	// Basic cp
	Cp([]string{src, dst}, env, nil, stdout, stderr)
	if data, _ := os.ReadFile(dst); string(data) != content {
		t.Errorf("expected %q, got %q", content, string(data))
	}

	// Recursive cp
	srcDir := filepath.Join(tmpDir, "srcdir")
	dstDir := filepath.Join(tmpDir, "dstdir")
	os.Mkdir(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte(content), 0644)

	Cp([]string{"-r", srcDir, dstDir}, env, nil, stdout, stderr)
	if data, _ := os.ReadFile(filepath.Join(dstDir, "file.txt")); string(data) != content {
		t.Errorf("recursive copy failed")
	}
}
