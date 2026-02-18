package builtin

import (
	"fmt"
	"io"
	"os"
)

func init() {
	RegisterBuiltin("env", Env)
}

func Env(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	for _, e := range os.Environ() {
		fmt.Fprintln(stdout, e)
	}
	return 0
}
