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

	input := "print \"hello world\" -> \"" + tempFile + "\"; cat \"" + tempFile + "\""
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

func TestForLoopLargeCount(t *testing.T) {
	env := NewEmptyEnvironment()
	input := "i := 0; for i < 1000 { i = i + 1 }; print i"
	stdout, stderr, _ := runKami(input, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "1000" {
		t.Errorf("expected 1000, got %q", strings.TrimSpace(stdout))
	}
}

func TestForLoopDecrement(t *testing.T) {
	env := NewEmptyEnvironment()
	input := "i := 5; for i > 0 { i = i - 1 }; print i"
	stdout, stderr, _ := runKami(input, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "0" {
		t.Errorf("expected 0, got %q", strings.TrimSpace(stdout))
	}
}

func TestForLoopEmptyBody(t *testing.T) {
	env := NewEmptyEnvironment()
	input := "i := 0; for i < 5 { i = i + 1 }; print i"
	stdout, stderr, _ := runKami(input, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "5" {
		t.Errorf("expected 5, got %q", strings.TrimSpace(stdout))
	}
}

func TestForLoopZeroIterations(t *testing.T) {
	env := NewEmptyEnvironment()
	input := "i := 10; for i < 5 { i = i + 1 }; print i"
	stdout, stderr, _ := runKami(input, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "10" {
		t.Errorf("expected 10, got %q", strings.TrimSpace(stdout))
	}
}

func TestForLoopMultiStatementBody(t *testing.T) {
	env := NewEmptyEnvironment()
	input := "i := 0; for i < 3 { x := i + 1; print x; i = i + 1 }"
	stdout, stderr, _ := runKami(input, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	expected := []string{"1", "2", "3"}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestForThreeClauseBasic(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`for i := 0; i < 5; i = i + 1 { print i }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"0", "1", "2", "3", "4"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestForThreeClausePostIncrement(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`for i := 0; i < 3; i = i + 1 { x := i + i; print x }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"0", "2", "4"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestForThreeClauseZeroIterations(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`for i := 10; i < 5; i = i + 1 { print i }; print "done"`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "done" {
		t.Errorf("expected done, got %q", strings.TrimSpace(stdout))
	}
}

func TestForThreeClauseCountDown(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`for i := 5; i > 0; i = i - 1 { print i }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"5", "4", "3", "2", "1"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestForThreeClauseWithReturn(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`func find(n) { for i := 0; i < n; i = i + 1 { if i == 3 { return i } }; return -1 }; print find(10)`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "3" {
		t.Errorf("expected 3, got %q", strings.TrimSpace(stdout))
	}
}

func TestBreakBasic(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`for i := 0; i < 10; i = i + 1 { if i == 5 { break }; print i }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"0", "1", "2", "3", "4"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestContinueBasic(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`for i := 0; i < 6; i = i + 1 { if i == 3 { continue }; print i }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"0", "1", "2", "4", "5"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestBreakWhileStyle(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`i := 0; for i < 100 { if i == 3 { break }; print i; i = i + 1 }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"0", "1", "2"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
}

func TestContinueWhileStyle(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`i := 0; for i < 5 { i = i + 1; if i == 3 { continue }; print i }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"1", "2", "4", "5"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestBreakNestedLoops(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`for i := 0; i < 3; i = i + 1 { for j := 0; j < 5; j = j + 1 { if j == 2 { break }; print j }; print "next" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"0", "1", "next", "0", "1", "next", "0", "1", "next"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
}

func TestArrayLiteral(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`arr := [1, 2, 3]; print arr`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "[1, 2, 3]" {
		t.Errorf("expected [1, 2, 3], got %q", strings.TrimSpace(stdout))
	}
}

func TestArrayIndex(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`arr := [10, 20, 30]; print arr[0]; print arr[1]; print arr[2]`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"10", "20", "30"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestArrayLen(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`arr := [1, 2, 3, 4, 5]; print len(arr)`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "5" {
		t.Errorf("expected 5, got %q", strings.TrimSpace(stdout))
	}
}

func TestArrayPush(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`arr := [1, 2, 3]; arr2 := push(arr, 4); print arr2`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "[1, 2, 3, 4]" {
		t.Errorf("expected [1, 2, 3, 4], got %q", strings.TrimSpace(stdout))
	}
}

func TestArrayPushTypeMismatch(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, _ := runKami(`arr := [1, 2, 3]; arr2 := push(arr, "hello")`, env)
	if !strings.Contains(stderr, "type mismatch") {
		t.Errorf("expected type mismatch error, got %q", stderr)
	}
}

func TestArrayHomogeneousType(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, _ := runKami(`arr := [1, "hello", true]`, env)
	if !strings.Contains(stderr, "type mismatch") {
		t.Errorf("expected mixed type error, got %q", stderr)
	}
}

func TestArrayOutOfBounds(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, _ := runKami(`arr := [1, 2, 3]; print arr[5]`, env)
	if !strings.Contains(stderr, "out of bounds") {
		t.Errorf("expected out of bounds error, got %q", stderr)
	}
}

func TestArrayString(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`arr := ["a", "b", "c"]; print arr[1]`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "b" {
		t.Errorf("expected b, got %q", strings.TrimSpace(stdout))
	}
}

func TestArrayEmpty(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`arr := []; print len(arr)`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "0" {
		t.Errorf("expected 0, got %q", strings.TrimSpace(stdout))
	}
}

func TestArrayIndexAssign(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`arr := [1, 2, 3]; arr[1] = 99; print arr`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "[1, 99, 3]" {
		t.Errorf("expected [1, 99, 3], got %q", strings.TrimSpace(stdout))
	}
}

func TestArrayIndexAssignTypeMismatch(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, _ := runKami(`arr := [1, 2, 3]; arr[0] = "hello"`, env)
	if !strings.Contains(stderr, "cannot assign") {
		t.Errorf("expected type mismatch error, got %q", stderr)
	}
}

func TestArrayValueSemantics(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`a := [1, 2, 3]; b := a; b[0] = 99; print a; print b`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if strings.TrimSpace(lines[0]) != "[1, 2, 3]" {
		t.Errorf("a should be unchanged, got %q", strings.TrimSpace(lines[0]))
	}
	if strings.TrimSpace(lines[1]) != "[99, 2, 3]" {
		t.Errorf("b should be modified, got %q", strings.TrimSpace(lines[1]))
	}
}

func TestArrayEqual(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`a := [1, 2, 3]; b := [1, 2, 3]; print a == b`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "true" {
		t.Errorf("expected true, got %q", strings.TrimSpace(stdout))
	}
}

func TestArrayNotEqual(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`a := [1, 2, 3]; b := [1, 2, 4]; print a == b`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "false" {
		t.Errorf("expected false, got %q", strings.TrimSpace(stdout))
	}
}

func TestVarArrayDeclaration(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`var arr array; print len(arr)`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "0" {
		t.Errorf("expected 0, got %q", strings.TrimSpace(stdout))
	}
}

func TestArrayReassignValueSemantics(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`a := [1, 2, 3]; b := [4, 5, 6]; a = b; b[0] = 99; print a; print b`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if strings.TrimSpace(lines[0]) != "[4, 5, 6]" {
		t.Errorf("a should be unchanged after b modified, got %q", strings.TrimSpace(lines[0]))
	}
	if strings.TrimSpace(lines[1]) != "[99, 5, 6]" {
		t.Errorf("b should be modified, got %q", strings.TrimSpace(lines[1]))
	}
}

func TestRangeIndexOnly(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`arr := [10, 20, 30]; for i := range arr { print i }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"0", "1", "2"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestRangeIndexAndValue(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`arr := [10, 20, 30]; for i, v := range arr { print i; print v }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"0", "10", "1", "20", "2", "30"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestRangeNoVars(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`for range [1, 2, 3] { print "tick" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
}

func TestRangeWithBreak(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`for i, v := range [10, 20, 30, 40, 50] { if v == 30 { break }; print v }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"10", "20"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
}

func TestRangeWithContinue(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`for i, v := range [10, 20, 30, 40, 50] { if i == 2 { continue }; print v }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"10", "20", "40", "50"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestRangeEmptyArray(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`for i, v := range [] { print i }; print "done"`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "done" {
		t.Errorf("expected done, got %q", strings.TrimSpace(stdout))
	}
}

func TestRangeStringArray(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`for i, v := range ["a", "b", "c"] { print v }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"a", "b", "c"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
}

func TestIterRangeSingleVar(t *testing.T) {
	env := NewEmptyEnvironment()
	input := `func countTo(n) { return func(yield) { i := 0; for i < n { if !yield(i) { return }; i = i + 1 } } }; for v := range countTo(5) { print v }`
	stdout, stderr, _ := runKami(input, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"0", "1", "2", "3", "4"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestIterRangeDualVar(t *testing.T) {
	env := NewEmptyEnvironment()
	input := `func enumerate(arr) { return func(yield) { for i := range arr { if !yield(i, arr[i]) { return } } } }; for k, v := range enumerate([10, 20, 30]) { print k; print v }`
	stdout, stderr, _ := runKami(input, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"0", "10", "1", "20", "2", "30"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestIterRangeBreak(t *testing.T) {
	env := NewEmptyEnvironment()
	input := `func countTo(n) { return func(yield) { i := 0; for i < n { if !yield(i) { return }; i = i + 1 } } }; for v := range countTo(100) { if v == 5 { break }; print v }`
	stdout, stderr, _ := runKami(input, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"0", "1", "2", "3", "4"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
}

func TestIterRangeContinue(t *testing.T) {
	env := NewEmptyEnvironment()
	input := `func countTo(n) { return func(yield) { i := 0; for i < n { if !yield(i) { return }; i = i + 1 } } }; for v := range countTo(5) { if v == 2 { continue }; print v }`
	stdout, stderr, _ := runKami(input, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"0", "1", "3", "4"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestIterRangeEmpty(t *testing.T) {
	env := NewEmptyEnvironment()
	input := `func empty() { return func(yield) { } }; for v := range empty() { print v }; print "done"`
	stdout, stderr, _ := runKami(input, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "done" {
		t.Errorf("expected done, got %q", strings.TrimSpace(stdout))
	}
}

func TestIterRangeWithReturn(t *testing.T) {
	env := NewEmptyEnvironment()
	input := `func countTo(n) { return func(yield) { i := 0; for i < n { if !yield(i) { return }; i = i + 1 } } }; func find(target) { for v := range countTo(10) { if v == target { return v } }; return -1 }; print find(7)`
	stdout, stderr, _ := runKami(input, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "7" {
		t.Errorf("expected 7, got %q", strings.TrimSpace(stdout))
	}
}

func TestIterRangeNested(t *testing.T) {
	env := NewEmptyEnvironment()
	input := `func countTo(n) { return func(yield) { i := 0; for i < n { if !yield(i) { return }; i = i + 1 } } }; for v := range countTo(3) { for w := range countTo(2) { print v; print w } }`
	stdout, stderr, _ := runKami(input, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := []string{"0", "0", "0", "1", "1", "0", "1", "1", "2", "0", "2", "1"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
	for i, val := range expected {
		if strings.TrimSpace(lines[i]) != val {
			t.Errorf("at line %d: expected %s, got %s", i, val, lines[i])
		}
	}
}

func TestForBuiltins(t *testing.T) {
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
	stdout, stderr, _ := runKami("x := nil", env)
	if !strings.Contains(stderr, "untyped nil cannot be used with :=") {
		t.Errorf("expected error about untyped nil, got stderr: %q", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty stdout, got %q", stdout)
	}
}

func TestVarNilDoesNotFreezeNullType(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("var x = nil", env)
	if !strings.Contains(stderr, "invalid var statement") {
		t.Errorf("expected error about invalid var statement, got stderr: %q", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty stdout, got %q", stdout)
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

func TestPointerAssignComplexExpression(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("x := 10; p := &x; *p = *p + 5; print x", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "15" {
		t.Errorf("expected 15, got %q", stdout)
	}
}

func TestPointerAssignWithFunction(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("func inc(p) { *p = *p + 1 }; x := 0; p := &x; inc(p); print x", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "1" {
		t.Errorf("expected 1, got %q", stdout)
	}
}

func TestSwitchBasicInt(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`x := 2; switch x { case 1: print "one" case 2: print "two" case 3: print "three" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "two" {
		t.Errorf("expected two, got %q", stdout)
	}
}

func TestSwitchMultipleValues(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`x := 4; switch x { case 1: print "one" case 2, 3, 4: print "two-four" case 5: print "five" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "two-four" {
		t.Errorf("expected two-four, got %q", stdout)
	}
}

func TestSwitchDefault(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`x := 99; switch x { case 1: print "one" default: print "other" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "other" {
		t.Errorf("expected other, got %q", stdout)
	}
}

func TestSwitchNoMatchNoDefault(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`x := 99; switch x { case 1: print "one" case 2: print "two" }; print "done"`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "done" {
		t.Errorf("expected done, got %q", stdout)
	}
}

func TestSwitchString(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`name := "b"; switch name { case "a": print "alpha" case "b": print "beta" case "c": print "charlie" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "beta" {
		t.Errorf("expected beta, got %q", stdout)
	}
}

func TestSwitchTagless(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`x := 10; switch { case x > 100: print "big" case x > 5: print "medium" case x > 0: print "small" default: print "zero" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "medium" {
		t.Errorf("expected medium, got %q", stdout)
	}
}

func TestSwitchNested(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`x := 1; y := 2; switch x { case 1: switch y { case 1: print "1-1" case 2: print "1-2" } case 2: print "2" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "1-2" {
		t.Errorf("expected 1-2, got %q", stdout)
	}
}

func TestSwitchWithReturn(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`func pick(x) { switch x { case 1: return "one" case 2: return "two" default: return "other" } }; print pick(2)`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "two" {
		t.Errorf("expected two, got %q", stdout)
	}
}

func TestSwitchDefaultFirst(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`x := 5; switch x { default: print "default" case 1: print "one" case 5: print "five" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "five" {
		t.Errorf("expected five (case should take priority over default position), got %q", stdout)
	}
}

func TestSwitchLargeIntBinarySearch(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`x := 15; switch x { case 1: print "1" case 3: print "3" case 5: print "5" case 7: print "7" case 9: print "9" case 11: print "11" case 13: print "13" case 15: print "15" case 17: print "17" case 19: print "19" default: print "?" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "15" {
		t.Errorf("expected 15, got %q", stdout)
	}
}

func TestSwitchLargeIntDefault(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`x := 99; switch x { case 1: print "1" case 2: print "2" case 3: print "3" case 4: print "4" case 5: print "5" case 6: print "6" case 7: print "7" case 8: print "8" case 9: print "9" case 10: print "10" default: print "default" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "default" {
		t.Errorf("expected default, got %q", stdout)
	}
}

func TestSwitchLargeStringMatch(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`x := "hello"; switch x { case "alpha": print "a" case "beta": print "b" case "hello": print "matched" case "world": print "w" default: print "?" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "matched" {
		t.Errorf("expected matched, got %q", stdout)
	}
}

func TestSwitchLargeStringDefault(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`x := "nope"; switch x { case "alpha": print "a" case "beta": print "b" case "hello": print "h" case "world": print "w" default: print "default" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "default" {
		t.Errorf("expected default, got %q", stdout)
	}
}

func TestSwitchIntFirstCase(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`x := 1; switch x { case 1: print "first" case 2: print "second" case 3: print "third" case 4: print "fourth" case 5: print "fifth" default: print "?" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "first" {
		t.Errorf("expected first, got %q", stdout)
	}
}

func TestSwitchIntLastCase(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`x := 5; switch x { case 1: print "first" case 2: print "second" case 3: print "third" case 4: print "fourth" case 5: print "fifth" default: print "?" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "fifth" {
		t.Errorf("expected fifth, got %q", stdout)
	}
}

func TestSwitchIntMultipleValues(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami(`x := 3; switch x { case 1, 2, 3: print "matched" case 4, 5, 6: print "other" default: print "?" }`, env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "matched" {
		t.Errorf("expected matched, got %q", stdout)
	}
}
