package builtin

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type errReader struct{}

func (errReader) Read(p []byte) (int, error) {
	return 0, errors.New("read failed")
}

type rmMockEnv struct {
	store map[string]any
}

func (m *rmMockEnv) Set(name string, val any) { m.store[name] = val }
func (m *rmMockEnv) Get(name string) (any, bool) {
	val, ok := m.store[name]
	return val, ok
}

func TestRm(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rm_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	origWD, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWD)

	// Test case 1: Remove a file
	f1 := "file1.txt"
	os.WriteFile(f1, []byte("test"), 0644)
	Rm([]string{f1}, &rmMockEnv{}, nil, os.Stdout, os.Stderr)
	if _, err := os.Stat(f1); !os.IsNotExist(err) {
		t.Errorf("file1.txt should have been removed")
	}

	// Test case 2: Remove a directory without -r (should fail)
	d1 := "dir1"
	os.Mkdir(d1, 0755)
	stderr := &bytes.Buffer{}
	code := Rm([]string{d1}, &rmMockEnv{}, nil, os.Stdout, stderr)
	if code == 0 || !strings.Contains(stderr.String(), "Is a directory") {
		t.Errorf("expected error when removing directory without -r, got code %d, stderr: %s", code, stderr.String())
	}

	// Test case 3: Remove a directory with -r
	code = Rm([]string{"-r", d1}, &rmMockEnv{}, nil, os.Stdout, os.Stderr)
	if code != 0 {
		t.Errorf("expected success when removing directory with -r, got code %d", code)
	}
	if _, err := os.Stat(d1); !os.IsNotExist(err) {
		t.Errorf("dir1 should have been removed with -r")
	}

	// Test case 4: Force remove non-existent file
	code = Rm([]string{"-f", "nonexistent"}, &rmMockEnv{}, nil, os.Stdout, os.Stderr)
	if code != 0 {
		t.Errorf("expected success with -f for nonexistent file, got code %d", code)
	}

	// Test case 5: Verbose
	f2 := "file2.txt"
	os.WriteFile(f2, []byte("test"), 0644)
	stdout := &bytes.Buffer{}
	Rm([]string{"-v", f2}, &rmMockEnv{}, nil, stdout, os.Stderr)
	if !strings.Contains(stdout.String(), "removed 'file2.txt'") {
		t.Errorf("expected verbose output, got: %s", stdout.String())
	}

	// Test case 6: Combined flags -rf
	d2 := "dir2"
	os.MkdirAll(filepath.Join(d2, "subdir"), 0755)
	os.WriteFile(filepath.Join(d2, "file.txt"), []byte("test"), 0644)
	code = Rm([]string{"-rf", d2}, &rmMockEnv{}, nil, os.Stdout, os.Stderr)
	if code != 0 {
		t.Errorf("expected success with -rf, got code %d", code)
	}
	if _, err := os.Stat(d2); !os.IsNotExist(err) {
		t.Errorf("dir2 should have been removed with -rf")
	}
}

func TestRmInteractive(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rm_interactive_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	origWD, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWD)

	f1 := "file1.txt"
	os.WriteFile(f1, []byte("test"), 0644)

	// Test interactive 'no'
	stdin := strings.NewReader("n\n")
	stdout := &bytes.Buffer{}
	Rm([]string{"-i", f1}, &rmMockEnv{}, stdin, stdout, os.Stderr)
	if _, err := os.Stat(f1); os.IsNotExist(err) {
		t.Errorf("file1.txt should NOT have been removed when responding 'n'")
	}

	// Test interactive 'yes'
	stdin = strings.NewReader("y\n")
	Rm([]string{"-i", f1}, &rmMockEnv{}, stdin, stdout, os.Stderr)
	if _, err := os.Stat(f1); !os.IsNotExist(err) {
		t.Errorf("file1.txt should have been removed when responding 'y'")
	}
}

func TestRmInteractiveReadError(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	f1 := "file1.txt"
	if err := os.WriteFile(f1, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := Rm([]string{"-i", f1}, &rmMockEnv{}, errReader{}, stdout, stderr)
	if code == 0 {
		t.Fatal("expected interactive read failure to return non-zero exit code")
	}
	if _, err := os.Stat(f1); err != nil {
		t.Fatalf("expected file to remain after failed prompt read, got %v", err)
	}
}

func TestRmRootProtection(t *testing.T) {
	stderr := &bytes.Buffer{}
	// Test rm -rf /
	code := Rm([]string{"-rf", "/"}, &rmMockEnv{}, nil, os.Stdout, stderr)
	if code == 0 {
		t.Errorf("expected error when removing root recursively")
	}
	if !strings.Contains(stderr.String(), "dangerous to operate recursively on '/'") {
		t.Errorf("expected protection message, got: %s", stderr.String())
	}

	stderr.Reset()
	// Test rm -rf --no-preserve-root /
	// We don't want to actually run this if we are root!
	// But we can check if it passes the protection check.
	// Actually, the protection check is BEFORE os.Lstat(target).
	// So if we have --no-preserve-root, it will proceed to os.Lstat("/").
	// We can't easily mock os.Lstat in this setup without more refactoring.
	// But we can at least verify that it DOES NOT print the protection message.

	// Let's use a non-existent path that is NOT root to see it proceeds.
	code = Rm([]string{"-rf", "--no-preserve-root", "/nonexistent_root_test"}, &rmMockEnv{}, nil, os.Stdout, stderr)
	// It should NOT contain the "dangerous" message.
	if strings.Contains(stderr.String(), "dangerous to operate recursively on '/'") {
		t.Errorf("protection message should not appear with --no-preserve-root")
	}
}

func TestRmDotProtection(t *testing.T) {
	stderr := &bytes.Buffer{}
	code := Rm([]string{"-rf", "."}, &rmMockEnv{}, nil, os.Stdout, stderr)
	if code == 0 {
		t.Errorf("expected error when removing '.'")
	}
	if !strings.Contains(stderr.String(), "refusing to remove '.' or '..' directory") {
		t.Errorf("expected dot protection message, got: %s", stderr.String())
	}

	stderr.Reset()
	code = Rm([]string{"-rf", ".."}, &rmMockEnv{}, nil, os.Stdout, stderr)
	if code == 0 {
		t.Errorf("expected error when removing '..'")
	}
	if !strings.Contains(stderr.String(), "refusing to remove '.' or '..' directory") {
		t.Errorf("expected dot protection message, got: %s", stderr.String())
	}
}
