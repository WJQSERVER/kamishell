package make

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"kamishell/builtin"
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

func TestNewBuildCommandUsesPackageSource(t *testing.T) {
	target := &Target{
		Name:     "kami",
		Package:  ".",
		BuildEnv: map[string]string{"GOOS": "linux"},
	}

	cmd := newBuildCommand(target)
	if got, want := strings.Join(cmd.Args, " "), "go build -o kami ."; got != want {
		t.Fatalf("expected args %q, got %q", want, got)
	}
}

func TestNormalizeTargetInputsAllowsPackageSource(t *testing.T) {
	sources, pkg, ok := normalizeTargetInputs([]string{"."}, &bytes.Buffer{})
	if !ok {
		t.Fatal("expected package source to be accepted")
	}
	if pkg != "." {
		t.Fatalf("expected package '.', got %q", pkg)
	}
	if len(sources) != 0 {
		t.Fatalf("expected no explicit sources, got %v", sources)
	}
}

func TestNormalizeTargetInputsRejectsMixedPackageAndFiles(t *testing.T) {
	stderr := &bytes.Buffer{}
	_, _, ok := normalizeTargetInputs([]string{".", "main.go"}, stderr)
	if ok {
		t.Fatal("expected mixed package source and file list to be rejected")
	}
	if !strings.Contains(stderr.String(), "cannot be mixed") {
		t.Fatalf("expected helpful error, got %q", stderr.String())
	}
}

func TestTargetEnvOverridesBuildSnapshotForCommand(t *testing.T) {
	target := &Target{
		Name:    "kami",
		Sources: []string{"main.go"},
		BuildEnv: map[string]string{
			"GOOS":        "linux",
			"GOARCH":      "amd64",
			"CGO_ENABLED": "0",
		},
	}

	setEnvValue(target.BuildEnv, "GOOS", "windows")
	setEnvValue(target.BuildEnv, "CGO_ENABLED", "1")

	cmd := newBuildCommand(target)
	joinedEnv := strings.Join(cmd.Env, "\n")
	for _, expected := range []string{"GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=1"} {
		if !strings.Contains(joinedEnv, expected) {
			t.Fatalf("expected command env to contain %q, got %q", expected, joinedEnv)
		}
	}
	if got := targetOutputName(target); got != "kami.exe" {
		t.Fatalf("expected target_env GOOS override to affect output name, got %q", got)
	}
}

func TestMakeUsesSpecifiedFilePath(t *testing.T) {
	tempDir := t.TempDir()
	buildFile := filepath.Join(tempDir, "custom_build.km")
	if err := os.WriteFile(buildFile, []byte("project \"path-script\"\n"), 0o644); err != nil {
		t.Fatalf("write build file failed: %v", err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := Make([]string{buildFile}, core.NewEnvironment(), bytes.NewReader(nil), stdout, stderr)
	if code != 0 {
		t.Fatalf("expected make to succeed, code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Building project: path-script") {
		t.Fatalf("expected project from specified file, got stdout=%q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr, got %q", stderr.String())
	}
}

func TestRegisterBuildFunctionsRestoresGlobalBuiltins(t *testing.T) {
	originalProject := builtin.Builtins["project"]
	originalAddExecutable := builtin.Builtins["add_executable"]
	originalAddLibrary := builtin.Builtins["add_library"]
	originalTargetLinkLibraries := builtin.Builtins["target_link_libraries"]
	originalTargetEnv := builtin.Builtins["target_env"]

	restore := registerBuildFunctions()
	restore()

	if builtin.Builtins["project"] != originalProject {
		t.Fatal("expected project builtin to be restored")
	}
	if builtin.Builtins["add_executable"] != originalAddExecutable {
		t.Fatal("expected add_executable builtin to be restored")
	}
	if builtin.Builtins["add_library"] != originalAddLibrary {
		t.Fatal("expected add_library builtin to be restored")
	}
	if builtin.Builtins["target_link_libraries"] != originalTargetLinkLibraries {
		t.Fatal("expected target_link_libraries builtin to be restored")
	}
	if builtin.Builtins["target_env"] != originalTargetEnv {
		t.Fatal("expected target_env builtin to be restored")
	}
}
