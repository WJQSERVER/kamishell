# Readline (Pure Go) 深度使用指南

`github.com/WJQSERVER/readline` 是一个完全由 Go 编写的、无 cgo 依赖的跨平台行编辑库。它不仅在 Unix 类系统（Linux/macOS）上表现卓越，也针对 Windows 控制台进行了深度优化。

> **⚠️ 实验状态说明**
> 本项目目前定义为**实验性原型项目**。虽然核心功能已经可用，但 API 设计仍在演进中。如果您在关键项目中使用，请务必关注版本更新说明。

---

## 📖 目录
1. [快速安装](#-快速安装)
2. [核心配置 (Config) 详解](#-核心配置-config-详解)
3. [自动补全 (Completer) 进阶](#-自动补全-completer-进阶)
4. [历史记录持久化方案](#-历史记录持久化方案)
5. [跨平台渲染与 Unicode](#-跨平台渲染与-unicode)
6. [并发安全与信号处理](#-并发安全与信号处理)
7. [高级快捷键与交互](#-高级快捷键与-交互)
8. [常见问题 (FAQ)](#-常见问题-faq)

---

## 🚀 快速安装

本项目要求使用 **Go 1.26** 或更高版本，以利用最新的语言特性和性能优化。

```bash
go get github.com/WJQSERVER/readline
```

---

## 🛠 核心配置 (Config) 详解

`Config` 结构体是控制 `Readline` 行为的关键。

```go
type Config struct {
    // Prompt: 每一行开始显示的提示符。支持包含 ANSI 颜色序列。
    Prompt string

    // History: 实现 History 接口的对象。
    // 如果为 nil，Readline 会自动创建一个内存 history。
    History History

    // Completer: 实现 Completer 接口的对象，用于处理 Tab 键补全。
    Completer Completer

    // Stdin: 数据读取源。默认为 os.Stdin。
    // 在测试环境或需要重定向输入时，可以将其设置为自定义的 *os.File。
    Stdin *os.File

    // Stdout: 数据输出源。默认为 os.Stdout。
    // 注意：Stdout 必须支持终端控制序列才能实现行编辑功能。
    Stdout *os.File
}
```

---

## 🧠 自动补全 (Completer) 进阶

### 1. 简单的静态补全
使用内置的 `PrefixCompleter`：

```go
completer := &readline.PrefixCompleter{
    Candidates: []string{"help", "status", "exit", "version", "你好"},
}
```

### 2. 动态上下文补全
通过实现 `Completer` 接口，您可以根据当前已输入的内容进行复杂的逻辑判断。

```go
type DynamicCompleter struct{}

func (d *DynamicCompleter) Do(line []rune, pos int) (candidates [][]rune, length int) {
    // line: 当前输入框内的完整字符切片
    // pos: 当前光标所在的逻辑位置

    currentLine := string(line[:pos])
    words := strings.Fields(currentLine)

    // 逻辑示例：如果第一个单词是 "get"，则补全资源名称
    if len(words) > 0 && words[0] == "get" {
        resources := []string{"users", "orders", "products"}
        // 获取当前正在输入的后缀
        lastWord := ""
        if !strings.HasSuffix(currentLine, " ") {
            lastWord = words[len(words)-1]
        }

        var matches [][]rune
        for _, r := range resources {
            if strings.HasPrefix(r, lastWord) {
                matches = append(matches, []rune(r))
            }
        }
        return matches, len([]rune(lastWord))
    }

    return nil, 0
}
```

---

## 📜 历史记录持久化方案

当前内置的历史记录是基于内存的。如果您需要跨会话保留历史记录，可以实现一个简单的装饰器：

```go
type PersistentHistory struct {
    readline.History // 组合内置的内存 history
    filepath string
}

func (ph *PersistentHistory) Append(line string) {
    ph.History.Append(line)
    // 实时同步到文件
    f, _ := os.OpenFile(ph.filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    defer f.Close()
    f.WriteString(line + "\n")
}

// 在初始化时从文件加载
func LoadHistory(path string) *PersistentHistory {
    h := &PersistentHistory{History: readline.NewHistory(), filepath: path}
    content, _ := os.ReadFile(path)
    lines := strings.Split(string(content), "\n")
    for _, l := range lines {
        if l != "" {
            h.History.Append(l)
        }
    }
    return h
}
```

---

## 🖥 跨平台渲染与 Unicode

### 宽字符对齐
我们在渲染引擎中集成了 `go-runewidth`，这使得 Readline 能够识别字符的视觉宽度。例如：
- `a` 的视觉宽度为 1。
- `你` 的视觉宽度为 2。
- `🚀` 的视觉宽度为 2。

这保证了在移动光标或删除字符时，屏幕显示不会发生错位。

### Windows VT 模式
在 Windows 10 及更高版本上，Readline 会尝试激活 **Virtual Terminal Processing**。这意味着您可以在 Windows 上直接使用 ANSI 颜色序列（如 `\x1b[32m>\x1b[0m `）作为提示符，且能获得与 Linux 下一致的平滑体验。

---

## ⚠️ 并发安全与信号处理

### ErrInterrupt (Ctrl+C)
当用户按下 `Ctrl+C` 时，`Readline()` 会立即返回 `readline.ErrInterrupt`。
**建议做法**：捕获该错误并清理当前行的显示状态。

```go
line, err := rl.Readline()
if err == readline.ErrInterrupt {
    // 仅清除当前行，不退出程序
    continue
}
```

### 并发调用
`Instance` **不是协程安全**的。您不应该在多个 goroutine 中同时调用同一个实例的 `Readline()` 方法。

---

## ⌨️ 高级快捷键与交互

除了基础操作，我们还支持以下高级快捷键：

| 快捷键 | 动作描述 |
| :--- | :--- |
| **Ctrl + A** / **Home** | 光标跳转至行首 |
| **Ctrl + E** / **End** | 光标跳转至行尾 |
| **Ctrl + F** / **→** | 光标向右移动 |
| **Ctrl + B** / **←** | 光标向左移动 |
| **Ctrl + L** | 强制刷新当前行（逻辑重绘） |
| **Ctrl + K** | 删除光标位置到行尾的所有内容 |
| **Ctrl + U** | 删除行首到光标位置的所有内容 |
| **Ctrl + W** | 删除光标前的一个单词 |
| **Ctrl + D** | 行首时退出 (EOF)，非行首时删除光标处字符 |

---

## ❓ 常见问题 (FAQ)

**Q: 为什么提示符中的 ANSI 颜色没有生效？**
A: 请确保您的终端支持颜色，并且 `Config.Stdout` 指向的是一个真实的终端设备。在某些 IDE 的内置控制台中，可能需要手动开启相关支持。

**Q: 如何处理窗口大小调整 (Window Resize)？**
A: 当前版本在每次渲染时会重新获取终端宽度。如果发生窗口调整，下一次输入或光标移动会触发自适应重绘。

**Q: 是否支持多行编辑？**
A: 目前版本仅支持单行编辑。多行编辑已列入 [Roadmap](../roadmap.md)。

---

## 🤝 贡献与反馈
由于本项目是实验性的，我们非常欢迎 Issue 和 PR。如果您发现了渲染错位或按键识别问题，请提供您的操作系统和终端类型信息。
