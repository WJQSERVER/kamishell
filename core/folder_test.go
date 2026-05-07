package core

import (
	"testing"
)

func TestFoldIntAddition(t *testing.T) {
	stdout, stderr, _ := runKami(`print 3 + 4`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "7" {
		t.Errorf("expected 7, got %s", trim(stdout))
	}
}

func TestFoldIntMultiplication(t *testing.T) {
	stdout, stderr, _ := runKami(`print 3 * 4 + 2`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "14" {
		t.Errorf("expected 14, got %s", trim(stdout))
	}
}

func TestFoldIntPrecedence(t *testing.T) {
	// 3 + 4 * 2 = 3 + 8 = 11 (not 14)
	stdout, stderr, _ := runKami(`print 3 + 4 * 2`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "11" {
		t.Errorf("expected 11, got %s", trim(stdout))
	}
}

func TestFoldIntSubtraction(t *testing.T) {
	stdout, stderr, _ := runKami(`print 10 - 3`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "7" {
		t.Errorf("expected 7, got %s", trim(stdout))
	}
}

func TestFoldIntDivision(t *testing.T) {
	stdout, stderr, _ := runKami(`print 20 / 4`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "5" {
		t.Errorf("expected 5, got %s", trim(stdout))
	}
}

func TestFoldIntModulo(t *testing.T) {
	stdout, stderr, _ := runKami(`print 17 % 5`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "2" {
		t.Errorf("expected 2, got %s", trim(stdout))
	}
}

func TestFoldFloatArithmetic(t *testing.T) {
	stdout, stderr, _ := runKami(`print 3.14 * 2.0`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "6.28" {
		t.Errorf("expected 6.28, got %s", trim(stdout))
	}
}

func TestFoldMixedIntFloat(t *testing.T) {
	stdout, stderr, _ := runKami(`print 3 + 4.5`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "7.5" {
		t.Errorf("expected 7.5, got %s", trim(stdout))
	}
}

func TestFoldStringConcatenation(t *testing.T) {
	stdout, stderr, _ := runKami(`print "hello " + "world"`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "hello world" {
		t.Errorf("expected 'hello world', got %s", trim(stdout))
	}
}

func TestFoldStringComparison(t *testing.T) {
	stdout, stderr, _ := runKami(`print "abc" == "abc"`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "true" {
		t.Errorf("expected true, got %s", trim(stdout))
	}
}

func TestFoldBooleanNot(t *testing.T) {
	stdout, stderr, _ := runKami(`print !true`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "false" {
		t.Errorf("expected false, got %s", trim(stdout))
	}
}

func TestFoldBooleanDoubleNot(t *testing.T) {
	stdout, stderr, _ := runKami(`print !!true`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "true" {
		t.Errorf("expected true, got %s", trim(stdout))
	}
}

func TestFoldIntComparison(t *testing.T) {
	stdout, stderr, _ := runKami(`print 3 > 2`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "true" {
		t.Errorf("expected true, got %s", trim(stdout))
	}
}

func TestFoldIntEquality(t *testing.T) {
	stdout, stderr, _ := runKami(`print 5 == 5`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "true" {
		t.Errorf("expected true, got %s", trim(stdout))
	}
}

func TestFoldIntNegation(t *testing.T) {
	stdout, stderr, _ := runKami(`print -5`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "-5" {
		t.Errorf("expected -5, got %s", trim(stdout))
	}
}

func TestFoldFloatNegation(t *testing.T) {
	stdout, stderr, _ := runKami(`print -3.14`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "-3.14" {
		t.Errorf("expected -3.14, got %s", trim(stdout))
	}
}

func TestFoldNilEquality(t *testing.T) {
	stdout, stderr, _ := runKami(`print nil == nil`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "true" {
		t.Errorf("expected true, got %s", trim(stdout))
	}
}

func TestFoldDivisionByZeroNotCrashing(t *testing.T) {
	// Division by zero should produce an error at runtime, not crash
	_, stderr, _ := runKami(`print 1 / 0`, NewEmptyEnvironment())
	if stderr == "" {
		t.Error("expected division by zero error")
	}
}

func TestFoldNestedExpression(t *testing.T) {
	// (3 + 4) * (2 + 5) = 7 * 7 = 49
	stdout, stderr, _ := runKami(`print (3 + 4) * (2 + 5)`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "49" {
		t.Errorf("expected 49, got %s", trim(stdout))
	}
}

func TestFoldInAssignment(t *testing.T) {
	stdout, stderr, _ := runKami(`x := 3 + 4; print x`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "7" {
		t.Errorf("expected 7, got %s", trim(stdout))
	}
}

func TestFoldInCondition(t *testing.T) {
	stdout, stderr, _ := runKami(`if 3 > 2 { print "yes" }`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "yes" {
		t.Errorf("expected 'yes', got %s", trim(stdout))
	}
}

func TestFoldWithVariableNotFolded(t *testing.T) {
	// Variables should NOT be folded — only pure literals
	stdout, stderr, _ := runKami(`x := 10; print x + 5`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "15" {
		t.Errorf("expected 15, got %s", trim(stdout))
	}
}

func TestFoldBooleanLogicChain(t *testing.T) {
	stdout, stderr, _ := runKami(`print (3 > 2) == (5 < 10)`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "true" {
		t.Errorf("expected true, got %s", trim(stdout))
	}
}

func TestFoldFloatDivision(t *testing.T) {
	stdout, stderr, _ := runKami(`print 10.0 / 4.0`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "2.5" {
		t.Errorf("expected 2.5, got %s", trim(stdout))
	}
}

func TestFoldLargeExpression(t *testing.T) {
	// 1 + 2 * 3 - 4 / 2 + 5 % 3 = 1 + 6 - 2 + 2 = 7
	stdout, stderr, _ := runKami(`print 1 + 2 * 3 - 4 / 2 + 5 % 3`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "7" {
		t.Errorf("expected 7, got %s", trim(stdout))
	}
}

func TestFoldStringMultiConcat(t *testing.T) {
	stdout, stderr, _ := runKami(`print "a" + "b" + "c"`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "abc" {
		t.Errorf("expected 'abc', got %s", trim(stdout))
	}
}

func TestFoldDoesNotAffectVariables(t *testing.T) {
	// Ensure folding doesn't break variable-based expressions
	stdout, stderr, _ := runKami(`a := 3; b := 4; print a * b + 1`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "13" {
		t.Errorf("expected 13, got %s", trim(stdout))
	}
}

func TestFoldSwitchCases(t *testing.T) {
	// Constants in switch cases should still work
	stdout, stderr, _ := runKami(`x := 2; switch x { case 1: print "one" case 2: print "two" }`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "two" {
		t.Errorf("expected 'two', got %s", trim(stdout))
	}
}
