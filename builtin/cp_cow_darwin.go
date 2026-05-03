//go:build darwin

package builtin

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func tryCopyOnWrite(src, dst string) (bool, error) {
	err := unix.Clonefile(src, dst, 0)
	if err != nil {
		return false, fmt.Errorf("clonefile: %w", err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		os.Remove(dst)
		return false, fmt.Errorf("stat cloned file: %w", err)
	}
	if info.IsDir() {
		os.Remove(dst)
		return false, fmt.Errorf("clonefile produced a directory")
	}

	return true, nil
}
