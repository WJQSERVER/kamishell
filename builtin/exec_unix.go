//go:build !windows

package builtin

import (
	"os"
)

// isExecutableUnix 使用Unix权限位检测可执行文件
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	mode := info.Mode()
	return mode&0111 != 0
}

// getPlatformSpecificExecExtensions 返回Unix平台可执行扩展名
func getPlatformSpecificExecExtensions() []string {
	return []string{} // Unix没有特定的可执行扩展名
}
