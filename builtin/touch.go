package builtin

import (
	"fmt"
	"io"
	"os"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "touch",
		Description: "创建空文件或更新时间戳",
		Usage:       "touch file...",
		Help:        "为每个目标创建空文件；如果文件已存在，则仅打开并关闭它。",
		Action:      Touch,
	})
}

func Touch(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if HandleBuiltinHelp(Builtins["touch"], args, stdout) {
		return 0
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, "touch: missing file operand")
		return 1
	}
	for _, arg := range args {
		f, err := os.OpenFile(arg, os.O_RDONLY|os.O_CREATE, 0644)
		if err != nil {
			fmt.Fprintf(stderr, "touch: %v\n", err)
			return 1
		}
		if err := f.Close(); err != nil {
			fmt.Fprintf(stderr, "touch: %v\n", err)
			return 1
		}
	}
	return 0
}
