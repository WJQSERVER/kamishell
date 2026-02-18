# 语法设计提案

## 1. 变量与赋值
*   **Shell 风格**: `VAR=value`, 使用 `$VAR` 访问。
*   **Go 风格**: `var x = 10` 或 `x := 10`。
*   **提案**: 默认支持 `x := ...` 这种简洁的 Go 语法用于脚本逻辑，而 `export VAR=value` 用于环境变量。

## 2. 命令执行
*   直接输入命令即可执行，如 `ls -la`。
*   **捕获输出**: `output, err := ls -la` (支持多返回值，捕获输出和错误)。

## 3. 控制流
*   使用 Go 的语法替代 Bash 繁琐的 `if [ ... ]; then`。
*   **示例**:
    ```go
    if x == "test" {
        print "Match"
    }
    ```

## 4. 管道与重定向
*   保留 `|`, `>`, `>>`。
*   **错误检查**: 管道中的最后一个命令如果出错，可以通过多返回值获取。

## 5. 并发
*   `command &` (传统后台)
*   `go { ... }` (运行一个代码块作为 Goroutine)

## 6. 函数定义
*   支持 Go 风格的函数定义，方便重用逻辑。
*   **示例**:
    ```go
    func check_logs(pattern string) error {
        err := grep $pattern /var/log/syslog | tail -n 5
        return err
    }
    ```

## 7. 错误处理
*   **核心原则**: 采用 Go 风格的 `if err != nil` 清晰语义。
*   **示例**:
    ```go
    err := cp src dest
    if err != nil {
        print "Copy failed: $err"
        return err
    }
    ```
    或者在捕获输出时：
    ```go
    out, err := ls "dir"
    if err != nil {
        handleError(err)
    }

## 8. 强制执行关键字 (`exec`)
*   用于解决命令名与关键字冲突的问题。
*   **示例**:
    ```go
    exec "go run ."
    exec "print -p 9090"
    ```
