package builtin

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "cat",
		Description: "连接文件并打印到标准输出",
		Usage:       "cat [-u] [file...]",
		Help:        "按顺序输出文件内容；文件名为 - 时从标准输入读取。",
		Action:      Cat,
	})
}

func Cat(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	args = PreprocessArgs(args)
	fs := flag.NewFlagSet("cat", flag.ContinueOnError)
	fs.SetOutput(stderr)

	_ = fs.Bool("u", false, "ignored; for POSIX compatibility")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	targets := fs.Args()
	if len(targets) == 0 {
		_, err := io.Copy(stdout, stdin)
		if err != nil {
			fmt.Fprintf(stderr, "cat: %v\n", err)
			return 1
		}
		return 0
	}

	exitCode := 0
	for _, target := range targets {
		var r io.Reader
		var closer io.Closer

		if target == "-" {
			r = stdin
		} else {
			f, err := os.Open(target)
			if err != nil {
				fmt.Fprintf(stderr, "cat: %s: %v\n", target, err)
				exitCode = 1
				continue
			}
			r = f
			closer = f
		}

		_, err := io.Copy(stdout, r)
		if closer != nil {
			if closeErr := closer.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
		}

		if err != nil {
			fmt.Fprintf(stderr, "cat: %s: %v\n", target, err)
			exitCode = 1
		}
	}

	return exitCode
}
