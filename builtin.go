package kamishell

import (
	"io"
)

type BuiltinFunc func(args []string, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int

var Builtins = map[string]BuiltinFunc{}

func RegisterBuiltin(name string, fn BuiltinFunc) {
	Builtins[name] = fn
}
