package builtin

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "export",
		Description: "设置环境变量",
		Usage:       "export name=value ...",
		Help:        "设置环境变量；不带参数时等同于 env。",
		Action:      Export,
	})
}

func Export(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if HandleBuiltinHelp(Builtins["export"], args, stdout) {
		return 0
	}
	if len(args) == 0 {
		return Env(args, env, stdin, stdout, stderr)
	}

	for _, arg := range args {
		pair := strings.SplitN(arg, "=", 2)
		if len(pair) != 2 {
			fmt.Fprintf(stderr, "export: usage: export name=value\n")
			return 1
		}
		os.Setenv(pair[0], pair[1])
		env.Set(pair[0], pair[1])
	}
	return 0
}
