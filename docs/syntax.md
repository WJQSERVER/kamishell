# Kamishell 语法指南

Kamishell 是一种混合了 Bash 简洁性和 Go 语言严谨性的跨平台 Shell。

## 1. 文件头 (Shebang)

Kamishell 支持标准的 Unix Shebang 文件头，允许脚本作为可执行文件运行。

```bash
#!/usr/bin/env kami
print "Hello from an executable script!"
```

## 2. 变量与赋值

使用 `:=` 进行变量声明和赋值。使用 `=` 对已存在的变量进行重新赋值。Kamishell 是动态类型的。

### 基础类型
- **Integer**: `x := 10`
- **String**: `name := "Kamishell"`
- **Boolean**: `isValid := true`
- **Nil**: `empty := nil`
- **Function**: `f := func() { ... }`

### 变量插值
在字符串字面量中，可以使用 `$VAR` 语法进行变量插值：
```go
name := "Jules"
print "Hello, $name"
```

## 3. 命令执行

直接输入命令及其参数即可执行。Kamishell 搜索顺序：
1. **当前作用域函数**
2. **Shell 内置命令** (如 `ls`, `cd`)
3. **系统 PATH 中的外部命令**

### 管道 (Pipes)
使用 `|` 连接多个命令，将前一个命令的 `stdout` 作为下一个命令的 `stdin`。
```bash
ls -la | grep "kami" | wc -l
```

### 重定向 (Redirection)
- `>`: 覆盖写入文件。
- `>>`: 追加写入文件。
```bash
print "log entry" >> access.log
ls /tmp > file_list.txt
```

## 4. 逻辑运算符

Kamishell 支持命令链式执行和逻辑判断：
- `&&`: 仅当前一个命令成功（退出码 0）时执行后续命令。
- `||`: 仅当前一个命令失败（退出码非 0）时执行后续命令。

```bash
mkdir new_dir && cd new_dir
ls non_existent || print "File not found"
```

## 5. 异步执行

### 后缀 `&` (Shell 风格)
将整个命令行放入后台运行。
```bash
sleep 10 &
print "I am not waiting for sleep"
```

### 关键字 `go` (Go 风格)
用于异步运行一个代码块或单个命令。
```go
go {
    sleep 5
    print "Background task done"
}

go updatedb
```

## 6. 函数定义 (`func`)

支持类 Go 的函数定义语法，支持参数传递和词法作用域。

```go
func greet(name) {
    print "Hello, " + name
}

greet "Kamishell"

// 闭包支持
x := 10
func check() {
    print x // 访问外部变量
}
```

## 7. 控制流

### If-Else 语句
注意：`{` 必须与 `if` 或 `else` 在同一行。

```go
x := 10
if x > 5 {
    print "High"
} else {
    print "Low"
}
```

### For 循环
目前支持基础的无限循环或带条件的循环。
```go
i := 0
for i < 3 {
    print i
    i = i + 1
}
```

## 8. 错误处理

Kamishell 运行时会自动维护一个名为 `err` 的特殊变量。每次命令执行后，该变量都会被更新。

- 如果命令成功，`err` 为 `nil`。
- 如果命令失败，`err` 为一个 Error 对象，包含以下字段：
  - `err.Message`: 错误描述信息。
  - `err.Code`: 退出码。
  - `err.Op`: 产生错误的命令名称。

```go
cp source.txt dest.txt
if err != nil {
    print "Operation failed: " + err.Message
}
```

## 9. 强制命令执行 (`exec`)

当命令名称与关键字冲突时使用：
```go
exec "go run ."
```

## 10. 注释

- **单行**: `//`
- **多行**: `/* ... */`
