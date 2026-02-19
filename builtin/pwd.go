package builtin

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "pwd",
		Description: "显示当前工作目录",
		Action:      Pwd,
	})
}

func Pwd(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	args = PreprocessArgs(args)
	fs := flag.NewFlagSet("pwd", flag.ContinueOnError)
	fs.SetOutput(stderr)

	logical := fs.Bool("L", true, "use PWD from environment, even if it contains symlinks")
	physical := fs.Bool("P", false, "avoid all symlinks")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	// POSIX: "If both -L and -P are specified, the last one shall apply."
	// flag package doesn't guarantee this if we just check *logical and *physical.
	// But we can check which one was set last in args.
	usePhysical := *physical
	for _, arg := range args {
		if arg == "-L" {
			usePhysical = false
		} else if arg == "-P" {
			usePhysical = true
		}
	}

	_ = logical // satisfy compiler if not used elsewhere

	if usePhysical {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "pwd: %v\n", err)
			return 1
		}
		physDir, err := filepath.EvalSymlinks(dir)
		if err == nil {
			dir = physDir
		}
		fmt.Fprintln(stdout, dir)
		return 0
	}

	// Logical path
	pwd, ok := env.Get("PWD")
	if ok {
		var pwdStr string
		if s, ok := pwd.(string); ok {
			pwdStr = s
		} else if ins, ok := pwd.(Inspector); ok {
			pwdStr = ins.Inspect()
		}

		if pwdStr != "" && filepath.IsAbs(pwdStr) {
			fi1, err1 := os.Stat(pwdStr)
			fi2, err2 := os.Stat(".")
			if err1 == nil && err2 == nil && os.SameFile(fi1, fi2) {
				fmt.Fprintln(stdout, pwdStr)
				return 0
			}
		}
	}

	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "pwd: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, dir)
	return 0
}
