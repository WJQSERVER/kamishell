# Kamishell 完整语法参考手册

Kamishell 是一种兼具传统 Shell 简洁性与现代编程语言（如 Go）严谨性的混合型 Shell 环境。本文档详细介绍了 Kamishell 支持的所有语法特性。

---

## 1. 基础语法

### 注释
Kamishell 支持两种风格的注释：
- **单行注释**: 使用 `//`
- **多行注释**: 使用 `/* ... */`

```go
// 这是一个单行注释
x := 10 /* 这是一个
          多行注释 */
```

### 语句分隔
Kamishell 能够智能识别行尾作为语句结束。你也可以使用分号 `;` 在同一行编写多个语句。
```bash
ls -la; echo "Done"
```

---

## 2. 数据类型与变量

### 变量声明与赋值
- **声明并初始化**: 使用 `:=`（自动类型推断）。
- **重新赋值**: 使用 `=`。

```go
name := "Kamishell" // String
count := 42         // Integer
isActive := true    // Boolean
data := nil         // Nil
```

### 变量插值
在双引号字符串中，可以使用 `$变量名` 或 `${变量名}` 语法插入变量值。
```go
user := "Admin"
echo "Welcome, $user!"
echo "Path: ${HOME}/bin"
```

---

## 3. 命令执行与流程控制

### 外部命令执行
直接输入命令及其参数即可。Kamishell 会自动在系统 `PATH` 中搜索对应的可执行文件。
```bash
git status
go build -o app main.go
```

### 逻辑运算符 (Command Chaining)
- `&&`: 前一个命令成功（退出码为 0）时执行。
- `||`: 前一个命令失败（退出码非 0）时执行。

```bash
mkdir build && cd build
ls /root || echo "Access denied"
```

### 管道与重定向
- **管道 (`|`)**: 将前一命令的输出作为后一命令的输入。
- **重定向 (`>`, `>>`)**: 覆盖写入或追加写入到文件。

```bash
cat logs.txt | grep "Error" | wc -l
echo "Initial config" > config.yaml
echo "New line" >> config.yaml
```

---

## 4. 控制结构

### 条件判断 (If-Else)
条件不需要括号，但大括号 `{` 必须与 `if`/`else` 在同一行。
```go
score := 85
if score >= 60 {
    echo "Passed"
} else {
    echo "Failed"
}
```

### 循环 (For)
Kamishell 提供了灵活的 `for` 循环：
- **条件循环**:
```go
i := 0
for i < 5 {
    echo "Count: $i"
    i = i + 1
}
```
- **无限循环**:
```go
for {
    // 使用 break 退出
    if some_condition { break }
}
```

---

## 5. 函数定义与调用

函数使用 `func` 关键字定义，支持参数传递和词法作用域。

```go
func say_hello(name, times) {
    i := 0
    for i < times {
        echo "Hello, $name! (count: $i)"
        i = i + 1
    }
}

say_hello "User" 3
```

---

## 6. 异步与并发执行

Kamishell 结合了 Shell 的便捷与 Go 的并发特性：
- **后台运行 (`&`)**: 传统的 Shell 后台执行。
- **并发块 (`go`)**: 类似于 Go 语言的 goroutine，可以异步运行代码块。

```bash
# Shell 风格
long_task &

# Go 风格
go {
    sleep 5
    echo "Background process finished"
}
```

---

## 7. 错误处理

Kamishell 内置了一个全局变量 `err`，它保存了最近一次执行命令的错误信息。

```go
rm "/protected/file"
if err != nil {
    echo "Error occurred!"
    echo "Message: " + err.Message
    echo "Exit Code: " + err.Code
}
```

---

## 8. 进阶特性

### 强制执行 (`exec`)
当你的命令名称与 Kamishell 的关键字（如 `if`, `for`, `func`）冲突时，使用 `exec`：
```go
exec "go run main.go"
```

### 环境变元访问
可以直接访问系统环境变量：
```go
echo $PATH
HOME_DIR := $HOME
```

---

## 9. 转义字符

在字符串中支持常见的转义序列：
- `\n`: 换行
- `\t`: 制表符
- `\"`: 双引号
- `\\`: 反斜杠
