package builtin

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "grep",
		Description: "在文件中搜索模式（支持正则表达式）",
		Usage:       "grep [-i] [-n] [-v] [-w] [-x] [-c] [-l] [-L] [-q] [-r] pattern [file...]",
		Help: `在输入流或文件中按正则表达式模式搜索内容。

选项:
  -i, --ignore-case         忽略大小写
  -n, --line-number         显示行号
  -v, --invert-match        反向匹配，显示不匹配的行
  -w, --word-regexp         仅匹配完整单词
  -x, --line-regexp         仅匹配整行
  -c, --count               只显示匹配行计数
  -l, --files-with-matches  只显示包含匹配的文件名
  -L, --files-without-match 只显示不包含匹配的文件名
  -q, --quiet               静默模式
  -r, --recursive           递归搜索目录

示例:
  grep "func.*main" *.go
  grep -i "error" log.txt
  grep -n "TODO" *.go
  cat file.txt | grep -v "^#"
  grep -w "test" file.txt
  grep -c "pattern" file.txt
  grep -r "pattern" src/`,
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
	recursive    bool
}

type grepResult struct {
	matched      bool
	selectedFile bool
	err          bool
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
	fs.BoolVar(&opts.recursive, "r", false, "recursive search")
	fs.BoolVar(&opts.recursive, "R", false, "recursive search")
	fs.BoolVar(&opts.recursive, "recursive", false, "recursive search")

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
		result := grepReader(stdin, pattern, opts, stdout, stderr, "")
		if result.err {
			return 1
		}
		if result.matched {
			return 0
		}
		return 1
	}

	hadError := false
	foundMatch := false
	selectedFileFound := false

	// 收集所有要搜索的文件
	var filesToSearch []string
	for _, filename := range files {
		info, err := os.Stat(filename)
		if err != nil {
			if !opts.quiet {
				fmt.Fprintf(stderr, "grep: %s: %v\n", filename, err)
			}
			hadError = true
			continue
		}

		if info.IsDir() {
			if opts.recursive {
				// 递归收集目录中的所有文件
				err := filepath.WalkDir(filename, func(path string, d iofs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if !d.IsDir() {
						filesToSearch = append(filesToSearch, path)
					}
					return nil
				})
				if err != nil {
					if !opts.quiet {
						fmt.Fprintf(stderr, "grep: %s: %v\n", filename, err)
					}
					hadError = true
				}
			} else {
				if !opts.quiet {
					fmt.Fprintf(stderr, "grep: %s: Is a directory\n", filename)
				}
				hadError = true
			}
		} else {
			filesToSearch = append(filesToSearch, filename)
		}
	}

	// 搜索所有收集到的文件
	for _, filename := range filesToSearch {
		f, err := os.Open(filename)
		if err != nil {
			if !opts.quiet {
				fmt.Fprintf(stderr, "grep: %s: %v\n", filename, err)
			}
			hadError = true
			continue
		}

		prefix := ""
		if opts.filesMatch || opts.filesNoMatch || opts.recursive || len(filesToSearch) > 1 {
			prefix = filename + ":"
		}

		result := grepReader(f, pattern, opts, stdout, stderr, prefix)
		if closeErr := f.Close(); closeErr != nil {
			if !opts.quiet {
				fmt.Fprintf(stderr, "grep: %s: %v\n", filename, closeErr)
			}
			hadError = true
		}

		if result.err {
			hadError = true
			continue
		}

		if result.selectedFile {
			selectedFileFound = true
		}

		if result.matched {
			foundMatch = true
			if opts.quiet {
				break
			}
		}
	}

	if hadError {
		return 1
	}

	if opts.filesNoMatch {
		if selectedFileFound {
			return 0
		}
		return 1
	}

	if foundMatch {
		return 0
	}

	return 1
}

func buildPattern(patternStr string, opts *grepOptions) (*regexp.Regexp, error) {
	// 根据选项调整模式
	adjustedPattern := patternStr

	if opts.wordRegexp {
		adjustedPattern = "\\b(" + patternStr + ")\\b"
	} else if opts.lineRegexp {
		adjustedPattern = "^(" + patternStr + ")$"
	}

	// 正则表达式选项
	regexOpts := ""
	if opts.ignoreCase {
		regexOpts = "(?i)"
	}

	finalPattern := regexOpts + adjustedPattern
	return regexp.Compile(finalPattern)
}

func grepReader(r io.Reader, pattern *regexp.Regexp, opts *grepOptions, stdout, stderr io.Writer, prefix string) grepResult {
	reader := bufio.NewReader(r)
	lineNum := 0
	matchCount := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			if !opts.quiet {
				fmt.Fprintf(stderr, "grep: error: %v\n", err)
			}
			return grepResult{matched: matchCount > 0, err: true}
		}
		if err == io.EOF && len(line) == 0 {
			break
		}

		lineNum++
		line = trimTrailingLineEnding(line)
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
				return grepResult{matched: true}
			}

			// -q: 静默模式
			if opts.quiet {
				return grepResult{matched: true}
			}

			// -L: 找到匹配，不输出
			if opts.filesNoMatch {
				if err == io.EOF {
					break
				}
				continue
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

		if err == io.EOF {
			break
		}
	}

	// -L: 显示不包含匹配的文件名
	if opts.filesNoMatch {
		if matchCount == 0 {
			filename := strings.TrimSuffix(prefix, ":")
			fmt.Fprintln(stdout, filename)
			return grepResult{selectedFile: true}
		}
		return grepResult{matched: true}
	}

	// -c: 输出匹配计数
	if opts.count && !opts.quiet {
		if prefix != "" {
			fmt.Fprintf(stdout, "%s%d\n", prefix, matchCount)
		} else {
			fmt.Fprintln(stdout, matchCount)
		}
	}

	return grepResult{matched: matchCount > 0}
}

func trimTrailingLineEnding(line string) string {
	if strings.HasSuffix(line, "\r\n") {
		return strings.TrimSuffix(line, "\r\n")
	}
	return strings.TrimSuffix(line, "\n")
}
