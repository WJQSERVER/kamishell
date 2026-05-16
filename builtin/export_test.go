package builtin

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

type testEnv struct {
	data map[string]any
}

func (e *testEnv) Set(name string, val any) {
	e.data[name] = val
}

func (e *testEnv) Get(name string) (any, bool) {
	v, ok := e.data[name]
	return v, ok
}

func (e *testEnv) SetString(name string, val string) {
	e.data[name] = val
}

func (e *testEnv) GetString(name string) (string, bool) {
	v, ok := e.data[name]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func newTestEnv() *testEnv {
	return &testEnv{data: make(map[string]any)}
}

type osEnv struct {
	data map[string]any
}

func newOsEnv() *osEnv {
	return &osEnv{data: make(map[string]any)}
}

func (e *osEnv) Set(name string, val any) {
	e.data[name] = val
	if s, ok := val.(string); ok {
		os.Setenv(name, s)
	}
}

func (e *osEnv) Get(name string) (any, bool) {
	v, ok := e.data[name]
	if !ok {
		return os.LookupEnv(name)
	}
	return v, ok
}

func (e *osEnv) SetString(name string, val string) {
	e.data[name] = val
	os.Setenv(name, val)
}

func (e *osEnv) GetString(name string) (string, bool) {
	v, ok := e.data[name]
	if !ok {
		return os.LookupEnv(name)
	}
	s, ok := v.(string)
	return s, ok
}

func TestExportSetsVariable(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	env := newTestEnv()
	env.Set("FOO", "original")

	code := Export([]string{"FOO=bar"}, env, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	val, ok := env.Get("FOO")
	if !ok {
		t.Fatal("expected FOO to be set in env")
	}
	if val != "bar" {
		t.Errorf("expected FOO=bar, got %q", val)
	}
}

func TestExportWithoutValueShowsEnv(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	env := newOsEnv()
	os.Setenv("TEST_EXPORT_VAR", "testvalue")

	code := Export(nil, env, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "TEST_EXPORT_VAR=testvalue") {
		t.Errorf("expected output to contain TEST_EXPORT_VAR=testvalue, got: %s", output)
	}

	os.Unsetenv("TEST_EXPORT_VAR")
}

func TestExportInvalidFormat(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	env := newOsEnv()

	code := Export([]string{"INVALID"}, env, stdin, stdout, stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "usage:") {
		t.Errorf("expected stderr to contain 'usage:', got: %s", errOutput)
	}
}

func TestExportOverwrites(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	env := newOsEnv()
	env.Set("MYVAR", "old")

	code := Export([]string{"MYVAR=new"}, env, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	val, ok := env.Get("MYVAR")
	if !ok {
		t.Fatal("expected MYVAR to exist")
	}
	if val != "new" {
		t.Errorf("expected MYVAR=new, got %q", val)
	}
}

func TestExportMultiple(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	env := newOsEnv()

	code := Export([]string{"A=1", "B=2", "C=3"}, env, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	tests := []struct {
		key   string
		value string
	}{
		{"A", "1"},
		{"B", "2"},
		{"C", "3"},
	}

	for _, tt := range tests {
		val, ok := env.Get(tt.key)
		if !ok {
			t.Errorf("expected %s to be set", tt.key)
		}
		if val != tt.value {
			t.Errorf("expected %s=%s, got %s", tt.key, tt.value, val)
		}
	}
}

func TestExportEmptyValue(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	env := newOsEnv()

	code := Export([]string{"EMPTY="}, env, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	val, ok := env.Get("EMPTY")
	if !ok {
		t.Fatal("expected EMPTY to be set")
	}
	if val != "" {
		t.Errorf("expected EMPTY='', got %q", val)
	}
}

func TestExportWithEqualsInValue(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	env := newOsEnv()

	code := Export([]string{"PATH=/usr/bin:/usr/local/bin"}, env, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	val, ok := env.Get("PATH")
	if !ok {
		t.Fatal("expected PATH to be set")
	}
	if val != "/usr/bin:/usr/local/bin" {
		t.Errorf("expected PATH=/usr/bin:/usr/local/bin, got %q", val)
	}
}

func TestExportHelp(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	env := newOsEnv()

	code := Export([]string{"--help"}, env, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 for --help, got %d", code)
	}

	helpOutput := stdout.String()
	if !strings.Contains(helpOutput, "export") {
		t.Errorf("expected help output to contain 'export', got: %s", helpOutput)
	}
}

func TestExportSetsOSEnvironment(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	env := newOsEnv()

	testKey := "KAMI_TEST_EXPORT_" + strings.Repeat("X", 10)
	testValue := "test_value"

	code := Export([]string{testKey + "=" + testValue}, env, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	osValue := os.Getenv(testKey)
	if osValue != testValue {
		t.Errorf("expected os.Getenv(%q)=%q, got %q", testKey, testValue, osValue)
	}

	os.Unsetenv(testKey)
}

func TestEnvBasic(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	env := newOsEnv()
	os.Setenv("SIMPLE_VAR", "simple_value")

	code := Env(nil, env, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "SIMPLE_VAR=simple_value") {
		t.Errorf("expected output to contain SIMPLE_VAR=simple_value, got: %s", output)
	}

	os.Unsetenv("SIMPLE_VAR")
}

func TestEnvHelp(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	env := newOsEnv()

	code := Env([]string{"--help"}, env, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 for --help, got %d", code)
	}

	helpOutput := stdout.String()
	if !strings.Contains(helpOutput, "env") {
		t.Errorf("expected help output to contain 'env', got: %s", helpOutput)
	}
}
