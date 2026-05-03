//go:build windows

package builtin

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procCopyFileExW  = kernel32.NewProc("CopyFileExW")
)

func tryCopyOnWrite(src, dst string) (bool, error) {
	srcPtr, err := syscall.UTF16PtrFromString(src)
	if err != nil {
		return false, fmt.Errorf("src path: %w", err)
	}
	dstPtr, err := syscall.UTF16PtrFromString(dst)
	if err != nil {
		return false, fmt.Errorf("dst path: %w", err)
	}

	// CopyFileExW can use COW on NTFS/ReFS internally
	ret, _, callErr := procCopyFileExW.Call(
		uintptr(unsafe.Pointer(srcPtr)),
		uintptr(unsafe.Pointer(dstPtr)),
		0, 0, 0, 0,
	)
	if ret == 0 {
		return false, fmt.Errorf("CopyFileEx: %v", callErr)
	}

	info, err := os.Stat(dst)
	if err != nil {
		os.Remove(dst)
		return false, fmt.Errorf("stat copied file: %w", err)
	}
	if info.IsDir() {
		os.Remove(dst)
		return false, fmt.Errorf("CopyFileEx produced a directory")
	}

	return true, nil
}
