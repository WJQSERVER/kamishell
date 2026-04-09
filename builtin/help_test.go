package builtin

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuiltinHelpRequested(t *testing.T) {
	if !BuiltinHelpRequested([]string{"--help"}) {
		t.Fatal("expected --help to be recognized")
	}
	if BuiltinHelpRequested([]string{"-h"}) {
		t.Fatal("did not expect -h to be treated as generic builtin help")
	}
}

func TestHelpBuiltinShowsCommandHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Help([]string{"http"}, &rmMockEnv{store: map[string]interface{}{}}, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "用法: http [flags] [METHOD] URL") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestNonFlagBuiltinSupportsDoubleDashHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Touch([]string{"--help"}, &rmMockEnv{store: map[string]interface{}{}}, nil, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "用法: touch file...") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}
