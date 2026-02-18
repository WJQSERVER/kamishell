package builtin

import (
	"io"
)

type Environment interface {
	Set(name string, val interface{})
	Get(name string) (interface{}, bool)
}

type BuiltinFunc func(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int

var Builtins = map[string]BuiltinFunc{}

func Register(name string, fn BuiltinFunc) {
	Builtins[name] = fn
}
