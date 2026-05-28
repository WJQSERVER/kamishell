package recompiler

import (
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WJQSERVER/kamishell/core"
)

var projectRoot string

func init() {
	// Find the project root (where go.mod is)
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		panic(err)
	}
	modFile := strings.TrimSpace(string(out))
	projectRoot = filepath.Dir(modFile)
}

func compileAndBuild(t *testing.T, name, source string) (string, string) {
	t.Helper()

	// Parse the source
	lexer := core.NewLexer(source)
	parser := core.NewParser(lexer)
	program := parser.ParseProgram()

	// Check for parse errors (InvalidStatement nodes)
	for _, stmt := range program.Statements {
		if inv, ok := stmt.(*core.InvalidStatement); ok {
			t.Fatalf("parse error: %s", inv.Message)
		}
	}

	// Compile to Go
	compiled, err := Compile(program)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	// Verify it's valid Go syntax
	_, err = format.Source([]byte(compiled.Source))
	if err != nil {
		t.Fatalf("generated code is not valid Go:\n%v\n---\n%s", err, compiled.Source)
	}

	// Write to a subdirectory under the project root so go.mod is picked up
	// This ensures kamishell/builtin and kamishell/recompiler resolve correctly
	tmpDir := filepath.Join(projectRoot, ".test_recompiler")
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(compiled.Source), 0644); err != nil {
		t.Fatalf("write main.go failed: %v", err)
	}

	// Build - go will walk up from tmpDir to find projectRoot/go.mod
	binary := filepath.Join(tmpDir, name)
	cmd := exec.Command("go", "build", "-o", binary, goFile)
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed:\n%s\n%s", string(out), compiled.Source)
	}

	return binary, tmpDir
}

func runBinary(t *testing.T, binary string) string {
	t.Helper()
	cmd := exec.Command(binary)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("binary returned error: %v", err)
	}
	return string(out)
}

func TestCompileEmpty(t *testing.T) {
	source := ""
	binary, _ := compileAndBuild(t, "empty", source)
	out := runBinary(t, binary)
	if out != "" {
		t.Fatalf("expected empty output, got: %q", out)
	}
}

func TestCompilePrint(t *testing.T) {
	source := `print "hello world"`
	binary, _ := compileAndBuild(t, "print", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "hello world" {
		t.Fatalf("expected 'hello world', got: %q", out)
	}
}

func TestCompileIntegerArithmetic(t *testing.T) {
	source := `x := 10 + 20
print x`
	binary, _ := compileAndBuild(t, "arith", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "30" {
		t.Fatalf("expected '30', got: %q", out)
	}
}

func TestCompileStringConcat(t *testing.T) {
	source := `x := "hello, "
y := "world"
print x + y`
	binary, _ := compileAndBuild(t, "string_concat", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "hello, world" {
		t.Fatalf("expected 'hello, world', got: %q", out)
	}
}

func TestCompileIfElse(t *testing.T) {
	source := `x := 10
if x > 5 {
    print "big"
} else {
    print "small"
}`
	binary, _ := compileAndBuild(t, "ifelse", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "big" {
		t.Fatalf("expected 'big', got: %q", out)
	}
}

func TestCompileForLoop(t *testing.T) {
	source := `sum := 0
for i := 0; i < 5; i = i + 1 {
    sum = sum + 1
}
print sum`
	binary, _ := compileAndBuild(t, "forloop", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "5" {
		t.Fatalf("expected '5', got: %q", out)
	}
}

func TestCompileBoolExpr(t *testing.T) {
	source := `x := true
if x {
    print "yes"
}`
	binary, _ := compileAndBuild(t, "bool", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "yes" {
		t.Fatalf("expected 'yes', got: %q", out)
	}
}

func TestCompileComparison(t *testing.T) {
	source := `x := 100
y := 200
if x < y {
    print "lt"
}`
	binary, _ := compileAndBuild(t, "cmp", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "lt" {
		t.Fatalf("expected 'lt', got: %q", out)
	}
}

func TestCompileVarDecl(t *testing.T) {
	source := `var count int = 42
print count`
	binary, _ := compileAndBuild(t, "vardecl", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "42" {
		t.Fatalf("expected '42', got: %q", out)
	}
}

func TestCompileSwitch(t *testing.T) {
	source := `x := 2
switch x {
case 1:
    print "one"
case 2:
    print "two"
default:
    print "other"
}`
	binary, _ := compileAndBuild(t, "switch", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "two" {
		t.Fatalf("expected 'two', got: %q", out)
	}
}

func TestCompileBuiltinLs(t *testing.T) {
	source := `ls`
	binary, _ := compileAndBuild(t, "builtin_ls", source)
	out := runBinary(t, binary)
	// ls runs in the build directory, just check it doesn't error
	if out == "" {
		t.Fatalf("expected non-empty output from ls, got empty")
	}
}

func TestCompileStringInterpolation(t *testing.T) {
	source := `name := "kami"
print "hello $name"`
	binary, _ := compileAndBuild(t, "interp", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "hello kami" {
		t.Fatalf("expected 'hello kami', got: %q", out)
	}
}

func TestCompileNestedIfElse(t *testing.T) {
	source := `x := 5
if x > 10 {
    print "a"
} else if x > 3 {
    print "b"
} else {
    print "c"
}`
	binary, _ := compileAndBuild(t, "nested_if", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "b" {
		t.Fatalf("expected 'b', got: %q", out)
	}
}

func TestCompileFullProgram(t *testing.T) {
	source := `x := 1
y := 2
z := x + y
if z == 3 {
    print "ok"
} else {
    print "fail"
}`
	binary, _ := compileAndBuild(t, "full", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "ok" {
		t.Fatalf("expected 'ok', got: %q", out)
	}
}

// --- Readability: literal optimization and paren removal ---

func TestLiteralOptimizationInt(t *testing.T) {
	source := `x := 42
print x`
	src := compileSource(t, source)
	// Should generate "var x int64 = 42" not "var x int64 = int64(42)"
	assertSourceContains(t, src, "var x int64 = 42")
	assertSourceNotContains(t, src, "int64(42)")
}

func TestLiteralOptimizationFloat(t *testing.T) {
	source := `x := 3.14
print x`
	src := compileSource(t, source)
	// Should generate "var x float64 = 3.14" not "var x float64 = float64(3.14)"
	assertSourceContains(t, src, "var x float64 = 3.14")
}

func TestArithmeticPrecedencePreserved(t *testing.T) {
	// a + b * c should be a + (b * c), not (a + b) * c
	source := `a := 2
b := 3
c := 4
result := a + b * c
print result`
	binary, _ := compileAndBuild(t, "precedence", source)
	out := runBinary(t, binary)
	// 2 + 3*4 = 2 + 12 = 14
	if strings.TrimSpace(out) != "14" {
		t.Fatalf("expected '14', got: %q", out)
	}
}

func TestArithmeticPrecedenceSubtraction(t *testing.T) {
	source := `a := 10
b := 3
c := 2
result := a - b * c
print result`
	binary, _ := compileAndBuild(t, "precedence_sub", source)
	out := runBinary(t, binary)
	// 10 - 3*2 = 10 - 6 = 4
	if strings.TrimSpace(out) != "4" {
		t.Fatalf("expected '4', got: %q", out)
	}
}

func TestArithmeticPrecedenceDivision(t *testing.T) {
	source := `a := 10
b := 2
c := 3
result := a + b / c
print result`
	binary, _ := compileAndBuild(t, "precedence_div", source)
	out := runBinary(t, binary)
	// 10 + 2/3 = 10 + 0 = 10 (integer division)
	if strings.TrimSpace(out) != "10" {
		t.Fatalf("expected '10', got: %q", out)
	}
}

func TestNestedFunctionCallsCorrect(t *testing.T) {
	source := `func add(a int, b int) int { return a + b }
func mul(a int, b int) int { return a * b }
result := add(mul(2, 3), 4)
print result`
	binary, _ := compileAndBuild(t, "nested_calls", source)
	out := runBinary(t, binary)
	// mul(2,3) = 6, add(6, 4) = 10
	if strings.TrimSpace(out) != "10" {
		t.Fatalf("expected '10', got: %q", out)
	}
}

func TestForLoopPostCorrect(t *testing.T) {
	source := `sum := 0
for i := 0; i < 5; i = i + 1 {
    sum = sum + i
}
print sum`
	binary, _ := compileAndBuild(t, "for_post", source)
	out := runBinary(t, binary)
	// 0+1+2+3+4 = 10
	if strings.TrimSpace(out) != "10" {
		t.Fatalf("expected '10', got: %q", out)
	}
}

func TestReturnWithoutParens(t *testing.T) {
	source := `func calc(a int, b int) int { return a + b }
print calc(3, 4)`
	binary, _ := compileAndBuild(t, "return_no_parens", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "7" {
		t.Fatalf("expected '7', got: %q", out)
	}
}

func TestPrintWithoutParens(t *testing.T) {
	source := `x := 10
y := 20
print x + y`
	binary, _ := compileAndBuild(t, "print_no_parens", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "30" {
		t.Fatalf("expected '30', got: %q", out)
	}
}

func TestStringConcatCorrect(t *testing.T) {
	source := `a := "hello"
b := " "
c := "world"
print a + b + c`
	binary, _ := compileAndBuild(t, "str_concat", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "hello world" {
		t.Fatalf("expected 'hello world', got: %q", out)
	}
}

func TestMixedArithmeticAndComparison(t *testing.T) {
	source := `x := 10
y := 3
if x % y == 1 {
    print "correct"
} else {
    print "wrong"
}`
	binary, _ := compileAndBuild(t, "mixed_ops", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "correct" {
		t.Fatalf("expected 'correct', got: %q", out)
	}
}

// --- Exec Recompiler Tests ---
// These tests constrain the expected behavior of exec in recompiler mode.

// 裸词形式：基本执行
func TestCompileExecBareWordBasic(t *testing.T) {
	source := `exec echo hello`
	binary, _ := compileAndBuild(t, "exec_bare", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "hello" {
		t.Fatalf("expected 'hello', got: %q", out)
	}
}

// 裸词形式：带引号
func TestCompileExecBareWordWithQuotes(t *testing.T) {
	source := `exec echo "my document.txt"`
	binary, _ := compileAndBuild(t, "exec_quotes", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "my document.txt" {
		t.Fatalf("expected 'my document.txt', got: %q", out)
	}
}

// 裸词形式：带变量
func TestCompileExecBareWordWithVariable(t *testing.T) {
	source := `msg := "hello world"
exec echo $msg`
	binary, _ := compileAndBuild(t, "exec_var", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "hello world" {
		t.Fatalf("expected 'hello world', got: %q", out)
	}
}

// 裸词形式：$var 依赖分析
func TestCompileExecBareWordDollarVarDependency(t *testing.T) {
	source := `msg := "hello"
exec echo $msg`
	src := compileSource(t, source)
	// msg is used in exec echo $msg via $var, so it needs env sync
	assertSourceContains(t, src, `kamiEnv.SetString("msg", msg)`)
}

// 函数形式：基本执行
func TestCompileExecFunctionBasic(t *testing.T) {
	source := `exec("echo hello")`
	binary, _ := compileAndBuild(t, "exec_func", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "hello" {
		t.Fatalf("expected 'hello', got: %q", out)
	}
}

// 函数形式：带变量
func TestCompileExecFunctionWithVariable(t *testing.T) {
	source := `cmd := "echo hello"
exec(cmd)`
	binary, _ := compileAndBuild(t, "exec_func_var", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "hello" {
		t.Fatalf("expected 'hello', got: %q", out)
	}
}

// 弃用字符串形式：基本执行
func TestCompileExecDeprecatedStringForm(t *testing.T) {
	source := `exec "echo hello"`
	binary, _ := compileAndBuild(t, "exec_deprecated", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "hello" {
		t.Fatalf("expected 'hello', got: %q", out)
	}
}

// 弃用字符串形式：带变量
func TestCompileExecDeprecatedStringFormWithVariable(t *testing.T) {
	source := `msg := "hello"
exec "echo $msg"`
	binary, _ := compileAndBuild(t, "exec_deprecated_var", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "hello" {
		t.Fatalf("expected 'hello', got: %q", out)
	}
}

// 裸词形式：多参数
func TestCompileExecBareWordMultipleArgs(t *testing.T) {
	source := `exec printf "hello %s" world`
	binary, _ := compileAndBuild(t, "exec_multi_args", source)
	out := runBinary(t, binary)
	if strings.TrimSpace(out) != "hello world" {
		t.Fatalf("expected 'hello world', got: %q", out)
	}
}
func TestConstantFoldIntArithmetic(t *testing.T) {
	// All arithmetic on two int literals should be folded at compile time.
	source := `a := 3 + 4
b := 10 - 3
c := 6 * 7
d := 20 / 4
e := 17 % 5
print a`
	src := compileSource(t, source)
	assertSourceContains(t, src, "int64(7)")
	assertSourceContains(t, src, "int64(42)")
	assertSourceContains(t, src, "int64(5)")
	assertSourceContains(t, src, "int64(2)")
	// The folded literals should appear as literal values, not runtime calls
	// Should NOT contain runtime dispatch for these
	assertSourceNotContains(t, src, "recompiler.Add")
	assertSourceNotContains(t, src, "recompiler.Sub")
	assertSourceNotContains(t, src, "recompiler.Mul")
	assertSourceNotContains(t, src, "recompiler.Div")
	assertSourceNotContains(t, src, "recompiler.Mod")
}

func TestConstantFoldFloatArithmetic(t *testing.T) {
	source := `a := 1.5 + 2.5
b := 10.0 - 3.0
c := 2.5 * 4.0
d := 9.0 / 3.0
print a`
	src := compileSource(t, source)
	assertSourceContains(t, src, "float64(4)")
	assertSourceContains(t, src, "float64(7)")
	assertSourceContains(t, src, "float64(10)")
	assertSourceContains(t, src, "float64(3)")
	assertSourceNotContains(t, src, "recompiler.Add")
}

func TestConstantFoldStringConcat(t *testing.T) {
	source := `a := "hello" + " " + "world"
print a`
	src := compileSource(t, source)
	// Inner "hello"+" " folds to "hello ", then native concat with "world"
	assertSourceContains(t, src, `"hello "`)
	assertSourceNotContains(t, src, "recompiler.Add")
}

func TestConstantFoldComparison(t *testing.T) {
	source := `a := 3 == 3
b := 3 != 4
c := 3 < 4
d := 5 > 2
e := 5 >= 5
f := "abc" == "abc"
g := true != false
print a`
	src := compileSource(t, source)
	assertSourceContains(t, src, "var a bool = true")
	assertSourceContains(t, src, "var b bool = true")
	assertSourceContains(t, src, "var c bool = true")
	assertSourceContains(t, src, "var d bool = true")
	assertSourceContains(t, src, "var e bool = true")
	assertSourceContains(t, src, "var f bool = true")
	assertSourceContains(t, src, "var g bool = true")
	assertSourceNotContains(t, src, "recompiler.Eq")
	assertSourceNotContains(t, src, "recompiler.NotEq")
}

func TestConstantFoldUnaryMinus(t *testing.T) {
	source := `a := -42
b := -3.14
print a`
	src := compileSource(t, source)
	assertSourceContains(t, src, "int64(-42)")
	assertSourceContains(t, src, "float64(-3.14)")
}

func TestConstantFoldUnaryNot(t *testing.T) {
	source := `a := !true
b := !false
print a`
	src := compileSource(t, source)
	assertSourceContains(t, src, "var a bool = false")
	assertSourceContains(t, src, "var b bool = true")
}

func TestConstantFoldMixedWithVariables(t *testing.T) {
	// Variable + literal should NOT be folded (only literal-literal pairs)
	source := `x := 10
y := x + 5
print y`
	src := compileSource(t, source)
	// x + 5 should use native + (both int), but not folded
	assertSourceContains(t, src, "+ int64(5)")
}

func TestConstantFoldDivisionByZeroNotFolded(t *testing.T) {
	// Division by zero should NOT be folded (safety)
	source := `a := 10 / 0
print a`
	src := compileSource(t, source)
	// Should still have runtime division expression, not a single folded literal
	assertSourceContains(t, src, "/ int64(0)")
}
