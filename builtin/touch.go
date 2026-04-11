package builtin

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "touch",
		Description: "更新文件访问和修改时间戳",
		Usage:       "touch [-a] [-c] [-d TIME] [-m] [-r FILE] [-t TIME] file...",
		Help: `更新文件访问和修改时间戳。
如果不存在文件，则创建（除非指定了 -c）。

选项:
  -a, --time=atime          仅更改访问时间戳
  -c, --no-create           不创建不存在的文件
  -d, --date=TIME           使用指定时间而非当前时间
                            TIME 可以是：YYYY-MM-DD, YYYY-MM-DD HH:MM:SS,
                            @UNIX_TIMESTAMP, 或 "2 days ago" 等
  -m, --time=mtime          仅更改修改时间戳
  -r, --reference=FILE      使用参考文件的时间戳
  -t [[CC]YY]MMDDhhmm[.ss]  使用指定时间戳（与 date -t 格式相同）

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
	atime     bool
	noCreate  bool
	date      string
	mtime     bool
	reference string
	time      string
}

func Touch(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	args = PreprocessArgs(args)

	if HandleBuiltinHelp(Builtins["touch"], args, stdout) {
		return 0
	}

	fs := flag.NewFlagSet("touch", flag.ContinueOnError)
	fs.SetOutput(stderr)

	opts := &touchOptions{}
	fs.BoolVar(&opts.atime, "a", false, "change only the access time")
	fs.BoolVar(&opts.atime, "time", false, "change only the access time")
	fs.BoolVar(&opts.noCreate, "c", false, "do not create the file if it does not exist")
	fs.BoolVar(&opts.noCreate, "no-create", false, "do not create the file if it does not exist")
	fs.StringVar(&opts.date, "d", "", "use DATE instead of current time")
	fs.StringVar(&opts.date, "date", "", "use DATE instead of current time")
	fs.BoolVar(&opts.mtime, "m", false, "change only the modification time")
	fs.StringVar(&opts.reference, "r", "", "use this file's times instead of current time")
	fs.StringVar(&opts.reference, "reference", "", "use this file's times instead of current time")
	fs.StringVar(&opts.time, "t", "", "use [[CC]YY]MMDDhhmm[.ss] instead of current time")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	files := fs.Args()
	if len(files) == 0 {
		fmt.Fprintln(stderr, "touch: missing file operand")
		return 1
	}

	// 解析目标时间戳
	targetTime, err := parseTime(opts, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "touch: %v\n", err)
		return 1
	}

	// 如果没有指定时间，使用当前时间
	if targetTime.IsZero() {
		targetTime = time.Now()
	}

	// 如果同时使用 -a 和 -m，两者都生效（都改变）
	// 如果只使用其中一个，另一个保持不变（设置为0表示不改变）
	// 在Unix系统中，我们不能单独改变atime或mtime，需要使用特定系统调用
	// 这里简化处理：atime = mtime = targetTime，除非选项指定

	atime := targetTime
	mtime := targetTime

	// 如果只指定了 -a，mtime 保持原值
	if opts.atime && !opts.mtime {
		mtime = time.Time{} // 使用零值表示不改变
	}

	// 如果只指定了 -m，atime 保持原值
	if opts.mtime && !opts.atime {
		atime = time.Time{} // 使用零值表示不改变
	}

	exitCode := 0
	for _, file := range files {
		// 检查文件是否存在
		info, err := os.Stat(file)
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
				f.Close()
				// 新文件创建成功，不需要修改时间戳
				continue
			}
			// 其他错误（如权限问题）
			fmt.Fprintf(stderr, "touch: cannot access '%s': %v\n", file, err)
			exitCode = 1
			continue
		}

		// 文件存在，获取当前时间戳
		currentAtime := info.ModTime() // 作为默认值
		currentMtime := info.ModTime()

		// 设置atime
		if atime.IsZero() {
			atime = currentAtime
		}
		// 设置mtime
		if mtime.IsZero() {
			mtime = currentMtime
		}

		// 修改文件时间戳
		err = os.Chtimes(file, atime, mtime)
		if err != nil {
			fmt.Fprintf(stderr, "touch: setting times of '%s': %v\n", file, err)
			exitCode = 1
		}
	}

	return exitCode
}

// parseTime 解析时间选项
func parseTime(opts *touchOptions, stderr io.Writer) (time.Time, error) {
	// 优先级：-r > -t > -d > 默认当前时间

	// -r: 使用参考文件的时间戳
	if opts.reference != "" {
		info, err := os.Stat(opts.reference)
		if err != nil {
			if os.IsNotExist(err) {
				return time.Time{}, fmt.Errorf("failed to get attributes of '%s': No such file or directory", opts.reference)
			}
			return time.Time{}, fmt.Errorf("failed to get attributes of '%s': %v", opts.reference, err)
		}
		return info.ModTime(), nil
	}

	// -t: 使用指定时间戳格式 [[CC]YY]MMDDhhmm[.ss]
	if opts.time != "" {
		return parseTimestamp(opts.time)
	}

	// -d: 使用日期字符串
	if opts.date != "" {
		return parseDateString(opts.date)
	}

	// 没有指定时间，返回零值表示使用当前时间
	return time.Time{}, nil
}

// parseTimestamp 解析 -t 格式的时间戳 [[CC]YY]MMDDhhmm[.ss]
func parseTimestamp(ts string) (time.Time, error) {
	// 尝试解析不同格式
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
		t, err := time.Parse(format, input)
		if err == nil {
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
		t, err := time.Parse(format, dateStr)
		if err == nil {
			return t, nil
		}
	}

	// 尝试解析 Unix 时间戳（以 @ 开头）
	if strings.HasPrefix(dateStr, "@") {
		tsStr := strings.TrimPrefix(dateStr, "@")
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

// abs 返回绝对路径（辅助函数）
func abs(path string) (string, error) {
	return filepath.Abs(path)
}
