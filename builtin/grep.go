package builtin

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func init() {
	RegisterBuiltin("grep", Grep)
}

func Grep(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "grep: search pattern required")
		return 1
	}

	pattern := args[0]
	files := args[1:]

	if len(files) == 0 {
		return grepReader(stdin, pattern, stdout, stderr, "")
	}

	exitCode := 0
	for _, filename := range files {
		f, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(stderr, "grep: %s: %v\n", filename, err)
			exitCode = 1
			continue
		}
		prefix := ""
		if len(files) > 1 {
			prefix = filename + ":"
		}
		if grepReader(f, pattern, stdout, stderr, prefix) != 0 {
			exitCode = 1
		}
		f.Close()
	}

	return exitCode
}

func grepReader(r io.Reader, pattern string, stdout, stderr io.Writer, prefix string) int {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, pattern) {
			fmt.Fprintf(stdout, "%s%s\n", prefix, line)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "grep: error: %v\n", err)
		return 1
	}
	return 0
}
