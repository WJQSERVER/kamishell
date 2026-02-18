package builtin

import (
	"fmt"
	"io"
	"os"
)

func init() {
	Register("pwd", Pwd)
}

func Pwd(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "pwd: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, dir)
	return 0
}
