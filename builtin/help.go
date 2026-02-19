package builtin

import (
	"fmt"
	"io"
	"sort"
)

func init() {
	RegisterBuiltin("help", Help)
}

var helpDescriptions = map[string]string{
	"help":   "显示此帮助信息",
	"ls":     "列出目录内容",
	"cd":     "切换工作目录",
	"pwd":    "显示当前工作目录",
	"cat":    "连接文件并打印到标准输出",
	"cp":     "复制文件或目录",
	"mv":     "移动或重命名文件或目录",
	"rm":     "删除文件或目录",
	"mkdir":  "创建目录",
	"touch":  "创建空文件或更新时间戳",
	"exit":   "退出 Shell",
	"export": "设置环境变量",
	"env":    "显示环境变量",
	"type":   "显示命令类型",
	"which":  "查找命令的可执行文件路径",
	"jobs":   "列出后台作业",
	"grep":   "在文件中搜索模式",
	"sed":    "流编辑器，用于过滤和转换文本",
}

func Help(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	fmt.Fprintln(stdout, "Kamishell 内建命令帮助:")
	fmt.Fprintln(stdout, "---------------------------")

	// Get all builtin names and sort them
	names := make([]string, 0, len(Builtins))
	for name := range Builtins {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		desc, ok := helpDescriptions[name]
		if !ok {
			desc = "(尚无详细说明)"
		}
		fmt.Fprintf(stdout, "  %-10s %s\n", name, desc)
	}

	fmt.Fprintln(stdout, "---------------------------")
	fmt.Fprintln(stdout, "提示: 输入 'help <command>' (未来支持) 或查看文档了解详情。")
	return 0
}
