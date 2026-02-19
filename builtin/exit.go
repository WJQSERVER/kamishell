package builtin

import (
	"fmt"
	"io"
	"os"
	"strconv"
)

func init() {
	RegisterBuiltin("exit", Exit)
}

func Exit(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	code := 0
	if len(args) > 0 {
		c, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(stderr, "exit: illegal number: %s\n", args[0])
			os.Exit(1)
		}
		code = c
	}
	os.Exit(code)
	return 0 // never reached
}
