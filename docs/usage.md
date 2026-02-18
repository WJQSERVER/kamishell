# 使用指南

## 1. 编译环境要求

- **Go SDK**: 1.26 或更高版本

## 2. 编译与运行

在项目根目录下执行以下命令编译：

```bash
go build -o kamishell ./cmd/kamishell/main.go
```

编译完成后，可以直接启动交互式 REPL：

```bash
./kamishell
```

## 3. REPL 命令

- **输入命令**: 在 `kami> ` 提示符后输入任何支持的语法。
- **退出**: 输入 `exit` 或按下 `Ctrl+D`。

## 4. 运行测试与基准测试

```bash
# 运行单元测试
go test ./...

# 运行性能基准测试
go test -bench=. ./...
```
