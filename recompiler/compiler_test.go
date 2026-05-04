package recompiler

import (
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"kamishell/core"
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