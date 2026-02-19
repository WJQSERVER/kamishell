package builtin

import (
	"bytes"
	"os"
	"testing"
)

func TestCat(t *testing.T) {
	content := "hello world"
	tmpFile := "test_cat.txt"
	os.WriteFile(tmpFile, []byte(content), 0644)
	defer os.Remove(tmpFile)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Test file
	Cat([]string{tmpFile}, &rmMockEnv{}, nil, stdout, stderr)
	if stdout.String() != content {
		t.Errorf("expected %q, got %q", content, stdout.String())
	}

	stdout.Reset()
	// Test stdin
	stdin := bytes.NewBufferString("stdin content")
	Cat([]string{"-"}, &rmMockEnv{}, stdin, stdout, stderr)
	if stdout.String() != "stdin content" {
		t.Errorf("expected %q, got %q", "stdin content", stdout.String())
	}
}
