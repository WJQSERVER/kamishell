//go:build windows
// +build windows

package builtin

import (
	"os"
	"path/filepath"
	"strings"
)

// 常见的Windows可执行扩展名
var windowsExecExtensions = []string{
	".exe", ".com", ".bat", ".cmd",
	".ps1", ".vbs", ".js",
	".msi", ".msp",
}

// isExecutableWindows Windows特定的可执行文件检测
func isExecutable(path string) bool {
	// 检查扩展名
	ext := strings.ToLower(filepath.Ext(path))
	for _, execExt := range windowsExecExtensions {
		if ext == execExt {
			return true
		}
	}

	// 如果没有扩展名，尝试添加.exe检查
	if ext == "" {
		if _, err := os.Stat(path + ".exe"); err == nil {
			return true
		}
	}

	return false
}

// getPlatformSpecificExecExtensions 返回Windows可执行扩展名
func getPlatformSpecificExecExtensions() []string {
	return windowsExecExtensions
}

// hasExecExtension 检查路径是否有可执行扩展名
func hasExecExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, execExt := range windowsExecExtensions {
		if ext == execExt {
			return true
		}
	}
	return false
}
