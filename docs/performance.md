# Kamishell 性能说明

本文档记录当前基准测试覆盖范围与已观察到的主要性能热点，便于后续持续优化。

## 当前 benchmark 覆盖

- `core/lexer_bench_test.go`: 词法分析
- `core/parser_bench_test.go`: 语法分析
- `core/environment_bench_test.go`: 环境查找与脚本包快照
- `core/runtime_bench_test.go`: 运行时求值、函数调用、管道、内置命令路径
- `make/make_bench_test.go`: 构建环境快照与构建命令生成

## 当前观察到的热点

以 `go test ./core ./make -run ^$ -bench . -benchtime=50ms` 的结果为参考：

- `BenchmarkEvalLoopProgram`: 循环执行分配明显偏高，说明解释器每轮求值路径仍有额外对象创建。
- `BenchmarkEvalPipelineProgram`: 管道实现会创建多个 goroutine 与 pipe，适合作为并发/IO 路径优化重点。
- `BenchmarkSnapshotBuildEnv`: 构建环境快照会复制整个进程环境并叠加脚本包变量，分配较多。
- `BenchmarkParseProgramControlFlow`: 复杂控制流解析的对象分配仍然偏多。

## 可行优化方向

### 1. 运行时环境与类型路径
- 让赋值直接更新命中的作用域，避免重复 fallback 写入。
- 为热点路径减少 `Object` 装箱与临时字符串转换。
- 为 `env` 包和普通变量查找建立更便宜的局部缓存。

### 2. 解析器
- 减少 AST 构造过程中的临时切片扩容。
- 对简单语句和常见表达式采用更轻量的分支路径。

### 3. 管道执行
- 评估是否可在纯 builtin 管道场景下降低 goroutine/pipe 创建成本。
- 对短管道做更紧凑的串行 fast path。

### 4. make 构建环境
- 避免每次快照都无差别复制全部 `os.Environ()`。
- 在目标未修改构建变量时复用快照结果。

## 使用方式

```bash
go test ./...
go test ./core ./make -run ^$ -bench .
```
