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
		Details:     "（计划中）用于迭代数组、映射或其他可迭代对象。",
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
		Description: "导入模块",
		Usage:       "import \"模块路径\"",
		Details:     "（计划中）用于加载外部脚本或库。",
	},
	"exec": {
		Description: "执行字符串命令",
		Usage:       "exec \"命令字符串\"",
		Details:     "将字符串作为 shell 命令执行。常用于动态执行命令或避开关键字冲突。",
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
	KeywordsDoc["alias"] = KeywordInfo{
		Description: "定义别名",
		Usage:       "alias 别名='命令'",
		Details:     "（计划中）为长命令定义一个简短的替代名称。",
	}
}
