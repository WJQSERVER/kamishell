package builtin

import (
	"fmt"
	"io"
	"os"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "env",
		Description: "显示环境变量",
		Usage:       "env",
		Help:        "打印当前进程环境变量列表。",
		Action:      Env,
	})
}

func Env(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if HandleBuiltinHelp(Builtins["env"], args, stdout) {
		return 0
	}
	for _, e := range os.Environ() {
		fmt.Fprintln(stdout, e)
	}
	return 0
}
