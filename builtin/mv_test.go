package builtin

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMv(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "mv_test")
	defer os.RemoveAll(tmpDir)

	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")
	content := "hello"
	os.WriteFile(src, []byte(content), 0644)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := &rmMockEnv{}

	// Basic mv
	Mv([]string{src, dst}, env, nil, stdout, stderr)
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("src should have been removed")
	}
	if data, _ := os.ReadFile(dst); string(data) != content {
		t.Errorf("expected %q, got %q", content, string(data))
	}

	// Move into directory
	src2 := filepath.Join(tmpDir, "src2.txt")
	os.WriteFile(src2, []byte(content), 0644)
	os.Mkdir(filepath.Join(tmpDir, "dir"), 0755)

	Mv([]string{src2, filepath.Join(tmpDir, "dir")}, env, nil, stdout, stderr)
	if _, err := os.Stat(filepath.Join(tmpDir, "dir", "src2.txt")); err != nil {
		t.Errorf("file should have been moved into directory")
	}
}

func TestMvInteractiveReadError(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")
	if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
		t.Fatalf("write src failed: %v", err)
	}
	if err := os.WriteFile(dst, []byte("old"), 0644); err != nil {
		t.Fatalf("write dst failed: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := Mv([]string{"-i", src, dst}, &rmMockEnv{}, errReader{}, stdout, stderr)
	if code == 0 {
		t.Fatal("expected interactive read failure to return non-zero exit code")
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("expected source to remain after failed prompt read, got %v", err)
	}
}
