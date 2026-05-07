# Kamishell 使用手册

## 1. 安装与编译

要求: **Go 1.26** 或更高版本。

```bash
go build -o kami .
```

## 2. 交互式模式 (REPL)

直接运行 `./kami` 即可进入交互式 Shell。

### REPL 特性
- **统一 Readline 实现**: 当前默认使用项目内维护的 `readline` 实现。
- **动态提示符**: 提示符会显示当前目录名，并在 `cd` 后自动刷新。
- **历史记录**: 使用方向键（上/下）浏览执行过的历史记录。历史保存在 `~/.kami_history`。
- **自动补全**: 输入部分命令、变量名或文件路径后按下 **Tab** 键。
- **启动配置**: 启动时自动加载并执行 `$HOME/.kamirc` 和当前目录下的 `.kamirc`。

## 3. 内置命令参考

为了提供一致的跨平台体验，以下工具由 Go 纯手工打造：

### 基础文件操作
- **`ls [-a] [-l] [-h] [-F] [target]`**: 列出目录或文件。
- **`cp source destination`**: 复制文件。
- **`mv source destination`**: 移动或重命名文件。
- **`mkdir directory`**: 创建目录。
- **`rm target`**: 删除文件或目录。
- **`touch file`**: 创建空文件或更新时间戳。

### 文本处理
- **`grep pattern [file...]`**: 在输入流或文件中搜索匹配项。
- **`sed s/old/new/ [file...]`**: 简单的全局文本替换。

### 网络请求
- **`http [METHOD] URL [flags]`**: 发送内嵌 HTTP 请求，支持结构化请求体、认证、重试和多种响应输出模式。

### 系统信息与状态
- **`pwd`**: 显示当前工作目录。
- **`cd [dir]`**: 切换目录（默认为 HOME）。
- **`type name`**: 显示名称的类型（函数、内置命令、外部路径或变量）。
- **`which name`**: 在 PATH 中搜索外部命令的完整路径。
- **`jobs`**: 列出正在后台运行或已完成的任务（由 `&` 或 `go` 启动）。
- **`help`**: 显示内建命令的帮助信息。
- **`print [arg...]`**: 向终端打印信息（支持插值和拼接）。
- **`exit [code]`**: 退出 Shell，可选返回状态码（默认 0）。

`http` 设计约定：

```text
- 默认方法为 GET
- 使用 --data / --json / --form 时，默认方法自动切换为 POST
- 仅允许一种请求体模式：raw、json、form
- 输出模式互斥：默认 body；--include = 元数据+body；--headers = 仅元数据；--status = 仅状态行
```

`http` 示例：

```bash
# 简单 GET
http https://example.com

# 发送 JSON（自动 POST + application/json）
http https://api.example.com/items --json '{"name":"kami"}'

# 发送表单
http https://api.example.com/items --form name=kami --form lang=zh

# 添加认证与请求头
http https://api.example.com/profile --auth kami:secret --header "Accept: application/json"

# 从管道读取请求体
print '{"name":"kami"}' | http --method POST --header "Content-Type: application/json" --data - https://api.example.com/items

# 仅查看响应头
http --headers https://example.com
```

## 4. 脚本模式

运行脚本文件：
```bash
./kami script.sh
```

或作为可执行文件（需配置 Shebang）：
```bash
./my_script.sh
```

## 5. 调试与开发

- 运行测试: `go test ./...`
- 性能评估: `go test -bench=. ./...`

## 6. 深入了解

- [分词器实现细节](tokenizer.md): 了解 Kamishell 如何解析命令以及处理交互层面的单词边界。
