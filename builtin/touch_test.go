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

func TestTouchReferencePreservesAccessTime(t *testing.T) {
	tmpDir := t.TempDir()
	refFile := filepath.Join(tmpDir, "ref.txt")
	testFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(refFile, []byte("ref"), 0644); err != nil {
		t.Fatalf("write ref failed: %v", err)
	}
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("write test failed: %v", err)
	}

	refAtime := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	refMtime := time.Now().Add(-1 * time.Hour).Truncate(time.Second)
	if err := os.Chtimes(refFile, refAtime, refMtime); err != nil {
		t.Fatalf("set ref times failed: %v", err)
	}
	refInfo, err := os.Stat(refFile)
	if err != nil {
		t.Fatalf("stat ref failed: %v", err)
	}
	expectedAtime, expectedMtime := currentFileTimes(refInfo)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := Touch([]string{"-r", refFile, testFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	testInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("stat test failed: %v", err)
	}
	actualAtime, actualMtime := currentFileTimes(testInfo)

	if !actualAtime.Equal(expectedAtime) {
		t.Fatalf("expected atime %v, got %v", expectedAtime, actualAtime)
	}
	if !actualMtime.Equal(expectedMtime) {
		t.Fatalf("expected mtime %v, got %v", expectedMtime, actualMtime)
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
	if modTime.Hour() != 12 {
		t.Errorf("expected local hour 12, got %d", modTime.Hour())
	}
	if modTime.Minute() != 0 {
		t.Errorf("expected minute 0, got %d", modTime.Minute())
	}
}

func TestTouchDateParsesInLocalTime(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.FixedZone("UTC+8", 8*60*60)
	defer func() {
		time.Local = originalLocal
	}()

	parsed, err := parseDateString("2023-01-15 12:00")
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}

	if parsed.Location() != time.Local {
		t.Fatalf("expected local timezone parse, got %v", parsed.Location())
	}
	if parsed.Hour() != 12 {
		t.Fatalf("expected local hour 12, got %d", parsed.Hour())
	}
}

func TestTouchTimeSelectorAtime(t *testing.T) {
	opts := &touchOptions{timeSelector: "atime"}
	if err := applyTouchTimeSelector(opts); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !opts.atime || opts.mtime {
		t.Fatalf("expected atime selector only, got atime=%v mtime=%v", opts.atime, opts.mtime)
	}
}

func TestTouchTimeSelectorMtime(t *testing.T) {
	opts := &touchOptions{timeSelector: "mtime"}
	if err := applyTouchTimeSelector(opts); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !opts.mtime || opts.atime {
		t.Fatalf("expected mtime selector only, got atime=%v mtime=%v", opts.atime, opts.mtime)
	}
}

func TestTouchTimeSelectorInvalid(t *testing.T) {
	opts := &touchOptions{timeSelector: "bad"}
	if err := applyTouchTimeSelector(opts); err == nil {
		t.Fatal("expected invalid --time selector error")
	}
}

func TestParseTimestampWithoutYearUsesCurrentYear(t *testing.T) {
	parsed, err := parseTimestamp("01011200")
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}

	currentYear := time.Now().In(time.Local).Year()
	if parsed.Year() != currentYear {
		t.Fatalf("expected current year %d, got %d", currentYear, parsed.Year())
	}
	if parsed.Month() != time.January || parsed.Day() != 1 || parsed.Hour() != 12 || parsed.Minute() != 0 {
		t.Fatalf("unexpected parsed timestamp: %v", parsed)
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
