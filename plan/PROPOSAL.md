# Kamishell 设计提案

## 1. 核心愿景
构建一个跨平台的 Shell，它既具备 Bash 执行命令的简洁性，又拥有 Go 语言在脚本逻辑处理上的严谨性。

## 2. 架构概览
*   **引擎**: 基于 Go 编写的递归下降解析器。
*   **跨平台**: 使用 Go 标准库屏蔽 OS 差异，核心命令 (ls, cd, pwd 等) 内置化。
*   **运行时**: 支持 Goroutine 并发，具备完善的作用域管理。

## 3. 语法规范 (预览)

### 命令与赋值
```bash
# 执行并捕获输出与错误
files, err := ls -la | grep ".go"
if err != nil {
    return err
}
print "Found ${len(files)} files"
```

### 逻辑控制
```go
if $USER == "root" {
    print "Warning: Running as root"
}

for _, f := range files {
    err := du -h $f
    if err != nil {
        log.Error(err)
    }
}
```

### 并发处理
```go
go {
    # 异步备份
    err := tar -czf backup.tar.gz ./data
    if err != nil {
        sendNotification("Backup failed")
    }
}
```

### 错误处理
完全遵循 Go 语言的错误处理哲学：
```go
err := some_command
if err != nil {
    # 处理错误
}
```

## 4. 后续规划
1. 实现 Lexer 和基本的 Token 识别。
2. 构建 AST 解析简单的命令调用。
3. 实现跨平台的内置命令集。

## 5. 关键字与符号
详见 [keywords.md](./keywords.md)。Kamishell 严格遵循 Go 的关键字集合，并扩展了 Shell 必要的环境管理关键字。
