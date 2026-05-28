package builtin

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "sed",
		Description: "流编辑器，用于过滤和转换文本",
		Usage:       "sed [-n] [-i[SUFFIX]] [-e script] 'command' [file...]",
		Help: `支持替换(s)、删除(d)、打印(p)命令，可选地址范围。

命令格式:
  s/old/new/flags   替换 (默认替换第一个; g 标志替换全部)
  d                 删除匹配行
  p                 打印匹配行 (常与 -n 配合)

地址格式:
  N                 行号
  $                 最后一行
  /pattern/         正则匹配
  addr1,addr2       地址范围 (含首尾)

选项:
  -n                取消默认输出，仅显式打印
  -i[SUFFIX]        直接修改文件 (可指定备份后缀, 如 -i.bak)
  -e script         添加编辑命令

示例:
  sed s/foo/bar/ file.txt
  sed -n /error/p log.txt
  sed -i.bak s/old/new/ file.txt
  sed 1,10d file.txt
  sed -e s/a/b/ -e 5d file.txt`,
		Action: Sed,
	})
}

type sedAddr struct {
	kind    byte   // 'n'=line num, 'r'=regex, '$'=last
	num     int    // line number for 'n'
	pattern string // regex pattern for 'r'
	re      *regexp.Regexp
}

type sedCommand struct {
	addrStart *sedAddr
	addrEnd   *sedAddr
	cmd       byte   // 's', 'd', 'p'
	oldStr    string // for 's'
	newStr    string // for 's'
	global    bool   // 'g' flag for 's'
}

func Sed(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if HandleBuiltinHelp(Builtins["sed"], args, stdout) {
		return 0
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, "sed: expression or -e required")
		return 1
	}

	inPlace := false
	backupSuffix := ""
	quiet := false
	hasScript := false
	var expressions []string
	var positional []string

	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "--" {
			i++
			break
		}
		if len(arg) > 0 && arg[0] == '-' && arg != "-" {
			switch {
			case arg == "-n" || arg == "--quiet" || arg == "--silent":
				quiet = true
				i++
			case arg == "-e" || arg == "--expression":
				if i+1 >= len(args) {
					fmt.Fprintln(stderr, "sed: -e requires an argument")
					return 1
				}
				expressions = append(expressions, args[i+1])
				hasScript = true
				i += 2
			case strings.HasPrefix(arg, "-i"):
				suffix := arg[2:]
				inPlace = true
				if suffix != "" {
					if suffix[0] != '.' {
						suffix = "." + suffix
					}
					backupSuffix = suffix
				}
				i++
			default:
				fmt.Fprintf(stderr, "sed: unknown option: %s\n", arg)
				return 1
			}
		} else {
			positional = append(positional, arg)
			i++
		}
	}

	positional = append(positional, args[i:]...)

	if !hasScript {
		if len(positional) == 0 {
			fmt.Fprintln(stderr, "sed: expression required")
			return 1
		}
		expressions = append(expressions, positional[0])
		positional = positional[1:]
	}

	if len(expressions) == 0 {
		fmt.Fprintln(stderr, "sed: expression required")
		return 1
	}

	var commands []*sedCommand
	for _, expr := range expressions {
		cmds, err := parseSedExpr(expr)
		if err != nil {
			fmt.Fprintf(stderr, "sed: %v\n", err)
			return 1
		}
		commands = append(commands, cmds...)
	}

	if len(commands) == 0 {
		fmt.Fprintln(stderr, "sed: no valid command")
		return 1
	}

	if len(positional) == 0 {
		return runStream(stdin, stdout, stderr, commands, quiet)
	}

	exitCode := 0
	for _, filename := range positional {
		code := runFile(filename, stdout, stderr, commands, quiet, inPlace, backupSuffix)
		if code != 0 {
			exitCode = code
		}
	}
	return exitCode
}

func parseSedExpr(expr string) ([]*sedCommand, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty expression")
	}

	// Bare s/old/new/flags — no address prefix
	if strings.HasPrefix(expr, "s") && len(expr) > 2 {
		delim := expr[1]
		end1 := strings.IndexByte(expr[2:], delim)
		if end1 >= 0 {
			end2 := strings.IndexByte(expr[2+end1+1:], delim)
			if end2 >= 0 {
				oldStr := expr[2 : 2+end1]
				newStr := expr[2+end1+1 : 2+end1+1+end2]
				flags := expr[2+end1+1+end2+1:]
				global := parseFlags(flags)
				return []*sedCommand{{
					cmd:    's',
					oldStr: oldStr,
					newStr: newStr,
					global: global,
				}}, nil
			}
		}
	}

	var cmds []*sedCommand
	pos := 0

	for pos < len(expr) {
		for pos < len(expr) && (expr[pos] == ';' || expr[pos] == ' ' || expr[pos] == '\t') {
			pos++
		}
		if pos >= len(expr) {
			break
		}

		addrStart, addrEnd, newPos, err := parseAddress(expr, pos)
		if err != nil {
			return nil, err
		}
		pos = newPos

		if pos >= len(expr) {
			return nil, fmt.Errorf("missing command after address")
		}

		cmdLetter := expr[pos]
		pos++

		switch cmdLetter {
		case 's':
			if pos >= len(expr) {
				return nil, fmt.Errorf("incomplete s command")
			}
			delim := expr[pos]
			pos++
			end1 := strings.IndexByte(expr[pos:], delim)
			if end1 < 0 {
				return nil, fmt.Errorf("unterminated s command")
			}
			oldStr := expr[pos : pos+end1]
			pos += end1 + 1
			end2 := strings.IndexByte(expr[pos:], delim)
			if end2 < 0 {
				return nil, fmt.Errorf("unterminated s command")
			}
			newStr := expr[pos : pos+end2]
			pos += end2 + 1
			global := parseFlags(expr[pos:])
			cmds = append(cmds, &sedCommand{
				addrStart: addrStart,
				addrEnd:   addrEnd,
				cmd:       's',
				oldStr:    oldStr,
				newStr:    newStr,
				global:    global,
			})

		case 'd':
			cmds = append(cmds, &sedCommand{
				addrStart: addrStart,
				addrEnd:   addrEnd,
				cmd:       'd',
			})

		case 'p':
			cmds = append(cmds, &sedCommand{
				addrStart: addrStart,
				addrEnd:   addrEnd,
				cmd:       'p',
			})

		default:
			return nil, fmt.Errorf("unknown command: %c", cmdLetter)
		}

		if pos < len(expr) && expr[pos] == ';' {
			pos++
		}
	}

	return cmds, nil
}

func parseAddress(expr string, pos int) (start, end *sedAddr, newPos int, err error) {
	for pos < len(expr) && (expr[pos] == ' ' || expr[pos] == '\t') {
		pos++
	}
	if pos >= len(expr) {
		return nil, nil, pos, nil
	}

	var addr *sedAddr
	switch {
	case expr[pos] == '$':
		addr = &sedAddr{kind: '$'}
		pos++
	case expr[pos] == '/':
		endIdx := strings.IndexByte(expr[pos+1:], '/')
		if endIdx < 0 {
			return nil, nil, pos, fmt.Errorf("unterminated regex address")
		}
		pattern := expr[pos+1 : pos+1+endIdx]
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, nil, pos, fmt.Errorf("invalid regex: %v", err)
		}
		addr = &sedAddr{kind: 'r', pattern: pattern, re: re}
		pos += endIdx + 2
	case expr[pos] >= '0' && expr[pos] <= '9':
		numStart := pos
		for pos < len(expr) && expr[pos] >= '0' && expr[pos] <= '9' {
			pos++
		}
		n, err := strconv.Atoi(expr[numStart:pos])
		if err != nil {
			return nil, nil, pos, fmt.Errorf("invalid address: %s", expr[numStart:pos])
		}
		addr = &sedAddr{kind: 'n', num: n}
	default:
		return nil, nil, pos, nil
	}

	if pos < len(expr) && expr[pos] == ',' {
		start = addr
		pos++
		for pos < len(expr) && (expr[pos] == ' ' || expr[pos] == '\t') {
			pos++
		}
		end, _, newPos2, err := parseAddress(expr, pos)
		if err != nil {
			return nil, nil, pos, err
		}
		if end == nil {
			return nil, nil, pos, fmt.Errorf("incomplete range after ','")
		}
		return start, end, newPos2, nil
	}

	return addr, nil, pos, nil
}

func runFile(filename string, stdout, stderr io.Writer, cmds []*sedCommand, quiet bool, inPlace bool, backupSuffix string) int {
	if !inPlace {
		f, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(stderr, "sed: %s: %v\n", filename, err)
			return 1
		}
		defer f.Close()
		return runStream(f, stdout, stderr, cmds, quiet)
	}

	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(stderr, "sed: %s: %v\n", filename, err)
		return 1
	}

	var buf strings.Builder
	code := runStream(f, &buf, stderr, cmds, quiet)
	f.Close()

	if code != 0 || buf.Len() == 0 {
		return code
	}

	if backupSuffix != "" {
		if err := copyFile(filename, filename+backupSuffix); err != nil {
			fmt.Fprintf(stderr, "sed: %s: cannot create backup: %v\n", filename, err)
			return 1
		}
	}

	if err := os.WriteFile(filename, []byte(buf.String()), 0644); err != nil {
		fmt.Fprintf(stderr, "sed: %s: cannot write: %v\n", filename, err)
		return 1
	}
	return 0
}

func runStream(r io.Reader, stdout, stderr io.Writer, cmds []*sedCommand, quiet bool) int {
	data, err := io.ReadAll(r)
	if err != nil {
		fmt.Fprintf(stderr, "sed: read error: %v\n", err)
		return 1
	}

	text := string(data)
	if text == "" {
		return 0
	}

	lines := strings.Split(text, "\n")
	totalLines := len(lines)
	if totalLines > 0 && lines[totalLines-1] == "" {
		totalLines--
		lines = lines[:totalLines]
	}

	rangeActive := make([]bool, len(cmds))

	for i, line := range lines {
		lineNum := i + 1
		deleted := false

		for ci, cmd := range cmds {
			if !lineMatches(cmd, lineNum, totalLines, line, &rangeActive[ci]) {
				continue
			}
			switch cmd.cmd {
			case 's':
				if cmd.global {
					line = strings.ReplaceAll(line, cmd.oldStr, cmd.newStr)
				} else {
					line = strings.Replace(line, cmd.oldStr, cmd.newStr, 1)
				}
			case 'd':
				deleted = true
			case 'p':
				fmt.Fprintln(stdout, line)
			}
			if deleted {
				break
			}
		}

		if deleted {
			continue
		}
		if !quiet {
			fmt.Fprintln(stdout, line)
		}
	}

	return 0
}

func lineMatches(cmd *sedCommand, lineNum, totalLines int, line string, rangeActive *bool) bool {
	if cmd.addrStart == nil {
		return true
	}

	startMatch := addrMatch(cmd.addrStart, lineNum, totalLines, line)
	if cmd.addrEnd == nil {
		return startMatch
	}

	if !*rangeActive && startMatch {
		*rangeActive = true
		return true
	}

	if *rangeActive {
		endMatch := addrMatch(cmd.addrEnd, lineNum, totalLines, line)
		if endMatch {
			*rangeActive = false
		}
		return true
	}

	return false
}

func addrMatch(addr *sedAddr, lineNum, totalLines int, line string) bool {
	switch addr.kind {
	case 'n':
		return lineNum == addr.num
	case '$':
		return totalLines > 0 && lineNum == totalLines
	case 'r':
		if addr.re == nil {
			return false
		}
		return addr.re.MatchString(line)
	default:
		return false
	}
}

func parseFlags(flags string) bool {
	global := false
	for _, f := range flags {
		if f == 'g' {
			global = true
		}
	}
	return global
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	_, err = io.Copy(d, s)
	return err
}
