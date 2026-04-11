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
		Usage:       "help [command] | help -k [keyword] | help --version",
		Help:        "显示 shell 总帮助、指定内建命令帮助或关键字说明。",
		Action:      Help,
	})
}

func Help(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 1 {
		if cmd, ok := Builtins[args[0]]; ok {
			PrintBuiltinHelp(cmd, stdout)
			return 0
		}
	}

	// Parse flags
	keywordArg := ""
	showKeywords := false

	for i, arg := range args {
		if arg == "--version" || arg == "-v" {
			printVersion(stdout)
			return 0
		}
		if arg == "-k" {
			showKeywords = true
			if i+1 < len(args) {
				keywordArg = args[i+1]
			}
			break // Priority to keyword help
		}
	}

	if showKeywords {
		if keywordArg != "" {
			return showKeywordDetail(keywordArg, stdout)
		}
		return listKeywords(stdout)
	}

	fmt.Fprintln(stdout, "Kamishell (kami) - 一个用 Go 编写的高级交互式 Shell")
	fmt.Fprintln(stdout, "--------------------------------------------------")

	printVersion(stdout)
	fmt.Fprintln(stdout, "LICENSE: Mozilla Public License 2.0")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "内建命令列表:")

	for _, name := range BuiltinNames() {
		cmd := Builtins[name]
		fmt.Fprintf(stdout, "  %-12s %s\n", name, cmd.Description)
	}

	fmt.Fprintln(stdout, "--------------------------------------------------")
	fmt.Fprintln(stdout, "提示: 输入 'help --version' 查看详细构建信息。")
	fmt.Fprintln(stdout, "      输入 'help <命令>' 查看特定内建命令帮助。")
	fmt.Fprintln(stdout, "      输入 'help -k' 查看关键字列表。")
	fmt.Fprintln(stdout, "      输入 'help -k <关键字>' 查看特定关键字详情。")
	return 0
}

func listKeywords(stdout io.Writer) int {
	fmt.Fprintln(stdout, "Kamishell 关键字与操作符注解:")
	fmt.Fprintln(stdout, "--------------------------------------------------")

	keys := make([]string, 0, len(KeywordsDoc))
	for k := range KeywordsDoc {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		info := KeywordsDoc[k]
		fmt.Fprintf(stdout, "  %-12s %s\n", k, info.Description)
	}
	fmt.Fprintln(stdout, "--------------------------------------------------")
	fmt.Fprintln(stdout, "使用 'help -k <关键字>' 获取详细用法。")
	return 0
}

func showKeywordDetail(key string, stdout io.Writer) int {
	info, ok := KeywordsDoc[key]
	if !ok {
		fmt.Fprintf(stdout, "未找到关键字 '%s' 的文档。\n", key)
		return 1
	}

	fmt.Fprintf(stdout, "关键字: %s\n", key)
	fmt.Fprintf(stdout, "描述:   %s\n", info.Description)
	fmt.Fprintf(stdout, "用法:   %s\n", info.Usage)
	fmt.Fprintln(stdout, "详情:")
	fmt.Fprintf(stdout, "  %s\n", info.Details)
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
