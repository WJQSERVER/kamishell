# Kamishell GNU Coreutils 对齐路线图

## 概述

本文档记录了 Kamishell 内置命令与 GNU Coreutils 实现的对齐状态和改进计划。

**对齐性总览:**
```
总体对齐性: ████████░░░░░░░░░░░░ 约 40%
核心功能对齐性: ██████████░░░░░░░░░░ 约 50%
完整功能对齐性: ████████░░░░░░░░░░░░ 约 35%
```

---

## 优先级定义

| 优先级 | 含义 | 时间规划 |
|--------|------|----------|
| 🔴 P0 | 核心功能，建议优先实现 | 1-2周 |
| 🟡 P1 | 增强功能，建议中期实现 | 1个月 |
| 🟢 P2 | 高级功能，可长期规划 | 2-3个月 |
| ⚪ P3 | 特殊功能，按需实现 | 待定 |

---

## 命令对齐详细计划

### 1. grep ⭐⭐⭐ (feature/grep-regex-support分支)

**当前完成度:** ~35%

#### 已完成功能 ✅
- [x] 正则表达式支持 (RE2语法)
- [x] `-i, --ignore-case` - 忽略大小写
- [x] `-n, --line-number` - 显示行号
- [x] `-v, --invert-match` - 反向匹配
- [x] `-w, --word-regexp` - 单词边界匹配
- [x] `-x, --line-regexp` - 整行匹配
- [x] `-c, --count` - 匹配计数
- [x] `-l, --files-with-matches` - 显示匹配文件名
- [x] `-L, --files-without-match` - 显示不匹配文件名
- [x] `-q, --quiet` - 静默模式
- [x] `-r, -R, --recursive` - 递归搜索目录

#### 🟡 P1 - 增强功能（建议下一步实现）
- [ ] `-A, --after-context=NUM` - 显示后N行
- [ ] `-B, --before-context=NUM` - 显示前N行
- [ ] `-C, --context=NUM` - 显示上下文N行
- [ ] `--include=PATTERN` - 递归时包含文件
- [ ] `--exclude=PATTERN` - 递归时排除文件
- [ ] `--exclude-dir=PATTERN` - 递归时排除目录
- [ ] `-e PATTERN` - 多模式指定
- [ ] `-f FILE` - 从文件读取模式
- [ ] `-E, --extended-regexp` - 扩展正则表达式
- [ ] `-F, --fixed-strings` - 固定字符串匹配
- [ ] `-m NUM, --max-count=NUM` - 最大匹配数

#### 🟢 P2 - 高级功能
- [ ] `--color[=WHEN]` - 彩色输出
- [ ] `-P, --perl-regexp` - Perl兼容正则表达式
- [ ] `-s, --no-messages` - 抑制错误消息
- [ ] `-h, --no-filename` - 不显示文件名
- [ ] `-H, --with-filename` - 总是显示文件名
- [ ] `-o, --only-matching` - 只显示匹配部分
- [ ] `-b, --byte-offset` - 显示字节偏移量

---

### 2. cat ⭐⭐⭐

**当前完成度:** ~70%

#### 已完成功能 ✅
- [x] 基础文件连接输出
- [x] 从stdin读取
- [x] `-u` (POSIX兼容性)
- [x] `-n, --number` - 显示行号
- [x] `-b, --number-nonblank` - 非空行显示行号
- [x] `-s, --squeeze-blank` - 压缩连续空行
- [x] `-E, --show-ends` - 显示行尾$
- [x] `-T, --show-tabs` - 显示制表符为^I
- [x] `-v, --show-nonprinting` - 显示非打印字符
- [x] `-A, --show-all` - 等价于-vET
- [x] `-e` - 等价于-vE
- [x] `-t` - 等价于-vT

#### 🟡 P1 - 增强功能
- [ ] `--help` - 显示帮助信息
- [ ] `--version` - 显示版本信息

---

### 3. touch ⭐⭐⭐

**当前完成度:** ~80%

#### 已完成功能 ✅
- [x] 创建空文件
- [x] 更新文件时间戳（访问/修改）
- [x] `-a, --time=atime/access/use` - 仅更改访问时间戳
- [x] `-m, --time=mtime/modify` - 仅更改修改时间戳
- [x] `-c, --no-create` - 不创建不存在的文件
- [x] `-d, --date=TIME` - 使用指定时间（支持多种格式）
- [x] `-r, --reference=FILE` - 使用参考文件的时间戳
- [x] `-t [[cc]yy]mmddhhmm[.ss]` - 使用指定时间戳

#### 🟡 P1 - 增强功能
- [ ] `-h, --no-dereference` - 更改符号链接本身（而非目标）
- [ ] `-f` - 忽略（BSD兼容性）

---

### 4. cp ⭐⭐⭐

**当前完成度:** ~35%

#### 已完成功能 ✅
- [x] `-r, -R, --recursive` - 递归复制
- [x] `-f, --force` - 强制覆盖
- [x] `-i, --interactive` - 交互式覆盖确认
- [x] `-p, --preserve` - 保留模式和时间戳
- [x] `-n, --no-clobber` - 不覆盖已存在文件
- [x] `-u, --update` - 仅在源文件较新时复制
- [x] `-v, --verbose` - 显示复制进度
- [x] 所有长选项支持

#### 🟡 P1 - 增强功能（建议下一步实现）
- [ ] `-d` - 复制符号链接为符号链接
- [ ] `-L, --dereference` - 跟随所有符号链接
- [ ] `-P, --no-dereference` - 不跟随符号链接
- [ ] `--backup[=CONTROL]` - 创建备份
- [ ] `-b` - 创建备份（简写）
- [ ] `--suffix=SUFFIX` - 指定备份后缀
- [ ] `-a, --archive` - 归档模式
- [ ] `--preserve[=ATTR_LIST]` - 精确控制保留的属性
- [ ] `-l, --link` - 创建硬链接而非复制
- [ ] `-s, --symbolic-link` - 创建符号链接而非复制
- [ ] `-t, --target-directory` - 指定目标目录
- [ ] `-T, --no-target-directory` - 将目标视为普通文件

#### 🟢 P2 - 高级功能
- [ ] `-x, --one-file-system` - 停留在同一文件系统
- [ ] `--sparse=WHEN` - 稀疏文件控制
- [ ] `--parents` - 保持源文件目录结构
- [ ] `--reflink[=WHEN]` - 使用写时复制克隆
- [ ] `--remove-destination` - 打开目标前删除

---

### 5. mv ⭐⭐⭐

**当前完成度:** ~40%

#### 已完成功能 ✅
- [x] `-f, --force` - 覆盖前不提示
- [x] `-i, --interactive` - 覆盖前提示确认
- [x] `-n, --no-clobber` - 不覆盖已存在文件
- [x] `-v, --verbose` - 显示移动进度
- [x] 所有长选项支持

#### 🟡 P1 - 增强功能（建议下一步实现）
- [ ] `-b, --backup` - 创建备份
- [ ] `--backup[=CONTROL]` - 备份控制
- [ ] `-S, --suffix=SUFFIX` - 指定备份后缀
- [ ] `-t, --target-directory` - 指定目标目录
- [ ] `-T, --no-target-directory` - 将目标视为普通文件

#### 🟢 P2 - 高级功能
- [ ] `--strip-trailing-slashes` - 删除尾部斜杠
- [ ] `-Z, --context` - 设置SELinux安全上下文

---

### 6. rm ⭐⭐

**当前完成度:** ~40%

#### 已完成功能 ✅
- [x] `-f, --force` - 强制删除
- [x] `-i, --interactive` - 交互式删除
- [x] `-r, -R, --recursive` - 递归删除
- [x] `-v, --verbose` - 详细输出
- [x] `--no-preserve-root` - 允许删除根目录
- [x] `--preserve-root` - 保护根目录（默认）

#### 🟡 P1 - 增强功能（建议下一步实现）
- [ ] `-I` - 一次提示（递归时每个目录提示一次）
- [ ] `--one-file-system` - 跨越文件系统时停止
- [ ] `--no-preserve-root` 更完善的行为控制

#### 🟢 P2 - 高级功能
- [ ] `-d, --dir` - 删除空目录
- [ ] `--help` - 显示帮助
- [ ] `--version` - 显示版本

---

### 7. ls ⭐⭐

**当前完成度:** ~15%

#### 已完成功能 ✅
- [x] `-a` - 显示隐藏文件
- [x] `-l` - 长列表格式（部分实现）
- [x] `-h` - 人类可读大小
- [x] `-F, --classify` - 追加类型指示符
- [x] `-R, --recursive` - 递归列出
- [x] `-r, --reverse` - 逆序排序
- [x] `-t` - 按修改时间排序
- [x] `-S` - 按大小排序
- [x] `-d` - 列出目录本身

#### 🔴 P0 - 高优先级（建议下一步实现）
- [ ] `-A, --almost-all` - 列出隐藏文件但不包括.和..
- [ ] `-L, --dereference` - 跟随符号链接
- [ ] `-H, --dereference-command-line` - 仅跟随命令行中的符号链接
- [ ] `-i, --inode` - 显示inode号
- [ ] 完善`-l`输出（硬链接数、所有者、组）

#### 🟡 P1 - 增强功能
- [ ] `-p, --indicator-style=slash` - 目录后追加/
- [ ] `-s, --size` - 显示块数
- [ ] `-c` - 显示ctime
- [ ] `-u` - 显示atime
- [ ] `--time=WORD` - 选择时间类型
- [ ] `--time-style=STYLE` - 时间格式控制
- [ ] `-v, --sort=version` - 版本号排序
- [ ] `-X, --sort=extension` - 按扩展名排序
- [ ] `--group-directories-first` - 目录排在文件前

#### 🟢 P2 - 高级功能
- [ ] `--color[=WHEN]` - 彩色输出
- [ ] `-C` - 多列输出
- [ ] `-1` - 每行一个文件
- [ ] `-m` - 逗号分隔
- [ ] `-x` - 横向排序
- [ ] `-g, -o, -G` - 简化长格式选项
- [ ] `-n, --numeric-uid-gid` - 数字UID/GID
- [ ] `-b, --escape` - C风格转义
- [ ] `-q, --hide-control-chars` - 隐藏控制字符

---

### 8. sed ⭐⭐

**当前完成度:** ~5%

#### 已完成功能 ✅
- [x] 基础替换 `s/old/new/`
- [x] 从文件/stdin读取
- [x] 输出到stdout

#### 🔴 P0 - 高优先级（建议下一步实现）
- [ ] `s/old/new/g` - 全局替换标志
- [ ] `s/old/new/N` - 替换第N个匹配
- [ ] `s/old/new/i` - 忽略大小写
- [ ] `d` 命令 - 删除行
- [ ] `p` 命令 - 打印行
- [ ] `-n` - 安静模式（只打印指定行）
- [ ] 行号寻址（如 `5s/old/new/`）
- [ ] 正则寻址（如 `/foo/s/old/new/`）
- [ ] 范围寻址（如 `1,10d`）

#### 🟡 P1 - 增强功能
- [ ] `a\` - 追加文本
- [ ] `i\` - 插入文本
- [ ] `c\` - 更改行
- [ ] `y/source/dest/` - 字符转换
- [ ] `=` - 打印行号
- [ ] `q` - 退出
- [ ] `w file` - 写入文件
- [ ] `-e script` - 多表达式
- [ ] `-f script-file` - 从文件读取脚本

#### 🟢 P2 - 高级功能
- [ ] `h/H/g/G/x` - 保持空间操作
- [ ] `n/N/P/D` - 多行操作
- [ ] `-i[SUFFIX]` - 原地编辑
- [ ] `-r, -E` - 扩展正则表达式
- [ ] `-s` - 将文件视为独立流
- [ ] 高级正则元字符（\w, \W, \b, \B等）

---

### 9. mkdir ⭐

**当前完成度:** ~70%

#### 已完成功能 ✅
- [x] `-p, --parents` - 按需创建父目录
- [x] `-m, --mode=MODE` - 设置权限模式
- [x] 基础目录创建

#### 🟡 P1 - 增强功能（建议下一步实现）
- [ ] `-v, --verbose` - 打印创建消息

#### 🟢 P2 - 高级功能
- [ ] `-Z, --context[=CTX]` - 设置SELinux安全上下文

---

### 10. pwd ⭐

**当前完成度:** ~90%

#### 已完成功能 ✅
- [x] `-L, --logical` - 使用PWD环境变量
- [x] `-P, --physical` - 解析物理路径
- [x] POSIX行为（-L和-P同时指定时后者生效）
- [x] PWD环境变量验证

#### 🟢 P2 - 建议下一步实现
- [ ] `--help` - 显示帮助
- [ ] `--version` - 显示版本
- [ ] 注意：当前默认-L，GNU默认-P（除非POSIXLY_CORRECT）

---

## 开发优先级建议

### P0 高优先级（建议近期完成）
1. **ls** - 实现 `-A, -i`, 完善 `-l` 输出
2. **sed** - 实现 `s/g` 全局替换、`-n` 安静模式、行号寻址
3. **mkdir** - 实现 `-v` 详细输出
4. **cp** - 实现符号链接处理 (`-d`, `-L`, `-P`)

### P1 中优先级（建议1个月内完成）
1. **grep** - 实现上下文显示 `-A, -B, -C`
2. **cp** - 实现备份功能 `-b, --backup`
3. **mv** - 实现备份功能 `-b, --backup`
4. **ls** - 实现时间戳相关选项 (`-c`, `-u`, `--time-style`)
5. **sed** - 实现 `d/p` 命令、`-e` 多表达式

### P2 低优先级（建议2-3个月内完成）
1. **grep** - 实现彩色输出 `--color`
2. **sed** - 实现完整脚本支持、空间操作
3. **ls** - 实现彩色输出 `--color`
4. **cp** - 实现归档模式 `-a`
5. **mv** - 实现完整备份系统

---

## 技术实现注意事项

### 正则表达式实现
- Go的`regexp`包使用RE2语法，与GNU grep的PCRE有差异
- 考虑是否需要引入`github.com/dlclark/regexp2`支持更多PCRE特性
- `-P`（Perl正则）实现难度较高，建议P2阶段考虑

### 符号链接处理
- 使用`os.Lstat` vs `os.Stat`的区别
- Windows平台符号链接支持有限
- 需要考虑循环链接检测

### 文件时间戳
- Go的`os.Chtimes`支持atime和mtime
- ctime（状态变更时间）在Go中不可直接设置
- 高精度时间戳需要平台特定实现

### 彩色输出
- 建议使用`github.com/fatih/color`或ANSI转义码
- 需要检测TTY（`isatty`）
- 支持`NO_COLOR`环境变量和`--color`选项

### 退出码规范
- 确保与GNU工具一致的退出码行为
- 0 = 成功/找到匹配
- 1 = 无匹配/无错误
- 2 = 错误（语法错误、文件不存在等）

---

## 贡献指南

### 开发流程
1. 从本路线图选择功能
2. 创建功能分支：`feature/<command>-<feature>`
3. 编写代码和测试
4. 确保测试覆盖率
5. 提交PR并关联本路线图

### 测试要求
- 每个新选项/功能需要对应测试用例
- 需要测试边界条件（空输入、错误模式等）
- 多平台测试（Linux/Windows/macOS）

### 文档要求
- 更新命令的`Help`文本
- 更新本文档对应功能状态
- 更新主README.md（如适用）

---

## 附录

### GNU Coreutils 参考版本
本文档基于 GNU Coreutils 9.x 版本的命令行接口。

### 相关文档
- [GNU Coreutils Manual](https://www.gnu.org/software/coreutils/manual/)
- [POSIX.1-2017 Shell and Utilities](https://pubs.opengroup.org/onlinepubs/9699919799/)

### 最后更新
2025年4月11日

---

### 版本历史

#### v1.0 (2025-04-11)
- ✅ 完成 grep 正则表达式支持和递归搜索
- ✅ 完成 cat 行号和特殊字符显示选项
- ✅ 完成 touch 时间戳控制选项
- ✅ 完成 cp -n, -u, -v 选项
- ✅ 完成 mv -n, -v 选项
- 📊 总体对齐性从 15-20% 提升至 40%

---

**注:** 本路线图是动态文档，随着功能实现持续更新。完成的功能请标记为✅并注明版本号。
