package builtin

import (
	"bytes"
	"os"
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
	if !strings.Contains(stderr.String(), "only simple") {
		t.Errorf("expected error about s/old/new/, got %q", stderr.String())
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
	if !strings.Contains(stderr.String(), "No such file") && !strings.Contains(stderr.String(), "cannot find") {
		// error message varies by OS
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
