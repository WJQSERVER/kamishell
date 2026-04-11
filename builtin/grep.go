package builtin

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "grep",
		Description: "在文件中搜索模式（支持正则表达式）",
		Usage:       "grep [-i] [-n] [-v] [-w] [-x] [-c] [-l] [-L] [-q] pattern [file...]",
		Help: `在输入流或文件中按正则表达式模式搜索内容。

选项:
  -i, --ignore-case    忽略大小写
  -n, --line-number    显示行号
  -v, --invert-match   反向匹配，显示不匹配的行
  -w, --word-regexp    仅匹配完整单词
  -x, --line-regexp    仅匹配整行
  -c, --count          只显示匹配行计数
  -l, --files-with-matches    只显示包含匹配的文件名
  -L, --files-without-match   只显示不包含匹配的文件名
  -q, --quiet          静默模式，不输出，用于脚本条件判断

示例:
  grep "func.*main" *.go
  grep -i "error" log.txt
  grep -n "TODO" *.go
  cat file.txt | grep -v "^#"
  grep -w "test" file.txt
  grep -c "pattern" file.txt`,
		Action: Grep,
	})
}

type grepOptions struct {
	ignoreCase   bool
	lineNumber   bool
	invertMatch  bool
	wordRegexp   bool
	lineRegexp   bool
	count        bool
	filesMatch   bool
	filesNoMatch bool
	quiet        bool
}

func Grep(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	args = PreprocessArgs(args)

	if HandleBuiltinHelp(Builtins["grep"], args, stdout) {
		return 0
	}

	fs := flag.NewFlagSet("grep", flag.ContinueOnError)
	fs.SetOutput(stderr)

	opts := &grepOptions{}
	fs.BoolVar(&opts.ignoreCase, "i", false, "ignore case distinctions")
	fs.BoolVar(&opts.ignoreCase, "ignore-case", false, "ignore case distinctions")
	fs.BoolVar(&opts.lineNumber, "n", false, "print line number")
	fs.BoolVar(&opts.lineNumber, "line-number", false, "print line number")
	fs.BoolVar(&opts.invertMatch, "v", false, "invert match")
	fs.BoolVar(&opts.invertMatch, "invert-match", false, "invert match")
	fs.BoolVar(&opts.wordRegexp, "w", false, "match whole words")
	fs.BoolVar(&opts.wordRegexp, "word-regexp", false, "match whole words")
	fs.BoolVar(&opts.lineRegexp, "x", false, "match whole lines")
	fs.BoolVar(&opts.lineRegexp, "line-regexp", false, "match whole lines")
	fs.BoolVar(&opts.count, "c", false, "print count of matching lines")
	fs.BoolVar(&opts.count, "count", false, "print count of matching lines")
	fs.BoolVar(&opts.filesMatch, "l", false, "list filenames with matches")
	fs.BoolVar(&opts.filesMatch, "files-with-matches", false, "list filenames with matches")
	fs.BoolVar(&opts.filesNoMatch, "L", false, "list filenames without matches")
	fs.BoolVar(&opts.filesNoMatch, "files-without-match", false, "list filenames without matches")
	fs.BoolVar(&opts.quiet, "q", false, "quiet mode")
	fs.BoolVar(&opts.quiet, "quiet", false, "quiet mode")
	fs.BoolVar(&opts.quiet, "silent", false, "silent mode")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	remainingArgs := fs.Args()
	if len(remainingArgs) == 0 {
		fmt.Fprintln(stderr, "grep: search pattern required")
		return 1
	}

	patternStr := remainingArgs[0]
	files := remainingArgs[1:]

	// 构建正则表达式
	pattern, err := buildPattern(patternStr, opts)
	if err != nil {
		fmt.Fprintf(stderr, "grep: invalid pattern: %v\n", err)
		return 1
	}

	if len(files) == 0 {
		return grepReader(stdin, pattern, opts, stdout, stderr, "")
	}

	exitCode := 0
	foundMatch := false
	foundNoMatch := false
	for _, filename := range files {
		f, err := os.Open(filename)
		if err != nil {
			if !opts.quiet {
				fmt.Fprintf(stderr, "grep: %s: %v\n", filename, err)
			}
			exitCode = 1
			continue
		}

		prefix := ""
		if opts.filesMatch || opts.filesNoMatch {
			prefix = filename + ":"
		} else if len(files) > 1 {
			prefix = filename + ":"
		}

		result := grepReader(f, pattern, opts, stdout, stderr, prefix)
		f.Close()

		// -l 选项: 只要有匹配就记录
		if opts.filesMatch {
			if result == 0 {
				foundMatch = true
			}
		} else if opts.filesNoMatch {
			// -L 选项: 只要有没有匹配的文件就记录
			if result == 0 {
				foundNoMatch = true
			}
		} else if result != 0 {
			exitCode = 1
		}
	}

	if opts.filesMatch {
		if foundMatch {
			return 0
		}
		return 1
	}

	if opts.filesNoMatch {
		if foundNoMatch {
			return 0
		}
		return 1
	}

	return exitCode
}

func buildPattern(patternStr string, opts *grepOptions) (*regexp.Regexp, error) {
	// 根据选项调整模式
	adjustedPattern := patternStr

	if opts.wordRegexp {
		// 单词边界匹配
		adjustedPattern = "\\b" + regexp.QuoteMeta(patternStr) + "\\b"
	} else if opts.lineRegexp {
		// 整行匹配
		adjustedPattern = "^" + regexp.QuoteMeta(patternStr) + "$"
	}

	// 正则表达式选项
	regexOpts := ""
	if opts.ignoreCase {
		regexOpts = "(?i)"
	}

	finalPattern := regexOpts + adjustedPattern
	return regexp.Compile(finalPattern)
}

func grepReader(r io.Reader, pattern *regexp.Regexp, opts *grepOptions, stdout, stderr io.Writer, prefix string) int {
	scanner := bufio.NewScanner(r)
	lineNum := 0
	matchCount := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		matches := pattern.MatchString(line)

		// 处理反向匹配
		if opts.invertMatch {
			matches = !matches
		}

		if matches {
			matchCount++

			// -l: 只显示包含匹配的文件名
			if opts.filesMatch {
				// 提取文件名（去掉末尾的冒号）
				filename := strings.TrimSuffix(prefix, ":")
				fmt.Fprintln(stdout, filename)
				return 0 // 找到一个匹配就返回
			}

			// -q: 静默模式
			if opts.quiet {
				return 0 // 找到匹配就返回
			}

			// -L: 找到匹配，不输出，返回1表示这个文件不匹配-L条件
			if opts.filesNoMatch {
				return 1
			}

			// -c: 只显示计数
			if opts.count {
				continue // 继续计数，不输出
			}

			// 构建输出行
			output := ""
			if prefix != "" {
				output += prefix
			}
			if opts.lineNumber {
				output += fmt.Sprintf("%d:", lineNum)
			}
			output += line
			fmt.Fprintln(stdout, output)
		}
	}

	// -L: 显示不包含匹配的文件名
	if opts.filesNoMatch {
		if matchCount == 0 {
			filename := strings.TrimSuffix(prefix, ":")
			fmt.Fprintln(stdout, filename)
			return 0
		}
		return 1
	}

	// 如果没有匹配且使用了 -l，返回非零
	if opts.filesMatch && matchCount == 0 {
		return 1
	}

	// -q: 静默模式，根据是否有匹配返回不同的退出码
	if opts.quiet {
		if matchCount > 0 {
			return 0
		}
		return 1
	}

	// -c: 输出匹配计数
	if opts.count && !opts.quiet {
		if prefix != "" {
			fmt.Fprintf(stdout, "%s%d\n", prefix, matchCount)
		} else {
			fmt.Fprintln(stdout, matchCount)
		}
	}

	if err := scanner.Err(); err != nil {
		if !opts.quiet {
			fmt.Fprintf(stderr, "grep: error: %v\n", err)
		}
		return 1
	}

	// 如果没有匹配且使用了 -l 或 -q，返回非零
	if opts.filesMatch && matchCount == 0 {
		return 1
	}

	return 0
}
