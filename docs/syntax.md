# Kamishell 当前语法与关键字参考

这份文档描述的是 Kamishell **当前代码实际支持** 的语法与关键字，而不是理想设计或长期规划。

如果某个能力仍在规划中，本文会明确标记为"未实现"或"部分实现"。

## 1. 语言定位

Kamishell 是一种混合型 Shell 语言：

- 可以像传统 Shell 一样直接执行命令、管道、重定向
- 也可以像轻量脚本语言一样写变量、条件、循环、函数
- 在 `.km` 构建脚本里还能作为 `make` DSL 使用
- 支持编译为原生二进制（`--compile`），生成高效 Go 代码

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

- `INTEGER` — 64 位有符号整数
- `FLOAT` — 64 位浮点数
- `STRING` — 字符串
- `BOOLEAN` — `true` / `false`
- `ARRAY` — 同构数组（元素必须同一类型）
- `FUNCTION` — 函数（含闭包），带有完整签名信息
- `NULL` — 空值，源码字面量写作 `nil`
- `ERROR` — 错误对象
- `PACKAGE` — 包对象（`env`、`sync` 等）

### 字面量

```go
42
3.14
"hello"
true
false
nil
[1, 2, 3]
```

### 变量静态类型约束

Kamishell 是静态类型语言，变量一旦赋值，类型即固定：

```go
x := 10
x = "hello"   // 错误：cannot assign STRING to variable of type INTEGER

arr := [1, 2, 3]
arr[0] = "a"  // 错误：cannot assign STRING to ARRAY[INTEGER] element
```

支持的显式类型声明：

- `int` / `integer`
- `float` / `float64`
- `string`
- `bool` / `boolean`
- `array`

注意：

- `nil` 是运行时空值，`:=` 不能用于 `nil`（`x := nil` 会报错：`untyped nil cannot be used with :=`）
- `nil` 可以赋给已声明的函数变量（引用类型语义）：`f := func(a int) int { return a }; f = nil`

### 函数签名类型

函数变量携带完整的签名信息。`func(a, b int) int` 和 `func(x string)` 是不同的类型：

```go
f := func(a, b int) int { return a + b }
f = func(a, b int) int { return a * b }   // ✅ 同签名
f = func(x string) { print x }            // ❌ 签名不兼容
f = nil                                    // ✅ nil 可赋给引用类型
```

## 4. 变量与赋值

### `:=` 短变量声明

在当前作用域声明变量并赋值，同时记录推断出的类型。

```go
x := 10
name := "kami"
ready := true
arr := [1, 2, 3]
```

### `=` 赋值

给已有变量重新赋值。

当前语义：

- 优先更新最近作用域中的同名变量
- 如果该变量有类型约束，则新值必须匹配
- 数组赋值遵循值语义（拷贝）

```go
x := 1
x = 2

a := [1, 2, 3]
b := a        // 值拷贝
b[0] = 99     // a 不变
```

### `var` 显式声明

`var` 支持显式类型声明和类型推断：

```go
// 显式类型 + 零值
var count int
var name string
var ready bool
var arr array

// 显式类型 + 初始值
var retries int = 3

// 类型推断（从值推断类型）
var title = "kami"     // 推断为 string
var x = 42             // 推断为 int
var ok = true          // 推断为 bool
var arr = [1, 2, 3]    // 推断为 array
```

注意：`var x = nil`（无类型 nil）**不合法**，因为 nil 无法推断类型。

### 零值规则

如果 `var` 只写类型不写初始值，则会生成零值：

- `int` -> `0`
- `string` -> `""`
- `bool` -> `false`
- `array` -> `[]`

```go
var count int
print count   // 0

var title string
print title   // 空字符串

var ok bool
print ok      // false
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

### 算术运算

```go
x := 10 + 20     // 整数加法
y := 3.14 * 2.0  // 浮点乘法
z := 10 - 3      // 减法
w := 100 / 5     // 除法
```

已实现：

- `+` 加法 / 字符串拼接
- `-` 减法
- `*` 乘法
- `/` 除法
- `==` 等于
- `!=` 不等于
- `>` 大于
- `<` 小于
- `!` 逻辑非（前缀）

未实现：

- `>=` 大于等于（lexer 未生成 token，runtime 有死代码但不可达）
- `<=` 小于等于（同上）
- `%` 取模

### 逻辑非 `!`

```go
x := true
print !x          // false

if !false {
    print "ok"
}
```

### 拼接规则

`+` 在当前语义下：

- 两边都是整数时做整数加法
- 两边都是浮点数时做浮点加法
- 任一边是字符串时做字符串拼接

```go
print 1 + 2
print "hello " + "kami"
print "count=" + 3
```

### 括号分组

```go
x := (10 + 20) * 3
y := (a + b) / 2
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

### `switch` / `case`

```go
x := 3
switch x {
case 1:
    print "one"
case 2:
    print "two"
case 3:
    print "three"
default:
    print "other"
}
```

特性：

- 支持整数、字符串、布尔值比较
- 整数 case 自动使用二分查找优化
- 字符串 case 使用直接比较优化
- 支持 `default` 分支
- 支持无 tag 的 switch（类似 if-else 链）

```go
x := 15
switch {
case x > 10:
    print "big"
case x > 5:
    print "medium"
default:
    print "small"
}
```

### `for` 循环

#### 条件循环（while 风格）

```go
i := 0
for i < 3 {
    print i
    i = i + 1
}
```

#### 无限循环

```go
for {
    print "loop"
}
```

#### 三段式循环（C 风格）

```go
for i := 0; i < 10; i = i + 1 {
    print i
}
```

#### 数组 range

```go
arr := [10, 20, 30]

// 仅索引
for i := range arr {
    print i
}

// 索引 + 值
for i, v := range arr {
    print i; print v
}

// 无变量
for range arr {
    print "tick"
}
```

#### 迭代器 range（range-over-func）

```go
// 定义迭代器
func countTo(n) {
    return func(yield) {
        i := 0
        for i < n {
            if !yield(i) { return }
            i = i + 1
        }
    }
}

// 使用
for v := range countTo(5) {
    print v
}

// 双变量迭代器
func enumerate(arr) {
    return func(yield) {
        for i := range arr {
            if !yield(i, arr[i]) { return }
        }
    }
}

for i, v := range enumerate([10, 20, 30]) {
    print i; print v
}
```

### `break` / `continue`

```go
for i := 0; i < 10; i = i + 1 {
    if i == 3 { continue }
    if i == 7 { break }
    print i
}
```

## 8. 数组

### 数组字面量

```go
arr := [1, 2, 3]
names := ["alice", "bob", "charlie"]
flags := [true, false, true]
```

数组是同构的——所有元素必须同一类型：

```go
[1, 2, 3]           // OK: ARRAY[INTEGER]
["a", "b", "c"]     // OK: ARRAY[STRING]
[1, "hello", true]  // 错误：mixed types
```

### 索引访问

```go
arr := [10, 20, 30]
print arr[0]   // 10
print arr[2]   // 30
```

### 索引赋值

```go
arr := [1, 2, 3]
arr[0] = 99
print arr      // [99, 2, 3]
```

### 值语义

数组赋值是值拷贝：

```go
a := [1, 2, 3]
b := a
b[0] = 99
print a   // [1, 2, 3] — a 不受影响
```

### 数组比较

```go
a := [1, 2, 3]
b := [1, 2, 3]
print a == b   // true

c := [1, 2, 4]
print a == c   // false
```

### 内置函数

```go
arr := [1, 2, 3]
print len(arr)        // 3
arr2 := push(arr, 4)  // [1, 2, 3, 4]
```

### 空数组

```go
arr := []
print len(arr)   // 0
```

## 9. 函数

### `func` 定义函数

函数参数建议使用类型注解。有类型注解的参数在运行时强制类型检查和参数数量检查：

```go
func greet(name string) {
    print "hello " + name
}
```

无类型注解的参数（如 `func foo(a, b) { }`）在 parser 中会被静默丢弃，**不推荐使用**。

### 多参数与类型共享

支持 Go 风格的 `(a, b T)` 简写：

```go
// 逐个声明
func add(a int, b int) int {
    return a + b
}

// 共享类型简写
func add(a, b int) int {
    return a + b
}

// 混合类型
func greet(name string, age int) {
    print name + " is " + age
}

// 三个参数共享类型
func sum(a, b, c int) int {
    return a + b + c
}
```

### 运行时类型强制

参数类型在运行时强制执行：

```go
func add(a, b int) int { return a + b }
add(1, 2)       // ✅
add("x", 2)     // ❌ parameter a: expected INTEGER, got STRING
add(1)           // ❌ expected 2 arguments, got 1
add(1, 2, 3)    // ❌ expected 2 arguments, got 3
```

使用 `any` 类型可接受任意参数：

```go
func echo(v any) { print v }
echo(42)        // ✅
echo("hello")   // ✅
```

### 函数常量

`func` 声明的函数名是**常量标识符**，不可重赋值：

```go
func add(a, b int) int { return a + b }
add = func(a, b int) int { return a * b }  // ❌ cannot assign to constant add
```

函数字面量赋值的变量可以重赋值（签名必须匹配）：

```go
f := func(a, b int) int { return a + b }
f = func(a, b int) int { return a * b }    // ✅ 同签名
f = func(x string) { print x }             // ❌ 签名不兼容
```

### 返回值类型

```go
// 单返回值
func add(a int, b int) int {
    return a + b
}

// 多返回值 (Go 风格)
func div(a int, b int) (int, error) {
    if b == 0 {
        return 0, error("division by zero")
    }
    return a / b, nil
}

// 无返回值 (void)
func greet(name string) {
    print "hello " + name
}
```

### 多值解包赋值

```go
ok, err := check_positive(5)
if err != nil {
    print "error: " + err
}
print ok
```

### `error()` 构造器

```go
func validate(age int) (bool, error) {
    if age < 0 {
        return false, error("negative age")
    }
    return true, nil
}
```

### 匿名函数

匿名函数同样需要类型注解，支持多返回值：

```go
add := func(a int, b int) int { return a + b }
print add(3, 4)   // 7

// 多返回值闭包
divmod := func(a, b int) (int, int) { return a / b, a - a / b * b }
q, r := divmod(17, 5)
print q   // 3
print r   // 2

// 无返回值闭包
greet := func(name string) { print "hello " + name }
greet("world")
```

### 调用函数

当前常见可用形式：

- 表达式调用：`greet("kami")`
- 命令式调用：`greet "kami"`

```go
func greet(name string) {
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

### `return`

```go
func add(a int, b int) int {
    return a + b
}
result := add(3, 4)
print result   // 7
```

## 10. 命令执行

### 直接命令执行

输入命令名和参数即可执行。

```bash
ls -la
pwd
go build
```

### 命令查找顺序

当前大致按以下顺序解析：

1. 原生函数（`len`、`push` 等）
2. 用户定义函数 / 环境中的可调用对象
3. 内置命令（`ls`、`cd`、`cat` 等）
4. 系统 PATH 下的外部命令

### `exec`

把字符串强制当命令执行，适合处理关键字冲突或动态拼接命令。

```go
exec "go run main.go"
```

## 11. 管道、重定向与逻辑链

### `|` 管道

```bash
print "line1\nline2" | cat
```

### `->` 覆盖重定向

```bash
print "hello" -> "out.txt"
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

## 12. 并发与后台执行

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

go sleep 5
```

### Task/Future

```go
t := go { return 42 }
result := t.Wait()
result := t.Wait(10)  // 带超时
```

### WaitGroup

```go
wg := sync.NewWaitGroup()
wg.Go { task1() }
wg.Go { task2() }
wg.Wait()
```

带超时：

```go
wg := sync.NewWaitGroup()
wg.Go { doWork() }
wg.Wait(5)  // 最多等待 5 秒
if err != nil {
    print "timeout"
}
```

### `wait` 命令

```go
go { task1() }
go { task2() }
wait           // 等待所有任务完成
wait(10)       // 等待最多 10 秒
```

## 13. 错误处理

运行时会维护一个特殊变量 `err`。

- 上一次执行失败时，`err` 是一个错误对象
- 成功时，`err` 为 `nil`

```go
ls missing_file
if err != nil {
    print err
}
```

## 14. 脚本内 `env` 包

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

## 15. `.km` / make DSL

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

## 16. Go 标准库导入

Kami 支持通过 `import` 语法导入 Go 标准库函数，编译时直接解析为原生 Go 调用。

### 语法

```go
import "Go/包名"
```

### 已支持的包

- `fmt` — 格式化输出（`Println`、`Printf`、`Sprintf`）
- `math` — 数学函数（`Sqrt`、`Abs`）
- `strings` — 字符串处理（`Contains`、`HasPrefix`、`HasSuffix`、`Replace`、`Split`、`Join`）
- `strconv` — 类型转换（`Itoa`、`Atoi`）
- `os` — 系统操作（`Getenv`、`Setenv`）

### 内置包（无需 import）

- `env` — 环境变量管理
- `sync` — 并发同步（`NewWaitGroup`、`wg.Go`、`wg.Wait`）

### 示例

```go
import "Go/fmt"
import "Go/strings"

fmt.Println("Hello, Kami!")
print strings.Contains("hello", "ell")   // true
print strings.Replace("hello", "l", "L", -1)  // heLLo
```

注意：解释模式下仅支持 `goStdlib` 中注册的函数（`Contains`、`HasPrefix`、`HasSuffix`、`Replace`、`Split`、`Join`）。编译模式下支持 Go 标准库的任意函数（直接生成 Go 调用）。

## 17. 关键字总览

### 已实现关键字

| 关键字 | 用途 |
|---|---|
| `if` / `else` | 条件分支 |
| `for` | 循环（含三段式、range） |
| `range` | 数组/迭代器遍历 |
| `func` | 函数定义 |
| `return` | 函数返回 |
| `go` | goroutine |
| `var` | 显式类型声明 |
| `print` | 输出 |
| `exec` | 强制命令执行 |
| `import` | Go 标准库导入 |
| `nil` | 空值 |
| `true` / `false` | 布尔字面量 |
| `switch` / `case` / `default` | 分支匹配 |
| `break` / `continue` | 循环控制 |
| `wait` | 等待 goroutine |

### 已实现的重要符号/操作符

| 符号 | 用途 |
|---|---|
| `:=` | 短变量声明 |
| `=` | 赋值 |
| `+` | 加法 / 字符串拼接 |
| `-` | 减法 / 重定向箭头（`->`） |
| `*` | 乘法 / 指针解引用 |
| `/` | 除法 |
| `!` | 逻辑非 |
| `==` / `!=` | 等于 / 不等于 |
| `>` / `<` | 大于 / 小于 |
| `\|` | 管道 |
| `->` | 覆盖重定向 |
| `>>` | 追加重定向 |
| `&&` / `\|\|` | 逻辑与 / 或 |
| `&` | 后台执行 / 取地址 |
| `$` | 变量插值 |
| `.` | 成员访问 |
| `[` `]` | 数组索引 |
| `(` `)` `{` `}` | 分组 / 块 |
| `,` / `;` | 分隔符 |

### make 相关关键字

- `make`
- `project`
- `add_executable`
- `add_library`
- `target_link_libraries`
- `target_env`

## 18. 当前未实现或未完整实现

下面这些名字可能出现在文档、帮助系统或规划里，但当前不能当成稳定能力使用：

- `const` — 常量声明（`func` 声明的函数名已是常量，但通用 `const` 语法未实现）
- 通用对象字段访问（仅支持已导入包的成员访问，如 `fmt.Println`；自定义对象成员访问不支持）
- map 类型
- 字符串 range（按字符迭代）
- 命名返回参数（Go 的 `func foo() (result int, err error)` 语法）
- `>=` / `<=` 比较运算符（lexer 未实现，runtime 有死代码）
- `%` 取模运算符（lexer 未实现）

## 19. 综合示例

```go
// 数组 + range + break/continue
arr := [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
for i, v := range arr {
    if v == 3 { continue }
    if v == 8 { break }
    print v
}

// 迭代器
func countTo(n int) {
    return func(yield func(int) bool) {
        i := 0
        for i < n {
            if !yield(i) { return }
            i = i + 1
        }
    }
}
for v := range countTo(5) { print v }

// switch/case（原生 Go switch，无 ToStr 开销）
x := 3
switch x {
case 1: print "one"
case 2: print "two"
case 3: print "three"
default: print "other"
}

// 强类型函数（运行时检查参数类型和数量）
func add(a, b int) int { return a + b }
print add(3, 4)   // 7

// 多返回值函数
func divmod(a, b int) (int, int) { return a / b, a - a / b * b }
q, r := divmod(17, 5)
print q   // 3
print r   // 2

// 匿名函数（支持多返回值）
f := func(a, b int) (int, int) { return a + b, a - b }
x, y := f(5, 3)
print x   // 8
print y   // 2

// 函数常量（不可重赋值）
func greet(name string) { print "hello " + name }
greet("world")

// 并发（Go 1.26 原生 wg.Go）
wg := sync.NewWaitGroup()
wg.Go { print "task1" }
wg.Go { print "task2" }
wg.Wait()

// WaitGroup 带超时
wg := sync.NewWaitGroup()
wg.Go { doWork() }
wg.Wait(5)
if err != nil { print "timeout" }

// Task/Future
t := go { return 42 }
result := t.Wait()
print result

// Go 标准库（编译期直接解析为原生调用）
import "Go/fmt"
import "Go/strings"
fmt.Println("Hello, Kami!")
print strings.Contains("hello", "ell")   // true
```
