package core

import (
	"io"
	"strings"
	"sync"
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

// --- Exec Keyword Form Runtime Tests (target behavior) ---
// These tests constrain the expected behavior of the new exec implementation.
// They should FAIL until the implementation is done.

// 关键字形式：基本裸词执行
func TestExecBareWordBasic(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`exec echo hello`},
		{`exec ls -la`},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		if evaluated != NULL && isError(evaluated) {
			t.Errorf("exec failed for input %s: %s", tt.input, evaluated.Inspect())
		}
	}
}

// 关键字形式：带引号的参数
func TestExecBareWordWithQuotes(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`exec echo "my document.txt"`},
		{`exec echo 'my document.txt'`},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		if evaluated != NULL && isError(evaluated) {
			t.Errorf("exec failed for input %s: %s", tt.input, evaluated.Inspect())
		}
	}
}

// 关键字形式：带变量插值
func TestExecBareWordWithVariable(t *testing.T) {
	input := `x := "hello"; exec echo $x`
	evaluated := testEval(input)
	if evaluated != NULL && isError(evaluated) {
		t.Errorf("exec failed for input %s: %s", input, evaluated.Inspect())
	}
}

// 关键字形式：URL 正确处理
func TestExecBareWordURL(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`exec curl http://localhost:8080`},
		{`exec wget https://example.com/file.txt`},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		// These should not error (even if the command fails)
		if isError(evaluated) {
			err, ok := evaluated.(*Error)
			if ok && err.Message == "exec expects a string" {
				t.Errorf("exec should not require string argument for bare word form: %s", tt.input)
			}
		}
	}
}

// 关键字形式：无参数返回 NULL
func TestExecBareWordNoArgs(t *testing.T) {
	input := `exec`
	evaluated := testEval(input)
	if evaluated != NULL {
		t.Errorf("expected NULL for exec with no args, got %v", evaluated)
	}
}

// --- Exec Function Form Runtime Tests (target behavior) ---
// These tests constrain the expected behavior of the exec() function.
// They should FAIL until the implementation is done.

// 函数形式：基本字符串执行
func TestExecFunctionBasic(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`exec("echo hello")`},
		{`exec("ls -la")`},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		if evaluated != NULL && isError(evaluated) {
			t.Errorf("exec failed for input %s: %s", tt.input, evaluated.Inspect())
		}
	}
}

// 函数形式：带引号的参数
func TestExecFunctionWithQuotes(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`exec("echo \"my document.txt\"")`},
		{`exec("echo 'my document.txt'")`},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		if evaluated != NULL && isError(evaluated) {
			t.Errorf("exec failed for input %s: %s", tt.input, evaluated.Inspect())
		}
	}
}

// 函数形式：带变量
func TestExecFunctionWithVariable(t *testing.T) {
	input := `cmd := "echo hello"; exec(cmd)`
	evaluated := testEval(input)
	if evaluated != NULL && isError(evaluated) {
		t.Errorf("exec failed for input %s: %s", input, evaluated.Inspect())
	}
}

// 函数形式：空字符串报错
func TestExecFunctionEmptyString(t *testing.T) {
	input := `exec("")`
	evaluated := testEval(input)
	if !isError(evaluated) {
		t.Errorf("expected error for exec with empty string, got %v", evaluated)
	}
}

// 函数形式：非字符串参数报错
func TestExecFunctionNonStringArg(t *testing.T) {
	input := `exec(123)`
	evaluated := testEval(input)
	if !isError(evaluated) {
		t.Errorf("expected error for exec with non-string arg, got %v", evaluated)
	}
	err, ok := evaluated.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", evaluated)
	}
	if !strings.Contains(err.Message, "string") {
		t.Errorf("expected error message to contain 'string', got %q", err.Message)
	}
}

// 函数形式：nil 参数报错
func TestExecFunctionNilArg(t *testing.T) {
	input := `exec(nil)`
	evaluated := testEval(input)
	if !isError(evaluated) {
		t.Errorf("expected error for exec with nil arg, got %v", evaluated)
	}
	err, ok := evaluated.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", evaluated)
	}
	if !strings.Contains(err.Message, "string") {
		t.Errorf("expected error message to contain 'string', got %q", err.Message)
	}
}

// 函数形式：参数数量错误报错
func TestExecFunctionWrongArgCount(t *testing.T) {
	input := `exec("echo", "hello")`
	evaluated := testEval(input)
	if !isError(evaluated) {
		t.Errorf("expected error for exec with wrong arg count, got %v", evaluated)
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
		Parameters: []Parameter{{Name: "value", TypeName: "any"}},
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

func TestConcurrentEvalOfSharedProgramMutatesFunctionStatementCache(t *testing.T) {
	program := func() *Program {
		l := NewLexer("func greet() { 1 }; greet()")
		p := NewParser(l)
		return p.ParseProgram()
	}()

	const workers = 16
	start := make(chan struct{})
	var wg sync.WaitGroup
	errCh := make(chan string, workers)

	for range workers {
		wg.Go(func() {
			<-start

			env := NewEmptyEnvironment()
			result := EvalWithIO(program, env, strings.NewReader(""), io.Discard, io.Discard)
			if isError(result) {
				errCh <- result.Inspect()
			}
		})
	}

	close(start)
	wg.Wait()
	close(errCh)

	for errMsg := range errCh {
		t.Fatalf("unexpected eval error: %s", errMsg)
	}
}

func TestEvalFloatExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"3.14", 3.14},
		{"0.5", 0.5},
		{".5", 0.5},
		{"123.456", 123.456},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testFloatObject(t, evaluated, tt.expected)
	}
}

func testFloatObject(t *testing.T, obj Object, expected float64) bool {
	result, ok := obj.(*Float)
	if !ok {
		t.Errorf("object is not Float. got=%T (%+v)", obj, obj)
		return false
	}
	delta := result.Value - expected
	if delta < 0 {
		delta = -delta
	}
	if delta > 0.0001 {
		t.Errorf("object has wrong value. expect=%f, got=%f", expected, result.Value)
		return false
	}
	return true
}

func TestFloatArithmetic(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"3.14 + 1.0", 4.14},
		{"2.5 + 2.5", 5.0},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testFloatObject(t, evaluated, tt.expected)
	}
}

func TestFloatIntegerMixedArithmetic(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"3.14 + 1", 4.14},
		{"10 + 0.5", 10.5},
		{"2.5 + 2", 4.5},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testFloatObject(t, evaluated, tt.expected)
	}
}

func TestFloatComparison(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"3.14 == 3.14", true},
		{"3.14 != 2.71", true},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func testBooleanObject(t *testing.T, obj Object, expected bool) bool {
	result, ok := obj.(*Boolean)
	if !ok {
		t.Errorf("object is not Boolean. got=%T (%+v)", obj, obj)
		return false
	}
	if result.Value != expected {
		t.Errorf("object has wrong value. expect=%t, got=%t", expected, result.Value)
		return false
	}
	return true
}

// --- NilLiteral ---

func TestNilLiteral(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("print nil", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "nil" {
		t.Errorf("expected 'nil', got %q", stdout)
	}
}

// --- Division by zero ---

func TestIntegerDivisionByZero(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, result := runKami("x := 10 / 0; print x", env)
	if !isError(result) && !strings.Contains(stderr, "division by zero") {
		t.Errorf("expected error for division by zero, got stderr=%q result=%s", stderr, result.Inspect())
	}
}

func TestFloatDivisionByZero(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, result := runKami("x := 10.0 / 0.0; print x", env)
	if !isError(result) && !strings.Contains(stderr, "division by zero") {
		t.Errorf("expected error for float division by zero, got stderr=%q result=%s", stderr, result.Inspect())
	}
}

func TestModuloByZero(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, result := runKami("x := 10 % 0; print x", env)
	if !isError(result) && !strings.Contains(stderr, "division by zero") {
		t.Errorf("expected error for modulo by zero, got stderr=%q result=%s", stderr, result.Inspect())
	}
}

// --- Float edge cases ---

func TestFloatEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"print 0.0", "0"},
	}

	for _, tt := range tests {
		env := NewEmptyEnvironment()
		stdout, stderr, _ := runKami(tt.input, env)
		if stderr != "" {
			t.Errorf("input %q: unexpected stderr: %s", tt.input, stderr)
		}
		if strings.TrimSpace(stdout) != tt.expected {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.expected, strings.TrimSpace(stdout))
		}
	}
}

func TestFloatComparisonEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"print 0.0 == 0.0", "true"},
		{"print 0.0 != 1.0", "true"},
		{"print 1.0 > 0.0", "true"},
		{"print 0.0 < 1.0", "true"},
		{"print 1.0 >= 1.0", "true"},
		{"print 1.0 <= 1.0", "true"},
	}

	for _, tt := range tests {
		env := NewEmptyEnvironment()
		stdout, stderr, _ := runKami(tt.input, env)
		if stderr != "" {
			t.Errorf("input %q: unexpected stderr: %s", tt.input, stderr)
		}
		if strings.TrimSpace(stdout) != tt.expected {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.expected, strings.TrimSpace(stdout))
		}
	}
}

// --- Negation ---

func TestPrefixNegation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"print -5", "-5"},
		{"print --5", "5"},
		{"print -0", "0"},
	}

	for _, tt := range tests {
		env := NewEmptyEnvironment()
		stdout, stderr, _ := runKami(tt.input, env)
		if stderr != "" {
			t.Errorf("input %q: unexpected stderr: %s", tt.input, stderr)
		}
		if strings.TrimSpace(stdout) != tt.expected {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.expected, strings.TrimSpace(stdout))
		}
	}
}

func TestPrefixNegationFloat(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"print -3.14", "-3.14"},
		{"print --3.14", "3.14"},
	}

	for _, tt := range tests {
		env := NewEmptyEnvironment()
		stdout, stderr, _ := runKami(tt.input, env)
		if stderr != "" {
			t.Errorf("input %q: unexpected stderr: %s", tt.input, stderr)
		}
		if strings.TrimSpace(stdout) != tt.expected {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.expected, strings.TrimSpace(stdout))
		}
	}
}

func TestPrefixNegationNonNumeric(t *testing.T) {
	env := NewEmptyEnvironment()
	_, stderr, result := runKami(`print -"hello"`, env)
	if !isError(result) && !strings.Contains(stderr, "cannot negate") {
		t.Errorf("expected error for negating string, got stderr=%q result=%s", stderr, result.Inspect())
	}
}

// --- Modulo ---

func TestModuloInteger(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"print 17 % 5", "2"},
		{"print 10 % 3", "1"},
		{"print 0 % 5", "0"},
		{"print -7 % 3", "-1"},
	}

	for _, tt := range tests {
		env := NewEmptyEnvironment()
		stdout, stderr, _ := runKami(tt.input, env)
		if stderr != "" {
			t.Errorf("input %q: unexpected stderr: %s", tt.input, stderr)
		}
		if strings.TrimSpace(stdout) != tt.expected {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.expected, strings.TrimSpace(stdout))
		}
	}
}

func TestModuloFloat(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("print 17.5 % 5.0", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "2.5" {
		t.Errorf("expected '2.5', got %q", strings.TrimSpace(stdout))
	}
}

// --- >= and <= operators ---

func TestGeqLeqInteger(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"5 >= 3", true},
		{"3 >= 5", false},
		{"3 >= 3", true},
		{"5 <= 3", false},
		{"3 <= 5", true},
		{"3 <= 3", true},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func TestGeqLeqString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`"abc" >= "abc"`, true},
		{`"abc" >= "abd"`, false},
		{`"abd" >= "abc"`, true},
		{`"abc" <= "abc"`, true},
		{`"abc" <= "abd"`, true},
		{`"abd" <= "abc"`, false},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

// --- Break/Continue ---

func TestBreakInLoop(t *testing.T) {
	input := `
x := 0
for x < 10 {
    x = x + 1
    if x == 5 {
        break
    }
}
print x
`
	env := NewEmptyEnvironment()
	stdout := &strings.Builder{}
	EvalWithIO(parse(input), env, strings.NewReader(""), stdout, io.Discard)
	if strings.TrimSpace(stdout.String()) != "5" {
		t.Errorf("expected 5, got %q", stdout.String())
	}
}

func TestContinueInLoop(t *testing.T) {
	input := `
sum := 0
for i := 0; i < 10; i = i + 1 {
    if i % 2 == 0 {
        continue
    }
    sum = sum + i
}
print sum
`
	env := NewEmptyEnvironment()
	stdout := &strings.Builder{}
	EvalWithIO(parse(input), env, strings.NewReader(""), stdout, io.Discard)
	if strings.TrimSpace(stdout.String()) != "25" {
		t.Errorf("expected 25, got %q", stdout.String())
	}
}

func parse(input string) *Program {
	l := NewLexer(input)
	p := NewParser(l)
	return p.ParseProgram()
}

// --- ImportStatement ---

func TestImportGoFmt(t *testing.T) {
	input := `
import "Go/fmt"
x := fmt.Sprintf("hello %s", "world")
print x
`
	env := NewEmptyEnvironment()
	stdout := &strings.Builder{}
	result := EvalWithIO(parse(input), env, strings.NewReader(""), stdout, io.Discard)
	if isError(result) {
		t.Fatalf("import failed: %s", result.Inspect())
	}
	if strings.TrimSpace(stdout.String()) != "hello world" {
		t.Errorf("expected 'hello world', got %q", stdout.String())
	}
}

// --- WaitStatement ---

func TestWaitAllWithNoJobs(t *testing.T) {
	input := `wait`
	evaluated := testEval(input)
	if evaluated != NULL && !isError(evaluated) {
		t.Errorf("expected NULL or error, got %T (%s)", evaluated, evaluated.Inspect())
	}
}

// --- GoExpression ---

func TestGoExpressionBasic(t *testing.T) {
	// Go clones the env, so changes inside don't affect parent
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("x := 1; go { x = 2 }; wait; print x", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	// x stays 1 because goroutine operates on cloned env
	if strings.TrimSpace(stdout) != "1" {
		t.Errorf("expected 1, got %q", strings.TrimSpace(stdout))
	}
}

// --- String comparison operators ---

func TestStringComparison(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`"a" < "b"`, true},
		{`"b" < "a"`, false},
		{`"a" > "b"`, false},
		{`"b" > "a"`, true},
		{`"a" == "a"`, true},
		{`"a" != "b"`, true},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

// --- Grouped expression as statement ---

func TestGroupedExpressionStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"(5 + 3)", 8},
		{"(10 - 2)", 8},
		{"(2 * 3)", 6},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestPrintGroupedExpression(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("print (5 + 3)", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "8" {
		t.Errorf("expected '8', got %q", strings.TrimSpace(stdout))
	}
}

func TestIfGroupedCondition(t *testing.T) {
	env := NewEmptyEnvironment()
	stdout, stderr, _ := runKami("x := 10; if (x > 5) { print \"big\" }", env)
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if strings.TrimSpace(stdout) != "big" {
		t.Errorf("expected 'big', got %q", strings.TrimSpace(stdout))
	}
}

func TestPrefixNotStatement(t *testing.T) {
	evaluated := testEval("(!false)")
	testBooleanObject(t, evaluated, true)
}

func TestPrefixNegationStatement(t *testing.T) {
	evaluated := testEval("(-5)")
	testIntegerObject(t, evaluated, -5)
}
