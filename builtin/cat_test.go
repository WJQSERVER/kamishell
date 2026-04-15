package builtin

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatBasic(t *testing.T) {
	content := "hello world\n"
	tmpFile := "test_cat.txt"
	os.WriteFile(tmpFile, []byte(content), 0644)
	defer os.Remove(tmpFile)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Test file
	code := Cat([]string{tmpFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != content {
		t.Errorf("expected %q, got %q", content, stdout.String())
	}
}

func TestCatStdin(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	stdin := bytes.NewBufferString("stdin content")
	code := Cat([]string{"-"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	expected := "stdin content"
	if stdout.String() != expected {
		t.Errorf("expected %q, got %q", expected, stdout.String())
	}
}

func TestCatNumber(t *testing.T) {
	content := "line1\nline2\nline3\n"
	tmpFile := "test_cat_n.txt"
	os.WriteFile(tmpFile, []byte(content), 0644)
	defer os.Remove(tmpFile)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cat([]string{"-n", tmpFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "1") {
		t.Errorf("expected line numbers, got: %s", output)
	}
	if !strings.Contains(output, "line1") {
		t.Errorf("expected 'line1', got: %s", output)
	}
}

func TestCatNumberNonblank(t *testing.T) {
	content := "line1\n\nline3\n"
	tmpFile := "test_cat_b.txt"
	os.WriteFile(tmpFile, []byte(content), 0644)
	defer os.Remove(tmpFile)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cat([]string{"-b", tmpFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	// line1 应该有行号 1
	// 空行不应该有行号
	// line3 应该有行号 2
	if len(lines) < 3 {
		t.Fatalf("expected 3 lines, got %d: %s", len(lines), output)
	}
	// 检查第一行有行号（6位宽度）
	if !strings.HasPrefix(lines[0], "     1") {
		t.Errorf("expected first line to have number, got: %s", lines[0])
	}
	// 检查第三行有行号2
	if !strings.HasPrefix(lines[2], "     2") {
		t.Errorf("expected third line to have number 2, got: %s", lines[2])
	}
}

func TestCatSqueezeBlank(t *testing.T) {
	content := "line1\n\n\nline2\n\nline3\n"
	tmpFile := "test_cat_s.txt"
	os.WriteFile(tmpFile, []byte(content), 0644)
	defer os.Remove(tmpFile)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cat([]string{"-s", tmpFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	// 原始：line1, "", "", line2, "", line3
	// 压缩后：line1, "", line2, "", line3 (5行)
	if len(lines) != 5 {
		t.Errorf("expected 5 lines after squeezing, got %d: %v", len(lines), lines)
	}
}

func TestCatSqueezeBlankDoesNotTreatWhitespaceAsEmpty(t *testing.T) {
	content := "line1\n \n\t\n\nline2\n"
	tmpFile := "test_cat_s_whitespace.txt"
	os.WriteFile(tmpFile, []byte(content), 0644)
	defer os.Remove(tmpFile)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cat([]string{"-s", tmpFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, " \n") || !strings.Contains(output, "\t\n") {
		t.Fatalf("expected whitespace-only lines to be preserved, got: %q", output)
	}
}

func TestCatLongLine(t *testing.T) {
	longLine := strings.Repeat("a", 70*1024)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	stdin := strings.NewReader(longLine + "\n")

	code := Cat([]string{"-"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if stdout.Len() != len(longLine)+1 {
		t.Fatalf("expected long line to be preserved, got len=%d", stdout.Len())
	}
}

func TestCatLongInputWithoutTrailingNewline(t *testing.T) {
	longLine := strings.Repeat("a", 2*1024*1024)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	stdin := strings.NewReader(longLine)

	code := Cat([]string{"-E", "-"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if stdout.Len() != len(longLine)+1 {
		t.Fatalf("expected output length %d, got %d", len(longLine)+1, stdout.Len())
	}
	if !strings.HasSuffix(stdout.String(), "$") {
		t.Fatalf("expected trailing $, got %q", stdout.String()[max(0, stdout.Len()-8):])
	}
	if strings.HasSuffix(stdout.String(), "$\n") {
		t.Fatalf("expected no extra newline for unterminated input")
	}
}

func TestReadStreamedLineLongInputWithoutTrailingNewline(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(strings.Repeat("a", 2*1024*1024)))
	line, err := readStreamedLine(reader)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer line.Close()
	if line.hadNewline {
		t.Fatal("expected no trailing newline")
	}
	if line.Empty() {
		t.Fatal("expected non-empty streamed line")
	}
}

func TestReadStreamedLineTrimsCRLFAcrossBufferBoundary(t *testing.T) {
	input := strings.Repeat("a", streamedLineMemoryLimit-1) + "\r\n"
	reader := bufio.NewReader(strings.NewReader(input))
	line, err := readStreamedLine(reader)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer line.Close()
	if !line.hadNewline {
		t.Fatal("expected newline to be detected")
	}
	buf := &bytes.Buffer{}
	if err := line.WriteForGrep(buf); err != nil {
		t.Fatalf("expected grep writer to succeed: %v", err)
	}
	if bytes.HasSuffix(buf.Bytes(), []byte{'\r'}) {
		t.Fatal("expected grep view to trim trailing CR across boundary")
	}
	if line.Empty() {
		t.Fatal("expected non-empty content")
	}
}

func TestCatSqueezeBlankWithLongCRLFBlankLine(t *testing.T) {
	input := strings.Repeat("a", streamedLineMemoryLimit-1) + "\r\n\r\n\r\nend\r\n"
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	stdin := strings.NewReader(input)

	code := Cat([]string{"-s", "-E", "-"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if strings.Count(stdout.String(), "$\n$\n") > 1 {
		t.Fatalf("expected repeated blank CRLF lines to be squeezed, got %q", stdout.String())
	}
}

type closeErrorReadCloser struct {
	reader io.Reader
}

func (c *closeErrorReadCloser) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (c *closeErrorReadCloser) Close() error {
	return io.ErrUnexpectedEOF
}

func TestCatShowEnds(t *testing.T) {
	content := "hello world"
	tmpFile := "test_cat_E.txt"
	os.WriteFile(tmpFile, []byte(content), 0644)
	defer os.Remove(tmpFile)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cat([]string{"-E", tmpFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.HasSuffix(strings.TrimSpace(output), "$") {
		t.Errorf("expected line to end with $, got: %s", output)
	}
}

func TestCatShowEndsWithoutTrailingNewline(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	stdin := bytes.NewBufferString("hello world")

	code := Cat([]string{"-E", "-"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "hello world$" {
		t.Fatalf("expected no extra newline, got %q", stdout.String())
	}
}

func TestCatShowTabs(t *testing.T) {
	content := "hello\tworld"
	tmpFile := "test_cat_T.txt"
	os.WriteFile(tmpFile, []byte(content), 0644)
	defer os.Remove(tmpFile)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cat([]string{"-T", tmpFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "^I") {
		t.Errorf("expected ^I for tab, got: %s", output)
	}
}

func TestCatShowAll(t *testing.T) {
	content := "hello\tworld"
	tmpFile := "test_cat_A.txt"
	os.WriteFile(tmpFile, []byte(content), 0644)
	defer os.Remove(tmpFile)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cat([]string{"-A", tmpFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	// -A 应该显示 $ 和 ^I
	if !strings.Contains(output, "^I") {
		t.Errorf("expected ^I for tab, got: %s", output)
	}
	if !strings.Contains(output, "$") {
		t.Errorf("expected $ at end, got: %s", output)
	}
}

func TestCatMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	os.WriteFile(file1, []byte("content1\n"), 0644)
	os.WriteFile(file2, []byte("content2\n"), 0644)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cat([]string{file1, file2}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if strings.Contains(output, "file1.txt:") || strings.Contains(output, "file2.txt:") {
		t.Fatalf("expected raw concatenated output without filename prefixes, got: %s", output)
	}
	if !strings.Contains(output, "content1") {
		t.Errorf("expected content1, got: %s", output)
	}
	if !strings.Contains(output, "content2") {
		t.Errorf("expected content2, got: %s", output)
	}
}

func TestCatNonExistentFile(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cat([]string{"/nonexistent/file.txt"}, nil, nil, stdout, stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	if stderr.Len() == 0 {
		t.Errorf("expected error message, got none")
	}
}

func TestCatCombinedOptions(t *testing.T) {
	content := "line1\nline2\n"
	tmpFile := "test_cat_combined.txt"
	os.WriteFile(tmpFile, []byte(content), 0644)
	defer os.Remove(tmpFile)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Cat([]string{"-n", "-E", tmpFile}, nil, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	// 应该有行号和行尾标记
	if !strings.Contains(output, "1") {
		t.Errorf("expected line number, got: %s", output)
	}
	if !strings.Contains(output, "$") {
		t.Errorf("expected $ at end, got: %s", output)
	}
}

func TestCatPreservesCRLFWithoutFormattingFlags(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	stdin := bytes.NewBufferString("line1\r\nline2\r\n")

	code := Cat([]string{"-"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "line1\r\nline2\r\n" {
		t.Fatalf("expected CRLF to be preserved, got %q", stdout.String())
	}
}

func TestCatShowNonprintingBinaryBytes(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	stdin := bytes.NewBuffer([]byte{0xff, '\n'})

	code := Cat([]string{"-v", "-"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "M-^?\n" {
		t.Fatalf("expected byte-wise nonprinting rendering, got %q", stdout.String())
	}
}
