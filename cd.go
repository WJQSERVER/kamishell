package kamishell

import (
	"fmt"
	"io"
	"os"
)

func init() {
	RegisterBuiltin("cd", Cd)
}

func Cd(args []string, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	var dir string
	if len(args) == 0 {
		dir = os.Getenv("HOME")
		if dir == "" {
			fmt.Fprintln(stderr, "cd: HOME not set")
			return 1
		}
	} else {
		dir = args[0]
	}

	err := os.Chdir(dir)
	if err != nil {
		fmt.Fprintf(stderr, "cd: %v\n", err)
		return 1
	}

	return 0
}
