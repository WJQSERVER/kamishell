package make

import (
	"runtime"
	"strings"
	"testing"

	"kamishell/core"
)

func TestSnapshotBuildEnvUsesScriptEnvPackage(t *testing.T) {
	env := core.NewEmptyEnvironment()
	env.SetPackageValue("env", "GOOS", "windows")
	env.SetPackageValue("env", "GOARCH", "amd64")
	env.SetPackageValue("env", "CGO_ENABLED", "0")
	env.SetPackageValue("env", "COUNT", "2")
	env.SetPackageValue("env", "ENABLED", "true")
	env.Set("GOOS", "linux")

	snapshot := snapshotBuildEnv(env)

	if got := snapshot["GOOS"]; got != "windows" {
		t.Fatalf("expected GOOS=windows, got %q", got)
	}
	if got := snapshot["GOARCH"]; got != "amd64" {
		t.Fatalf("expected GOARCH=amd64, got %q", got)
	}
	if got := snapshot["CGO_ENABLED"]; got != "0" {
		t.Fatalf("expected CGO_ENABLED=0, got %q", got)
	}
	if got := snapshot["COUNT"]; got != "2" {
		t.Fatalf("expected COUNT=2, got %q", got)
	}
	if got := snapshot["ENABLED"]; got != "true" {
		t.Fatalf("expected ENABLED=true, got %q", got)
	}
	if got := snapshot["GOOS"]; got != "windows" {
		t.Fatalf("expected script env GOOS to override variable store, got %q", got)
	}
}

func TestTargetOutputNameUsesTargetGOOS(t *testing.T) {
	target := &Target{
		Name:     "kami",
		BuildEnv: map[string]string{"GOOS": "windows"},
	}

	if got := targetOutputName(target); got != "kami.exe" {
		t.Fatalf("expected windows target to end with .exe, got %q", got)
	}

	target.BuildEnv["GOOS"] = "linux"
	if got := targetOutputName(target); got != "kami" {
		t.Fatalf("expected non-windows target to keep name, got %q", got)
	}
}

func TestNewBuildCommandUsesTargetEnvironment(t *testing.T) {
	target := &Target{
		Name:    "kami",
		Sources: []string{"main.go"},
		BuildEnv: map[string]string{
			"GOOS":        "linux",
			"GOARCH":      "arm64",
			"CGO_ENABLED": "0",
		},
	}

	cmd := newBuildCommand(target)

	if got, want := strings.Join(cmd.Args, " "), "go build -o kami main.go"; got != want {
		t.Fatalf("expected args %q, got %q", want, got)
	}

	joinedEnv := strings.Join(cmd.Env, "\n")
	for _, expected := range []string{"GOOS=linux", "GOARCH=arm64", "CGO_ENABLED=0"} {
		if !strings.Contains(joinedEnv, expected) {
			t.Fatalf("expected command env to contain %q, got %q", expected, joinedEnv)
		}
	}
}

func TestTargetGOOSFallsBackToHost(t *testing.T) {
	target := &Target{Name: "kami"}
	if got := targetGOOS(target); got != runtime.GOOS {
		t.Fatalf("expected fallback GOOS %q, got %q", runtime.GOOS, got)
	}
}
