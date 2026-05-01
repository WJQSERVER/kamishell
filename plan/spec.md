# Kamishell 语言规范 (Spec)

## 1. 类型系统 (Type System)

Kamishell 是动态弱类型语言，支持以下核心对象类型：

*   **INTEGER**: 基础 64 位有符号整数。零值为 `0`。
*   **FLOAT**: 64 位浮点数。零值为 `0.0`。
*   **STRING**: 字符串对象，支持 `+` 拼接。零值为 `""`。
*   **BOOLEAN**: `true` 或 `false`。零值为 `false`。
*   **FUNCTION**: 存储参数列表、函数体和词法环境。零值为 `nil`。
*   **ERROR**: 存储操作失败信息的结构化对象。零值为 `nil`。
*   **NULL**: 特殊值（关键字 `nil`），不是独立类型，而是引用类型（FUNCTION、ERROR）的零值表示。

### 1.1 nil 语义

`nil` 是一个预声明的特殊值，不是独立类型。其语义与 Go 一致：

*   `nil` 是引用类型（FUNCTION、ERROR）的零值。
*   `nil` 无类型，不能用于 `:=` 的类型推断。
*   `nil` 只能赋给已声明类型的变量，且该变量必须是引用类型。

```kami
// ✅ 有效
var f func
f = nil            // func 是引用类型，可以接受 nil

var e error
e = nil            // error 是引用类型，可以接受 nil

func findUser(id) {
    if id < 0 {
        return nil // 函数返回 nil（返回值无类型约束）
    }
    return "user"
}
```

```kami
// ❌ 无效
x := nil           // 错误：nil 无类型，无法推断
var x              // 错误：必须声明类型
var x = nil        // 错误：必须声明类型
var n int = nil    // 错误：int 不能接受 nil
n = nil            // 错误：int 变量不能赋 nil
```

### 1.2 变量类型约束

Kami 要求所有变量必须有明确的类型约束：

*   `var x int` 会为变量建立显式类型约束。
*   `x := 1` 会根据初始值推断类型并建立类型约束。
*   `var x`（无类型）是语法错误。
*   `var x = nil`（无类型）是语法错误。
*   `=` 会优先更新最近作用域中的同名变量，并遵守其已有类型约束。

### 1.3 Error 类型

运行时通过全局 `err` 变量自动暴露最近一次命令的结果。

*   **字段**:
    *   `Message`: 错误描述字符串。
    *   `Code`: 退出状态码 (exit code)。
    *   `Op`: 产生错误的命令或操作名称。
*   **布尔上下文**: 非空 Error 对象在判断中被视为 `true`。

## 2. 语法构造 (Grammar)

### 2.1 赋值与作用域

*   `:=`: 声明并赋值，必须有明确类型（不能用于 nil）。
*   `=`: 重新赋值，遵守类型约束。
*   `var x TYPE`: 显式类型声明。
*   `var x TYPE = value`: 显式类型声明并初始化。
*   支持词法作用域（Lexical Scoping）。

### 2.2 流程控制

*   **If-Else**: 强制使用大括号，大括号需跟在关键字同一行。
*   **For**: 支持带条件的循环。
*   **Func**: 函数定义支持位置参数，返回值无类型约束。

### 2.3 命令组合

*   **管道 (|)**: 标准输出流连接。
*   **重定向 (->)**: 输出重定向到文件。
*   **逻辑运算符 (&&, ||)**: 基于退出码的短路执行。
*   **异步执行 (&)**: 后缀符号，将整行转入后台协程。
*   **Go 关键字**: 前缀符号，将后续代码块或命令转入后台协程。

## 3. 运行时行为 (Runtime)

### 3.1 变量查找顺序

1.  本地作用域（函数内部）。
2.  外部闭包作用域。
3.  环境变量。
4.  Shell 内置命令库。
5.  系统 PATH。

### 3.2 作业控制 (Job Control)

*   所有后台任务在 `builtin.Jobs` 全局注册表中维护。
*   任务状态分为 `Running` 和 `Done`。

### 3.3 强制命令执行 (exec)

提供规避关键字冲突的逃生通道。

## 4. 并发模型 (Concurrency)

### 4.1 Goroutine

```kami
go { ... }            // 块语法
go name(args)         // 函数调用语法
```

### 4.2 Task/Future

```kami
t := go { return val }  // 返回 Task 对象
result := t.Wait()      // 等待结果
result := t.Wait(10)    // 带超时
```

### 4.3 WaitGroup

```kami
wg := sync.NewWaitGroup()
wg.Go { task1() }
wg.Go { task2() }
wg.Wait()              // 等待所有任务
wg.Wait(10)            // 带超时
```

### 4.4 wait 命令

```kami
go { task1() }
go { task2() }
wait                   // 等待所有 goroutine
wait(10)               // 带超时
```
