package builtin

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestCpInteractiveReadError(t *testing.T) {
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
	code := Cp([]string{"-i", src, dst}, &rmMockEnv{}, errReader{}, stdout, stderr)
	if code == 0 {
		t.Fatal("expected interactive read failure to return non-zero exit code")
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst failed: %v", err)
	}
	if string(data) != "old" {
		t.Fatalf("expected destination unchanged, got %q", string(data))
	}
}

func TestCpToDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "src.txt")
	dstDir := filepath.Join(tmpDir, "dest")

	os.WriteFile(srcFile, []byte("content"), 0644)
	os.MkdirAll(dstDir, 0755)

	stdin := bytes.NewReader([]byte{})
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cp([]string{srcFile, dstDir}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// 检查文件是否复制到目录中
	copiedFile := filepath.Join(dstDir, "src.txt")
	if _, err := os.Stat(copiedFile); os.IsNotExist(err) {
		t.Errorf("file was not copied to directory")
	}
}

func TestCpNoClobber(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "src.txt")
	dstFile := filepath.Join(tmpDir, "dst.txt")

	os.WriteFile(srcFile, []byte("new content"), 0644)
	os.WriteFile(dstFile, []byte("old content"), 0644)

	stdin := bytes.NewReader([]byte{})
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cp([]string{"-n", srcFile, dstFile}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// 检查目标文件内容是否保持不变
	content, _ := os.ReadFile(dstFile)
	if string(content) != "old content" {
		t.Errorf("file should not have been overwritten, got '%s'", string(content))
	}
}

func TestCpUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "src.txt")
	dstFile := filepath.Join(tmpDir, "dst.txt")

	// 创建源文件（较旧）
	os.WriteFile(srcFile, []byte("source"), 0644)
	// 创建目标文件（较新）
	os.WriteFile(dstFile, []byte("destination"), 0644)
	// 将目标文件时间戳设置为未来
	os.Chtimes(dstFile, time.Now().Add(time.Hour), time.Now().Add(time.Hour))

	stdin := bytes.NewReader([]byte{})
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cp([]string{"-u", srcFile, dstFile}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// 检查目标文件内容是否保持不变（因为源文件较旧）
	content, _ := os.ReadFile(dstFile)
	if string(content) != "destination" {
		t.Errorf("file should not have been overwritten with -u (src is older), got '%s'", string(content))
	}
}

func TestCpUpdateNewer(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "src.txt")
	dstFile := filepath.Join(tmpDir, "dst.txt")

	// 创建目标文件（较旧）
	os.WriteFile(dstFile, []byte("old destination"), 0644)
	// 设置为过去的时间
	pastTime := time.Now().Add(-time.Hour)
	os.Chtimes(dstFile, pastTime, pastTime)

	// 创建源文件（较新，默认是当前时间）
	os.WriteFile(srcFile, []byte("new source"), 0644)
	// 确保源文件时间戳更新
	time.Sleep(100 * time.Millisecond)
	os.Chtimes(srcFile, time.Now(), time.Now())

	stdin := bytes.NewReader([]byte{})
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cp([]string{"-u", srcFile, dstFile}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// 检查目标文件内容是否更新（因为源文件较新）
	content, _ := os.ReadFile(dstFile)
	if string(content) != "new source" {
		t.Errorf("file should have been overwritten with -u (src is newer), got '%s'", string(content))
	}
}

func TestCpVerbose(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "src.txt")
	dstFile := filepath.Join(tmpDir, "dst.txt")

	os.WriteFile(srcFile, []byte("content"), 0644)

	stdin := bytes.NewReader([]byte{})
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cp([]string{"-v", srcFile, dstFile}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// 检查是否输出了详细信息
	output := stdout.String()
	if !strings.Contains(output, "copying") {
		t.Errorf("expected verbose output, got: %s", output)
	}
}

func TestCpMissingOperand(t *testing.T) {
	stdin := bytes.NewReader([]byte{})
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cp([]string{}, nil, stdin, stdout, stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	if stderr.Len() == 0 {
		t.Errorf("expected error message, got none")
	}
}
