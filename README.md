# Kamishell

Kamishell 是一个基于 Go 语言开发的跨平台 Shell 实现。它的设计目标是结合 **Bash 的命令执行便捷性** 与 **Go 语言的逻辑严谨性**。

## 🌟 核心特性

- **跨平台一致性**: 核心功能（如路径处理、内置命令）在 Windows、Linux 和 macOS 上表现一致。
- **混合语法**:
  - 像执行 Bash 一样执行外部命令：`ls -la`
  - 像写 Go 一样编写逻辑：`x := 10; if x { ... }`
- **强类型对象系统**: 内部使用对象系统处理字符串、整数、布尔值和错误。
- **显式错误处理**: 遵循 `if err != nil` 的设计哲学。
- **高性能**: 采用手写的递归下降解析器，并配有完善的基准测试。

## 🚀 快速开始

### 安装要求

- Go 1.26+

### 编译与运行

```bash
git clone <repository-url>
cd kamishell
go build -o kamishell ./cmd/kamishell/main.go
./kamishell
```

### 简单示例

```go
kami> print "Welcome to Kamishell"
Welcome to Kamishell
kami> files := ls
kami> if files != nil { print "Success" }
Success
```

## 📖 文档

- [语法指南](./docs/syntax.md)
- [使用与开发指南](./docs/usage.md)
- [架构设计提案 (开发者参考)](./plan/architecture.md)

## 🛠 开发与测试

```bash
# 运行所有测试
go test ./...

# 运行性能测试
go test -bench=. ./...
```

## 📄 开源协议

本项目采用 [LICENSE](./LICENSE) 协议。
