package core

import (
	"io"
	"strings"
	"testing"
)

func TestScriptEnvironmentDoesNotInheritOuterPackageScope(t *testing.T) {
	outer := NewEmptyEnvironment()
	outer.SetPackageValue("env", "GOOS", "linux")

	scriptEnv := NewScriptEnvironment(outer)
	if _, ok := scriptEnv.GetPackageValue("env", "GOOS"); ok {
		t.Fatalf("did not expect script env to inherit outer package scope")
	}

	scriptEnv.SetPackageValue("env", "GOARCH", "arm64")
	if _, ok := outer.GetPackageValue("env", "GOARCH"); ok {
		t.Fatalf("did not expect outer package scope to be mutated by script env")
	}
}

func TestEnclosedEnvironmentSharesScriptPackageScope(t *testing.T) {
	scriptEnv := NewScriptEnvironment(NewEmptyEnvironment())
	scriptEnv.SetPackageValue("env", "GOOS", "linux")

	enclosed := NewEnclosedEnvironment(scriptEnv)
	if got, ok := enclosed.GetPackageValue("env", "GOOS"); !ok || got != "linux" {
		t.Fatalf("expected enclosed env to see script package value, got %q, %v", got, ok)
	}

	enclosed.SetPackageValue("env", "GOARCH", "arm64")
	if got, ok := scriptEnv.GetPackageValue("env", "GOARCH"); !ok || got != "arm64" {
		t.Fatalf("expected enclosed env to share script package scope, got %q, %v", got, ok)
	}
}

func TestScriptEnvironmentKeepsVariableScopeSeparate(t *testing.T) {
	outer := NewEmptyEnvironment()
	outer.Set("GOOS", "windows")

	scriptEnv := NewScriptEnvironment(outer)
	scriptEnv.Set("GOOS", "linux")

	if got, _ := scriptEnv.Get("GOOS"); got.(*String).Value != "linux" {
		t.Fatalf("expected script env variable override, got %v", got)
	}
	if got, _ := outer.Get("GOOS"); got.(*String).Value != "windows" {
		t.Fatalf("expected outer env variable to stay unchanged, got %v", got)
	}
}

func TestSetObjectSyncsWithRefStore(t *testing.T) {
	env := NewEmptyEnvironment()
	env.SetObject("x", getIntegerObject(1))

	ref, ok := env.GetRef("x")
	if !ok {
		t.Fatal("expected GetRef to succeed")
	}
	if ref.Value.(*Integer).Value != 1 {
		t.Fatalf("expected ref.Value=1, got %v", ref.Value)
	}

	env.SetObject("x", getIntegerObject(99))
	if ref.Value.(*Integer).Value != 99 {
		t.Fatalf("expected ref.Value to update to 99 after SetObject, got %v", ref.Value)
	}
}

func TestSetObjectDoesNotPanicWithoutRefStore(t *testing.T) {
	env := NewEmptyEnvironment()
	env.SetObject("x", getIntegerObject(1))

	val, ok := env.GetObject("x")
	if !ok || val.(*Integer).Value != 1 {
		t.Fatalf("expected x=1, got %v, %v", val, ok)
	}
}

func TestEvalStatementsSetsErrOnlyOnError(t *testing.T) {
	// Successful statements should NOT touch "err" — only errors set it
	env := NewEmptyEnvironment()

	program := mustParseProgram(t, `x := 1 + 2`)
	result := EvalWithIO(program, env, strings.NewReader(""), io.Discard, io.Discard)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.Inspect())
	}

	// After successful eval, "err" should remain unset (nil/absent)
	// because evalStatements only writes err on actual errors
	errVal, ok := env.GetObject("err")
	if ok && errVal != nil && errVal != NULL {
		t.Fatalf("expected err to not be set after successful eval, got %v", errVal)
	}
}

func TestEvalStatementsSetsErrOnRuntimeError(t *testing.T) {
	env := NewEmptyEnvironment()

	program := mustParseProgram(t, `x := 1 - "hello"`)
	result := EvalWithIO(program, env, strings.NewReader(""), io.Discard, io.Discard)
	if !isError(result) {
		t.Fatalf("expected error, got %v", result)
	}

	errVal, _ := env.GetObject("err")
	if errVal == nil || errVal == NULL {
		t.Fatal("expected err to be set after runtime error")
	}
}

func mustParseProgram(t *testing.T, input string) *Program {
	t.Helper()
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()
	if program == nil {
		t.Fatal("expected parsed program")
	}
	return program
}
