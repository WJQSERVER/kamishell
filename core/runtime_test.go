package core

import (
	"io"
	"strings"
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

func TestCallExpressionWithMemberAccess(t *testing.T) {
	input := `env.Set("GOOS", "linux"); env.Get("GOOS")`
	evaluated := testEval(input)
	if evaluated.Inspect() != "linux" {
		t.Errorf("expected linux, got %s", evaluated.Inspect())
	}
}

func TestEvalStringEqualityDoesNotRequireStringificationFallback(t *testing.T) {
	result := evalInfixExpression("==", &String{Value: "kami"}, &String{Value: "kami"})
	if result != TRUE {
		t.Fatalf("expected TRUE, got %s", result.Inspect())
	}

	result = evalInfixExpression("!=", &String{Value: "kami"}, &String{Value: "shell"})
	if result != TRUE {
		t.Fatalf("expected TRUE, got %s", result.Inspect())
	}
}

func TestEvalBooleanEqualityUsesTypedComparison(t *testing.T) {
	result := evalInfixExpression("==", TRUE, TRUE)
	if result != TRUE {
		t.Fatalf("expected TRUE, got %s", result.Inspect())
	}

	result = evalInfixExpression("!=", TRUE, FALSE)
	if result != TRUE {
		t.Fatalf("expected TRUE, got %s", result.Inspect())
	}
}

func TestExecuteCommandRunsUserFunctionWithoutStringifyingArguments(t *testing.T) {
	env := NewEmptyEnvironment()
	fn := &Function{
		Parameters: []string{"value"},
		Body: &BlockStatement{Statements: []Statement{
			&ExpressionStatement{Expression: &Identifier{Value: "value"}},
		}},
		Env: env,
	}
	env.Set("identity", fn)

	result := executeCommand("identity", []Expression{
		&IntegerLiteral{Value: 7, Obj: getIntegerObject(7)},
	}, env, strings.NewReader(""), io.Discard, io.Discard)

	str, ok := result.(*String)
	if !ok {
		t.Fatalf("expected string result, got %T", result)
	}
	if str.Value != "7" {
		t.Fatalf("expected 7, got %q", str.Value)
	}
}

func TestRepeatedEvalReusesParsedFunctionStatementWithCurrentEnv(t *testing.T) {
	program := func() *Program {
		l := NewLexer("prefix := \"first\"; func greet() { print prefix }; greet()")
		p := NewParser(l)
		return p.ParseProgram()
	}()

	env1 := NewEmptyEnvironment()
	stdout1 := &strings.Builder{}
	result1 := EvalWithIO(program, env1, strings.NewReader(""), stdout1, io.Discard)
	if isError(result1) {
		t.Fatalf("first eval returned error: %s", result1.Inspect())
	}
	if strings.TrimSpace(stdout1.String()) != "first" {
		t.Fatalf("expected first eval to print first, got %q", stdout1.String())
	}

	assign := program.Statements[0].(*AssignStatement)
	assign.Value = &StringLiteral{Value: "second", Obj: &String{Value: "second"}}

	env2 := NewEmptyEnvironment()
	stdout2 := &strings.Builder{}
	result2 := EvalWithIO(program, env2, strings.NewReader(""), stdout2, io.Discard)
	if isError(result2) {
		t.Fatalf("second eval returned error: %s", result2.Inspect())
	}
	if strings.TrimSpace(stdout2.String()) != "second" {
		t.Fatalf("expected second eval to print second, got %q", stdout2.String())
	}
}
