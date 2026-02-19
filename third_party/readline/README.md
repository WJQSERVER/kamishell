# Readline (Pure Go)

> **⚠️ 实验性原型项目 (Experimental Prototype)**
> 本项目目前处于实验阶段，旨在探索纯 Go 实现的跨平台 Readline 库。API 可能会发生剧烈变化，不建议在生产环境中使用。

`github.com/WJQSERVER/readline` 是一个纯 Go 编写的、无 cgo 依赖的多平台兼容 Readline 库。它支持 Linux, macOS 和 Windows，并对中文等 Unicode 字符有良好的显示支持。

## 核心特性

- **纯 Go 实现**: 无需 cgo，跨平台编译更简单。
- **多平台兼容**: 完美支持 Unix (termios) 和 Windows (Console API)。
- **Unicode 支持**: 准确处理中文字符的显示宽度（基于 `go-runewidth`）。
- **功能完备**: 支持基础编辑、历史记录、自动补全。
- **实验性**: 灵活的架构设计，易于扩展。

## 快速开始

### 安装

```bash
go get github.com/WJQSERVER/readline
```

### 简单示例

```go
package main

import (
	"fmt"
	"io"
	"github.com/WJQSERVER/readline"
)

func main() {
	rl, err := readline.NewInstance(&readline.Config{
		Prompt: "> ",
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println("Error:", err)
			continue
		}
		fmt.Println("Typed:", line)
	}
}
```

## 更多文档

- [使用指南](docs/USAGE.md)
- [架构设计](plan/ARCHITECTURE.md)
- [发展路线图](roadmap.md)

## 许可证

Mozilla Public License 2.0
