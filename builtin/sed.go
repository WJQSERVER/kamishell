package builtin

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "sed",
		Description: "流编辑器，用于过滤和转换文本",
		Action:      Sed,
	})
}

func Sed(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "sed: replacement expression required (e.g., s/old/new/)")
		return 1
	}

	expr := args[0]
	if !strings.HasPrefix(expr, "s/") {
		fmt.Fprintln(stderr, "sed: only simple 's/old/new/' substitution is supported")
		return 1
	}

	parts := strings.Split(expr, "/")
	if len(parts) < 3 {
		fmt.Fprintln(stderr, "sed: invalid substitution expression")
		return 1
	}

	old := parts[1]
	new := parts[2]
	files := args[1:]

	if len(files) == 0 {
		return sedReader(stdin, old, new, stdout, stderr)
	}

	exitCode := 0
	for _, filename := range files {
		f, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(stderr, "sed: %s: %v\n", filename, err)
			exitCode = 1
			continue
		}
		if sedReader(f, old, new, stdout, stderr) != 0 {
			exitCode = 1
		}
		f.Close()
	}

	return exitCode
}

func sedReader(r io.Reader, old, new string, stdout, stderr io.Writer) int {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		newLine := strings.ReplaceAll(line, old, new)
		fmt.Fprintln(stdout, newLine)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "sed: error: %v\n", err)
		return 1
	}
	return 0
}
