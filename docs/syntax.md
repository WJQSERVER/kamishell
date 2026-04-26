# Kamishell 当前语法与关键字参考

这份文档描述的是 Kamishell **当前代码实际支持** 的语法与关键字，而不是理想设计或长期规划。

如果某个能力仍在规划中，本文会明确标记为“未实现”或“部分实现”。

## 1. 语言定位

Kamishell 是一种混合型 Shell 语言：

- 可以像传统 Shell 一样直接执行命令、管道、重定向
- 也可以像轻量脚本语言一样写变量、条件、循环、函数
- 在 `.km` 构建脚本里还能作为 `make` DSL 使用

## 2. 基础语法

### 注释

支持两种注释：

- 单行注释：`// ...`
- 多行注释：`/* ... */`

```go
// 这是单行注释
x := 10 /* 这是
一个多行注释 */
```

### 语句分隔

支持两种方式：

- 换行
- 分号 `;`

```go
x := 1
y := 2

name := "kami"; print name
```

## 3. 数据类型

当前运行时核心对象类型：

- `INTEGER`
- `STRING`
- `BOOLEAN`
- `FUNCTION`
- `NULL`，源码字面量写作 `nil`
- `ERROR`
- `PACKAGE`，目前主要用于脚本内 `env` 包

### 字面量

```go
1
"hello"
true
false
nil
```

### 当前变量静态类型约束

显式和推断出的变量类型目前主要支持：

- `int` / `integer`
- `string`
- `bool` / `boolean`

注意：

- `nil` 是运行时空值，不会被记录为变量静态类型
- 所以 `x := nil; x = 1` 是允许的

## 4. 变量与赋值

### `:=` 短变量声明

在当前作用域声明变量并赋值，同时记录推断出的类型。

```go
x := 10
name := "kami"
ready := true
```

### `=` 赋值

给已有变量重新赋值。

当前语义：

- 优先更新最近作用域中的同名变量
- 如果该变量有类型约束，则新值必须匹配

```go
x := 1
x = 2
```

### `var` 显式声明

`var` 支持：

- 仅声明类型
- 声明类型并初始化
- 不写类型、只给初始值

```go
var count int
var name string
var ready bool

var retries int = 3
var title = "kami"
```

### 零值规则

如果 `var` 只写类型不写初始值，则会生成零值：

- `int` -> `0`
- `string` -> `""`
- `bool` -> `false`

```go
var count int
print count   // 0

var title string
print title   // 空字符串

var ok bool
print ok      // false
```

### `nil` 与变量类型

`nil` 只是空值，不是变量静态类型。

```go
x := nil
x = 1

var y = nil
y = true
```

## 5. 字符串与插值

### 字符串字面量

```go
msg := "hello"
```

### 插值

支持在字符串中用 `$变量名` 读取变量值：

```go
name := "kami"
print "hello $name"
```

也支持独立插值：

```go
print $name
```

### 环境变量回退

如果当前脚本变量中找不到某个名字，字符串展开时会继续尝试读取系统环境变量。

```go
print $PATH
```

### 转义字符

当前支持常见转义：

- `\n`
- `\t`
- `\r`
- `\"`
- `\\`

## 6. 表达式

当前已实现的核心表达式：

- 加法 / 字符串拼接：`+`
- 等于：`==`
- 不等于：`!=`
- 大于：`>`
- 小于：`<`
- 括号分组：`( ... )`

```go
x := 10 + 20
print x

if x == 30 {
    print "ok"
}
```

### 拼接规则

`+` 在当前语义下：

- 两边都是整数时做整数加法
- 任一边是字符串时做字符串拼接

```go
print 1 + 2
print "hello " + "kami"
print "count=" + 3
```

## 7. 控制结构

### `if` / `else`

```go
x := 10
if x > 5 {
    print "high"
} else {
    print "low"
}
```

说明：

- 条件不需要括号
- 块必须使用 `{ ... }`
- `else` 可选

### `for`

当前支持：

- 条件循环
- 无限循环

```go
i := 0
for i < 3 {
    print i
    i = i + 1
}
```

```go
for {
    print "loop"
}
```

## 8. 函数

### `func` 定义函数

```go
func greet(name) {
    print "hello " + name
}
```

### 调用函数

当前常见可用形式：

- 表达式调用：`greet("kami")`
- 命令式调用：`greet "kami"`

```go
func greet(name) {
    print name
}

greet("kami")
greet "shell"
```

### 作用域

函数支持词法作用域，可以读取外层变量，也可以更新最近作用域中的同名变量。

```go
x := 1

func update() {
    x = 2
}

update()
print x
```

## 9. 命令执行

### 直接命令执行

输入命令名和参数即可执行。

```bash
ls -la
pwd
go build
```

### 命令查找顺序

当前大致按以下顺序解析：

1. 原生函数
2. 用户定义函数 / 环境中的可调用对象
3. 内置命令
4. 系统 PATH 下的外部命令

### `exec`

把字符串强制当命令执行，适合处理关键字冲突或动态拼接命令。

```go
exec "go run main.go"
```

## 10. 管道、重定向与逻辑链

### `|` 管道

```bash
print "line1\nline2" | cat
```

### `>` 覆盖重定向

```bash
print "hello" > "out.txt"
```

### `>>` 追加重定向

```bash
print "hello" >> "out.txt"
```

### `&&` 逻辑与命令链

```bash
mkdir build && cd build
```

### `||` 逻辑或命令链

```bash
ls missing || print "not found"
```

## 11. 并发与后台执行

### `&` 后台执行

把当前语句放到后台执行。

```bash
sleep 10 &
```

### `go`

以 goroutine 风格异步执行命令或代码块。

```go
go {
    print "async"
}
```

```go
go sleep 5
```

## 12. 错误处理

运行时会维护一个特殊变量 `err`。

- 上一次执行失败时，`err` 是一个错误对象
- 成功时，`err` 为 `nil`

```go
ls missing_file
if err != nil {
    print err
}
```

注意：

- 目前 `err` 作为变量可用
- 但像 `err.Message` 这种通用对象字段访问，目前还不是完整对象系统的一部分，文档里不要把它当成稳定能力依赖

## 13. 脚本内 `env` 包

当前内置了脚本级 `env` 包，用来保存脚本内部键值状态，不和普通变量混用。

### 已实现函数

- `env.Set("KEY", "VALUE")`
- `env.Get("KEY")`
- `env.Unset("KEY")`

```go
env.Set("GOOS", "linux")
print env.Get("GOOS")
env.Unset("GOOS")
```

这个作用域特别适合：

- 构建变量
- 脚本内部配置
- `.km` 中的目标级参数传递

## 14. `.km` / make DSL

Kamishell 内置了 `make` 构建系统，使用 `.km` 脚本。

### 入口命令

```bash
make
make build.km
```

### 已实现关键字

- `project`
- `add_executable`
- `add_library`
- `target_link_libraries`
- `target_env`

### 示例

```go
project "Demo"

env.Set("GOOS", "linux")
env.Set("GOARCH", "amd64")

add_executable "app" "main.go"
target_env "app" "CGO_ENABLED=0"
```

更详细的构建说明见 `docs/make.md`。

## 15. Go 标准库导入

Kami 支持通过 `import` 语法导入 Go 标准库函数。

### 语法

```go
import "Go/包名"
```

### 已支持的包

- `fmt` - 格式化输出
- `math` - 数学函数
- `strings` - 字符串处理
- `strconv` - 字符串转换
- `os` - 操作系统功能
- `sync` - 并发同步（WaitGroup）

### 示例

```go
import "Go/fmt"
import "Go/math"
import "Go/strings"

// 使用 fmt 包
fmt.Println("Hello, Kami!")
fmt.Printf("Name: %s, Age: %d\n", "Kami", 1)

// 使用 math 包
x := math.Sqrt(16)
print "sqrt(16) = $x"

// 使用 strings 包
s := "Hello, World!"
contains := strings.Contains(s, "World")
print "contains 'World': $contains"
```

## 16. Go 协程支持

Kami 支持使用 `go` 关键字启动协程。

### 语法

```go
go {
    // 协程代码块
}

go 函数名(参数)
```

### 示例

```go
import "Go/fmt"

// 协程代码块
go {
    fmt.Println("Inside goroutine")
    x := 10 + 20
    fmt.Printf("Result: %d\n", x)
}

// 协程函数调用
func backgroundJob() {
    fmt.Println("Background job started")
    // 模拟工作
    i := 0
    for i < 5 {
        fmt.Printf("Working... %d\n", i)
        i = i + 1
    }
    fmt.Println("Background job completed")
}

go backgroundJob()
```

## 17. WaitGroup 同步

Kami 支持使用 `wg.Go { ... }` 语法进行并发任务同步。

### 语法

```go
import "Go/sync"

wg := sync.NewWaitGroup()
wg.Go { 任务1 }
wg.Go { 任务2 }
wg.Wait()
```

### 示例

```go
import "Go/fmt"
import "Go/sync"

func processTask(id) {
    fmt.Printf("Task %d started\n", id)
    // 模拟工作
    fmt.Printf("Task %d completed\n", id)
}

wg := sync.NewWaitGroup()
wg.Go { processTask(1) }
wg.Go { processTask(2) }
wg.Go { processTask(3) }
wg.Wait()
print "All tasks completed"
```

### 说明

- `wg.Go { ... }` 自动处理 `wg.Add(1)` 和 `wg.Done()`
- `wg.Wait()` 等待所有任务完成
- 任务在独立的 goroutine 中执行

## 18. 当前关键字总览

### 已实现关键字

- `if`
- `else`
- `for`
- `func`
- `go`
- `var`
- `print`
- `exec`
- `import`
- `nil`
- `true`
- `false`

### 已实现的重要符号/操作符

- `:=`
- `=`
- `+`
- `==`
- `!=`
- `>`
- `<`
- `|`
- `>>`
- `&&`
- `||`
- `&`
- `$`
- `.`
- `,`
- `;`
- `(` `)` `{` `}`

### make 相关关键字

- `make`
- `project`
- `add_executable`
- `add_library`
- `target_link_libraries`
- `target_env`

## 16. 当前未实现或未完整实现

下面这些名字可能出现在文档、帮助系统或规划里，但当前不能当成稳定能力使用：

- `return`
- `range`
- `const`
- `import`
- `break`
- `continue`
- `>=`
- `<=`
- `-`
- `*`
- `/`
- 通用对象字段访问
- 完整集合类型（数组、map 等）

## 17. 一个当前真实可用的例子

```go
var count int
name := "kami"

env.Set("GOOS", "linux")

func greet(user) {
    if user == "kami" {
        print "hello " + user
    } else {
        print "unknown"
    }
}

greet(name)

i := 0
for i < 3 {
    print i
    i = i + 1
}

print env.Get("GOOS")
```
