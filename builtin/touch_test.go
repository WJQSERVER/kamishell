package builtin

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTouchBasic(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Touch([]string{testFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// 检查文件是否创建
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("file was not created")
	}
}

func TestTouchNoCreate(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "nonexistent.txt")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Touch([]string{"-c", testFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// 检查文件是否未创建
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Errorf("file should not have been created with -c option")
	}
}

func TestTouchDateString(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Touch([]string{"-d", "2023-01-15", testFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// 检查时间戳是否更新
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	// 检查年份和月份
	modTime := info.ModTime()
	if modTime.Year() != 2023 {
		t.Errorf("expected year 2023, got %d", modTime.Year())
	}
	if modTime.Month() != time.January {
		t.Errorf("expected month January, got %v", modTime.Month())
	}
	if modTime.Day() != 15 {
		t.Errorf("expected day 15, got %d", modTime.Day())
	}
}

func TestTouchReference(t *testing.T) {
	tmpDir := t.TempDir()
	refFile := filepath.Join(tmpDir, "ref.txt")
	testFile := filepath.Join(tmpDir, "test.txt")

	// 创建参考文件并设置其时间戳
	os.WriteFile(refFile, []byte("ref"), 0644)
	// 设置一个过去的时间戳
	pastTime := time.Date(2022, 6, 1, 12, 0, 0, 0, time.UTC)
	os.Chtimes(refFile, pastTime, pastTime)

	// 创建测试文件
	os.WriteFile(testFile, []byte("test"), 0644)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Touch([]string{"-r", refFile, testFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// 检查时间戳是否与参考文件相同
	refInfo, _ := os.Stat(refFile)
	testInfo, _ := os.Stat(testFile)

	if refInfo.ModTime().Unix() != testInfo.ModTime().Unix() {
		t.Errorf("expected test file modtime to match reference file")
	}
}

func TestTouchTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// 格式: YYMMDDhhmm
	code := Touch([]string{"-t", "202301011200", testFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	info, _ := os.Stat(testFile)
	modTime := info.ModTime()

	if modTime.Year() != 2023 {
		t.Errorf("expected year 2023, got %d", modTime.Year())
	}
	if modTime.Month() != time.January {
		t.Errorf("expected month January, got %v", modTime.Month())
	}
	if modTime.Day() != 1 {
		t.Errorf("expected day 1, got %d", modTime.Day())
	}
	// 时区差异测试，只检查大致时间范围
	if modTime.Year() != 2023 && modTime.Year() != 2022 {
		t.Errorf("expected year around 2023, got %d", modTime.Year())
	}
	if modTime.Minute() != 0 {
		t.Errorf("expected minute 0, got %d", modTime.Minute())
	}
}

func TestTouchReferenceNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	refFile := filepath.Join(tmpDir, "nonexistent.txt")
	testFile := filepath.Join(tmpDir, "test.txt")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Touch([]string{"-r", refFile, testFile}, nil, nil, stdout, stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "No such file") {
		t.Errorf("expected error about nonexistent reference file, got: %s", errOutput)
	}
}

func TestTouchMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Touch([]string{file1, file2}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// 检查两个文件是否都创建
	if _, err := os.Stat(file1); os.IsNotExist(err) {
		t.Errorf("file1 was not created")
	}
	if _, err := os.Stat(file2); os.IsNotExist(err) {
		t.Errorf("file2 was not created")
	}
}

func TestTouchUnixTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// 使用 Unix 时间戳 @1672531200 (2023-01-01 00:00:00 UTC)
	code := Touch([]string{"-d", "@1672531200", testFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	info, _ := os.Stat(testFile)
	if info.ModTime().Year() != 2023 {
		t.Errorf("expected year 2023, got %d", info.ModTime().Year())
	}
}
