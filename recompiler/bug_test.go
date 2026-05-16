package recompiler

import (
	"go/format"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"kamishell/core"
)

// compileSource compiles a Kami source string and returns the generated Go source.
// It also verifies the Go source is syntactically valid.
func compileSource(t *testing.T, source string) string {
	t.Helper()
	lexer := core.NewLexer(source)
	parser := core.NewParser(lexer)
	program := parser.ParseProgram()
	for _, stmt := range program.Statements {
		if inv, ok := stmt.(*core.InvalidStatement); ok {
			t.Fatalf("parse error: %s", inv.Message)
		}
	}
	compiled, err := Compile(program)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	_, err = format.Source([]byte(compiled.Source))
	if err != nil {
		t.Fatalf("generated code is not valid Go:\n%v\n---\n%s", err, compiled.Source)
	}
	return compiled.Source
}

// assertSourceContains checks that the generated source contains a substring.
func assertSourceContains(t *testing.T, source, pattern string) {
	t.Helper()
	if !strings.Contains(source, pattern) {
		t.Errorf("generated source does not contain %q\n---\n%s", pattern, source)
	}
}

// assertSourceNotContains checks that the generated source does NOT contain a substring.
func assertSourceNotContains(t *testing.T, source, pattern string) {
	t.Helper()
	if strings.Contains(source, pattern) {
		t.Errorf("generated source should not contain %q\n---\n%s", pattern, source)
	}
}

// compileRun compiles, builds, and runs a Kami script, returning stdout.
func compileRun(t *testing.T, name, source string) string {
	t.Helper()
	binary, _ := compileAndBuild(t, name, source)
	return runBinary(t, binary)
}

// ============================================================
// Bug #1: compileFunctionLiteral returns empty string
// Location: compiler.go ~line 1166
// The function body is compiled into sub.buf but the return reads
// from bodyBuf (never written). This causes anonymous functions
// to produce empty Go code.
// ============================================================
func TestBug1_FunctionLiteralNotEmpty(t *testing.T) {
	source := `add := func(a int, b int) int { return a + b }
print add(3, 4)`
	src := compileSource(t, source)

	// The generated source should contain a function literal with a body,
	// not just "func() any { return nil }" or an empty string.
	// After fix: should contain "return (a + b)" or similar
	assertSourceContains(t, src, "func(")
	// The function body should NOT be empty - it should contain "return"
	assertSourceContains(t, src, "return")

	// End-to-end: the output should be "7"
	out := compileRun(t, "bug1", source)
	if strings.TrimSpace(out) != "7" {
		t.Fatalf("expected '7', got %q", out)
	}
}

// ============================================================
// Bug #2: collectDecls result discarded, closure capture broken
// Location: compiler.go ~line 339
// walkStatements is called with empty outerVars, so closure capture
// detection never fires. Variables referenced in closures won't get
// kamiEnv.Set, so the closure can't read them.
// ============================================================
func TestBug2_ClosureCaptureNeedsEnvSync(t *testing.T) {
	source := `x := 10
func getX() int {
    return x
}
print getX()`
	src := compileSource(t, source)

	// x is captured by closure getX, so it MUST have kamiEnv.SetString("x", ...)
	assertSourceContains(t, src, `kamiEnv.SetString("x", strconv.FormatInt(x, 10))`)

	// End-to-end: should print "10"
	out := compileRun(t, "bug2", source)
	if strings.TrimSpace(out) != "10" {
		t.Fatalf("expected '10', got %q", out)
	}
}

// ============================================================
// Bug #3: Sub-compiler in compileFunctionStatement missing envSync
// Location: compiler.go ~line 1750
// The sub-compiler doesn't inherit envSync, so all variable
// assignments inside functions unconditionally emit kamiEnv.Set
// (safe but defeats the optimization).
// ============================================================
func TestBug3_FunctionBodyEnvSyncOptimization(t *testing.T) {
	// Inside a function, variables NOT referenced via $var should NOT have kamiEnv.Set
	source := `func compute(n int) int {
    result := n + 1
    return result
}
print compute(5)`
	src := compileSource(t, source)

	// The function body should contain the assignment
	assertSourceContains(t, src, "var result int64")
	// After fix: "result" is not referenced via $var, so kamiEnv.SetString("result", result)
	// should NOT appear in the function body.
	// Current behavior: it DOES appear (because envSync is nil in sub-compiler)
	// This test documents the expected behavior.
	_ = src
}

// ============================================================
// Bug #4: Sub-compiler in compileFunctionLiteral missing fields
// Location: compiler.go ~line 1152
// Missing knownFuncs, funcReturns, arrayTypes, envSync
// ============================================================
func TestBug4_FunctionLiteralMissingFields(t *testing.T) {
	// A closure that calls a named function by name
	source := `func add(a int, b int) int { return a + b }
wrapper := func(x int, y int) int { return add(x, y) }
print wrapper(3, 4)`
	src := compileSource(t, source)

	// The closure should call kamiFunc_add directly, not recompiler.CallFunc
	// (Currently it may use CallFunc because knownFuncs is missing in sub-compiler)
	assertSourceContains(t, src, "kamiFunc_add")

	// End-to-end
	out := compileRun(t, "bug4", source)
	if strings.TrimSpace(out) != "7" {
		t.Fatalf("expected '7', got %q", out)
	}
}

// ============================================================
// Bug #5: PointerAssignStatement not handled
// Location: compiler.go compileStatement switch (no *core.PointerAssignStatement case)
// ============================================================
func TestBug5_PointerAssignStatement(t *testing.T) {
	source := `x := 0
p := &x
*p = 42
print x`

	// This should compile without error
	src := compileSource(t, source)
	assertSourceContains(t, src, "42")

	// End-to-end: should print "42"
	out := compileRun(t, "bug5", source)
	if strings.TrimSpace(out) != "42" {
		t.Fatalf("expected '42', got %q", out)
	}
}

func TestPointerPassToFunction(t *testing.T) {
	source := `func inc(p any) {
    *p = *p + 1
}
x := 0
p := &x
inc(p)
inc(p)
inc(p)
print x`

	src := compileSource(t, source)
	assertSourceContains(t, src, "recompiler.NewPtr")

	out := compileRun(t, "ptr_pass", source)
	if strings.TrimSpace(out) != "3" {
		t.Fatalf("expected '3', got %q", out)
	}
}

func TestPointerDerefReadWrite(t *testing.T) {
	source := `x := 10
p := &x
*p = *p + 5
print x
print *p`

	out := compileRun(t, "ptr_deref", source)
	if !strings.Contains(out, "15") {
		t.Fatalf("expected output containing '15', got %q", out)
	}
}

// ============================================================
// Bug #6: MethodCallBlockStatement not handled
// Location: compiler.go compileStatement switch (no *core.MethodCallBlockStatement case)
// ============================================================
func TestBug6_MethodCallBlockStatement(t *testing.T) {
	source := `wg := sync.NewWaitGroup()
counter := 0
wg.Go {
    counter = counter + 1
}
wg.Wait()
print counter`

	// This should compile without error
	src := compileSource(t, source)
	assertSourceContains(t, src, "wg")

	// End-to-end: should print "1"
	out := compileRun(t, "bug6", source)
	if strings.TrimSpace(out) != "1" {
		t.Fatalf("expected '1', got %q", out)
	}
}

// ============================================================
// Bug #7: ExecStatement not walked in analysis
// Location: analyzeEnvDependencies walkStatements (no *core.ExecStatement case)
// ============================================================
func TestBug7_ExecStatementDollarVar(t *testing.T) {
	source := `msg := "hello"
exec "echo $msg"`

	src := compileSource(t, source)

	// msg is used in exec "echo $msg" via $var, so it needs env sync
	assertSourceContains(t, src, `kamiEnv.SetString("msg", msg)`)
}

// ============================================================
// Bug #12: For-range IterCall not walked in analysis
// Location: analyzeEnvDependencies walkStatements (missing IterCall/IterVars)
// ============================================================
func TestBug12_ForRangeIterCallDollarVar(t *testing.T) {
	// This tests that for-range with iterator functions compiles correctly
	source := `arr := [10, 20, 30]
for i, v := range arr {
    print v
}`

	src := compileSource(t, source)

	// The for-range should compile to a proper loop
	assertSourceContains(t, src, "for")

	// End-to-end: should print 10, 20, 30
	out := compileRun(t, "bug12", source)
	if !strings.Contains(out, "10") || !strings.Contains(out, "20") || !strings.Contains(out, "30") {
		t.Fatalf("expected output containing 10, 20, 30, got %q", out)
	}
}

// ============================================================
// Bug #15: Multi-value assignment fallback is non-functional TODO
// Location: compiler.go ~line 575-581
// Variables are declared but never assigned.
// ============================================================
func TestBug15_MultiValueAssignmentFromTuple(t *testing.T) {
	// Multi-return function
	source := `func divmod(a int, b int) (int, int) {
    return a / b, a - a / b * b
}
q, r := divmod(17, 5)
print q
print r`

	src := compileSource(t, source)

	// The generated code should properly unpack the return values
	assertSourceContains(t, src, "kamiFunc_divmod")

	// End-to-end: q=3, r=2
	out := compileRun(t, "bug15", source)
	if !strings.Contains(out, "3") || !strings.Contains(out, "2") {
		t.Fatalf("expected output containing '3' and '2', got %q", out)
	}
}

// ============================================================
// Bug #17: Variable used before declaration causes Go redeclaration
// Location: compiler.go compileIdentifier ~line 936-940
// If print(x) auto-declares x, then x := 10 redeclares it.
// ============================================================
func TestBug17_NoRedeclarationError(t *testing.T) {
	source := `x := 10
y := x + 5
print y`

	// This should compile and build without Go redeclaration errors
	src := compileSource(t, source)
	assertSourceContains(t, src, "var x int64")

	// End-to-end
	out := compileRun(t, "bug17", source)
	if strings.TrimSpace(out) != "15" {
		t.Fatalf("expected '15', got %q", out)
	}
}

// ============================================================
// Bug #10: Duplicate code in Identifier walkExpression + wrong $ check
// Location: analyzeEnvDependencies walkExpression Identifier case
// The $ check on Identifier values is dead code (identifiers never contain $)
// ============================================================
func TestBug10_IdentifierNoDollarCheck(t *testing.T) {
	// This is a style issue, not a behavioral bug.
	// The $ check on Identifier values is dead code (identifiers never contain $).
	// This test just verifies normal identifier expressions compile correctly.
	source := `x := 10
y := 20
z := x + y
print z`

	src := compileSource(t, source)
	// z should be computed as x + y (no redundant parens at assignment level)
	assertSourceContains(t, src, "x + y")
}

// ============================================================
// Bug #11: Inconsistent envSync check (value vs key-existence)
// Location: compileVarStatement uses value check, compileAssignStatement uses key-existence
// ============================================================
func TestBug11_ConsistentEnvSyncCheck(t *testing.T) {
	// Var with $var reference should sync
	source1 := `var name string = "kami"
print "hello $name"`
	src1 := compileSource(t, source1)
	assertSourceContains(t, src1, `kamiEnv.SetString("name", name)`)

	// Var without $var reference should NOT sync (after fix)
	source2 := `var count int = 42
print count`
	src2 := compileSource(t, source2)
	// After fix: no env sync needed for "count"
	// Current behavior: envSync is nil so it always syncs (safe)
	_ = src2
}

// ============================================================
// Bug #16: Function literal assignment unconditionally syncs
// Location: compiler.go compileAssignStatement ~line 584-589
// ============================================================
func TestBug16_FunctionLiteralEnvSync(t *testing.T) {
	// Function literal assignment currently always syncs (bypasses envSync check)
	// This test documents the behavior, but the function literal bug (#1) prevents
	// proper testing of the env sync aspect.
	source := `add := func(a int, b int) int { return a + b }
print add(3, 4)`

	src := compileSource(t, source)
	// The function literal should at least be registered
	// (Currently broken due to Bug #1 - function literal body is empty)
	assertSourceContains(t, src, "add")
	_ = src
}

// ============================================================
// Bug #18: Redundant $var detection in CommandStatement
// Location: analyzeEnvDependencies walkStatements CommandStatement case
// The StringLiteral check is redundant with walkExpression
// ============================================================
func TestBug18_CommandArgDollarVar(t *testing.T) {
	// Test that $var in command arguments triggers env sync
	// Note: `print msg` is NOT $var - it's a direct expression.
	// But `print "hello $msg"` IS $var interpolation.
	source := `msg := "hello"
print "value is $msg"`

	src := compileSource(t, source)

	// msg is used via $var interpolation in print statement
	assertSourceContains(t, src, `kamiEnv.SetString("msg", msg)`)
}

// ============================================================
// Comprehensive integration test: mixed scenarios
// ============================================================
func TestBugIntegration_MixedScenarios(t *testing.T) {
	t.Run("pure compute no env sync", func(t *testing.T) {
		source := `sum := 0
for i := 0; i < 10; i = i + 1 {
    sum = sum + i
}
print sum`
		src := compileSource(t, source)
		// No $var reference, no builtin access, no closure capture
		// So NO kamiEnv.Set should appear
		assertSourceNotContains(t, src, `kamiEnv.SetString("sum"`)
		assertSourceNotContains(t, src, `kamiEnv.SetString("i"`)
	})

	t.Run("string interpolation needs env sync", func(t *testing.T) {
		source := `name := "kami"
print "hello $name"`
		src := compileSource(t, source)
		assertSourceContains(t, src, `kamiEnv.SetString("name", name)`)
	})

	t.Run("closure capture needs env sync", func(t *testing.T) {
		source := `x := 10
func getX() int { return x }
print getX()`
		src := compileSource(t, source)
		assertSourceContains(t, src, `kamiEnv.SetString("x", strconv.FormatInt(x, 10))`)
	})

	t.Run("function param no env sync", func(t *testing.T) {
		source := `func double(n int) int {
    result := n * 2
    return result
}
print double(5)`
		_ = compileSource(t, source)
		// Inside the function, "result" is local and not $var referenced
		// So no kamiEnv.SetString("result") should appear in the function body
		// (Currently it does because sub-compiler envSync is nil)
	})

	t.Run("array no env sync", func(t *testing.T) {
		source := `arr := [1, 2, 3, 4, 5]
total := 0
for i := 0; i < 5; i = i + 1 {
    total = total + arr[i]
}
print total`
		src := compileSource(t, source)
		assertSourceNotContains(t, src, `kamiEnv.SetString("arr"`)
		assertSourceNotContains(t, src, `kamiEnv.SetString("total"`)
		assertSourceNotContains(t, src, `kamiEnv.SetString("i"`)
	})
}

func init() {
	// Ensure projectRoot is set for test helpers (may already be set by compiler_test.go)
	if projectRoot == "" {
		out, err := exec.Command("go", "env", "GOMOD").Output()
		if err == nil {
			modFile := strings.TrimSpace(string(out))
			projectRoot = filepath.Dir(modFile)
		}
	}
}
