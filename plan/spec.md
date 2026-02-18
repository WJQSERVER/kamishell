# Kamishell 语言规范 (Spec)

## 1. 类型系统 (Type System)

Kamishell 是动态类型的，但支持以下核心值类型：

*   **String**: 默认的命令输出和输入类型。
*   **Int / Float**: 基础数值类型。
*   **Bool**: `true` 或 `false`。
*   **Array / Slice**: 对象的有序集合（例如：`ls` 的结果可以作为 `[]File` 或 `[]string`）。
*   **Map**: 键值对集合。
*   **Error**: 专门用于错误处理的类型。

### 1.1 Error 类型
`Error` 是 Kamishell 处理失败情况的核心。
*   **语义**: 遵循 Go 的 `err != nil` 模式。
*   **结构**:
    *   `msg`: 错误描述字符串。
    *   `code`: 退出状态码 (exit code)。
    *   `op`: 产生错误的命令或操作名称。
*   **行为**:
    *   当一个命令执行成功时，返回的 `err` 值为 `nil`。
    *   `nil` 在布尔上下文中被视为 `false`（用于 `if err != nil`）。
    *   可以直接打印 `err` 获取其 `msg`。

## 2. 语法构造 (Grammar)

### 2.1 变量声明与赋值
使用 `:=` 进行类型推断声明，使用 `=` 进行赋值。
```go
out, err := ls -la
```

### 2.2 管道 (Pipes)
管道连接命令流。
```bash
cat file.txt | grep "pattern" | wc -l
```

### 2.3 函数 (Functions)
```go
func myFunc(arg string) error {
    print "Argument: " + arg
    return nil
}
```

## 3. 运行时行为 (Runtime)

### 3.1 跨平台抽象
*   **Path**: 自动处理 `/` (Unix) 和 `\` (Windows)。
*   **Environment**: 统一访问环境变量。

### 3.2 内置命令 (Built-ins)
*   `print`: 替代传统的 `echo`。
*   `cd`, `pwd`, `exit`, `var`, `export` 等。
