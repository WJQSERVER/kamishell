package builtin

import (
	"bytes"
	"strings"
	"testing"
)

func TestTypeBuiltin(t *testing.T) {
	stdout := &bytes.Buffer{}
	code := Type([]string{"cd"}, &mockEnv{}, nil, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "shell builtin") {
		t.Errorf("expected 'shell builtin', got %q", stdout.String())
	}
}

func TestTypeExternalCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	code := Type([]string{"sh"}, &mockEnv{}, nil, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.Len() == 0 {
		t.Error("expected output, got empty")
	}
}

func TestTypeNotFound(t *testing.T) {
	stderr := &bytes.Buffer{}
	code := Type([]string{"nonexistent_command_xyz"}, &mockEnv{}, nil, &bytes.Buffer{}, stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Errorf("expected 'not found', got %q", stderr.String())
	}
}

func TestTypeNoArgs(t *testing.T) {
	code := Type([]string{}, &mockEnv{}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

type mockEnv struct{}

func (m *mockEnv) Get(name string) (any, bool)          { return nil, false }
func (m *mockEnv) Set(name string, val any)              {}
func (m *mockEnv) GetString(name string) (string, bool)  { return "", false }
func (m *mockEnv) SetString(name string, val string)     {}
