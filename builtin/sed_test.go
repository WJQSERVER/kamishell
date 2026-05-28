package builtin

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSedBasicReplacement(t *testing.T) {
	stdin := strings.NewReader("hello world\n")
	stdout := &bytes.Buffer{}
	code := Sed([]string{"s/world/kami/"}, nil, stdin, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if strings.TrimSpace(stdout.String()) != "hello kami" {
		t.Errorf("expected 'hello kami', got %q", stdout.String())
	}
}

func TestSedMultipleLines(t *testing.T) {
	stdin := strings.NewReader("foo bar\nfoo baz\n")
	stdout := &bytes.Buffer{}
	code := Sed([]string{"s/foo/REPLACED/"}, nil, stdin, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "REPLACED bar" {
		t.Errorf("expected 'REPLACED bar', got %q", lines[0])
	}
	if lines[1] != "REPLACED baz" {
		t.Errorf("expected 'REPLACED baz', got %q", lines[1])
	}
}

func TestSedNoMatch(t *testing.T) {
	stdin := strings.NewReader("hello world\n")
	stdout := &bytes.Buffer{}
	code := Sed([]string{"s/xyz/abc/"}, nil, stdin, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if strings.TrimSpace(stdout.String()) != "hello world" {
		t.Errorf("expected 'hello world', got %q", stdout.String())
	}
}

func TestSedEmptyExpression(t *testing.T) {
	code := Sed([]string{}, nil, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestSedInvalidExpression(t *testing.T) {
	stderr := &bytes.Buffer{}
	code := Sed([]string{"invalid"}, nil, strings.NewReader(""), &bytes.Buffer{}, stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Errorf("expected error about unknown command, got %q", stderr.String())
	}
}

func TestSedFromFile(t *testing.T) {
	tmpFile := t.TempDir() + "/test.txt"
	if err := writeFile(tmpFile, "line one\nline two\n"); err != nil {
		t.Fatal(err)
	}
	stdout := &bytes.Buffer{}
	code := Sed([]string{"s/line/LINE/", tmpFile}, nil, nil, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "LINE one") {
		t.Errorf("expected 'LINE one', got %q", stdout.String())
	}
}

func TestSedNonexistentFile(t *testing.T) {
	stderr := &bytes.Buffer{}
	code := Sed([]string{"s/a/b/", "/nonexistent/file.txt"}, nil, nil, &bytes.Buffer{}, stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestSedGlobalFlag(t *testing.T) {
	stdin := strings.NewReader("a b a b a b\n")
	stdout := &bytes.Buffer{}
	code := Sed([]string{"s/a/X/g"}, nil, stdin, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if strings.TrimSpace(stdout.String()) != "X b X b X b" {
		t.Errorf("expected 'X b X b X b', got %q", stdout.String())
	}
}

func TestSedNoGlobalOnlyFirst(t *testing.T) {
	stdin := strings.NewReader("a b a b a b\n")
	stdout := &bytes.Buffer{}
	code := Sed([]string{"s/a/X/"}, nil, stdin, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if strings.TrimSpace(stdout.String()) != "X b a b a b" {
		t.Errorf("expected 'X b a b a b', got %q", stdout.String())
	}
}

func TestSedDeleteLine(t *testing.T) {
	stdin := strings.NewReader("line1\nline2\nline3\n")
	stdout := &bytes.Buffer{}
	code := Sed([]string{"2d"}, nil, stdin, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 || lines[0] != "line1" || lines[1] != "line3" {
		t.Errorf("expected 'line1\\nline3', got %q", stdout.String())
	}
}

func TestSedDeleteRange(t *testing.T) {
	stdin := strings.NewReader("a\nb\nc\nd\ne\n")
	stdout := &bytes.Buffer{}
	code := Sed([]string{"2,4d"}, nil, stdin, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "e" {
		t.Errorf("expected 'a\\ne', got %q", stdout.String())
	}
}

func TestSedQuietMode(t *testing.T) {
	stdin := strings.NewReader("hit\nmiss\nhit\n")
	stdout := &bytes.Buffer{}
	code := Sed([]string{"-n", "/hit/p"}, nil, stdin, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 || lines[0] != "hit" || lines[1] != "hit" {
		t.Errorf("expected 'hit\\nhit', got %q", stdout.String())
	}
}

func TestSedPrintMatch(t *testing.T) {
	stdin := strings.NewReader("a\nb\nc\n")
	stdout := &bytes.Buffer{}
	code := Sed([]string{"2p"}, nil, stdin, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	// p prints the line AND auto-print also prints it, so we see line2 twice
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 4 || lines[0] != "a" || lines[1] != "b" || lines[2] != "b" || lines[3] != "c" {
		t.Errorf("unexpected output: %q", stdout.String())
	}
}

func TestSedRegexAddress(t *testing.T) {
	stdin := strings.NewReader("foo\nbar\nbaz\nfoo\n")
	stdout := &bytes.Buffer{}
	code := Sed([]string{"/bar/d"}, nil, stdin, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 3 || lines[0] != "foo" || lines[1] != "baz" || lines[2] != "foo" {
		t.Errorf("expected 'foo\\nbaz\\nfoo', got %q", stdout.String())
	}
}

func TestSedRegexRange(t *testing.T) {
	stdin := strings.NewReader("a\nSTART\nb\nc\nEND\nd\n")
	stdout := &bytes.Buffer{}
	code := Sed([]string{"/START/,/END/d"}, nil, stdin, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "d" {
		t.Errorf("expected 'a\\nd', got %q", stdout.String())
	}
}

func TestSedLastLineAddress(t *testing.T) {
	stdin := strings.NewReader("a\nb\nc\n")
	stdout := &bytes.Buffer{}
	code := Sed([]string{"$d"}, nil, stdin, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if strings.TrimSpace(stdout.String()) != "a\nb" && strings.TrimSpace(stdout.String()) != "a\nb\n" {
		t.Errorf("expected 'a\\nb', got %q", stdout.String())
	}
}

func TestSedMultipleExpressions(t *testing.T) {
	stdin := strings.NewReader("foo bar\nbaz qux\n")
	stdout := &bytes.Buffer{}
	code := Sed([]string{"-e", "s/foo/F/", "-e", "s/bar/B/"}, nil, stdin, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 || lines[0] != "F B" || lines[1] != "baz qux" {
		t.Errorf("expected 'F B\\nbaz qux', got %q", stdout.String())
	}
}

func TestSedInPlace(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	if err := writeFile(file, "hello world\nfoo bar\n"); err != nil {
		t.Fatal(err)
	}

	code := Sed([]string{"-i", "s/world/kami/", file}, nil, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "hello kami") {
		t.Errorf("expected file to contain 'hello kami', got %q", string(data))
	}
}

func TestSedInPlaceWithBackup(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	original := "hello world\n"
	if err := writeFile(file, original); err != nil {
		t.Fatal(err)
	}

	code := Sed([]string{"-i.bak", "s/world/kami/", file}, nil, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "hello kami") {
		t.Errorf("expected modified content, got %q", string(data))
	}

	backupData, err := os.ReadFile(file + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if string(backupData) != original {
		t.Errorf("expected backup to contain original, got %q", string(backupData))
	}
}

func TestSedSubstituteWithDifferentDelimiter(t *testing.T) {
	stdin := strings.NewReader("a/b/c\n")
	stdout := &bytes.Buffer{}
	code := Sed([]string{"s|/|-|g"}, nil, stdin, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if strings.TrimSpace(stdout.String()) != "a-b-c" {
		t.Errorf("expected 'a-b-c', got %q", stdout.String())
	}
}

func TestSedDeleteThenSubstitute(t *testing.T) {
	// Complex command: delete line 1, then substitute on remaining
	// Note: sed processes all commands per-line. Line 1 gets deleted.
	stdin := strings.NewReader("header\nbody1\nbody2\n")
	stdout := &bytes.Buffer{}
	code := Sed([]string{"-e", "1d", "-e", "s/body/item/"}, nil, stdin, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 || lines[0] != "item1" || lines[1] != "item2" {
		t.Errorf("expected 'item1\\nitem2', got %q", stdout.String())
	}
}

func writeFile(path, content string) error {
	f, err := createFile(path)
	if err != nil {
		return err
	}
	_, err = f.WriteString(content)
	f.Close()
	return err
}

func createFile(path string) (*os.File, error) {
	return os.Create(path)
}
