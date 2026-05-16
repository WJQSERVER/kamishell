package builtin

import (
	"fmt"
	"io"
	"os/exec"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "which",
		Description: "查找命令的位置（内置命令或可执行文件路径）",
		Usage:       "which name...",
		Help: `显示命令的完整路径；如果是内置命令则提示为 shell builtin。

示例:
  which ls
  which cd
  which go`,
		Action: Which,
	})
	SetArgCompleter("which", completeCommandNames)
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
		found := false

		if _, ok := Builtins[name]; ok {
			fmt.Fprintf(stdout, "%s: shell builtin\n", name)
			found = true
		}

		if !found && env != nil {
			if _, ok := env.Get(name); ok {
				fmt.Fprintf(stdout, "%s\n", name)
				found = true
			}
		}

		if found {
			continue
		}

		path, err := exec.LookPath(name)
		if err == nil {
			fmt.Fprintln(stdout, path)
		} else {
			exitCode = 1
		}
	}

	return exitCode
}
