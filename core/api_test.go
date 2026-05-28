package core

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// --- Simplified API tests: Run / RunWithIO ---

func TestRunSimpleExpression(t *testing.T) {
	var buf strings.Builder
	_, err := RunWithIO(`x := 1 + 2; print x`, os.Stdin, &buf, os.Stderr, WithEnvironment(NewEmptyEnvironment()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "3" {
		t.Fatalf("expected 3, got %q", buf.String())
	}
}

func TestRunReturnValue(t *testing.T) {
	result, err := Run(`func add(a int, b int) int { return a + b }; add(3, 4)`, WithEnvironment(NewEmptyEnvironment()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type() != INTEGER_OBJ {
		t.Fatalf("expected INTEGER, got %s", result.Type())
	}
	if result.Inspect() != "7" {
		t.Fatalf("expected 7, got %s", result.Inspect())
	}
}

func TestRunPrintCapturesStdout(t *testing.T) {
	var buf strings.Builder
	result, err := RunWithIO(`print "hello kamu"`, os.Stdin, &buf, os.Stderr, WithEnvironment(NewEmptyEnvironment()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if out != "hello kamu" {
		t.Fatalf("expected 'hello kamu', got %q", out)
	}
	if result != NULL {
		t.Fatalf("expected NULL result, got %v", result)
	}
}

func TestRunWithCustomEnvironment(t *testing.T) {
	var stdout, stderr strings.Builder
	env := NewEmptyEnvironment()
	env.SetString("MY_VAR", "hello")
	result, err := RunWithIO(`print $MY_VAR`, os.Stdin, &stdout, &stderr, WithEnvironment(env))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := strings.TrimSpace(stdout.String())
	if out != "hello" {
		t.Fatalf("expected 'hello', got %q", out)
	}
	if result != NULL {
		t.Fatalf("expected NULL, got %v", result)
	}
}

func TestRunWithStdoutCapture(t *testing.T) {
	var buf strings.Builder
	_, err := RunWithIO(`print "hello"`, os.Stdin, &buf, os.Stderr, WithEnvironment(NewEmptyEnvironment()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "hello" {
		t.Fatalf("expected 'hello', got %q", buf.String())
	}
}

func TestRunParseError(t *testing.T) {
	_, err := RunWithIO(`(1 + 2`, os.Stdin, os.Stdout, os.Stderr, WithEnvironment(NewEmptyEnvironment()))
	if err == nil {
		t.Fatal("expected error for invalid syntax")
	}
	if !strings.Contains(err.Error(), "parse error") {
		t.Fatalf("expected parse error, got: %v", err)
	}
}

func TestParseSyntaxError(t *testing.T) {
	_, errs := Parse(`(1 + 2`)
	if len(errs) == 0 {
		t.Fatal("expected parse errors for invalid syntax")
	}
}

func TestRunTimeout(t *testing.T) {
	var buf strings.Builder
	_, err := RunWithIO(`for i := 0; i < 9999999999; i = i + 1 { }`, os.Stdin, &buf, os.Stderr,
		WithEnvironment(NewEmptyEnvironment()),
		WithTimeout(100*time.Millisecond),
	)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "deadline exceeded") && !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("expected timeout/cancelled error, got: %v", err)
	}
}

func TestRunFunctionAndReturn(t *testing.T) {
	result, err := Run(`func add(a int, b int) int { return a + b }; add(3, 4)`, WithEnvironment(NewEmptyEnvironment()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type() != INTEGER_OBJ {
		t.Fatalf("expected INTEGER, got %s", result.Type())
	}
	if result.Inspect() != "7" {
		t.Fatalf("expected 7, got %s", result.Inspect())
	}
}

func TestRunMultiReturn(t *testing.T) {
	var buf strings.Builder
	_, err := RunWithIO(`func div(a int, b int) (int, error) { return a / b, nil }; q, e := div(10, 2); print q; print e`, os.Stdin, &buf, os.Stderr, WithEnvironment(NewEmptyEnvironment()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 1 || strings.TrimSpace(lines[0]) != "5" {
		t.Fatalf("expected 5, got %v", lines)
	}
}

func TestRunArrayOperations(t *testing.T) {
	result, err := Run(`arr := [1, 2, 3]; len(arr)`, WithEnvironment(NewEmptyEnvironment()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Inspect() != "3" {
		t.Fatalf("expected 3, got %s", result.Inspect())
	}
}

func TestRunForLoop(t *testing.T) {
	var buf strings.Builder
	_, err := RunWithIO(`s := 0; for i := 0; i < 5; i = i + 1 { s = s + i }; print s`, os.Stdin, &buf, os.Stderr, WithEnvironment(NewEmptyEnvironment()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "10" {
		t.Fatalf("expected 10, got %q", buf.String())
	}
}

func TestRunWhileLoop(t *testing.T) {
	var buf strings.Builder
	_, err := RunWithIO(`i := 0; for i < 3 { print i; i = i + 1 }`, os.Stdin, &buf, os.Stderr, WithEnvironment(NewEmptyEnvironment()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 || lines[0] != "0" || lines[1] != "1" || lines[2] != "2" {
		t.Fatalf("expected 0,1,2 on separate lines, got %q", buf.String())
	}
}

func TestRunSwitch(t *testing.T) {
	var buf strings.Builder
	_, err := RunWithIO(`x := 2; switch x { case 1: print "one"; case 2: print "two"; default: print "other" }`, os.Stdin, &buf, os.Stderr, WithEnvironment(NewEmptyEnvironment()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "two" {
		t.Fatalf("expected 'two', got %q", buf.String())
	}
}

func TestRunRuntimeErrorPropagated(t *testing.T) {
	result, err := Run(`x := 1; y := x / 0`, WithEnvironment(NewEmptyEnvironment()))
	if err == nil {
		t.Fatal("expected error for division by zero")
	}
	if result == nil {
		t.Fatalf("expected result to be non-nil (error object), got nil")
	}
	if result.Type() != ERROR_OBJ {
		t.Fatalf("expected ERROR_OBJ, got %s", result.Type())
	}
}

func TestRunSandboxDefaultBlocksExternal(t *testing.T) {
	_, err := Run(`nonexistent_cmd_xyz`)
	if err == nil {
		t.Fatal("expected sandbox to block external command")
	}
	if !strings.Contains(err.Error(), "not allowed in sandbox") {
		t.Fatalf("expected sandbox error, got: %v", err)
	}
}

func TestRunSandboxBlocksBuiltin(t *testing.T) {
	env := NewSandboxEnvironment()
	env.SetBlockedBuiltins([]string{"http"})
	_, err := RunWithIO(`http "https://example.com"`, os.Stdin, os.Stdout, os.Stderr, WithEnvironment(env))
	if err == nil {
		t.Fatal("expected sandbox to block http builtin")
	}
	if !strings.Contains(err.Error(), "not allowed in sandbox") {
		t.Fatalf("expected sandbox block error, got: %v", err)
	}
}

func TestRunSandboxBuiltinWhitelist(t *testing.T) {
	env := NewSandboxEnvironment()
	env.SetAllowedBuiltins([]string{"print"})
	var buf strings.Builder
	_, err := RunWithIO(`print "hello"`, os.Stdin, &buf, os.Stderr, WithEnvironment(env))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "hello" {
		t.Fatalf("expected 'hello', got %q", buf.String())
	}
}

func TestRunSandboxBuiltinWhitelistRejectsOther(t *testing.T) {
	env := NewSandboxEnvironment()
	env.SetAllowedBuiltins([]string{"print"})
	_, err := RunWithIO(`ls`, os.Stdin, os.Stdout, os.Stderr, WithEnvironment(env))
	if err == nil {
		t.Fatal("expected sandbox to reject 'ls' builtin")
	}
	if !strings.Contains(err.Error(), "not allowed in sandbox") {
		t.Fatalf("expected sandbox block error, got: %v", err)
	}
}

func TestSandboxModeBlocksExternal(t *testing.T) {
	_, err := Run(`nonexistent_cmd_xyz`, SandboxMode())
	if err == nil {
		t.Fatal("expected SandboxMode to block external command")
	}
	if !strings.Contains(err.Error(), "not allowed in sandbox") {
		t.Fatalf("expected sandbox error, got: %v", err)
	}
}

func TestRunWithTimeout(t *testing.T) {
	_, err := Run(`for i := 0; i < 9999999999; i = i + 1 { }`, WithEnvironment(NewEmptyEnvironment()), WithTimeout(100*time.Millisecond))
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "deadline exceeded") && !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("expected timeout/cancelled error, got: %v", err)
	}
}

func TestSandboxEnvironmentIsolated(t *testing.T) {
	env := NewSandboxEnvironment()
	if env.IsSandboxed() != true {
		t.Fatal("expected sandboxed to be true")
	}
	val, ok := env.Get("PATH")
	if ok {
		t.Fatalf("expected sandbox to not have system env vars, got PATH=%v", val)
	}
}

func TestSandboxSetAllowExternalCmd(t *testing.T) {
	env := NewSandboxEnvironment()
	if env.allowExternalCmd != false {
		t.Fatal("expected allowExternalCmd to default to false")
	}
	env.SetAllowExternalCmd(true)
	if env.allowExternalCmd != true {
		t.Fatal("expected allowExternalCmd to be true after SetAllowExternalCmd(true)")
	}
}

func TestSandboxSetMaxRecursionDepth(t *testing.T) {
	env := NewSandboxEnvironment()
	if env.maxRecursionDepth != 100 {
		t.Fatalf("expected default maxRecursionDepth=100, got %d", env.maxRecursionDepth)
	}
	env.SetMaxRecursionDepth(50)
	if env.maxRecursionDepth != 50 {
		t.Fatalf("expected maxRecursionDepth=50, got %d", env.maxRecursionDepth)
	}
}

// --- Parse tests ---

func TestParseValidProgram(t *testing.T) {
	program, errs := Parse(`x := 1 + 2; print x`)
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if program == nil {
		t.Fatal("expected non-nil program")
	}
	if len(program.Statements) == 0 {
		t.Fatal("expected at least one statement")
	}
}

func TestNewSandboxEnvironmentDefaults(t *testing.T) {
	env := NewSandboxEnvironment()
	if env.sandboxed != true {
		t.Error("expected sandboxed=true")
	}
	if env.allowExternalCmd != false {
		t.Error("expected allowExternalCmd=false")
	}
	if env.maxRecursionDepth != 100 {
		t.Errorf("expected maxRecursionDepth=100, got %d", env.maxRecursionDepth)
	}
	if env.allowedBuiltins != nil {
		t.Error("expected allowedBuiltins=nil")
	}
	if env.blockedBuiltins != nil {
		t.Error("expected blockedBuiltins=nil")
	}
}

func TestIsBuiltinAllowedAllAllowed(t *testing.T) {
	env := NewEmptyEnvironment()
	if env.IsBuiltinAllowed("ls") != true {
		t.Error("expected ls to be allowed in non-sandbox env")
	}
}

func TestIsBuiltinAllowedWhitelist(t *testing.T) {
	env := NewSandboxEnvironment()
	env.SetAllowedBuiltins([]string{"print", "cd"})
	if env.IsBuiltinAllowed("print") != true {
		t.Error("expected print to be allowed")
	}
	if env.IsBuiltinAllowed("ls") != false {
		t.Error("expected ls to be blocked")
	}
}

func TestIsBuiltinAllowedBlacklist(t *testing.T) {
	env := NewEmptyEnvironment()
	env.SetBlockedBuiltins([]string{"http"})
	if env.IsBuiltinAllowed("http") != false {
		t.Error("expected http to be blocked")
	}
	if env.IsBuiltinAllowed("print") != true {
		t.Error("expected print to be allowed")
	}
}

func TestRunWithFullEnvironment(t *testing.T) {
	env := NewEmptyEnvironment()
	var buf strings.Builder
	_, err := RunWithIO(`x := 1 + 2; print x`, os.Stdin, &buf, os.Stderr, WithEnvironment(env))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "3" {
		t.Fatalf("expected 3, got %q", buf.String())
	}
}

func TestRunChainedOptions(t *testing.T) {
	var buf strings.Builder
	env := NewEmptyEnvironment()
	env.SetString("MSG", "chained")
	_, err := RunWithIO(`print $MSG`, os.Stdin, &buf, os.Stderr,
		WithEnvironment(env),
		WithTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "chained" {
		t.Fatalf("expected 'chained', got %q", buf.String())
	}
}

func TestRunPointerOperations(t *testing.T) {
	var buf strings.Builder
	_, err := RunWithIO(`x := 10; p := &x; *p = 20; print x`, os.Stdin, &buf, os.Stderr, WithEnvironment(NewEmptyEnvironment()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "20" {
		t.Fatalf("expected 20, got %q", buf.String())
	}
}

func TestRunStringInterpolation(t *testing.T) {
	var buf strings.Builder
	_, err := RunWithIO(`name := "kami"; print "hello $name"`, os.Stdin, &buf, os.Stderr, WithEnvironment(NewEmptyEnvironment()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "hello kami" {
		t.Fatalf("expected 'hello kami', got %q", buf.String())
	}
}

func TestSandboxModePreservesEnvVarAccess(t *testing.T) {
	_, err := Run(`x := 1 + 2; print x`, SandboxMode())
	if err != nil {
		t.Fatalf("expected sandbox mode to work, got %v", err)
	}
}

func TestSandboxModeWithNonSandboxEnv(t *testing.T) {
	env := NewEmptyEnvironment()
	_, err := RunWithIO(`nonexistent_cmd_xyz`, os.Stdin, os.Stdout, os.Stderr, WithEnvironment(env), SandboxMode())
	if err == nil {
		t.Fatal("expected SandboxMode to block external command even on non-sandbox env")
	}
	if !strings.Contains(err.Error(), "not allowed in sandbox") {
		t.Fatalf("expected sandbox error, got: %v", err)
	}
}

func TestRunImportAndUseStdlib(t *testing.T) {
	var buf strings.Builder
	_, err := RunWithIO(`import "Go/strings"; print strings.Contains("hello", "ell")`, os.Stdin, &buf, os.Stderr, WithEnvironment(NewEmptyEnvironment()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "true" {
		t.Fatalf("expected 'true', got %q", buf.String())
	}
}

func TestRunRecursionDepthExceeded(t *testing.T) {
	env := NewSandboxEnvironment()
	env.SetMaxRecursionDepth(3)
	_, err := RunWithIO(`func f() { f() }; f()`, os.Stdin, os.Stdout, os.Stderr, WithEnvironment(env))
	if err == nil {
		t.Fatal("expected max recursion depth error")
	}
	if !strings.Contains(err.Error(), "max recursion depth") {
		t.Fatalf("expected recursion depth error, got: %v", err)
	}
}

func TestRunRecursionWithinLimit(t *testing.T) {
	env := NewSandboxEnvironment()
	env.SetMaxRecursionDepth(50)
	var buf strings.Builder
	_, err := RunWithIO(`func fact(n int) int { if n <= 1 { return 1 }; return n * fact(n - 1) }; print fact(5)`, os.Stdin, &buf, os.Stderr, WithEnvironment(env))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "120" {
		t.Fatalf("expected 120, got %q", buf.String())
	}
}

// Benchmark for the simplified API
func BenchmarkRun(b *testing.B) {
	input := `x := 1; for i := 0; i < 100; i = i + 1 { x = x + i }; print x`
	for i := 0; i < b.N; i++ {
		RunWithIO(input, os.Stdin, &bytes.Buffer{}, &bytes.Buffer{}, WithEnvironment(NewEmptyEnvironment()))
	}
}

func ExampleRun() {
	result, err := Run(`func add(a int, b int) int { return a + b }; add(3, 4)`, WithEnvironment(NewEmptyEnvironment()))
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}
	fmt.Println(result.Inspect())
	// Output: 7
}

func ExampleRunWithIO() {
	var buf strings.Builder
	_, err := RunWithIO(`print "hello from kami"`, nil, &buf, os.Stderr, WithEnvironment(NewEmptyEnvironment()))
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}
	fmt.Print(buf.String())
	// Output: hello from kami
}

func ExampleParse() {
	program, errs := Parse(`x := 1; if x > 0 { print x }`)
	if len(errs) > 0 {
		fmt.Printf("parse errors: %v", errs)
		return
	}
	fmt.Printf("parsed %d statements", len(program.Statements))
	// Output: parsed 2 statements
}