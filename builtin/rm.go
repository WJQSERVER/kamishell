package builtin

import (
	"fmt"
	"io"
	"os"
)

func init() {
	RegisterBuiltin("rm", Rm)
}

func Rm(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	for _, arg := range args {
		err := os.RemoveAll(arg)
		if err != nil {
			fmt.Fprintf(stderr, "rm: %v\n", err)
			return 1
		}
	}
	return 0
}
