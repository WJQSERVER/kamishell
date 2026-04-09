package builtin

import (
	"fmt"
	"io"
	"os/exec"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "which",
		Description: "查找命令的可执行文件路径",
		Usage:       "which name...",
		Help:        "仅在 PATH 中查找外部可执行文件，不解析内建命令。",
		Action:      Which,
	})
}

func Which(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if HandleBuiltinHelp(Builtins["which"], args, stdout) {
		return 0
	}
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
