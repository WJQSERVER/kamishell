# Kamishell 语法指南

Kamishell 是一种混合了 Bash 简洁性和 Go 语言严谨性的跨平台 Shell。

## 1. 变量与赋值

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

## 2. 外部命令执行

直接输入命令及其参数即可执行，就像在 Bash 中一样：

```bash
ls -la
grep "main" cmd/kamishell/main.go
```

## 3. 内置命令

### `print`
用于向标准输出打印内容，替代了传统的 `echo`。

```go
print "Hello, Kamishell!"
```

## 4. 控制流

### If-Else 语句
语法采用类 Go 的风格，不需要 `then` 或 `fi`。

```go
isValid := true
if isValid {
    print "It is valid"
} else {
    print "It is invalid"
}
```

## 5. 错误处理

Kamishell 鼓励显式的错误处理。当命令执行失败时，会返回一个 Error 对象。

```go
err := ls non_existent_folder
if err != nil {
    print "发生错误了"
}
```
*(注意：目前的实现中，错误会自动打印到标准错误流，并在赋值时捕获)*
