package core

import "testing"

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
