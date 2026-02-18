package builtin

import (
	"io"
)

type BuiltinFunc func(args []string, stdout io.Writer, stderr io.Writer) int

var Builtins = map[string]BuiltinFunc{}

func Register(name string, fn BuiltinFunc) {
	Builtins[name] = fn
}
