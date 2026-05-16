package builtin

import (
	"bytes"
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
