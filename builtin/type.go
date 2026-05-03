package builtin

import (
	"fmt"
	"io"
	"os/exec"
	"reflect"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "type",
		Description: "显示命令类型",
		Usage:       "type name...",
		Help:        "显示名称是函数、变量、内建命令还是外部可执行文件。",
		Action:      Type,
	})
	SetArgCompleter("type", completeCommandNames)
}

func Type(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if HandleBuiltinHelp(Builtins["type"], args, stdout) {
		return 0
	}
	if len(args) == 0 {
		return 0
	}

	exitCode := 0
	for _, name := range args {
		found := false

		// 1. Check functions and other environment variables
		val, ok := env.Get(name)
		if ok {
			v := reflect.ValueOf(val)
			method := v.MethodByName("Type")
			if method.IsValid() {
				results := method.Call(nil)
				if len(results) > 0 {
					typeStr := fmt.Sprintf("%v", results[0].Interface())
					if typeStr == "FUNCTION" {
						fmt.Fprintf(stdout, "%s is a function\n", name)
						found = true
					} else {
						fmt.Fprintf(stdout, "%s is a variable of type %s\n", name, typeStr)
						found = true
					}
				}
			}
		}

		if found {
			continue
		}

		// 2. Check builtins
		if _, ok := Builtins[name]; ok {
			fmt.Fprintf(stdout, "%s is a shell builtin\n", name)
			found = true
		}

		if found {
			continue
		}

		// 3. Check external commands
		path, err := exec.LookPath(name)
		if err == nil {
			fmt.Fprintf(stdout, "%s is %s\n", name, path)
			found = true
		}

		if !found {
			fmt.Fprintf(stderr, "type: %s: not found\n", name)
			exitCode = 1
		}
	}

	return exitCode
}
