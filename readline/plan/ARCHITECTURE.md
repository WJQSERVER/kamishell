# PureGo Readline 架构设计

## 核心目标
- **纯 Go 实现**: 不依赖 cgo，方便跨平台编译。
- **多平台兼容**: 完美支持 Linux, macOS 和 Windows。
- **Unicode 支持**: 完美处理中文字符等多字节字符。
- **功能完备**: 包含行编辑、历史记录、自动补全、快捷键绑定等。

## 模块化设计

### 1. 终端抽象层 (Terminal Abstraction - `internal/term`)
负责底层终端状态管理：
- **Raw Mode**: 在 Unix 系统使用 termios，在 Windows 使用 Console API 切换原始模式。
- **窗口大小**: 监听窗口大小变化（Unix 的 SIGWINCH，Windows 的事件轮询）。
- **光标控制**: 通过标准的 ANSI 转义序列进行光标移动和屏幕清理。

### 2. 输入解析层 (Input Parsing - `internal/input`)
- 负责从 `os.Stdin` 读取字节流并解析为符号（Runes）或特殊按键（如方向键、功能键）。
- 处理复杂的转义序列。

### 3. 编辑缓冲区 (Edit Buffer - `internal/buffer`)
- 内部使用 `[]rune` 存储当前行内容，确保 Unicode 字符被视为单个单位。
- 维护逻辑光标位置与视觉偏移。
- 提供插入、删除、删除到行首/行尾等原子操作。

### 4. 渲染引擎 (Rendering Engine - `internal/render`)
- 差分渲染：只更新变化的部分，减少闪烁。
- 处理长行自动换行显示。
- 处理提示符（Prompt）的展示。

### 5. 历史记录管理 (History - `history.go`)
- 内存缓冲区存储最近命令。
- 可选的文件持久化支持。
- 通过上下方向键或 Ctrl+R 进行搜索导航。

### 6. 自动补全 (Completion - `completion.go`)
- 插件化接口 `Completer`。
- 支持 Tab 键循环补全或展示补全列表。

## 关键技术挑战
- **Windows 兼容性**: 旧版 Windows 控制台对 ANSI 转义序列支持有限，需要适配。
- **宽度计算**: 中文字符通常占用两个显示宽度，光标移动需要考虑 `runewidth`。
- **并发安全**: 确保在读取输入时能安全地处理中断信号。


## Windows 适配策略
1. **VT 序列支持**: 优先尝试通过 `SetConsoleMode` 开启 `ENABLE_VIRTUAL_TERMINAL_PROCESSING`，使 Windows Console 支持 ANSI 转义序列。
2. **Fallback**: 如果无法开启 VT 模式，则通过直接调用 Windows API (如 `SetConsoleCursorPosition`) 来实现光标移动（虽然这会增加复杂度，但能保证兼容性）。
3. **输入处理**: 使用 `ReadConsoleInput` 来获取按键事件，以准确识别各种组合键，或者保持 Read 字节流并手动解析。

## 性能与渲染
- 采用 **双缓冲区 (Double Buffering)** 或 **差分更新** 思想，计算新旧屏幕内容的差异，仅输出必要的 ANSI 序列，降低卡顿。
