package builtin

import (
	"fmt"
	"io"
	"runtime/debug"
	"sort"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "help",
		Description: "显示此帮助信息",
		Action:      Help,
	})
}

func Help(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	// Parse flags
	for _, arg := range args {
		if arg == "--version" || arg == "-v" {
			printVersion(stdout)
			return 0
		}
	}

	fmt.Fprintln(stdout, "Kamishell (kami) - 一个用 Go 编写的高级交互式 Shell")
	fmt.Fprintln(stdout, "--------------------------------------------------")

	printVersion(stdout)
	fmt.Fprintln(stdout, "LICENSE: Mozilla Public License 2.0")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "内建命令列表:")

	// Get all builtin names and sort them
	names := make([]string, 0, len(Builtins))
	for name := range Builtins {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		cmd := Builtins[name]
		fmt.Fprintf(stdout, "  %-12s %s\n", name, cmd.Description)
	}

	fmt.Fprintln(stdout, "--------------------------------------------------")
	fmt.Fprintln(stdout, "提示: 输入 'help --version' 查看详细构建信息。")
	fmt.Fprintln(stdout, "      内建关键字 (如 if, for, func, print) 请参考文档。")
	return 0
}

func printVersion(stdout io.Writer) {
	version := "unknown"
	revision := "none"
	time := "unknown"

	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" {
			version = info.Main.Version
		}
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				revision = setting.Value
			}
			if setting.Key == "vcs.time" {
				time = setting.Value
			}
		}
	}

	fmt.Fprintf(stdout, "版本: %s\n", version)
	fmt.Fprintf(stdout, "提交: %s\n", revision)
	fmt.Fprintf(stdout, "构建时间: %s\n", time)
}
