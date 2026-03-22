# 核心接口与组件设计

## 1. 主实例 (`Instance`)
`Instance` 是用户交互的核心类。

```go
type Instance struct {
    Config    *Config
    Terminal  Terminal
    Buffer    *Buffer
    History   History
    Completer Completer
}

func NewInstance(cfg *Config) (*Instance, error)
func (i *Instance) Readline() (string, error)
func (i *Instance) SetPrompt(prompt string)
func (i *Instance) Close() error
```

## 2. 终端接口 (`Terminal`)
用于抽象跨平台的底层操作。

```go
type Terminal interface {
    // Read 从终端读取数据
    Read(p []byte) (n int, err error)
    // Write 向终端写入数据
    Write(p []byte) (n int, err error)
    // GetSize 获取当前终端窗口大小
    GetSize() (width, height int, err error)
    // SetRaw 进入原始模式，并返回恢复函数
    SetRaw() (restore func(), err error)
}
```

## 3. 编辑缓冲区 (`Buffer`)
管理当前行内容和光标。

```go
type Buffer struct {
    data   []rune
    cursor int // 逻辑索引
}

func (b *Buffer) Insert(r rune)
func (b *Buffer) Delete()
func (b *Buffer) MoveLeft()
func (b *Buffer) MoveRight()
func (b *Buffer) String() string
```

## 4. 自动补全 (`Completer`)
允许用户自定义补全逻辑。

```go
type Completer interface {
    // Complete 返回补全建议和建议替换的长度
    Complete(line string, pos int) (suggestions []string, prefixLen int)
}
```

## 5. 历史记录 (`History`)

```go
type History interface {
    Append(line string)
    Get(index int) (string, bool)
    Len() int
    Save() error
    Load() error
}
```

## 6. 配置项 (`Config`)

```go
type Config struct {
    Prompt          string
    HistoryFile     string
    HistoryLimit    int
    Completer       Completer
    InterruptPrompt string // 如 "^C"
}
```
