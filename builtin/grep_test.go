package builtin

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type panicAfterMatchReader struct {
	data    string
	read    bool
	trigger string
}

func (r *panicAfterMatchReader) Read(p []byte) (int, error) {
	if r.read {
		return 0, errors.New("reader continued after first chunk")
	}
	r.read = true
	return copy(p, r.data), nil
}

func TestGrepBasicPattern(t *testing.T) {
	stdin := strings.NewReader("hello world\nfoo bar\nhello test\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"hello"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "hello world") {
		t.Errorf("expected output to contain 'hello world', got: %s", output)
	}
	if !strings.Contains(output, "hello test") {
		t.Errorf("expected output to contain 'hello test', got: %s", output)
	}
	if strings.Contains(output, "foo bar") {
		t.Errorf("expected output NOT to contain 'foo bar', got: %s", output)
	}
}

func TestGrepRegexPattern(t *testing.T) {
	stdin := strings.NewReader("func main()\nvar x = 1\nfunc test()\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"func .*\\("}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "func main()") {
		t.Errorf("expected output to contain 'func main()', got: %s", output)
	}
	if !strings.Contains(output, "func test()") {
		t.Errorf("expected output to contain 'func test()', got: %s", output)
	}
	if strings.Contains(output, "var x = 1") {
		t.Errorf("expected output NOT to contain 'var x = 1', got: %s", output)
	}
}

func TestGrepIgnoreCase(t *testing.T) {
	stdin := strings.NewReader("Hello World\nHELLO TEST\nfoo bar\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"-i", "hello"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "Hello World") {
		t.Errorf("expected output to contain 'Hello World', got: %s", output)
	}
	if !strings.Contains(output, "HELLO TEST") {
		t.Errorf("expected output to contain 'HELLO TEST', got: %s", output)
	}
}

func TestGrepLineNumber(t *testing.T) {
	stdin := strings.NewReader("line one\nline two\ntarget line\nline four\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"-n", "target"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "3:target line") {
		t.Errorf("expected output to contain '3:target line', got: %s", output)
	}
}

func TestGrepInvertMatch(t *testing.T) {
	stdin := strings.NewReader("hello world\nfoo bar\nhello test\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"-v", "hello"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "foo bar") {
		t.Errorf("expected output to contain 'foo bar', got: %s", output)
	}
	if strings.Contains(output, "hello") {
		t.Errorf("expected output NOT to contain 'hello', got: %s", output)
	}
}

func TestGrepWordRegexp(t *testing.T) {
	stdin := strings.NewReader("test\ntesting\nmy test here\ncontest\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"-w", "test"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "test") {
		t.Errorf("expected output to contain 'test', got: %s", output)
	}
	if !strings.Contains(output, "my test here") {
		t.Errorf("expected output to contain 'my test here', got: %s", output)
	}
	if strings.Contains(output, "testing") {
		t.Errorf("expected output NOT to contain 'testing', got: %s", output)
	}
	if strings.Contains(output, "contest") {
		t.Errorf("expected output NOT to contain 'contest', got: %s", output)
	}
}

func TestGrepLineRegexp(t *testing.T) {
	stdin := strings.NewReader("test\nmy test\ntest only\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"-x", "test"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "test") {
		t.Errorf("expected output to contain 'test', got: %s", output)
	}
	if strings.Contains(output, "my test") {
		t.Errorf("expected output NOT to contain 'my test', got: %s", output)
	}
	if strings.Contains(output, "test only") {
		t.Errorf("expected output NOT to contain 'test only', got: %s", output)
	}
}

func TestGrepWordRegexpPreservesRegex(t *testing.T) {
	stdin := strings.NewReader("a1b\nxa1by\na[0-9]b\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"-w", "a[0-9]b"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "a1b") {
		t.Fatalf("expected regex word match, got: %s", output)
	}
	if strings.Contains(output, "xa1by") {
		t.Fatalf("expected whole-word filtering, got: %s", output)
	}
	if strings.Contains(output, "a[0-9]b") {
		t.Fatalf("expected regex semantics instead of literal matching, got: %s", output)
	}
}

func TestGrepLineRegexpPreservesRegex(t *testing.T) {
	stdin := strings.NewReader("a1b\nxa1by\na[0-9]b\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"-x", "a[0-9]b"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "a1b") {
		t.Fatalf("expected regex whole-line match, got: %s", output)
	}
	if strings.Contains(output, "xa1by") || strings.Contains(output, "a[0-9]b") {
		t.Fatalf("expected regex whole-line semantics, got: %s", output)
	}
}

func TestGrepCount(t *testing.T) {
	stdin := strings.NewReader("hello world\nfoo bar\nhello test\nhello again\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"-c", "hello"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := strings.TrimSpace(stdout.String())
	if output != "3" {
		t.Errorf("expected output to be '3', got: %s", output)
	}
}

func TestGrepQuiet(t *testing.T) {
	stdin := strings.NewReader("hello world\nfoo bar\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"-q", "hello"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if stdout.Len() != 0 {
		t.Errorf("expected no output in quiet mode, got: %s", stdout.String())
	}

	// Test no match case
	stdin2 := strings.NewReader("foo bar\nbaz qux\n")
	stdout2 := &bytes.Buffer{}
	code2 := Grep([]string{"-q", "hello"}, nil, stdin2, stdout2, stderr)
	if code2 != 1 {
		t.Fatalf("expected exit code 1 when no match, got %d", code2)
	}
}

func TestGrepFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line one\nhello world\nline three\nhello test\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"hello", testFile}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "hello world") {
		t.Errorf("expected output to contain 'hello world', got: %s", output)
	}
	if !strings.Contains(output, "hello test") {
		t.Errorf("expected output to contain 'hello test', got: %s", output)
	}
}

func TestGrepMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	os.WriteFile(file1, []byte("hello from file1\nother line\n"), 0644)
	os.WriteFile(file2, []byte("hello from file2\nanother line\n"), 0644)

	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"hello", file1, file2}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "file1.txt:") {
		t.Errorf("expected output to contain filename prefix, got: %s", output)
	}
	if !strings.Contains(output, "file2.txt:") {
		t.Errorf("expected output to contain filename prefix, got: %s", output)
	}
}

func TestGrepFilesWithMatches(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	os.WriteFile(file1, []byte("hello from file1\n"), 0644)
	os.WriteFile(file2, []byte("no match here\n"), 0644)

	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"-l", "hello", file1, file2}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := strings.TrimSpace(stdout.String())
	if !strings.Contains(output, "file1.txt") {
		t.Errorf("expected output to contain 'file1.txt', got: %s", output)
	}
	if strings.Contains(output, "file2.txt") {
		t.Errorf("expected output NOT to contain 'file2.txt', got: %s", output)
	}
}

func TestGrepFilesWithMatchesListsAllMatchingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")

	os.WriteFile(file1, []byte("hello from file1\n"), 0644)
	os.WriteFile(file2, []byte("hello from file2\n"), 0644)
	os.WriteFile(file3, []byte("no match here\n"), 0644)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"-l", "hello", file1, file2, file3}, nil, strings.NewReader(""), stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "file1.txt") || !strings.Contains(output, "file2.txt") {
		t.Fatalf("expected all matching filenames, got: %q", output)
	}
	if strings.Contains(output, "file3.txt") {
		t.Fatalf("expected non-matching file to be omitted, got: %q", output)
	}
}

func TestGrepFilesWithoutMatch(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	os.WriteFile(file1, []byte("hello from file1\n"), 0644)
	os.WriteFile(file2, []byte("no match here\n"), 0644)

	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"-L", "hello", file1, file2}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := strings.TrimSpace(stdout.String())
	if strings.Contains(output, "file1.txt") {
		t.Errorf("expected output NOT to contain 'file1.txt', got: %s", output)
	}
	if !strings.Contains(output, "file2.txt") {
		t.Errorf("expected output to contain 'file2.txt', got: %s", output)
	}
}

func TestGrepMultipleFilesExitCodeUsesAnyMatch(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	os.WriteFile(file1, []byte("hello\n"), 0644)
	os.WriteFile(file2, []byte("nope\n"), 0644)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := Grep([]string{"hello", file1, file2}, nil, strings.NewReader(""), stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 when any file matches, got %d", code)
	}
}

func TestGrepFilesWithoutMatchExitCodeUsesAnyMatch(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	os.WriteFile(file1, []byte("hello\n"), 0644)
	os.WriteFile(file2, []byte("nope\n"), 0644)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := Grep([]string{"-L", "hello", file1, file2}, nil, strings.NewReader(""), stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 when any file matches pattern, got %d", code)
	}
	if !strings.Contains(stdout.String(), "file2.txt") {
		t.Fatalf("expected -L to print non-matching file, got %q", stdout.String())
	}
}

func TestGrepFilesWithoutMatchExitCodeWhenAnyFilePrinted(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	os.WriteFile(file1, []byte("hello\n"), 0644)
	os.WriteFile(file2, []byte("world\n"), 0644)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := Grep([]string{"-L", "hello", file1, file2}, nil, strings.NewReader(""), stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 when -L outputs at least one filename, got %d", code)
	}
}

func TestGrepPreservesBareCarriageReturn(t *testing.T) {
	stdin := strings.NewReader("hello\rworld")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"world"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "hello\rworld") {
		t.Fatalf("expected bare carriage return to be preserved, got %q", stdout.String())
	}
}

func TestGrepFilesWithoutMatchReturnsEarlyAfterMatch(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	reader := &panicAfterMatchReader{data: "hello\nrest that should not be read"}

	result := grepReader(reader, regexp.MustCompile("hello"), &grepOptions{filesNoMatch: true}, stdout, stderr, "sample:")
	if result.err {
		t.Fatalf("expected no error, got stderr=%q", stderr.String())
	}
	if !result.matched {
		t.Fatal("expected file to be classified as matched for -L")
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no output for matched file in -L mode, got %q", stdout.String())
	}
}

func TestGrepLongLine(t *testing.T) {
	longLine := strings.Repeat("a", 70*1024)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	stdin := strings.NewReader(longLine + "\n")

	code := Grep([]string{"a+"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected long line to be processed, got code=%d stderr=%q", code, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Fatal("expected output for long matching line")
	}
}

func TestGrepLongInputWithoutTrailingNewline(t *testing.T) {
	longLine := strings.Repeat("a", 2*1024*1024)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	stdin := strings.NewReader(longLine)

	code := Grep([]string{"a+"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected long unterminated line to be processed, got code=%d stderr=%q", code, stderr.String())
	}
	if stdout.Len() != len(longLine) {
		t.Fatalf("expected output length %d, got %d", len(longLine), stdout.Len())
	}
}

func TestGrepLineRegexpWithLongCRLFLine(t *testing.T) {
	line := strings.Repeat("a", streamedLineMemoryLimit-1)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	stdin := strings.NewReader(line + "\r\n")

	code := Grep([]string{"-x", line}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected CRLF-terminated long line to match, got code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), line) {
		t.Fatalf("expected matched long line in output")
	}
}

func TestStreamedLineMatchRegexpTrimsCRLFAcrossBoundary(t *testing.T) {
	line := strings.Repeat("a", streamedLineMemoryLimit-1)
	reader := bufio.NewReader(strings.NewReader(line + "\r\n"))
	streamed, err := readStreamedLine(reader)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer streamed.Close()

	matched, err := streamed.MatchRegexp(regexp.MustCompile("^(" + line + ")$"))
	if err != nil {
		t.Fatalf("unexpected match error: %v", err)
	}
	if !matched {
		t.Fatal("expected CRLF-normalized streamed line to match regexp")
	}
}

func TestGrepNoPattern(t *testing.T) {
	stdin := strings.NewReader("hello world\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{}, nil, stdin, stdout, stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "search pattern required") {
		t.Errorf("expected error message about pattern, got: %s", errOutput)
	}
}

func TestGrepInvalidPattern(t *testing.T) {
	stdin := strings.NewReader("hello world\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// 无效的正则表达式模式
	code := Grep([]string{"[invalid"}, nil, stdin, stdout, stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "invalid pattern") {
		t.Errorf("expected error message about invalid pattern, got: %s", errOutput)
	}
}

func TestGrepCombinedOptions(t *testing.T) {
	stdin := strings.NewReader("Hello World\nHELLO Test\nfoo bar\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"-i", "-n", "-c", "hello"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := strings.TrimSpace(stdout.String())
	if output != "2" {
		t.Errorf("expected count to be '2', got: %s", output)
	}
}

func TestGrepAnchors(t *testing.T) {
	stdin := strings.NewReader("test line\nanother test\ntest\nend test\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// 匹配以 "test" 开头的行
	code := Grep([]string{"^test"}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "test line") {
		t.Errorf("expected output to contain 'test line', got: %s", output)
	}
	if !strings.Contains(output, "test") {
		t.Errorf("expected output to contain standalone 'test', got: %s", output)
	}
	if strings.Contains(output, "another test") {
		t.Errorf("expected output NOT to contain 'another test', got: %s", output)
	}
	if strings.Contains(output, "end test") {
		t.Errorf("expected output NOT to contain 'end test', got: %s", output)
	}
}

func TestGrepRecursive(t *testing.T) {
	tmpDir := t.TempDir()
	// 创建目录结构
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)

	// 在根目录创建文件
	file1 := filepath.Join(tmpDir, "file1.txt")
	os.WriteFile(file1, []byte("hello from root\n"), 0644)

	// 在子目录创建文件
	file2 := filepath.Join(subDir, "file2.txt")
	os.WriteFile(file2, []byte("hello from subdir\n"), 0644)

	// 在子目录创建另一个文件
	file3 := filepath.Join(subDir, "file3.txt")
	os.WriteFile(file3, []byte("no match here\n"), 0644)

	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"-r", "hello", tmpDir}, nil, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "hello from root") {
		t.Errorf("expected output to contain 'hello from root', got: %s", output)
	}
	if !strings.Contains(output, "hello from subdir") {
		t.Errorf("expected output to contain 'hello from subdir', got: %s", output)
	}
	// 递归搜索应该显示完整路径
	if !strings.Contains(output, "file1.txt") {
		t.Errorf("expected output to contain file paths, got: %s", output)
	}
	if !strings.Contains(output, "file2.txt") {
		t.Errorf("expected output to contain 'file2.txt', got: %s", output)
	}
}

func TestGrepDirectoryWithoutRecursive(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("hello world\n"), 0644)

	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Grep([]string{"hello", tmpDir}, nil, stdin, stdout, stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "Is a directory") {
		t.Errorf("expected error about directory, got: %s", errOutput)
	}
}
