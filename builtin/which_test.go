package builtin

import (
	"bytes"
	"strings"
	"testing"
)

func TestWhichFindsSh(t *testing.T) {
	stdout := &bytes.Buffer{}
	code := Which([]string{"sh"}, nil, nil, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.Len() == 0 {
		t.Error("expected output, got empty")
	}
}

func TestWhichNotFound(t *testing.T) {
	code := Which([]string{"nonexistent_command_xyz"}, nil, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestWhichNoArgs(t *testing.T) {
	code := Which([]string{}, nil, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestWhichFindsBuiltin(t *testing.T) {
	stdout := &bytes.Buffer{}
	code := Which([]string{"cd"}, nil, nil, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "shell builtin") {
		t.Errorf("expected output to indicate shell builtin, got %q", stdout.String())
	}
}

func TestWhichFindsMultipleBuiltins(t *testing.T) {
	stdout := &bytes.Buffer{}
	code := Which([]string{"cd", "ls", "cat"}, nil, nil, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), stdout.String())
	}
	for _, line := range lines {
		if !strings.Contains(line, "shell builtin") {
			t.Errorf("expected each line to indicate shell builtin, got %q", line)
		}
	}
}

func TestWhichBuiltinAndExternal(t *testing.T) {
	stdout := &bytes.Buffer{}
	code := Which([]string{"cd", "sh"}, nil, nil, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), stdout.String())
	}
	if !strings.Contains(lines[0], "shell builtin") {
		t.Errorf("expected first line to be builtin, got %q", lines[0])
	}
}
