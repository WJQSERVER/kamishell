package builtin

import (
	"fmt"
	"io"
	"os/exec"
)

func init() {
	RegisterBuiltin("which", Which)
}

func Which(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		return 0
	}

	exitCode := 0
	for _, name := range args {
		path, err := exec.LookPath(name)
		if err == nil {
			fmt.Fprintln(stdout, path)
		} else {
			exitCode = 1
		}
	}

	return exitCode
}
