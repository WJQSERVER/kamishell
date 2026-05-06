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
		Usage:       "sed s/old/new/ [file...]",
		Help: `当前仅支持简单的全局替换表达式 s/old/new/。

示例:
  sed s/foo/bar/ file.txt
  print "foo" | sed s/foo/bar/`,
		Action: Sed,
	})
}

func Sed(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if HandleBuiltinHelp(Builtins["sed"], args, stdout) {
		return 0
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, "sed: replacement expression required (e.g., s/old/new/)")
		return 1
	}

	expr := args[0]
	if len(expr) < 2 || expr[0] != 's' {
		fmt.Fprintln(stderr, "sed: only simple 's/old/new/' substitution is supported")
		return 1
	}

	// Use the character after 's' as the delimiter (usually '/')
	delim := rune(expr[1])

	// Find the second delimiter (end of old pattern)
 secondIdx := -1
 for i := 2; i < len(expr); i++ {
		if rune(expr[i]) == delim {
			secondIdx = i
			break
		}
	}
	if secondIdx < 0 {
		fmt.Fprintln(stderr, "sed: invalid substitution expression: missing second delimiter")
		return 1
	}

	// Find the third delimiter (end of new pattern)
 thirdIdx := -1
 for i := secondIdx + 1; i < len(expr); i++ {
		if rune(expr[i]) == delim {
			thirdIdx = i
			break
		}
	}
	if thirdIdx < 0 {
		fmt.Fprintln(stderr, "sed: invalid substitution expression: missing third delimiter")
		return 1
	}

	old := expr[2:secondIdx]
	new := expr[secondIdx+1 : thirdIdx]
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
