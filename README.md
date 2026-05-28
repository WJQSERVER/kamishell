# Kamishell

Kamishell 是一个用 Go 实现的交互式 Shell，内置轻量脚本语言。提供 REPL 环境和脚本执行能力，内置常用命令的纯 Go 实现，支持跨平台。

## 快速开始

```bash
go build -o kami .
./kami          # 进入 REPL
./kami script.km  # 运行脚本
./kami --compile myapp script.km  # 编译为原生二进制
```

## 功能

### Shell 层
- REPL（历史记录、Tab 补全、`.kamirc` 自启配置）
- 管道 `|`、重定向 `->`（覆盖）/ `>>`（追加）
- 逻辑链 `&&` `||`、后台执行 `&`
- Shebang 支持（`#!/usr/bin/env kami`）

### 脚本语言
- 变量声明 `:=`，显式类型 `var name type`
- 类型：`int` `float` `string` `bool` `array`（同构、值语义）
- 字符串插值 `$var`
- 控制流：`if/else` `switch/case/default` `for`（三段式/while/range）
- 循环控制：`break` `continue`
- 函数定义 `func`（参数类型注解、多返回值、闭包）
- 数组操作：索引访问/赋值、`len()`、`push()`
- 指针：`&` 取址、`*` 解引用、指针赋值 `*p = val`
- 函数名是常量，不可重赋值
- 并行：`go {}` 块、`t := go { return x }` / `t.Wait()`
- `sync.NewWaitGroup()` / `wg.Go {}` / `wg.Wait()`
- 显式错误处理：自动 `err` 变量、`error()` 构造器
- `import "Go/fmt"` 调用 Go 标准库（解释器模式限注册函数）

### 内置包
- `env.Get()` / `env.Set()` / `env.Unset()` / `env.GetOS()` / `env.GetArch()`
- `param.Get(key)`
- `sync.NewWaitGroup()`

### 内置命令
- 文件：`ls` `cp` `mv` `rm` `mkdir` `touch` `cat` `cd` `pwd`
- 文本：`grep` `sed`
- 网络：`http`（GET/POST、JSON/form、认证、重试）
- 系统：`type` `which` `jobs` `help` `exit` `print`
- 环境：`env` `export`
- 构建：`make`（`.km` 构建脚本）

### 编译模式
`--compile myapp script.km` 将脚本编译为 Go 原生二进制，消除解释器开销。
`--source output.go script.km` 仅输出 Go 源码。

### 沙箱
`NewSandboxEnvironment()` 创建受限环境，可控制外部命令执行、内置命令白名单、递归深度。

## 示例

```go
// 命令
ls -la | grep "go"
print "hello" -> out.txt

// 变量与插值
name := "kami"
print "hello $name"

// 条件
x := 10
if x > 5 {
    print "big"
} else {
    print "small"
}

// 循环
for i := 0; i < 5; i = i + 1 {
    print i
}

// 函数
func add(a int, b int) int {
    return a + b
}
print add(3, 4)

// 指针
x := 10
p := &x
*p = 20

// 并发
go {
    print "async"
}
wait
```

## 文档

- [语法参考](./docs/syntax.md)
- [使用手册](./docs/usage.md)
- [构建系统](./docs/make.md)

## 构建要求

Go 1.26+

## 测试

```bash
go test ./...
go test -bench=. ./...
```

## 许可

[LICENSE](./LICENSE)
