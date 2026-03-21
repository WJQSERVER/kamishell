package main

import (
	"io"
	"testing"

	"kamishell/builtin"
	"kamishell/core"
)

func TestRunBuiltinArgsPassesRawArguments(t *testing.T) {
	defer delete(builtin.Builtins, "test_builtin_raw_args")

	called := false
	builtin.RegisterBuiltin(&builtin.BuiltinCommand{
		Name: "test_builtin_raw_args",
		Action: func(args []string, env builtin.Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
			called = true
			if len(args) != 1 {
				t.Fatalf("expected 1 arg, got %d", len(args))
			}
			if args[0] != ".\\tmp_make_env_set.km" {
				t.Fatalf("expected raw relative windows path, got %q", args[0])
			}
			return 0
		},
	})

	runBuiltinArgs([]string{"test_builtin_raw_args", ".\\tmp_make_env_set.km"}, core.NewEnvironment())
	if !called {
		t.Fatal("expected builtin to be called")
	}
}
