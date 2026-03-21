# Kamishell 构建系统 (Make) 编写指南

`make` 是 Kamishell 内置的自动化构建工具。它允许你使用 Kamishell 原生的脚本语法（`.km` 文件）来定义项目结构、编译目标和依赖关系，从而简化构建流程。

## 1. 快速入门

### 运行命令
在包含构建脚本的目录下执行：
```bash
make
```
或者指定特定的脚本文件：
```bash
make my_project.km
```

### 默认搜索规则
如果不指定文件名，`make` 会按以下逻辑搜索脚本：
1. 如果目录下只有一个 `.km` 文件，则直接使用。
2. 如果有多个，优先寻找 `Kami.km` 或 `build.km`。
3. 否则，你需要手动指定文件名。

---

## 2. 脚本语法与内置函数

`.km` 脚本使用 Kamishell 的标准语法，你可以使用变量、字符串拼接等特性。

### `project`
定义项目的显示名称。
- **语法**: `project "项目名称"`
- **示例**: `project "MyAwesomeApp"`

### `add_executable`
定义一个可执行二进制文件目标。
- **语法**: `add_executable "目标名称" "源文件1" "源文件2" ...`
- **示例**: `add_executable "server" "main.go" "router.go"`
- **Go 包入口**: 也支持 `add_executable "server" "."`，此时会直接执行 `go build -o server .`
- **注意**: 在 Windows 平台上，构建系统会自动为目标名称添加 `.exe` 后缀。

### `add_library`
定义一个库文件（或模块）目标。
- **语法**: `add_library "库名称" "源文件1" ...`
- **示例**: `add_library "utils" "logger.go" "helpers.go"`
- **Go 包入口**: 也支持 `add_library "utils" "."`，按当前目录包构建。

### `target_link_libraries`
声明一个目标对其他库的依赖关系。
- **语法**: `target_link_libraries "目标名称" "库1" "库2" ...`
- **示例**: `target_link_libraries "server" "utils"`
- **说明**: 目前该函数主要用于声明逻辑依赖，实际编译时会优先确保依赖项的存在。

### `target_env`
为指定目标显式设置构建环境变量。
- **语法**: `target_env "目标名称" "变量1=值1" "变量2=值2" ...`
- **示例**: `target_env "server" "GOOS=linux" "GOARCH=arm64" "CGO_ENABLED=0"`
- **说明**: 适合给不同目标设置不同的构建变量，且不会影响其他目标。
- **重要**: 这些变量会传给 `go build` 构建过程本身，不会自动嵌入到生成程序的运行时环境中。

### 构建时变量快照
`make` 在执行 `add_executable` / `add_library` 时，会自动快照当前脚本内 `env` 包中的变量，并把它们作为该目标的构建环境。

- **常见变量**: `GOOS`, `GOARCH`, `CGO_ENABLED`, `CC`, `CXX`, `AR`, `PKG_CONFIG`
- **适用范围**: 这些变量会传给实际执行的 `go build`
- **设置方式**: 使用 `env.Set("变量名", "值")`、`env.Get("变量名")`、`env.Unset("变量名")`
- **注意**: 只有目标定义之前已经设置好的值会进入该目标；如果你想在目标创建后再覆盖，使用 `target_env`

---

## 3. 实战演示

### 示例 A：基础单目标项目
**文件名: `build.km`**
```bash
# 设置项目名
project "HelloWorld"

# 定义主程序及其源码
add_executable "hello" "main.go"
```

### 示例 A-2：直接构建当前 Go 包
**文件名: `build.km`**
```bash
project "HelloPkg"

# 使用 go build . 的方式构建当前包
add_executable "hello" "."
```

### 示例 B：使用变量管理多源文件
**文件名: `Kami.km`**
```bash
project "WebScanner"

# 定义源码变量 (Kamishell 语法)
core_src := "scanner.go parser.go"
network_src := "client.go proxy.go"

# 构建可执行文件
add_executable "scanner_bin" "main.go" $core_src $network_src
```

### 示例 C：模块化构建
**文件名: `build.km`**
```bash
project "SystemMonitor"

# 1. 定义一个工具库
add_library "sysutils" "cpu.go" "mem.go"

# 2. 定义主程序并链接库
add_executable "monitor" "main.go"
target_link_libraries "monitor" "sysutils"
```

### 示例 D：跨平台构建变量
**文件名: `build.km`**
```bash
project "CrossBuild"

# 目标创建前定义到 env 包中的构建变量会自动绑定到该目标
env.Set("GOOS", "windows")
env.Set("GOARCH", "amd64")
env.Set("CGO_ENABLED", "0")
add_executable "kami-win" "main.go"

# 继续创建另一个目标时，可以重新设置变量
env.Set("GOOS", "linux")
env.Set("GOARCH", "arm64")
add_executable "kami-linux" "main.go"

# 或者在目标创建后按目标覆盖
target_env "kami-linux" "CGO_ENABLED=1"
```

---

## 4. 技术细节

- **编译器**: 当前版本的 `make` 底层调用 `go build`。
- **构建命令**: 实际执行的命令格式为 `go build -o <目标> <所有源文件>`。
- **包模式**: 如果源写成 `"."`，实际执行命令会变成 `go build -o <目标> .`。
- **环境变量传递**: 目标的构建变量会作为环境变量传递给 `go build`，支持交叉编译。
- **跨平台**: `make` 会根据目标的 `GOOS` 判断是否为可执行文件补全 `.exe` 后缀，而不是只看当前宿主系统。

## 5. 获取帮助
在终端中直接输入以下命令可以查看简易版的内置帮助：
```bash
make help
```
