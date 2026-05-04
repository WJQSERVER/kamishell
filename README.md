# Kamishell

Kamishell 是一个基于 Go 语言开发的跨平台 Shell 实现。它的设计目标是结合 **Bash 的命令执行便捷性** 与 **Go 语言的逻辑严谨性**。

## 🌟 核心特性

- **跨平台一致性**: 核心功能（如路径处理、内置工具集）在 Windows、Linux 和 macOS 上表现一致。
- **混合语法**:
  - 像执行 Bash 一样执行外部命令和管道：`ls -la | grep kami`
  - 像写 Go 一样编写逻辑：`x := 10; if x > 5 { print "Large" }`
- **并发原生支持**:
  - 支持后缀 `&` 后台执行。
  - 支持关键字 `go` 开启 Goroutine 代码块。
- **强类型对象系统**: 内部使用对象系统处理字符串、整数、布尔值、函数和错误。
- **Go 风格函数**: 参数强制类型注解，支持多返回值 `func div(a int, b int) (int, error)`。
- **显式错误处理**: 运行时自动维护 `err` 变量，遵循 `if err != nil` 的设计哲学。
- **交互式 REPL**: 集成 Readline，支持历史记录、Tab 自动补全和 `.kamirc` 配置文件。
- **内置工具集**: 纯 Go 实现的 `ls`, `cp`, `mv`, `grep`, `sed`, `http`, `jobs`, `type`, `which` 等。

## 🚀 快速开始

### 安装要求

- Go 1.26+

### 编译与运行

```bash
git clone <repository-url>
cd kamishell
go build -o kami .
./kami
```

### 简单示例

```go
// 变量赋值与数学运算
x := 5 + 5
print "Result is $x"

// 管道与内置过滤工具
ls | grep "go"

// 发送 HTTP 请求
http "https://example.com/health"

// 发送 JSON 请求体
http "https://api.example.com/items" --json "{\"name\":\"kami\"}"

// 函数定义（参数必须有类型注解）与后台执行
func longTask(msg string) {
    sleep 2
    print msg
}
longTask "Task Finished" &

// 多返回值
func div(a int, b int) (int, error) {
    if b == 0 {
        return 0, error("division by zero")
    }
    return a / b, nil
}
result, err := div(10, 3)
if err != nil {
    print "error: " + err
}
print result

// 显式错误处理
ls non_existent_file
if err != nil {
    print "Captured error: " + err.Message
}
```

## 📖 文档

- [当前语法指南](./docs/syntax.md) - 当前代码真实支持的语法、关键字与未实现项总览。
- [make 构建文档](./docs/make.md) - `.km` 构建脚本与构建变量说明。
- [使用手册](./docs/usage.md) - REPL 和内置命令的使用说明。
- [语言规范](./plan/spec.md) - 类型系统与运行时行为定义。
- [性能说明](./docs/performance.md) - benchmark 覆盖与热点观察。
- [路线图](./plan/roadmap.md) - 项目开发计划。

## 🛠 开发与测试

```bash
# 运行所有测试
go test ./...

# 运行性能测试
go test -bench=. ./...
```

## 📄 开源协议

本项目采用 [LICENSE](./LICENSE) 协议。
