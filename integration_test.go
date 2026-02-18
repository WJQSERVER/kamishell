package kamishell

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

func runKami(input string, env *Environment) (string, string, Object) {
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	result := EvalWithIO(program, env, os.Stdin, stdout, stderr)

	if isError(result) {
		fmt.Fprintf(stderr, "%s\n", result.Inspect())
	}

	return stdout.String(), stderr.String(), result
}

func TestVariablesAndArithmetic(t *testing.T) {
	env := NewEmptyEnvironment()
	input := "x := 10 + 20; print x"
	stdout, stderr, _ := runKami(input, env)

	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "30" {
		t.Errorf("expected 30, got %s", stdout)
	}
}

func TestReassignment(t *testing.T) {
	env := NewEmptyEnvironment()
	input := "x := 1; x = 2; print x"
	stdout, stderr, _ := runKami(input, env)

	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "2" {
		t.Errorf("expected 2, got %s", stdout)
	}
}

func TestStringInterpolation(t *testing.T) {
	env := NewEmptyEnvironment()
	input := "name := \"kami\"; print \"hello $name\""
	stdout, stderr, _ := runKami(input, env)

	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "hello kami" {
		t.Errorf("expected 'hello kami', got %s", stdout)
	}
}

func TestStandaloneInterpolation(t *testing.T) {
	env := NewEmptyEnvironment()
	input := "name := \"kami\"; print $name"
	stdout, stderr, _ := runKami(input, env)

	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "kami" {
		t.Errorf("expected 'kami', got %s", stdout)
	}
}

func TestRedirection(t *testing.T) {
	env := NewEmptyEnvironment()
	tempFile := "test_redir.txt"
	defer os.Remove(tempFile)

	input := "print \"hello world\" > \"" + tempFile + "\"; cat \"" + tempFile + "\""
	stdout, stderr, _ := runKami(input, env)

	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "hello world" {
		t.Errorf("expected 'hello world', got %s", stdout)
	}
}

func TestPipeline(t *testing.T) {
	env := NewEmptyEnvironment()
	input := "print \"line1\nline2\" | cat"
	stdout, stderr, _ := runKami(input, env)

	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if !strings.Contains(stdout, "line1") || !strings.Contains(stdout, "line2") {
		t.Errorf("pipeline output missing lines, got %s", stdout)
	}
}

func TestForLoop(t *testing.T) {
	env := NewEmptyEnvironment()
	input := "i := 0; for i < 3 { print i; i = i + 1 }"
	stdout, stderr, result := runKami(input, env)

	if stderr != "" {
		t.Errorf("unexpected stderr: %s (result: %v)", stderr, result)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	expected := []string{"0", "1", "2"}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestBuiltins(t *testing.T) {
	env := NewEmptyEnvironment()
	dirName := "test_dir_builtin"
	defer os.RemoveAll(dirName)

	origWD, _ := os.Getwd()
	defer os.Chdir(origWD)

	input := "mkdir \"" + dirName + "\"; cd \"" + dirName + "\"; pwd"
	stdout, stderr, _ := runKami(input, env)

	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if !strings.Contains(stdout, dirName) {
		t.Errorf("pwd output missing dir name, got %s", stdout)
	}
}

func TestEnvironment(t *testing.T) {
	env := NewEmptyEnvironment()
	input := "export \"KAMI_TEST=123\"; print $KAMI_TEST"
	stdout, stderr, _ := runKami(input, env)

	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "123" {
		t.Errorf("expected 123, got %s", stdout)
	}
}
