package builtin

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "cat",
		Description: "连接文件并打印到标准输出",
		Usage:       "cat [-n] [-b] [-s] [-E] [-T] [-v] [-A] [-e] [-t] [-u] [file...]",
		Help: `按顺序输出文件内容；文件名为 - 时从标准输入读取。

选项:
  -n, --number             对所有输出行编号
  -b, --number-nonblank    对非空输出行编号
  -s, --squeeze-blank      压缩连续空行
  -E, --show-ends          在每行行尾显示 $
  -T, --show-tabs          将制表符显示为 ^I
  -v, --show-nonprinting   使用 ^ 和 M- 符号显示非打印字符
  -A, --show-all           等价于 -vET
  -e                       等价于 -vE
  -t                       等价于 -vT
  -u                       忽略；为 POSIX 兼容性

示例:
  cat file.txt
  cat -n file.txt
  cat -b file.txt
  cat -E file.txt
  cat -A file.txt`,
		Action: Cat,
	})
}

type catOptions struct {
	number          bool
	numberNonblank  bool
	squeezeBlank    bool
	showEnds        bool
	showTabs        bool
	showNonprinting bool
}

func Cat(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	args = PreprocessArgs(args)

	if HandleBuiltinHelp(Builtins["cat"], args, stdout) {
		return 0
	}

	fs := flag.NewFlagSet("cat", flag.ContinueOnError)
	fs.SetOutput(stderr)

	opts := &catOptions{}
	fs.BoolVar(&opts.number, "n", false, "number all output lines")
	fs.BoolVar(&opts.number, "number", false, "number all output lines")
	fs.BoolVar(&opts.numberNonblank, "b", false, "number nonempty output lines")
	fs.BoolVar(&opts.numberNonblank, "number-nonblank", false, "number nonempty output lines")
	fs.BoolVar(&opts.squeezeBlank, "s", false, "suppress repeated empty output lines")
	fs.BoolVar(&opts.squeezeBlank, "squeeze-blank", false, "suppress repeated empty output lines")
	fs.BoolVar(&opts.showEnds, "E", false, "display $ at end of each line")
	fs.BoolVar(&opts.showEnds, "show-ends", false, "display $ at end of each line")
	fs.BoolVar(&opts.showTabs, "T", false, "display TAB characters as ^I")
	fs.BoolVar(&opts.showTabs, "show-tabs", false, "display TAB characters as ^I")
	fs.BoolVar(&opts.showNonprinting, "v", false, "use ^ and M- notation for non-printing characters")
	fs.BoolVar(&opts.showNonprinting, "show-nonprinting", false, "use ^ and M- notation for non-printing characters")

	// -A, --show-all 等价于 -vET
	showAll := fs.Bool("A", false, "equivalent to -vET")
	fs.BoolVar(showAll, "show-all", false, "equivalent to -vET")

	// -e 等价于 -vE
	equivVE := fs.Bool("e", false, "equivalent to -vE")

	// -t 等价于 -vT
	equivVT := fs.Bool("t", false, "equivalent to -vT")

	// -u 是 POSIX 兼容性选项，忽略
	_ = fs.Bool("u", false, "ignored; for POSIX compatibility")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	// 处理组合选项
	if *showAll {
		opts.showNonprinting = true
		opts.showTabs = true
		opts.showEnds = true
	}
	if *equivVE {
		opts.showNonprinting = true
		opts.showEnds = true
	}
	if *equivVT {
		opts.showNonprinting = true
		opts.showTabs = true
	}

	targets := fs.Args()
	if len(targets) == 0 {
		return catReader(stdin, stdout, stderr, opts)
	}

	exitCode := 0
	for _, target := range targets {
		var r io.Reader
		var closer io.Closer

		if target == "-" {
			r = stdin
		} else {
			f, err := os.Open(target)
			if err != nil {
				fmt.Fprintf(stderr, "cat: %s: %v\n", target, err)
				exitCode = 1
				continue
			}
			r = f
			closer = f
		}

		result := catReader(r, stdout, stderr, opts)
		if closer != nil {
			if err := closer.Close(); err != nil && result == 0 {
				result = 1
				fmt.Fprintf(stderr, "cat: %s: %v\n", target, err)
			}
		}
		if result != 0 {
			exitCode = result
		}
	}

	return exitCode
}

func catReader(r io.Reader, stdout, stderr io.Writer, opts *catOptions) int {
	if !requiresFormattedCatOutput(opts) {
		if _, err := io.Copy(stdout, r); err != nil {
			fmt.Fprintf(stderr, "cat: %v\n", err)
			return 1
		}
		return 0
	}

	reader := bufio.NewReader(r)
	lineNum := 0
	lastLineWasEmpty := false

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			fmt.Fprintf(stderr, "cat: %v\n", err)
			return 1
		}
		if err == io.EOF && len(line) == 0 {
			break
		}

		hadNewline := strings.HasSuffix(line, "\n")
		content := strings.TrimSuffix(line, "\n")
		isEmpty := len(content) == 0

		// 压缩空行
		if opts.squeezeBlank && isEmpty {
			if lastLineWasEmpty {
				if err == io.EOF {
					break
				}
				continue
			}
			lastLineWasEmpty = true
		} else {
			lastLineWasEmpty = isEmpty
		}

		// 行号处理
		// -b 和 -n 互斥，-b 优先级更高
		if opts.numberNonblank && !isEmpty {
			lineNum++
		} else if opts.number && !opts.numberNonblank {
			lineNum++
		}

		// 构建输出行
		var output bytes.Buffer

		// 添加行号
		if opts.numberNonblank {
			if !isEmpty {
				fmt.Fprintf(&output, "%6d\t", lineNum)
			}
		} else if opts.number {
			fmt.Fprintf(&output, "%6d\t", lineNum)
		}

		// 处理显示非打印字符
		if opts.showNonprinting || opts.showTabs {
			output.Write(processNonprinting([]byte(content), opts.showTabs, opts.showNonprinting))
		} else {
			output.WriteString(content)
		}

		// 显示行尾
		if opts.showEnds {
			output.WriteByte('$')
		}

		if hadNewline || opts.showEnds {
			output.WriteByte('\n')
		}
		if _, writeErr := stdout.Write(output.Bytes()); writeErr != nil {
			fmt.Fprintf(stderr, "cat: %v\n", writeErr)
			return 1
		}

		if err == io.EOF {
			break
		}
	}

	return 0
}

func requiresFormattedCatOutput(opts *catOptions) bool {
	return opts.number || opts.numberNonblank || opts.squeezeBlank || opts.showEnds || opts.showTabs || opts.showNonprinting
}

// processNonprinting 处理非打印字符显示
func processNonprinting(data []byte, showTabs, showNonprinting bool) []byte {
	result := make([]byte, 0, len(data))
	for _, b := range data {
		if b == '\t' && showTabs {
			result = append(result, '^', 'I')
			continue
		}
		if !showNonprinting {
			result = append(result, b)
			continue
		}

		switch {
		case b < 32 && b != '\t':
			result = append(result, '^', 'A'-1+b)
		case b == 127:
			result = append(result, '^', '?')
		case b >= 128:
			result = append(result, 'M', '-')
			low := b & 0x7f
			switch {
			case low < 32:
				result = append(result, '^', 'A'-1+low)
			case low == 127:
				result = append(result, '^', '?')
			default:
				result = append(result, low)
			}
		default:
			result = append(result, b)
		}
	}
	return result
}
