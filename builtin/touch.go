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
		Action:      Touch,
	})
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
