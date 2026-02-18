package builtin

import (
	"fmt"
	"io"
	"os"
)

func init() {
	RegisterBuiltin("mkdir", Mkdir)
}

func Mkdir(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	for _, arg := range args {
		err := os.MkdirAll(arg, 0755)
		if err != nil {
			fmt.Fprintf(stderr, "mkdir: %v\n", err)
			return 1
		}
	}
	return 0
}
