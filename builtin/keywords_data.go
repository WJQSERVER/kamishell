package builtin

type KeywordInfo struct {
	Description string
	Usage       string
	Details     string
}

var KeywordsDoc = map[string]KeywordInfo{
	"func": {
		Description: "定义函数",
		Usage:       "func 函数名(参数列表) { 函数体 }",
		Details:     "用于声明一个新的函数。支持位置参数，函数体内可以访问闭包环境中的变量。",
	},
	"return": {
		Description: "从函数返回",
		Usage:       "return [值]",
		Details:     "用于从当前执行的函数中退出，并可选地返回一个值给调用者。",
	},
	"if": {
		Description: "条件判断",
		Usage:       "if 条件 { 后果 } [else { 替代 }]",
		Details:     "根据布尔条件的真假来执行相应的代码块。支持可选的 else 分支。",
	},
	"else": {
		Description: "条件分支",
		Usage:       "if ... { ... } else { ... }",
		Details:     "if 语句的备选分支，当 if 条件不满足时执行。",
	},
	"for": {
		Description: "循环结构",
		Usage:       "for [条件] { 循环体 }",
		Details:     "基本的循环结构。如果省略条件，则为无限循环。",
	},
	"range": {
		Description: "迭代集合",
		Usage:       "for 变量 := range 集合 { ... }",
		Details:     "用于迭代数组或迭代器。支持数组 range（for i, v := range arr）和迭代器 range-over-func（for v := range iter(args)）。",
	},
	"go": {
		Description: "启动 Goroutine (异步执行)",
		Usage:       "go 表达式",
		Details:     "在新的 Goroutine 中并发执行指定的函数调用或语句块。",
	},
	"var": {
		Description: "显式变量声明",
		Usage:       "var 变量名 [类型] [= 初始值]",
		Details:     "声明一个具有可选类型约束的变量。如果指定了类型，后续赋值必须匹配该类型。",
	},
	"const": {
		Description: "常量定义",
		Usage:       "const 常量名 = 值",
		Details:     "定义一个不可修改的常量值。",
	},
	"import": {
		Description: "导入 Go 标准库",
		Usage:       "import \"Go/包名\"",
		Details:     "导入 Go 标准库函数，编译时直接解析为原生 Go 调用。已支持的包：fmt、math、strings、strconv、os。",
	},
	"exec": {
		Description: "执行外部命令",
		Usage:       "exec <command> [args...] 或 exec(cmd)",
		Details:     "关键字形式：exec echo hello - 直接执行命令，参数按空格分割，支持引号。函数形式：exec(cmd) - 执行字符串命令。注意：exec \"...\" 已弃用。",
	},
	"export": {
		Description: "设置环境变量",
		Usage:       "export 变量名=值",
		Details:     "设置并导出环境变量，使其对子进程可见。",
	},
	"exit": {
		Description: "退出 Shell",
		Usage:       "exit [退出码]",
		Details:     "终止当前的 Kamishell 会话。可选提供退出状态码（默认为 0）。",
	},
	"true": {
		Description: "布尔真",
		Usage:       "true",
		Details:     "布尔逻辑值中的 '真'。",
	},
	"false": {
		Description: "布尔假",
		Usage:       "false",
		Details:     "布尔逻辑值中的 '假'。",
	},
	"nil": {
		Description: "空值",
		Usage:       "nil",
		Details:     "表示缺失的值或空指针。在错误处理中常用作无错误状态。",
	},
	":=": {
		Description: "短变量声明",
		Usage:       "变量名 := 初始值",
		Details:     "自动推断类型的变量声明并赋值。后续可以通过 '=' 重新赋值，但类型固定。",
	},
	"|": {
		Description: "管道操作符",
		Usage:       "命令1 | 命令2",
		Details:     "将前一个命令的标准输出重定向为后一个命令的标准输入。",
	},
	">": {
		Description: "重定向（覆盖）",
		Usage:       "命令 > 文件",
		Details:     "将命令的标准输出重定向到指定文件，并覆盖原有内容。",
	},
	">>": {
		Description: "重定向（追加）",
		Usage:       "命令 >> 文件",
		Details:     "将命令的标准输出重定向到指定文件，并追加到末尾。",
	},
	"&": {
		Description: "后台运行",
		Usage:       "命令 &",
		Details:     "在后台异步执行命令，不阻塞当前 shell。",
	},
}

func init() {
	// Add more keywords/operators from plan/keywords.md
	KeywordsDoc["="] = KeywordInfo{
		Description: "赋值操作符",
		Usage:       "变量名 = 新值",
		Details:     "将一个新值赋给已声明的变量。如果变量有类型约束，新值必须符合该类型。",
	}
	KeywordsDoc["&&"] = KeywordInfo{
		Description: "逻辑与 / 命令链",
		Usage:       "条件1 && 条件2",
		Details:     "逻辑与操作。在命令链中，只有当前一个命令执行成功（退出码为 0）时，才执行后一个命令。",
	}
	KeywordsDoc["||"] = KeywordInfo{
		Description: "逻辑或 / 命令备选",
		Usage:       "条件1 || 条件2",
		Details:     "逻辑或操作。在命令链中，只有当前一个命令执行失败（退出码非 0）时，才执行后一个命令。",
	}
	KeywordsDoc["!"] = KeywordInfo{
		Description: "逻辑非",
		Usage:       "! 表达式",
		Details:     "对布尔值取反。",
	}
	KeywordsDoc["=="] = KeywordInfo{
		Description: "等于比较",
		Usage:       "值1 == 值2",
		Details:     "检查两个值是否相等。",
	}
	KeywordsDoc["!="] = KeywordInfo{
		Description: "不等于比较",
		Usage:       "值1 != 值2",
		Details:     "检查两个值是否不相等。",
	}
	KeywordsDoc["$"] = KeywordInfo{
		Description: "变量插值",
		Usage:       "\"... $变量名 ...\"",
		Details:     "在双引号字符串中引用变量的值。",
	}
	KeywordsDoc["defer"] = KeywordInfo{
		Description: "延迟执行",
		Usage:       "defer 表达式",
		Details:     "（计划中）在函数返回前执行指定的表达式。",
	}
	KeywordsDoc["make"] = KeywordInfo{
		Description: "构建系统 (类似 CMake)",
		Usage:       "make [脚本文件.km]",
		Details:     "默认搜寻当前目录下以 .km 结尾的文件（如 Kami.km 或 build.km）。它会执行脚本并根据定义的项目目标调用编译器（目前默认支持 Go 语言）。",
	}
	KeywordsDoc["project"] = KeywordInfo{
		Description: "定义项目名称 (仅限 make)",
		Usage:       "project 项目名",
		Details:     "设置当前构建项目的名称，用于日志输出和默认目标标识。",
	}
	KeywordsDoc["add_executable"] = KeywordInfo{
		Description: "定义可执行文件目标 (仅限 make)",
		Usage:       "add_executable 目标名 源文件1 [源文件2 ...]",
		Details:     "指定一个可执行程序目标及其源代码。构建时会调用编译器生成对应的可执行文件。",
	}
	KeywordsDoc["add_library"] = KeywordInfo{
		Description: "定义库文件目标 (仅限 make)",
		Usage:       "add_library 库名 源文件1 [源文件2 ...]",
		Details:     "指定一个库文件目标及其源代码。构建时会调用编译器生成对应的库文件。",
	}
	KeywordsDoc["target_link_libraries"] = KeywordInfo{
		Description: "定义链接依赖 (仅限 make)",
		Usage:       "target_link_libraries 目标名 依赖库1 [依赖库2 ...]",
		Details:     "指定一个目标需要链接的外部库或其他库目标。",
	}
	KeywordsDoc["target_env"] = KeywordInfo{
		Description: "设置目标构建变量 (仅限 make)",
		Usage:       "target_env 目标名 变量=值 [变量=值 ...]",
		Details:     "为指定目标追加或覆盖构建环境变量，例如 GOOS、GOARCH、CGO_ENABLED。make 也会在创建目标时快照脚本内 env 包中的变量，并在构建时传给 go build。",
	}
}
