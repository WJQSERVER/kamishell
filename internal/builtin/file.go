package builtin

import (
	"fmt"
	"io"
	"os"
)

func init() {
	Register("cat", Cat)
	Register("rm", Rm)
	Register("mkdir", Mkdir)
	Register("touch", Touch)
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

func Touch(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	for _, arg := range args {
		f, err := os.OpenFile(arg, os.O_RDONLY|os.O_CREATE, 0644)
		if err != nil {
			fmt.Fprintf(stderr, "touch: %v\n", err)
			return 1
		}
		f.Close()
	}
	return 0
}
