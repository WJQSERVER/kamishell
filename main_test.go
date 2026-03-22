package main

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"kamishell/builtin"
	"kamishell/core"
)

func TestRunBuiltinArgsPassesRawRelativeWindowsPath(t *testing.T) {
	runBuiltinArgsCase(t, []string{"test_builtin_raw_args_rel", ".\\tmp_make_env_set.km"}, []string{".\\tmp_make_env_set.km"})
}

func TestRunBuiltinArgsPassesRawAbsoluteWindowsPath(t *testing.T) {
	runBuiltinArgsCase(t, []string{"test_builtin_raw_args_abs", `D:\programs\alina\tmp_make_env_set.km`}, []string{`D:\programs\alina\tmp_make_env_set.km`})
}

func TestRunBuiltinArgsPreservesSpacesAndMultipleArgs(t *testing.T) {
	runBuiltinArgsCase(t,
		[]string{"test_builtin_raw_args_multi", `.\dir with space\build file.km`, `GOOS=windows`, `CGO_ENABLED=0`},
		[]string{`.\dir with space\build file.km`, `GOOS=windows`, `CGO_ENABLED=0`},
	)
}

func TestShouldRunAsBuiltinWhenBuiltinExistsAndFileDoesNot(t *testing.T) {
	name := "test_builtin_dispatch_missing"
	builtin.RegisterBuiltin(&builtin.BuiltinCommand{Name: name, Action: noopBuiltin})
	defer delete(builtin.Builtins, name)

	if !shouldRunAsBuiltin(name) {
		t.Fatalf("expected %q to dispatch as builtin when file does not exist", name)
	}
}

func TestShouldRunAsBuiltinWhenPathIsDirectory(t *testing.T) {
	tempDir := t.TempDir()
	name := filepath.Join(tempDir, "tools")
	if err := os.Mkdir(name, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	builtin.RegisterBuiltin(&builtin.BuiltinCommand{Name: name, Action: noopBuiltin})
	defer delete(builtin.Builtins, name)

	if !shouldRunAsBuiltin(name) {
		t.Fatalf("expected builtin dispatch for directory path %q", name)
	}
}

func TestShouldNotRunAsBuiltinWhenRegularFileExists(t *testing.T) {
	tempDir := t.TempDir()
	name := filepath.Join(tempDir, "make")
	if err := os.WriteFile(name, []byte("print \"script\"\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	builtin.RegisterBuiltin(&builtin.BuiltinCommand{Name: name, Action: noopBuiltin})
	defer delete(builtin.Builtins, name)

	if shouldRunAsBuiltin(name) {
		t.Fatalf("did not expect builtin dispatch when regular file exists: %q", name)
	}
}

func runBuiltinArgsCase(t *testing.T, args []string, expected []string) {
	t.Helper()
	name := args[0]
	defer delete(builtin.Builtins, name)

	called := false
	builtin.RegisterBuiltin(&builtin.BuiltinCommand{
		Name: name,
		Action: func(actual []string, env builtin.Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
			called = true
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("expected raw args %v, got %v", expected, actual)
			}
			return 0
		},
	})

	runBuiltinArgs(args, core.NewEnvironment())
	if !called {
		t.Fatal("expected builtin to be called")
	}
}

func noopBuiltin(args []string, env builtin.Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	return 0
}

func TestBuildPromptUsesCurrentDirectoryBaseName(t *testing.T) {
	tempDir := t.TempDir()
	childDir := filepath.Join(tempDir, "prompt-target")
	if err := os.Mkdir(childDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(childDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	prompt := buildPrompt(false)
	if !strings.Contains(prompt, "prompt-target") {
		t.Fatalf("expected prompt to contain current dir base name, got %q", prompt)
	}
	if !strings.Contains(prompt, "kami>") {
		t.Fatalf("expected prompt to contain kami marker, got %q", prompt)
	}
}

func TestBuildPromptWithColorKeepsANSIAndPath(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	prompt := buildPrompt(true)
	if !strings.Contains(prompt, "\033[") {
		t.Fatalf("expected colored prompt to contain ANSI escape, got %q", prompt)
	}
	if !strings.Contains(prompt, filepath.Base(tempDir)) {
		t.Fatalf("expected colored prompt to contain current dir name, got %q", prompt)
	}
}
