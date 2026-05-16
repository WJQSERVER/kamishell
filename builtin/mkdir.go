package builtin

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "mkdir",
		Description: "创建目录",
		Usage:       "mkdir [-p] [-m mode] directory...",
		Help:        "创建目录；可按需创建父目录，并支持八进制权限模式。",
		Action:      Mkdir,
	})
}

func Mkdir(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	args = PreprocessArgs(args)
	fs := flag.NewFlagSet("mkdir", flag.ContinueOnError)
	fs.SetOutput(stderr)

	m := RegisterMeta("mkdir")
	parents := BoolFlag(fs, m, "p", "p", false, "no error if existing, make parent directories as needed")
	modeStr := StringFlag(fs, m, "m", "m", "0755", "set file mode (octal)")
	m.SetFlagCompleter("m", func(cmdName string, argIndex int, prefix string) []string {
		return []string{"0755", "0644", "0777", "0700", "0600", "0750", "0640"}
	})

	if err := fs.Parse(args); err != nil {
		return 1
	}

	mode, err := strconv.ParseUint(*modeStr, 8, 32)
	if err != nil {
		fmt.Fprintf(stderr, "mkdir: invalid mode: %s\n", *modeStr)
		return 1
	}

	targets := fs.Args()
	if len(targets) == 0 {
		fmt.Fprintln(stderr, "mkdir: missing operand")
		return 1
	}

	exitCode := 0
	for _, target := range targets {
		var err error
		if *parents {
			err = os.MkdirAll(target, os.FileMode(mode))
		} else {
			err = os.Mkdir(target, os.FileMode(mode))
		}

		if err != nil {
			fmt.Fprintf(stderr, "mkdir: %s: %v\n", target, err)
			exitCode = 1
		}
	}

	return exitCode
}
