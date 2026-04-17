package core

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
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

func TestHTTPBuiltin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		_, _ = w.Write([]byte("kami-http"))
	}))
	defer server.Close()

	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("http \""+server.URL+"\"", env)

	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "kami-http" {
		t.Errorf("expected kami-http, got %q", stdout)
	}
}

func TestBuiltinHelpIntegration(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("help http", env)

	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if !strings.Contains(stdout, "用法: http [flags] [METHOD] URL") {
		t.Errorf("unexpected stdout: %q", stdout)
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

func TestScriptEnvPackage(t *testing.T) {
	env := NewEmptyEnvironment()
	input := "env.Set(\"GOOS\", \"linux\"); print env.Get(\"GOOS\"); env.Unset(\"GOOS\"); print env.Get(\"GOOS\")"
	stdout, stderr, _ := runKami(input, env)

	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}

	lines := strings.Split(stdout, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if strings.TrimSpace(lines[0]) != "linux" {
		t.Errorf("expected first line linux, got %q", lines[0])
	}
	if lines[1] != "" {
		t.Errorf("expected second line empty after unset, got %q", lines[1])
	}
	if lines[2] != "" {
		t.Errorf("expected trailing newline terminator, got %q", lines[2])
	}
}

func TestScriptEnvPackageExpressionResult(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, result := runKami("env.Set(\"GOOS\", \"linux\"); env.Get(\"GOOS\")", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if result == nil || result.Inspect() != "linux" {
		t.Errorf("expected linux, got %v", result)
	}
}

func TestVarWithTypeZeroValue(t *testing.T) {
	env := NewEmptyEnvironment()
	input := "var count int; var name string; var ready bool; print count; print name; print ready"
	stdout, stderr, _ := runKami(input, env)

	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}

	lines := strings.Split(stdout, "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "0" || lines[1] != "" || lines[2] != "false" {
		t.Errorf("unexpected zero values: %v", lines)
	}
	if lines[3] != "" {
		t.Errorf("expected trailing newline terminator, got %q", lines[3])
	}
}

func TestVarTypeMismatch(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, _ := runKami("var count int = true", env)
	if !strings.Contains(stderr, "cannot initialize INTEGER with value of type BOOLEAN") {
		t.Errorf("expected type mismatch error, got %q", stderr)
	}
}

func TestAssignmentUpdatesOuterScope(t *testing.T) {
	env := NewEmptyEnvironment()
	input := "x := 1; func update() { x = 2 }; update(); print x"
	stdout, stderr, _ := runKami(input, env)

	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "2" {
		t.Errorf("expected 2, got %q", stdout)
	}
}

func TestTypedAssignmentRejectsMismatch(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, _ := runKami("var count int = 1; count = true", env)
	if !strings.Contains(stderr, "cannot assign BOOLEAN to variable of type INTEGER") {
		t.Errorf("expected typed assignment failure, got %q", stderr)
	}
}

func TestNilDoesNotBecomeTrackedVariableType(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("x := nil; x = 1; print x", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "1" {
		t.Errorf("expected 1, got %q", stdout)
	}
	if tracked, ok := env.GetType("x"); ok && tracked == string(NULL_OBJ) {
		t.Errorf("did not expect x to be typed as NULL")
	}
}

func TestVarNilDoesNotFreezeNullType(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("var x = nil; x = true; print x", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "true" {
		t.Errorf("expected true, got %q", stdout)
	}
	if tracked, ok := env.GetType("x"); ok && tracked == string(NULL_OBJ) {
		t.Errorf("did not expect x to be typed as NULL")
	}
}

func TestBackgroundExecutionDoesNotMutateParentVariableScope(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, _ := runKami("x := 1; go { x = 2 }", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	time.Sleep(20 * time.Millisecond)
	value, ok := env.GetObject("x")
	if !ok {
		t.Fatal("expected x to exist in parent env")
	}
	if value.(*Integer).Value != 1 {
		t.Fatalf("expected parent x to stay 1, got %s", value.Inspect())
	}
}

func TestBackgroundExecutionDoesNotMutateParentScriptEnvPackage(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, _ := runKami("env.Set(\"GOOS\", \"linux\"); go { env.Set(\"GOOS\", \"windows\") }", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	time.Sleep(20 * time.Millisecond)
	value, ok := env.GetPackageValue("env", "GOOS")
	if !ok {
		t.Fatal("expected GOOS in parent script env package")
	}
	if value != "linux" {
		t.Fatalf("expected parent GOOS to stay linux, got %q", value)
	}
}

func TestAssignmentWithSpacesStillWorks(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("x := 1; x = 2; print x", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "2" {
		t.Errorf("expected 2, got %q", stdout)
	}
}

func TestAssignmentWithoutSpacesReportsSyntaxError(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, _ := runKami("x=1", env)
	if !strings.Contains(stderr, "syntax error") {
		t.Errorf("expected syntax error for x=1, got %q", stderr)
	}
}

func TestUserFunctionKeepsIntegerArguments(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("func add(a, b) { print a + b }; add(1, 2)", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "3" {
		t.Errorf("expected 3, got %q", stdout)
	}
}

func TestUserFunctionKeepsBooleanArguments(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("func pick(flag) { if flag == true { print \"yes\" } else { print \"no\" } }; pick(true)", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "yes" {
		t.Errorf("expected yes, got %q", stdout)
	}
}

func TestCommandAndExecUserFunctionUseStringArguments(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("func describe(v) { print v }; describe 7; exec \"describe 7\"", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "7" || lines[1] != "7" {
		t.Fatalf("expected both command paths to print 7, got %v", lines)
	}
}

func TestUnicodeVariableNameWorks(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("变量 := 1; print 变量", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "1" {
		t.Errorf("expected 1, got %q", stdout)
	}
}

func TestIntegerLiteralOverflowReportsError(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, _ := runKami("print 999999999999999999999999999999999999", env)
	if !strings.Contains(stderr, "invalid integer literal") {
		t.Errorf("expected invalid integer literal error, got %q", stderr)
	}
}

func TestEnvGetOS(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("os := env.GetOS(); print os", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Errorf("expected non-empty OS string, got empty")
	}
}

func TestEnvGetArch(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("arch := env.GetArch(); print arch", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Errorf("expected non-empty architecture string, got empty")
	}
}

func TestEnvGetOSNoArgs(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, _ := runKami("env.GetOS(\"invalid\")", env)
	if !strings.Contains(stderr, "expects no arguments") {
		t.Errorf("expected 'expects no arguments' error, got %q", stderr)
	}
}

func TestEnvGetArchNoArgs(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, _ := runKami("env.GetArch(\"invalid\")", env)
	if !strings.Contains(stderr, "expects no arguments") {
		t.Errorf("expected 'expects no arguments' error, got %q", stderr)
	}
}
