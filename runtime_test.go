package kamishell

import (
	"testing"
)

func TestEvalIntegerExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"5", 5},
		{"10", 10},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func testEval(input string) Object {
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()
	env := NewEnvironment()

	return Eval(program, env)
}

func testIntegerObject(t *testing.T, obj Object, expected int64) bool {
	result, ok := obj.(*Integer)
	if !ok {
		t.Errorf("object is not Integer. got=%T (%+v)", obj, obj)
		return false
	}
	if result.Value != expected {
		t.Errorf("object has wrong value. expect=%d, got=%d", expected, result.Value)
		return false
	}
	return true
}

func TestIfElseExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"if true { 10 }", "10"},
		{"if false { 10 } else { 20 }", "20"},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		if evaluated.Inspect() != tt.expected {
			t.Errorf("expected=%s, got=%s", tt.expected, evaluated.Inspect())
		}
	}
}

func TestExecStatement(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`exec "ls -la"`},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		if evaluated != NULL && isError(evaluated) {
			t.Errorf("exec failed for input %s: %s", tt.input, evaluated.Inspect())
		}
	}
}

func TestInterpolation(t *testing.T) {
	input := `x := "world"; print $x`
	// Since testEval returns the result of the last statement, and print returns NULL
	evaluated := testEval(input)
	if evaluated != NULL {
		t.Errorf("expected NULL from print, got %v", evaluated)
	}
}
