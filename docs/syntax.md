# Kamishell 语法指南

Kamishell 是一种混合了 Bash 简洁性和 Go 语言严谨性的跨平台 Shell。

## 1. 文件头 (Shebang)

Kamishell 支持标准的 Unix Shebang 文件头，允许脚本作为可执行文件运行。

```bash
#!/usr/bin/env kami
print "Hello from an executable script!"
```

## 2. 变量与赋值

使用 `:=` 进行变量声明和赋值。Kamishell 是动态类型的，支持以下基础类型：

- **Integer**: `x := 10`
- **String**: `name := "Kamishell"`
- **Boolean**: `isValid := true`
- **Nil**: `empty := nil`

### 变量使用
在命令中可以直接使用变量名（如果是字符串或基础类型），或者在逻辑控制中使用。

```go
count := 5
print count
```

## 3. 外部命令与内置命令执行

直接输入命令及其参数即可执行。Kamishell 会优先查找内置命令，如果没有找到，则在系统的 PATH 中查找外部命令。

```bash
grep "main" cmd/kamishell/main.go
```

## 4. 内置核心工具

为了保证跨平台的一致性，Kamishell 重新实现了一些常用的核心工具。

### `ls`
列出目录内容。支持以下参数：
- `-a`: 显示所有文件（包括以 `.` 开头的隐藏文件）。
- `-l`: 使用长格式列表显示详细信息。
- `-h`: 配合 `-l` 使用，以易读的格式（如 1K, 234M）显示文件大小。
- `-F`: 在条目后添加文件类型分类符（如 `/` 表示目录，`*` 表示可执行文件）。

### `cd`
切换当前工作目录。
```bash
cd internal
cd ..
```

### `pwd`
显示当前绝对路径。

### `print`
用于向标准输出打印内容，替代了传统的 `echo`。

```go
print "Hello, Kamishell!"
```

## 5. 控制流

### If-Else 语句
语法采用类 Go 的风格。注意：`{` 必须与 `if` 在同一行，或者在不触发自动分号插入的情况下换行。

```go
isValid := true
if isValid {
    print "It is valid"
} else {
    print "It is invalid"
}
```

## 6. 分号 (Semicolons)

分号在 Kamishell 中是**可选的**。

- 你可以省略行尾的分号。
- 你可以使用分号在同一行分隔多个命令。

```go
print "first"; print "second"
x := 1; y := 2
```

## 7. 强制命令执行 (`exec`)

当命令名称与 Kamishell 的关键字（如 `go`, `print`, `if` 等）冲突时，可以使用 `exec` 关键字配合字符串来强制执行外部命令：

```go
exec "go run ."
exec "print -p 9090"
```

## 8. 注释

Kamishell 遵循 Go 的注释语法：

- **单行注释**: 使用 `//`
- **多行注释**: 使用 `/* ... */`

```go
// 这是一个单行注释
print "hello"

/*
  这是一个
  多行注释
*/
```

## 9. 错误处理

Kamishell 鼓励显式的错误处理。当命令执行失败时，会返回一个 Error 对象。

```go
err := ls non_existent_folder
if err != nil {
    print "发生错误了"
}
```
