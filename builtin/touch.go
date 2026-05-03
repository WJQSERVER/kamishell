package builtin

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "touch",
		Description: "更新文件访问和修改时间戳",
		Usage:       "touch [-a] [-c] [-d TIME] [-m] [-r FILE] [-t TIME] [--time=WORD] file...",
		Help: `更新文件访问和修改时间戳。
如果不存在文件，则创建（除非指定了 -c）。

选项:
	  -a                        仅更改访问时间戳
	  -c, --no-create           不创建不存在的文件
	  -d, --date=TIME           使用指定时间而非当前时间
	                            TIME 可以是：YYYY-MM-DD, YYYY-MM-DD HH:MM:SS,
	                            @UNIX_TIMESTAMP, 或 "2 days ago" 等
	  -m                        仅更改修改时间戳
	  -r, --reference=FILE      使用参考文件的时间戳
	  -t [[CC]YY]MMDDhhmm[.ss]  使用指定时间戳（与 date -t 格式相同）
	  --time=WORD               更改指定时间: access, atime, use, modify, mtime

示例:
  touch file.txt
  touch -c nonexistent.txt
  touch -d "2023-01-01" file.txt
  touch -r ref.txt target.txt
  touch -t 202301011200 file.txt`,
		Action: Touch,
	})
}

type touchOptions struct {
	atime        bool
	noCreate     bool
	date         string
	mtime        bool
	reference    string
	timestamp    string
	timeSelector string
}

type touchTimeSource struct {
	atime time.Time
	mtime time.Time
}

func Touch(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	args = PreprocessArgs(args)

	if HandleBuiltinHelp(Builtins["touch"], args, stdout) {
		return 0
	}

	fs := flag.NewFlagSet("touch", flag.ContinueOnError)
	fs.SetOutput(stderr)

	m := RegisterMeta("touch")
	opts := &touchOptions{}
	BoolFlagVar(fs, m, &opts.atime, "a", "a", false, "change only the access time")
	BoolFlagVar(fs, m, &opts.noCreate, "no-create", "c", false, "do not create the file if it does not exist")
	StringFlagVar(fs, m, &opts.date, "date", "d", "", "use DATE instead of current time")
	BoolFlagVar(fs, m, &opts.mtime, "m", "m", false, "change only the modification time")
	StringFlagVar(fs, m, &opts.reference, "reference", "r", "", "use this file's times instead of current time")
	StringFlagVar(fs, m, &opts.timestamp, "t", "t", "", "use [[CC]YY]MMDDhhmm[.ss] instead of current time")
	StringFlagVar(fs, m, &opts.timeSelector, "time", "", "", "change the specified time: access, atime, use, modify, mtime")
	m.SetFlagCompleter("time", func(cmdName string, argIndex int, prefix string) []string {
		return []string{"access", "atime", "use", "modify", "mtime"}
	})

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if err := applyTouchTimeSelector(opts); err != nil {
		fmt.Fprintf(stderr, "touch: %v\n", err)
		return 1
	}

	files := fs.Args()
	if len(files) == 0 {
		fmt.Fprintln(stderr, "touch: missing file operand")
		return 1
	}

	// 解析目标时间戳
	targetTimes, err := parseTimeSource(opts)
	if err != nil {
		fmt.Fprintf(stderr, "touch: %v\n", err)
		return 1
	}

	// 如果没有指定时间，使用当前时间
	if targetTimes.atime.IsZero() && targetTimes.mtime.IsZero() {
		now := time.Now()
		targetTimes = touchTimeSource{atime: now, mtime: now}
	}

	exitCode := 0
	for _, file := range files {
		// 检查文件是否存在
		info, err := os.Stat(file)
		created := false
		if err != nil {
			if os.IsNotExist(err) {
				// 文件不存在
				if opts.noCreate {
					// -c 选项：不创建文件，也不报错
					continue
				}
				// 创建新文件
				f, createErr := os.Create(file)
				if createErr != nil {
					fmt.Fprintf(stderr, "touch: cannot touch '%s': %v\n", file, createErr)
					exitCode = 1
					continue
				}
				if closeErr := f.Close(); closeErr != nil {
					fmt.Fprintf(stderr, "touch: cannot touch '%s': %v\n", file, closeErr)
					exitCode = 1
					continue
				}
				info, err = os.Stat(file)
				if err != nil {
					fmt.Fprintf(stderr, "touch: cannot access '%s': %v\n", file, err)
					exitCode = 1
					continue
				}
				created = true
			}
			// 其他错误（如权限问题）
			if err != nil && !created {
				fmt.Fprintf(stderr, "touch: cannot access '%s': %v\n", file, err)
				exitCode = 1
				continue
			}
		}

		currentAtime, currentMtime := currentFileTimes(info)
		desiredAtime := targetTimes.atime
		desiredMtime := targetTimes.mtime

		if opts.atime && !opts.mtime {
			desiredMtime = currentMtime
		}

		if opts.mtime && !opts.atime {
			desiredAtime = currentAtime
		}

		if !opts.atime && !opts.mtime {
			desiredAtime = targetTimes.atime
			desiredMtime = targetTimes.mtime
		}

		// 修改文件时间戳
		err = os.Chtimes(file, desiredAtime, desiredMtime)
		if err != nil {
			fmt.Fprintf(stderr, "touch: setting times of '%s': %v\n", file, err)
			exitCode = 1
		}
	}

	return exitCode
}

// parseTimeSource 解析时间选项
func parseTimeSource(opts *touchOptions) (touchTimeSource, error) {
	// 优先级：-r > -t > -d > 默认当前时间

	// -r: 使用参考文件的时间戳
	if opts.reference != "" {
		info, err := os.Stat(opts.reference)
		if err != nil {
			if os.IsNotExist(err) {
				return touchTimeSource{}, fmt.Errorf("failed to get attributes of '%s': No such file or directory", opts.reference)
			}
			return touchTimeSource{}, fmt.Errorf("failed to get attributes of '%s': %v", opts.reference, err)
		}
		atime, mtime := currentFileTimes(info)
		return touchTimeSource{atime: atime, mtime: mtime}, nil
	}

	// -t: 使用指定时间戳格式 [[CC]YY]MMDDhhmm[.ss]
	if opts.timestamp != "" {
		t, err := parseTimestamp(opts.timestamp)
		if err != nil {
			return touchTimeSource{}, err
		}
		return touchTimeSource{atime: t, mtime: t}, nil
	}

	// -d: 使用日期字符串
	if opts.date != "" {
		t, err := parseDateString(opts.date)
		if err != nil {
			return touchTimeSource{}, err
		}
		return touchTimeSource{atime: t, mtime: t}, nil
	}

	// 没有指定时间，返回零值表示使用当前时间
	return touchTimeSource{}, nil
}

func applyTouchTimeSelector(opts *touchOptions) error {
	if opts.timeSelector == "" {
		return nil
	}

	switch strings.ToLower(opts.timeSelector) {
	case "access", "atime", "use":
		opts.atime = true
	case "modify", "mtime":
		opts.mtime = true
	default:
		return fmt.Errorf("invalid argument %q for --time", opts.timeSelector)
	}

	return nil
}

// parseTimestamp 解析 -t 格式的时间戳 [[CC]YY]MMDDhhmm[.ss]
func parseTimestamp(ts string) (time.Time, error) {
	// 尝试解析不同格式
	now := time.Now().In(time.Local)
	formats := []string{
		"200601021504.05", // CCYYMMDDhhmm.ss
		"200601021504",    // CCYYMMDDhhmm
		"0601021504.05",   // YYMMDDhhmm.ss
		"0601021504",      // YYMMDDhhmm
		"01021504.05",     // MMDDhhmm.ss (当前世纪)
		"01021504",        // MMDDhhmm (当前世纪)
	}

	// 尝试将输入数字转换为时间格式
	input := strings.TrimSpace(ts)

	// 如果没有小数点，添加一个
	if !strings.Contains(input, ".") && len(input) >= 12 {
		// 可能是 CCYYMMDDhhmm.ss 格式但没有秒
		formats = []string{
			"20060102150405", // CCYYMMDDhhmmss
			"200601021504",   // CCYYMMDDhhmm
			"060102150405",   // YYMMDDhhmmss
			"0601021504",     // YYMMDDhhmm
		}
	}

	for _, format := range formats {
		t, err := time.ParseInLocation(format, input, time.Local)
		if err == nil {
			if strings.HasPrefix(format, "0102") {
				t = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
			}
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid date format '%s'", ts)
}

// parseDateString 解析 -d 格式的日期字符串
func parseDateString(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)

	// 支持的时间格式
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006/01/02",
		"Jan 2, 2006",
		"January 2, 2006",
		"2 Jan 2006",
		"02-Jan-2006",
		time.RFC3339,
		time.RFC1123,
	}

	// 尝试每种格式
	for _, format := range formats {
		t, err := time.ParseInLocation(format, dateStr, time.Local)
		if err == nil {
			return t, nil
		}
	}

	// 尝试解析 Unix 时间戳（以 @ 开头）
	if after, ok := strings.CutPrefix(dateStr, "@"); ok {
		tsStr := after
		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err == nil {
			return time.Unix(ts, 0), nil
		}
	}

	// 尝试使用自然语言解析（简化版）
	if strings.Contains(dateStr, "ago") {
		return parseRelativeTime(dateStr)
	}

	return time.Time{}, fmt.Errorf("invalid date format '%s'", dateStr)
}

// parseRelativeTime 解析相对时间（如 "2 days ago"）
func parseRelativeTime(dateStr string) (time.Time, error) {
	dateStr = strings.ToLower(strings.TrimSpace(dateStr))
	now := time.Now()

	// 移除 "ago"
	dateStr = strings.TrimSpace(strings.TrimSuffix(dateStr, "ago"))

	// 解析数字和单位
	parts := strings.Fields(dateStr)
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid relative time format")
	}

	amount, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid relative time amount")
	}

	unit := strings.TrimSuffix(parts[1], "s") // 处理复数形式

	switch unit {
	case "second":
		return now.Add(-time.Duration(amount) * time.Second), nil
	case "minute":
		return now.Add(-time.Duration(amount) * time.Minute), nil
	case "hour":
		return now.Add(-time.Duration(amount) * time.Hour), nil
	case "day":
		return now.AddDate(0, 0, -amount), nil
	case "week":
		return now.AddDate(0, 0, -amount*7), nil
	case "month":
		return now.AddDate(0, -amount, 0), nil
	case "year":
		return now.AddDate(-amount, 0, 0), nil
	default:
		return time.Time{}, fmt.Errorf("unknown time unit: %s", unit)
	}
}
