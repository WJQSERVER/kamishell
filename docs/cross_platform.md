# Kamishell 跨平台兼容性说明

## 概述

Kamishell 使用 Go 语言实现，天生具有良好的跨平台特性。本文档说明各命令在不同平台上的兼容性状态。

---

## 兼容性状态

### ✅ 完全兼容 (所有平台)

| 命令 | Linux | macOS | Windows | 说明 |
|------|-------|-------|---------|------|
| grep | ✅ | ✅ | ✅ | 纯 Go 正则表达式 |
| cat | ✅ | ✅ | ✅ | 纯 I/O 操作 |
| rm | ✅ | ✅ | ✅ | 根目录检测已适配 |

### ⚠️ 有限支持 (平台特性差异)

| 命令 | Linux | macOS | Windows | 限制说明 |
|------|-------|-------|---------|----------|
| touch | ✅ | ✅ | ⚠️ | -a/-m 选项无法真正单独修改 |
| cp | ✅ | ✅ | ⚠️ | 权限保留在 Windows 上有限 |
| mv | ✅ | ✅ | ⚠️ | 权限保留在 Windows 上有限 |
| ls | ✅ | ✅ | ⚠️ | 权限显示、可执行检测差异 |
| mkdir | ✅ | ✅ | ⚠️ | -m 权限模式在 Windows 忽略 |

---

## 平台特定实现

### 可执行文件检测

使用条件编译实现平台特定的检测：

- **Unix (Linux/macOS)**: `exec_unix.go`
  - 使用文件权限位检测 (mode & 0111)
  - 无特定扩展名

- **Windows**: `exec_windows.go`
  - 使用文件扩展名检测 (.exe, .com, .bat, .cmd, .ps1, etc.)
  - 自动适应 Windows 习惯

### 符号链接处理

统一使用 `os.Lstat` 进行文件检查：
- 不跟随符号链接
- 源文件和目标文件处理一致
- 在 Windows 需要管理员权限创建符号链接

---

## Windows 特定说明

### 权限系统差异

Windows 使用 ACL (Access Control List) 而非 Unix 权限模式：

| Unix 权限 | Windows 行为 |
|-----------|--------------|
| 0755 (rwxr-xr-x) | 可写文件 |
| 0644 (rw-r--r--) | 可写文件 |
| 0444 (r--r--r--) | 只读文件 |

**注意**: `os.Chmod` 在 Windows 上仅支持设置只读属性。

### 时间戳修改

- Windows 不支持单独修改 atime (访问时间)
- `os.Chtimes` 始终同时修改 atime 和 mtime
- touch 的 `-a` 和 `-m` 选项行为受限

### 符号链接

- Windows 创建符号链接需要管理员权限
- 软链接和硬链接支持有限
- 建议使用 cp -r 代替符号链接复制

---

## 最佳实践

### 编写跨平台脚本

```kami
# 避免使用平台特定的权限设置
mkdir mydir              # ✅ 推荐
mkdir -m 0755 mydir      # ⚠️ Windows 忽略权限

# 文件复制
 cp file dest/          # ✅ 推荐
 cp -p file dest/       # ⚠️ Windows 权限保留有限

# 可执行检测（自动适配平台）
ls -F                   # ✅ 自动使用平台特定检测
```

### 避免的操作

| 避免 | 原因 | 替代方案 |
|------|------|----------|
| `cp -p` 依赖精确权限 | Windows ACL 不同 | 使用 `cp` 后手动设置 |
| `touch -a` 单独修改访问时间 | Windows 不支持 | 使用 `touch` 不带选项 |
| `mkdir -m 755` | Windows 忽略权限 | 创建后使用 `chmod`（Unix） |

---

## 测试建议

在不同平台上运行测试：

```bash
# Linux/macOS
go test ./...

# Windows
go test ./...  # 需要 Windows 环境
```

---

## 版本历史

### v1.0 (2025-04-11)
- 统一 cp/mv 符号链接处理
- 添加 Windows 可执行文件检测
- 更新跨平台文档

---

**注**: 本文档随版本更新，建议查看最新版本。
