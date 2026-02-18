package builtin

import (
	"fmt"
	"io"
	"os"
)

func init() {
	RegisterBuiltin("cat", Cat)
}

func Cat(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		io.Copy(stdout, stdin)
		return 0
	}

	for _, arg := range args {
		content, err := os.ReadFile(arg)
		if err != nil {
			fmt.Fprintf(stderr, "cat: %v\n", err)
			return 1
		}
		fmt.Fprint(stdout, string(content))
	}
	return 0
}
